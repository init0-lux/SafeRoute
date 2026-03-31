package reports

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, input CreateParams) (*StoredReport, error)
	GetByID(ctx context.Context, id string) (*StoredReport, error)
	ListEvidenceIDs(ctx context.Context, reportID string) ([]string, error)
	ListNearby(ctx context.Context, input NearbyParams) ([]NearbyReportRow, error)
	CountNearby(ctx context.Context, input NearbyParams) (int64, error)
}

type CreateParams struct {
	UserID      *string
	Category    string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  time.Time
	Source      string
}

type GormRepository struct {
	db *gorm.DB
}

type StoredReport struct {
	ID          string
	UserID      *string
	Category    string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  time.Time
	CreatedAt   time.Time
	Source      string
	TrustScore  float64
}

type NearbyParams struct {
	Latitude  float64
	Longitude float64
	Radius    float64
	Limit     int
	Offset    int
}

type NearbyReportRow struct {
	StoredReport
	DistanceMeters float64
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, input CreateParams) (*StoredReport, error) {
	query := `
		WITH inserted AS (
			INSERT INTO reports (
				user_id,
				category,
				description,
				location,
				occurred_at,
				source
			)
			VALUES (
				?,
				?,
				?,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
				?,
				?
			)
			RETURNING
				id,
				user_id,
				category,
				description,
				location,
				occurred_at,
				created_at,
				source
		)
		SELECT
			i.id,
			i.user_id,
			i.category,
			i.description,
			ST_Y(i.location::geometry) AS latitude,
			ST_X(i.location::geometry) AS longitude,
			i.occurred_at,
			i.created_at,
			i.source,
			COALESCE(u.trust_score, 0.3) AS trust_score
		FROM inserted i
		LEFT JOIN users u ON u.id = i.user_id
	`

	var report StoredReport
	result := r.db.WithContext(ctx).Raw(
		query,
		input.UserID,
		input.Category,
		input.Description,
		input.Longitude,
		input.Latitude,
		input.OccurredAt,
		input.Source,
	).Scan(&report)
	if result.Error != nil {
		return nil, result.Error
	}

	if input.UserID != nil && strings.TrimSpace(*input.UserID) != "" {
		report.UserID = input.UserID
	}

	return &report, nil
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*StoredReport, error) {
	query := `
		SELECT
			r.id,
			r.user_id,
			r.category,
			r.description,
			ST_Y(r.location::geometry) AS latitude,
			ST_X(r.location::geometry) AS longitude,
			r.occurred_at,
			r.created_at,
			r.source,
			COALESCE(u.trust_score, 0.3) AS trust_score
		FROM reports r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.id = ?
		LIMIT 1
	`

	var report StoredReport
	result := r.db.WithContext(ctx).Raw(query, id).Scan(&report)
	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, ErrReportNotFound
	}

	return &report, nil
}

func (r *GormRepository) ListEvidenceIDs(ctx context.Context, reportID string) ([]string, error) {
	type evidenceIDRow struct {
		ID string
	}

	var rows []evidenceIDRow
	if err := r.db.WithContext(ctx).
		Raw(`SELECT id FROM evidence WHERE report_id = ? ORDER BY created_at DESC`, reportID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids, nil
}

func (r *GormRepository) ListNearby(ctx context.Context, input NearbyParams) ([]NearbyReportRow, error) {
	query := `
		SELECT
			r.id,
			r.user_id,
			r.category,
			r.description,
			ST_Y(r.location::geometry) AS latitude,
			ST_X(r.location::geometry) AS longitude,
			r.occurred_at,
			r.created_at,
			r.source,
			ST_Distance(
				r.location,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
			) AS distance_meters,
			COALESCE(u.trust_score, 0.3) AS trust_score
		FROM reports r
		LEFT JOIN users u ON u.id = r.user_id
		WHERE ST_DWithin(
			r.location,
			ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
			?
		)
		ORDER BY COALESCE(u.trust_score, 0.3) DESC, r.created_at DESC
		LIMIT ? OFFSET ?
	`

	var rows []NearbyReportRow
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.Longitude,
		input.Latitude,
		input.Longitude,
		input.Latitude,
		input.Radius,
		input.Limit,
		input.Offset,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *GormRepository) CountNearby(ctx context.Context, input NearbyParams) (int64, error) {
	type countRow struct {
		Count int64
	}

	query := `
		SELECT COUNT(*) AS count
		FROM reports r
		WHERE ST_DWithin(
			r.location,
			ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
			?
		)
	`

	var row countRow
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.Longitude,
		input.Latitude,
		input.Radius,
	).Scan(&row).Error; err != nil {
		return 0, err
	}

	return row.Count, nil
}
