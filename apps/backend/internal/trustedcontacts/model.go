package trustedcontacts

import (
	"time"

	"saferoute-backend/internal/auth"
)

type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusAccepted  RequestStatus = "accepted"
	RequestStatusCancelled RequestStatus = "cancelled"
	RequestStatusExpired   RequestStatus = "expired"
)

type TrustedContact struct {
	ID         string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     string    `gorm:"type:uuid;not null;uniqueIndex:ux_trusted_contacts_user_phone,priority:1;index"`
	RequestID  *string   `gorm:"column:request_id;type:uuid;uniqueIndex"`
	Name       string    `gorm:"type:text;not null"`
	Phone      string    `gorm:"type:text;not null;uniqueIndex:ux_trusted_contacts_user_phone,priority:2"`
	Email      *string   `gorm:"type:text"`
	AcceptedAt time.Time `gorm:"column:accepted_at;type:timestamptz;not null;default:now()"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
	User       auth.User `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
	// Field for joined query
	PushToken  string    `gorm:"-"`
}

type TrustedContactRequest struct {
	ID                string          `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID            string          `gorm:"type:uuid;not null;index"`
	Name              string          `gorm:"type:text;not null"`
	Phone             string          `gorm:"type:text;not null;index"`
	Email             *string         `gorm:"type:text"`
	Status            RequestStatus   `gorm:"column:status;type:trusted_contact_request_status;not null;default:'pending';index"`
	InviteTokenHash   string          `gorm:"column:invite_token_hash;type:text;not null;uniqueIndex"`
	ExpiresAt         time.Time       `gorm:"column:expires_at;type:timestamptz;not null;index"`
	RespondedAt       *time.Time      `gorm:"column:responded_at;type:timestamptz"`
	AcceptedContactID *string         `gorm:"column:accepted_contact_id;type:uuid;uniqueIndex"`
	CreatedAt         time.Time       `gorm:"type:timestamptz;not null;default:now()"`
	User              auth.User       `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
	AcceptedContact   *TrustedContact `gorm:"constraint:OnDelete:SET NULL;foreignKey:AcceptedContactID;references:ID"`
}

func (TrustedContact) TableName() string {
	return "trusted_contacts"
}

func (TrustedContactRequest) TableName() string {
	return "trusted_contact_requests"
}
