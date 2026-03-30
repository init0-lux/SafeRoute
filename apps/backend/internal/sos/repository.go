package sos

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	CreateSession(ctx context.Context, session *SOSSession) error
	GetSessionByID(ctx context.Context, sessionID string) (*SOSSession, error)
	GetActiveSessionByUserID(ctx context.Context, userID string) (*SOSSession, error)
	UpdateSession(ctx context.Context, session *SOSSession) error
	CreateLocationPing(ctx context.Context, sessionID string, latitude, longitude float64, recordedAt time.Time) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) CreateSession(ctx context.Context, session *SOSSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *GormRepository) GetSessionByID(ctx context.Context, sessionID string) (*SOSSession, error) {
	var session SOSSession
	if err := r.db.WithContext(ctx).First(&session, "id = ?", sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	}

	return &session, nil
}

func (r *GormRepository) GetActiveSessionByUserID(ctx context.Context, userID string) (*SOSSession, error) {
	var session SOSSession
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, SessionStatusActive).
		Order("started_at DESC").
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	}

	return &session, nil
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *SOSSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *GormRepository) CreateLocationPing(ctx context.Context, sessionID string, latitude, longitude float64, recordedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&LocationPing{}).Create(map[string]interface{}{
		"session_id":  sessionID,
		"location":    gorm.Expr("ST_SetSRID(ST_MakePoint(?, ?), 4326)", longitude, latitude),
		"recorded_at": recordedAt,
	}).Error
}
