package sos

import (
	"time"

	"saferoute-backend/internal/auth"
)

type SessionStatus string

const (
	SessionStatusActive SessionStatus = "active"
	SessionStatusEnded  SessionStatus = "ended"
)

type SOSSession struct {
	ID            string         `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID        *string        `gorm:"type:uuid;index"`
	User          *auth.User     `gorm:"constraint:OnDelete:SET NULL;foreignKey:UserID;references:ID"`
	Status        SessionStatus  `gorm:"type:sos_session_status;not null;default:active"`
	StartedAt     time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	EndedAt       *time.Time     `gorm:"type:timestamptz"`
	LocationPings []LocationPing `gorm:"constraint:OnDelete:CASCADE;foreignKey:SessionID;references:ID"`
}

type LocationPing struct {
	ID         string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SessionID  string     `gorm:"type:uuid;not null"`
	Session    SOSSession `gorm:"constraint:OnDelete:CASCADE;foreignKey:SessionID;references:ID"`
	Location   string     `gorm:"type:geometry(Point,4326);not null"`
	RecordedAt time.Time  `gorm:"type:timestamptz;not null"`
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (SOSSession) TableName() string {
	return "sos_sessions"
}

func (LocationPing) TableName() string {
	return "location_pings"
}
