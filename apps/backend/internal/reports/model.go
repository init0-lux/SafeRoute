package reports

import (
	"time"

	"saferoute-backend/internal/auth"
	"saferoute-backend/internal/sos"
)

type ComplaintEventStatus string

const (
	ComplaintEventStatusSubmitted   ComplaintEventStatus = "submitted"
	ComplaintEventStatusUnderReview ComplaintEventStatus = "under_review"
	ComplaintEventStatusEscalated   ComplaintEventStatus = "escalated"
	ComplaintEventStatusResolved    ComplaintEventStatus = "resolved"
)

type Report struct {
	ID              string           `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID          *string          `gorm:"type:uuid;index"`
	User            *auth.User       `gorm:"constraint:OnDelete:SET NULL;foreignKey:UserID;references:ID"`
	Category        string           `gorm:"type:text;not null"`
	Description     *string          `gorm:"type:text"`
	Location        string           `gorm:"type:geography(POINT,4326);not null"`
	Address         *string          `gorm:"type:text"`
	OccurredAt      time.Time        `gorm:"type:timestamptz;not null"`
	CreatedAt       time.Time        `gorm:"type:timestamptz;not null;default:now()"`
	Source          string           `gorm:"type:text;not null;default:app"`
	EvidenceItems   []Evidence       `gorm:"constraint:OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	ComplaintEvents []ComplaintEvent `gorm:"constraint:OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
}

type Evidence struct {
	ID           string          `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       *string         `gorm:"type:uuid;index"`
	User         *auth.User      `gorm:"constraint:OnDelete:SET NULL;foreignKey:UserID;references:ID"`
	ReportID     *string         `gorm:"type:uuid;index"`
	Report       *Report         `gorm:"constraint:OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	SessionID    *string         `gorm:"type:uuid;index"`
	Session      *sos.SOSSession `gorm:"constraint:OnDelete:SET NULL;foreignKey:SessionID;references:ID"`
	Kind         string          `gorm:"column:kind;type:text;not null;default:image"`
	StorageKey   string          `gorm:"column:storage_key;type:text;not null;uniqueIndex"`
	StorageProvider string       `gorm:"column:storage_provider;type:text;not null;default:local"`
	SHA256       string          `gorm:"column:sha256;type:text;not null;index"`
	PreviousHash *string         `gorm:"column:previous_hash;type:text"`
	MediaType    string          `gorm:"column:media_type;type:text;not null"`
	SizeBytes    *int64          `gorm:"column:size_bytes;type:bigint"`
	OriginalFilename *string     `gorm:"column:original_filename;type:text"`
	IPFSCID      *string         `gorm:"column:ipfs_cid;type:text"`
	IPFSStatus   *string         `gorm:"column:ipfs_status;type:text"`
	IPFSRequestedAt *time.Time   `gorm:"column:ipfs_requested_at;type:timestamptz"`
	IPFSUploadedAt *time.Time    `gorm:"column:ipfs_uploaded_at;type:timestamptz"`
	SignedAt     *time.Time      `gorm:"column:signed_at;type:timestamptz"`
	CreatedAt    time.Time       `gorm:"type:timestamptz;not null;default:now()"`
}

type ComplaintEvent struct {
	ID        string               `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ReportID  string               `gorm:"type:uuid;not null"`
	Report    Report               `gorm:"constraint:OnDelete:CASCADE;foreignKey:ReportID;references:ID"`
	Status    ComplaintEventStatus `gorm:"type:complaint_event_status;not null"`
	Actor     string               `gorm:"type:text;not null"`
	Note      *string              `gorm:"type:text"`
	CreatedAt time.Time            `gorm:"type:timestamptz;not null;default:now()"`
}

func (Report) TableName() string {
	return "reports"
}

func (Evidence) TableName() string {
	return "evidence"
}

func (ComplaintEvent) TableName() string {
	return "complaint_events"
}
