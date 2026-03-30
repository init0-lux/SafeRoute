package safety

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	GetAggregates(ctx context.Context, input AggregateParams) (*Aggregates, error)
}

type AggregateParams struct {
	Latitude    float64
	Longitude   float64
	Radius      float64
	RecentSince time.Time
}

type Aggregates struct {
	RecentReports         int64
	HistoricalReports     int64
	RecentTrustWeight     float64
	HistoricalTrustWeight float64
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetAggregates(ctx context.Context, input AggregateParams) (*Aggregates, error) {
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
