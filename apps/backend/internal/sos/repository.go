package sos

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	ExistsSession(ctx context.Context, id string) (bool, error)
	CreateSession(ctx context.Context, session *SOSSession) error
	GetSessionByID(ctx context.Context, sessionID string) (*SOSSession, error)
	GetActiveSessionByUserID(ctx context.Context, userID string) (*SOSSession, error)
	UpdateSession(ctx context.Context, session *SOSSession) error
	CreateLocationPing(ctx context.Context, sessionID string, latitude, longitude float64, recordedAt time.Time) error
	GetLatestLocationPing(ctx context.Context, sessionID string) (*LocationSnapshot, error)
	CreateViewerGrant(ctx context.Context, grant *SOSViewerGrant) error
	RevokeActiveViewerGrantBySessionContact(ctx context.Context, sessionID, trustedContactID string, revokedAt time.Time) error
	GetActiveViewerGrantBySessionContact(ctx context.Context, sessionID, trustedContactID string, now time.Time) (*SOSViewerGrant, error)
	GetViewerGrantByToken(ctx context.Context, tokenHash string) (*SOSViewerGrant, error)
	IsTrustedContactOwnedByUser(ctx context.Context, userID, trustedContactID string) (bool, error)
	ListActiveSessionAlertsByViewerPhone(ctx context.Context, viewerPhone string) ([]ActiveSessionAlert, error)
}

type ActiveSessionAlert struct {
	SessionID        string     `gorm:"column:session_id"`
	UserID           string     `gorm:"column:user_id"`
	TrustedContactID string     `gorm:"column:trusted_contact_id"`
	ReporterName     string     `gorm:"column:reporter_name"`
	ReporterPhone    string     `gorm:"column:reporter_phone"`
	StartedAt        time.Time  `gorm:"column:started_at"`
	Latitude         *float64   `gorm:"column:lat"`
	Longitude        *float64   `gorm:"column:lng"`
	RecordedAt       *time.Time `gorm:"column:recorded_at"`
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) ExistsSession(ctx context.Context, id string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&SOSSession{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
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

func (r *GormRepository) GetLatestLocationPing(ctx context.Context, sessionID string) (*LocationSnapshot, error) {
	type latestLocationRow struct {
		Latitude   float64   `gorm:"column:lat"`
		Longitude  float64   `gorm:"column:lng"`
		RecordedAt time.Time `gorm:"column:recorded_at"`
	}

	var row latestLocationRow
	err := r.db.WithContext(ctx).
		Raw(
			`SELECT
				ST_Y(location) AS lat,
				ST_X(location) AS lng,
				recorded_at
			FROM location_pings
			WHERE session_id = ?
			ORDER BY recorded_at DESC
			LIMIT 1`,
			sessionID,
		).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.RecordedAt.IsZero() {
		return nil, nil
	}

	return &LocationSnapshot{
		Latitude:   row.Latitude,
		Longitude:  row.Longitude,
		RecordedAt: row.RecordedAt.UTC(),
	}, nil
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

func (r *GormRepository) RevokeActiveViewerGrantBySessionContact(ctx context.Context, sessionID, trustedContactID string, revokedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&SOSViewerGrant{}).
		Where("session_id = ? AND trusted_contact_id = ? AND revoked_at IS NULL AND expires_at > ?", sessionID, trustedContactID, revokedAt).
		Update("revoked_at", revokedAt.UTC()).
		Error
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

func (r *GormRepository) ListActiveSessionAlertsByViewerPhone(ctx context.Context, viewerPhone string) ([]ActiveSessionAlert, error) {
	var alerts []ActiveSessionAlert

	err := r.db.WithContext(ctx).
		Raw(
			`SELECT
				s.id AS session_id,
				s.user_id AS user_id,
				tc.id AS trusted_contact_id,
				COALESCE(reporter.username, reporter.phone, '') AS reporter_name,
				COALESCE(reporter.phone, '') AS reporter_phone,
				s.started_at AS started_at,
				latest.lat AS lat,
				latest.lng AS lng,
				latest.recorded_at AS recorded_at
			FROM sos_sessions s
			JOIN trusted_contacts tc
				ON tc.user_id = s.user_id
				AND tc.phone = ?
			LEFT JOIN users reporter
				ON reporter.id = s.user_id
			LEFT JOIN LATERAL (
				SELECT
					ST_Y(lp.location) AS lat,
					ST_X(lp.location) AS lng,
					lp.recorded_at AS recorded_at
				FROM location_pings lp
				WHERE lp.session_id = s.id
				ORDER BY lp.recorded_at DESC
				LIMIT 1
			) latest ON TRUE
			WHERE s.status = ?
			ORDER BY s.started_at DESC`,
			viewerPhone,
			SessionStatusActive,
		).
		Scan(&alerts).Error
	if err != nil {
		return nil, err
	}

	return alerts, nil
}
