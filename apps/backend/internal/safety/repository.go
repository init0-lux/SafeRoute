package safety

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetPointAggregates(ctx context.Context, input PointAggregateParams) (*Aggregates, error)
	GetRouteSignals(ctx context.Context, input RouteSignalParams) ([]RouteSignal, error)
}

type PointAggregateParams struct {
	Latitude    float64
	Longitude   float64
	Radius      float64
	RecentSince time.Time
}

type RouteSignalParams struct {
	LineStringWKT  string
	CorridorRadius float64
	RecentSince    time.Time
}

type Aggregates struct {
	RecentReports         int64
	HistoricalReports     int64
	RecentTrustWeight     float64
	HistoricalTrustWeight float64
}

type RouteSignal struct {
	Fraction    float64
	IsRecent    bool
	TrustWeight float64
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetPointAggregates(ctx context.Context, input PointAggregateParams) (*Aggregates, error) {
	query := `
		SELECT
			COUNT(*) FILTER (
				WHERE r.created_at >= ?
			) AS recent_reports,
			COUNT(*) AS historical_reports,
			COALESCE(SUM(
				CASE
					WHEN r.created_at >= ? THEN COALESCE(u.trust_score, 0.3)
					ELSE 0
				END
			), 0) AS recent_trust_weight,
			COALESCE(SUM(COALESCE(u.trust_score, 0.3)), 0) AS historical_trust_weight
		FROM reports r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE ST_DWithin(
			r.location,
			ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
			?
		)
	`

	var aggregates Aggregates
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.RecentSince,
		input.RecentSince,
		input.Longitude,
		input.Latitude,
		input.Radius,
	).Scan(&aggregates).Error; err != nil {
		return nil, err
	}

	return &aggregates, nil
}

func (r *GormRepository) GetRouteSignals(ctx context.Context, input RouteSignalParams) ([]RouteSignal, error) {
	query := `
		WITH route_line AS (
			SELECT
				ST_GeomFromText(?, 4326) AS geom,
				ST_GeomFromText(?, 4326)::geography AS geog
		)
		SELECT
			LEAST(1.0, GREATEST(0.0, ST_LineLocatePoint(rl.geom, r.location::geometry))) AS fraction,
			CASE WHEN r.created_at >= ? THEN TRUE ELSE FALSE END AS is_recent,
			COALESCE(u.trust_score, 0.3) AS trust_weight
		FROM reports r
		CROSS JOIN route_line rl
		LEFT JOIN users u ON u.id = r.user_id
		WHERE ST_DWithin(
			r.location,
			rl.geog,
			?
		)
		ORDER BY fraction ASC, r.created_at DESC
	`

	var signals []RouteSignal
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.LineStringWKT,
		input.LineStringWKT,
		input.RecentSince,
		input.CorridorRadius,
	).Scan(&signals).Error; err != nil {
		return nil, err
	}

	return signals, nil
}
