package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"saferoute-backend/internal/reports"

	"github.com/google/uuid"
)

var allowedMediaTypes = map[string]string{
	"image/jpeg": "image",
	"image/png":  "image",
	"audio/mpeg": "audio",
	"audio/wav":  "audio",
	"audio/webm": "audio",
}

type ReportLookup interface {
	GetByID(ctx context.Context, id string) (*reports.ReportDetails, error)
}

type SessionLookup interface {
	ExistsSession(ctx context.Context, id string) (bool, error)
}

type Service struct {
	repo    Repository
	storage Storage
	reports ReportLookup
	sessions SessionLookup
	cfg     ServiceConfig
}

type ServiceConfig struct {
	MaxFileSizeBytes int64
}

type UploadInput struct {
	UserID    string
	ReportID  string
	SessionID string
	File      multipart.File
	Filename  string
	Size      int64
}

type Metadata struct {
	ID               string   `json:"id"`
	UserID           string   `json:"user_id"`
	ReportID         *string  `json:"report_id,omitempty"`
	SessionID        *string  `json:"session_id,omitempty"`
	Type             string   `json:"type"`
	URL              string   `json:"url"`
	Hash             string   `json:"hash"`
	MediaType        string   `json:"media_type"`
	SizeBytes        int64    `json:"size_bytes"`
	OriginalFilename string   `json:"original_filename"`
	CreatedAt        time.Time `json:"created_at"`
}

type Download struct {
	Metadata *Metadata
	Reader   io.ReadCloser
}

func NewService(repo Repository, storage Storage, reports ReportLookup, sessions SessionLookup, cfg ServiceConfig) *Service {
	if cfg.MaxFileSizeBytes <= 0 {
		cfg.MaxFileSizeBytes = 10 << 20
	}

	return &Service{
		repo:     repo,
		storage:  storage,
		reports:  reports,
		sessions: sessions,
		cfg:      cfg,
	}
}

func (s *Service) Upload(ctx context.Context, input UploadInput, baseURL string) (*Metadata, error) {
	defer input.File.Close()

	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return nil, ErrForbiddenEvidenceAccess
	}

	reportID := optionalString(input.ReportID)
	sessionID := optionalString(input.SessionID)
	if reportID == nil && sessionID == nil {
		return nil, ErrAttachmentTargetRequired
	}
	if reportID != nil && sessionID != nil {
		return nil, ErrAttachmentTargetConflict
	}
	if reportID != nil {
		if _, err := uuid.Parse(*reportID); err != nil {
			return nil, ErrInvalidReportID
		}
	}
	if sessionID != nil {
		if _, err := uuid.Parse(*sessionID); err != nil {
			return nil, ErrInvalidSessionID
		}
	}

	if input.Size <= 0 {
		return nil, ErrFileRequired
	}

	if input.Size > s.cfg.MaxFileSizeBytes {
		return nil, ErrFileTooLarge
	}

	if reportID != nil && s.reports != nil {
		if _, err := s.reports.GetByID(ctx, *reportID); err != nil {
			return nil, ErrParentNotFound
		}
	}

	if sessionID != nil && s.sessions != nil {
		exists, err := s.sessions.ExistsSession(ctx, *sessionID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrParentNotFound
		}
	}

	head := make([]byte, 512)
	n, err := input.File.Read(head)
	if err != nil && err != io.EOF {
		return nil, err
	}

	detectedType := http.DetectContentType(head[:n])
	kind, ok := allowedMediaTypes[detectedType]
	if !ok {
		return nil, ErrUnsupportedMediaType
	}

	safeName := sanitizeFilename(input.Filename)
	if safeName == "" {
		safeName = "upload.bin"
	}

	key := buildStorageKey(userID, safeName)
	hasher := sha256.New()
	body := io.MultiReader(bytes.NewReader(head[:n]), input.File)
	tee := io.TeeReader(body, hasher)

	if err := s.storage.Put(ctx, key, tee); err != nil {
		return nil, err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	record := &StoredEvidence{
		UserID:           userID,
		ReportID:         reportID,
		SessionID:        sessionID,
		Kind:             kind,
		StorageKey:       key,
		StorageProvider:  "local",
		SHA256:           hash,
		MediaType:        detectedType,
		SizeBytes:        input.Size,
		OriginalFilename: safeName,
	}
	if err := s.repo.Create(ctx, record); err != nil {
		return nil, err
	}

	return s.toMetadata(record, baseURL), nil
}

func (s *Service) GetByID(ctx context.Context, evidenceID, requesterID, baseURL string) (*Metadata, error) {
	evidenceID = strings.TrimSpace(evidenceID)
	if evidenceID == "" {
		return nil, ErrInvalidEvidenceID
	}
	if _, err := uuid.Parse(evidenceID); err != nil {
		return nil, ErrInvalidEvidenceID
	}

	record, err := s.repo.GetByID(ctx, evidenceID)
	if err != nil {
		return nil, err
	}

	if record.UserID != strings.TrimSpace(requesterID) {
		return nil, ErrForbiddenEvidenceAccess
	}

	return s.toMetadata(record, baseURL), nil
}

func (s *Service) Download(ctx context.Context, evidenceID, requesterID, baseURL string) (*Download, error) {
	metadata, err := s.GetByID(ctx, evidenceID, requesterID, baseURL)
	if err != nil {
		return nil, err
	}

	record, err := s.repo.GetByID(ctx, evidenceID)
	if err != nil {
		return nil, err
	}

	reader, err := s.storage.Open(ctx, record.StorageKey)
	if err != nil {
		return nil, err
	}

	return &Download{
		Metadata: metadata,
		Reader:   reader,
	}, nil
}

func (s *Service) toMetadata(record *StoredEvidence, baseURL string) *Metadata {
	url := strings.TrimRight(baseURL, "/") + "/api/v1/evidence/" + record.ID + "/content"
	return &Metadata{
		ID:               record.ID,
		UserID:           record.UserID,
		ReportID:         record.ReportID,
		SessionID:        record.SessionID,
		Type:             record.Kind,
		URL:              url,
		Hash:             record.SHA256,
		MediaType:        record.MediaType,
		SizeBytes:        record.SizeBytes,
		OriginalFilename: record.OriginalFilename,
		CreatedAt:        record.CreatedAt,
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}

func sanitizeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	return name
}

func buildStorageKey(userID, filename string) string {
	now := time.Now().UTC()
	return filepath.Join(
		"evidence",
		userID,
		now.Format("2006"),
		now.Format("01"),
		uuid.NewString()+"-"+filename,
	)
}
