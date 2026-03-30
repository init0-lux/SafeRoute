package reports

import (
	"context"
	"strings"
	"time"
)

const maxDescriptionLength = 1000

type Service struct {
	repo Repository
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
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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
	}, nil
}

func normalizeReportType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeDescription(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
