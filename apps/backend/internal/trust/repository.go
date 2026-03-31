package trust

import (
	"context"
	"errors"
	"time"

	"saferoute-backend/internal/auth"

	"gorm.io/gorm"
)

type Repository interface {
	GetByUserID(ctx context.Context, userID string) (*auth.User, error)
	IncrementReportCount(ctx context.Context, userID string) error
	SetVerificationStatus(ctx context.Context, userID string, verified bool, verifiedAt *time.Time) error
	UpdateTrustScore(ctx context.Context, userID string, score float64) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) GetByUserID(ctx context.Context, userID string) (*auth.User, error) {
	var user auth.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, auth.ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (r *GormRepository) IncrementReportCount(ctx context.Context, userID string) error {
	result := r.db.WithContext(ctx).
		Model(&auth.User{}).
		Where("id = ?", userID).
		UpdateColumn("report_count", gorm.Expr("report_count + 1"))
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return auth.ErrUserNotFound
	}

	return nil
}

func (r *GormRepository) SetVerificationStatus(ctx context.Context, userID string, verified bool, verifiedAt *time.Time) error {
	updates := map[string]any{
		"verified":    verified,
		"verified_at": verifiedAt,
	}

	result := r.db.WithContext(ctx).
		Model(&auth.User{}).
		Where("id = ?", userID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return auth.ErrUserNotFound
	}

	return nil
}

func (r *GormRepository) UpdateTrustScore(ctx context.Context, userID string, score float64) error {
	result := r.db.WithContext(ctx).
		Model(&auth.User{}).
		Where("id = ?", userID).
		Update("trust_score", score)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return auth.ErrUserNotFound
	}

	return nil
}
