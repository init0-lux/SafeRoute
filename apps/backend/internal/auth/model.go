package auth

import "time"

type User struct {
	ID                  string             `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Phone               string             `gorm:"type:text;not null;uniqueIndex"`
	Email               *string            `gorm:"type:text"`
	PasswordHash        string             `gorm:"column:password_hash;type:text;not null"`
	TrustScore          float64            `gorm:"type:double precision;not null;default:0.3"`
	ReportCount         int                `gorm:"not null;default:0"`
	CorroborationCount  int                `gorm:"not null;default:0"`
	Verified            bool               `gorm:"not null;default:false"`
	VerifiedAt          *time.Time         `gorm:"type:timestamptz"`
	CreatedAt           time.Time          `gorm:"type:timestamptz;not null;default:now()"`
	TrustedContacts     []TrustedContact   `gorm:"constraint:OnDelete:CASCADE;"`
	VerificationRecords []UserVerification `gorm:"constraint:OnDelete:CASCADE;"`
}

type TrustedContact struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:ux_trusted_contacts_user_phone,priority:1"`
	Name      string    `gorm:"type:text;not null"`
	Phone     string    `gorm:"type:text;not null;uniqueIndex:ux_trusted_contacts_user_phone,priority:2"`
	Email     *string   `gorm:"type:text"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now()"`
	User      User      `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
}

// optional external verification (integration with aadhar etc later)
type UserVerification struct {
	ID          string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID      string     `gorm:"type:uuid;not null;index"`
	Provider    string     `gorm:"type:text;not null;uniqueIndex:ux_user_verifications_provider_ref,priority:1"`
	ProviderRef string     `gorm:"column:provider_ref;type:text;not null;uniqueIndex:ux_user_verifications_provider_ref,priority:2"`
	ProofHash   string     `gorm:"column:proof_hash;type:text;not null;uniqueIndex"`
	VerifiedAt  time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	RevokedAt   *time.Time `gorm:"type:timestamptz"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	User        User       `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
}

func (User) TableName() string {
	return "users"
}

func (TrustedContact) TableName() string {
	return "trusted_contacts"
}

func (UserVerification) TableName() string {
	return "user_verifications"
}
