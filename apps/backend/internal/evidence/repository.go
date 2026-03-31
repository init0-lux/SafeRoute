package evidence

import (
	"context"
	"errors"
	"time"

	"saferoute-backend/internal/reports"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, record *StoredEvidence) error
	GetByID(ctx context.Context, id string) (*StoredEvidence, error)
}

type GormRepository struct {
	db *gorm.DB
}

type StoredEvidence struct {
	ID               string
	UserID           string
	ReportID         *string
	SessionID        *string
	Kind             string
	StorageKey       string
	StorageProvider  string
	SHA256           string
	MediaType        string
	SizeBytes        int64
	OriginalFilename string
	OnChainTx        *string
	OnChainVerified  bool
	OnChainVerifiedAt *time.Time
	CreatedAt        time.Time
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, record *StoredEvidence) error {
	sizeBytes := record.SizeBytes
	originalFilename := record.OriginalFilename
	model := reports.Evidence{
		UserID:           &record.UserID,
		ReportID:         record.ReportID,
		SessionID:        record.SessionID,
		Kind:             record.Kind,
		StorageKey:       record.StorageKey,
		StorageProvider:  record.StorageProvider,
		SHA256:           record.SHA256,
		MediaType:        record.MediaType,
		SizeBytes:        &sizeBytes,
		OriginalFilename: &originalFilename,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}

	record.ID = model.ID
	record.CreatedAt = model.CreatedAt
	return nil
}

func (r *GormRepository) GetByID(ctx context.Context, id string) (*StoredEvidence, error) {
	var model reports.Evidence
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEvidenceNotFound
		}

		return nil, err
	}

	return &StoredEvidence{
		ID:               model.ID,
		UserID:           derefString(model.UserID),
		ReportID:         model.ReportID,
		SessionID:        model.SessionID,
		Kind:             model.Kind,
		StorageKey:       model.StorageKey,
		StorageProvider:  model.StorageProvider,
		SHA256:           model.SHA256,
		MediaType:        model.MediaType,
		SizeBytes:        derefInt64(model.SizeBytes),
		OriginalFilename: derefString(model.OriginalFilename),
		OnChainTx:        model.OnChainTx,
		OnChainVerified:  model.OnChainVerified,
		OnChainVerifiedAt: model.OnChainVerifiedAt,
		CreatedAt:        model.CreatedAt,
	}, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}

	return *value
}
