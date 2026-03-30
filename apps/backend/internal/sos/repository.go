package sos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	CreateSession(ctx context.Context, session *SOSSession) error
	GetSessionByID(ctx context.Context, sessionID string) (*SOSSession, error)
	GetActiveSessionByUserID(ctx context.Context, userID string) (*SOSSession, error)
	UpdateSession(ctx context.Context, session *SOSSession) error
	CreateLocationPing(ctx context.Context, sessionID string, latitude, longitude float64, recordedAt time.Time) error
	CreateViewerGrant(ctx context.Context, grant *SOSViewerGrant) error
	GetActiveViewerGrantBySessionContact(ctx context.Context, sessionID, trustedContactID string, now time.Time) (*SOSViewerGrant, error)
	GetViewerGrantByToken(ctx context.Context, tokenHash string) (*SOSViewerGrant, error)
	IsTrustedContactOwnedByUser(ctx context.Context, userID, trustedContactID string) (bool, error)
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

func (r *GormRepository) CreateViewerGrant(ctx context.Context, grant *SOSViewerGrant) error {
	if err := r.db.WithContext(ctx).Create(grant).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrViewerGrantConflict
		}

		return err
	}

	return nil
}

func (r *GormRepository) GetActiveViewerGrantBySessionContact(ctx context.Context, sessionID, trustedContactID string, now time.Time) (*SOSViewerGrant, error) {
	var grant SOSViewerGrant
	if err := r.db.WithContext(ctx).
		Where("session_id = ? AND trusted_contact_id = ? AND revoked_at IS NULL AND expires_at > ?", sessionID, trustedContactID, now).
		Order("created_at DESC").
		First(&grant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrViewerGrantNotFound
		}

		return nil, err
	}

	return &grant, nil
}

func (r *GormRepository) GetViewerGrantByToken(ctx context.Context, tokenHash string) (*SOSViewerGrant, error) {
	var grant SOSViewerGrant
	if err := r.db.WithContext(ctx).
		Select("id", "session_id", "user_id", "trusted_contact_id", "token_hash", "revoked_at", "expires_at", "created_at").
		First(&grant, "token_hash = ?", tokenHash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrViewerGrantNotFound
		}

		return nil, err
	}

	return &grant, nil
}

func (r *GormRepository) IsTrustedContactOwnedByUser(ctx context.Context, userID, trustedContactID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Table("trusted_contacts").
		Where("id = ? AND user_id = ?", trustedContactID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
