package sos

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"time"

	"saferoute-backend/internal/auth"
	"saferoute-backend/internal/trustedcontacts"
)

type SessionStatus string

const (
	SessionStatusActive SessionStatus = "active"
	SessionStatusEnded  SessionStatus = "ended"
)

type SOSSession struct {
	ID            string           `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        *string          `gorm:"type:uuid;index"`
	User          *auth.User       `gorm:"constraint:OnDelete:SET NULL;foreignKey:UserID;references:ID"`
	Status        SessionStatus    `gorm:"type:sos_session_status;not null;default:active"`
	StartedAt     time.Time        `gorm:"type:timestamptz;not null;default:now()"`
	EndedAt       *time.Time       `gorm:"type:timestamptz"`
	LocationPings []LocationPing   `gorm:"constraint:OnDelete:CASCADE;foreignKey:SessionID;references:ID"`
	ViewerGrants  []SOSViewerGrant `gorm:"constraint:OnDelete:CASCADE;foreignKey:SessionID;references:ID"`
}

type LocationPing struct {
	ID         string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID  string     `gorm:"type:uuid;not null"`
	Session    SOSSession `gorm:"constraint:OnDelete:CASCADE;foreignKey:SessionID;references:ID"`
	Location   string     `gorm:"type:geometry(Point,4326);not null"`
	RecordedAt time.Time  `gorm:"type:timestamptz;not null"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

type SOSViewerGrant struct {
	ID               string                         `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID        string                         `gorm:"type:uuid;not null;index;uniqueIndex:ux_sos_viewer_grants_session_contact_active,priority:1"`
	Session          SOSSession                     `gorm:"constraint:OnDelete:CASCADE;foreignKey:SessionID;references:ID"`
	UserID           string                         `gorm:"type:uuid;not null;index"`
	TrustedContactID string                         `gorm:"column:trusted_contact_id;type:uuid;not null;index;uniqueIndex:ux_sos_viewer_grants_session_contact_active,priority:2"`
	TrustedContact   trustedcontacts.TrustedContact `gorm:"constraint:OnDelete:CASCADE;foreignKey:TrustedContactID;references:ID"`
	Token            string                         `gorm:"column:token;type:text"`
	TokenHash        string                         `gorm:"column:token_hash;type:text;not null;uniqueIndex"`
	RevokedAt        *time.Time                     `gorm:"column:revoked_at;type:timestamptz;uniqueIndex:ux_sos_viewer_grants_session_contact_active,priority:3"`
	ExpiresAt        time.Time                      `gorm:"column:expires_at;type:timestamptz;not null;index"`
	CreatedAt        time.Time                      `gorm:"type:timestamptz;not null;default:now()"`
}

func (SOSSession) TableName() string {
	return "sos_sessions"
}

func (LocationPing) TableName() string {
	return "location_pings"
}

func (SOSViewerGrant) TableName() string {
	return "sos_viewer_grants"
}

func GenerateViewerToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}

	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, HashViewerToken(token), nil
}

func HashViewerToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CompareViewerToken(hash, token string) bool {
	expected := HashViewerToken(token)
	return subtle.ConstantTimeCompare([]byte(hash), []byte(expected)) == 1
}
