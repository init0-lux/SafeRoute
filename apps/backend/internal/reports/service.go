package reports

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxDescriptionLength = 1000

var allowedReportTypes = map[string]struct{}{
	"harassment":         {},
	"unsafe_area":        {},
	"stalking":           {},
	"assault":            {},
	"theft":              {},
	"suspicious_activity": {},
}

type Service struct {
	repo Repository
	cfg  ServiceConfig
	trust ReportTrustRecorder
}

type ServiceConfig struct {
	DefaultNearbyLimit int
	MaxNearbyLimit     int
	MaxNearbyRadiusM   float64
}

type ReportTrustRecorder interface {
	RecordReportSubmission(ctx context.Context, userID string) error
}

type CreateReportInput struct {
	UserID      string
	Type        string
	Description string
	Latitude    float64
	Longitude   float64
}

type CreatedReport struct {
	ID          string
	UserID      string
	Type        string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  time.Time
	CreatedAt   time.Time
	Source      string
	TrustScore  float64
}

type ReportDetails struct {
	ID          string
	UserID      string
	Type        string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  time.Time
	CreatedAt   time.Time
	Source      string
	EvidenceIDs []string
	TrustScore  float64
}

type NearbyReportsInput struct {
	Latitude  float64
	Longitude float64
	Radius    float64
	Limit     int
	Offset    int
}

type NearbyReport struct {
	ID             string
	UserID         string
	Type           string
	Description    *string
	Latitude       float64
	Longitude      float64
	OccurredAt     time.Time
	CreatedAt      time.Time
	Source         string
	DistanceMeters float64
	TrustScore     float64
}

type NearbyReportsPage struct {
	Reports []NearbyReport
	Count   int64
	Limit   int
	Offset  int
}

func NewService(repo Repository, cfg ServiceConfig, trust ReportTrustRecorder) *Service {
	if cfg.DefaultNearbyLimit <= 0 {
		cfg.DefaultNearbyLimit = 20
	}

	if cfg.MaxNearbyLimit <= 0 {
		cfg.MaxNearbyLimit = 50
	}

	if cfg.MaxNearbyRadiusM <= 0 {
		cfg.MaxNearbyRadiusM = 5000
	}

	return &Service{
		repo: repo,
		cfg:  cfg,
		trust: trust,
	}
}

func (s *Service) Create(ctx context.Context, input CreateReportInput) (*CreatedReport, error) {
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return nil, ErrUnauthorizedReport
	}

	reportType := normalizeReportType(input.Type)
	if reportType == "" {
		return nil, ErrInvalidReportType
	}
	if !isAllowedReportType(reportType) {
		return nil, ErrUnsupportedReportType
	}

	if input.Latitude < -90 || input.Latitude > 90 {
		return nil, ErrInvalidLatitude
	}

	if input.Longitude < -180 || input.Longitude > 180 {
		return nil, ErrInvalidLongitude
	}

	description := normalizeDescription(input.Description)
	if description != nil && len(*description) > maxDescriptionLength {
		return nil, ErrDescriptionTooLong
	}

	occurredAt := time.Now().UTC()
	userIDCopy := userID
	report, err := s.repo.Create(ctx, CreateParams{
		UserID:      &userIDCopy,
		Category:    reportType,
		Description: description,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		OccurredAt:  occurredAt,
		Source:      "app",
	})
	if err != nil {
		return nil, err
	}

	if s.trust != nil {
		if err := s.trust.RecordReportSubmission(ctx, userID); err != nil {
			return nil, err
		}
	}

	return &CreatedReport{
		ID:          report.ID,
		UserID:      userID,
		Type:        report.Category,
		Description: description,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		OccurredAt:  report.OccurredAt,
		CreatedAt:   report.CreatedAt,
		Source:      report.Source,
		TrustScore:  report.TrustScore,
	}, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*ReportDetails, error) {
	reportID := strings.TrimSpace(id)
	if reportID == "" {
		return nil, ErrInvalidReportID
	}
	if _, err := uuid.Parse(reportID); err != nil {
		return nil, ErrInvalidReportID
	}

	report, err := s.repo.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}

	evidenceIDs, err := s.repo.ListEvidenceIDs(ctx, reportID)
	if err != nil {
		return nil, err
	}

	return &ReportDetails{
		ID:          report.ID,
		UserID:      derefString(report.UserID),
		Type:        report.Category,
		Description: report.Description,
		Latitude:    report.Latitude,
		Longitude:   report.Longitude,
		OccurredAt:  report.OccurredAt,
		CreatedAt:   report.CreatedAt,
		Source:      report.Source,
		EvidenceIDs: evidenceIDs,
		TrustScore:  report.TrustScore,
	}, nil
}

func (s *Service) ListNearby(ctx context.Context, input NearbyReportsInput) (*NearbyReportsPage, error) {
	if input.Latitude < -90 || input.Latitude > 90 {
		return nil, ErrInvalidLatitude
	}

	if input.Longitude < -180 || input.Longitude > 180 {
		return nil, ErrInvalidLongitude
	}

	if input.Radius <= 0 || input.Radius > s.cfg.MaxNearbyRadiusM {
		return nil, ErrInvalidRadius
	}

	limit := input.Limit
	if limit == 0 {
		limit = s.cfg.DefaultNearbyLimit
	}
	if limit < 1 || limit > s.cfg.MaxNearbyLimit {
		return nil, ErrInvalidLimit
	}

	if input.Offset < 0 {
		return nil, ErrInvalidOffset
	}

	rows, err := s.repo.ListNearby(ctx, NearbyParams{
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		Radius:    input.Radius,
		Limit:     limit,
		Offset:    input.Offset,
	})
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountNearby(ctx, NearbyParams{
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		Radius:    input.Radius,
	})
	if err != nil {
		return nil, err
	}

	reports := make([]NearbyReport, 0, len(rows))
	for _, row := range rows {
		reports = append(reports, NearbyReport{
			ID:             row.ID,
			UserID:         derefString(row.UserID),
			Type:           row.Category,
			Description:    row.Description,
			Latitude:       row.Latitude,
			Longitude:      row.Longitude,
			OccurredAt:     row.OccurredAt,
			CreatedAt:      row.CreatedAt,
			Source:         row.Source,
			DistanceMeters: row.DistanceMeters,
			TrustScore:     row.TrustScore,
		})
	}

	return &NearbyReportsPage{
		Reports: reports,
		Count:   count,
		Limit:   limit,
		Offset:  input.Offset,
	}, nil
}

func normalizeReportType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isAllowedReportType(value string) bool {
	_, ok := allowedReportTypes[value]
	return ok
}

func normalizeDescription(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
