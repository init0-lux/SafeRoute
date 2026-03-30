package reports

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, input CreateParams) (*Report, error)
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

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, input CreateParams) (*Report, error) {
	query := `
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
			ST_SetSRID(ST_MakePoint(?, ?), 4326),
			?,
			?
		)
		RETURNING
			id,
			user_id,
			category,
			description,
			occurred_at,
			created_at,
			source
	`

	var report Report
	if err := r.db.WithContext(ctx).Raw(
		query,
		input.UserID,
		input.Category,
		input.Description,
		input.Longitude,
		input.Latitude,
		input.OccurredAt,
		input.Source,
	).Scan(&report).Error; err != nil {
		return nil, err
	}

	if input.UserID != nil && strings.TrimSpace(*input.UserID) != "" {
		report.UserID = input.UserID
	}

	return &report, nil
}
