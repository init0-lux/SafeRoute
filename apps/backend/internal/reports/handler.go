package reports

import (
	"errors"
	"strconv"

	"saferoute-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   *Service
	authGuard fiber.Handler
}

type createReportRequest struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lng"`
}

type createReportResponse struct {
	Report reportResponse `json:"report"`
}

type reportResponse struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Type        string   `json:"type"`
	Description *string  `json:"description,omitempty"`
	Latitude    float64  `json:"lat"`
	Longitude   float64  `json:"lng"`
	OccurredAt  string   `json:"occurred_at"`
	CreatedAt   string   `json:"created_at"`
	Source      string   `json:"source"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
	TrustScore  *float64 `json:"trust_score,omitempty"`
	DistanceM   *float64 `json:"distance_meters,omitempty"`
}

type listReportsResponse struct {
	Reports []reportResponse `json:"reports"`
	Count   int64            `json:"count"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

type historyResponse struct {
	Reports []historyReportResponse `json:"reports"`
}

type historyReportResponse struct {
	reportResponse
	Status string                 `json:"status"`
	Events []ComplaintEventResult `json:"events"`
}

func NewHandler(service *Service, authGuard fiber.Handler) *Handler {
	return &Handler{
		service:   service,
		authGuard: authGuard,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/reports", h.authGuard, h.create)
	router.Get("/reports/me", h.authGuard, h.listUserHistory)
	router.Get("/reports/:id", h.getByID)
	router.Get("/reports", h.listNearby)
}

func (h *Handler) create(c *fiber.Ctx) error {
	var req createReportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeReportError(c, ErrUnauthorizedReport)
	}

	report, err := h.service.Create(c.UserContext(), CreateReportInput{
		UserID:      currentUser.ID,
		Type:        req.Type,
		Description: req.Description,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
	})
	if err != nil {
		return writeReportError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(createReportResponse{
		Report: newReportResponse(reportResponseInput{
			ID:          report.ID,
			UserID:      report.UserID,
			Type:        report.Type,
			Description: report.Description,
			Latitude:    report.Latitude,
			Longitude:   report.Longitude,
			OccurredAt:  report.OccurredAt,
			CreatedAt:   report.CreatedAt,
			Source:      report.Source,
			TrustScore:  &report.TrustScore,
		}),
	})
}

func (h *Handler) getByID(c *fiber.Ctx) error {
	report, err := h.service.GetByID(c.UserContext(), c.Params("id"))
	if err != nil {
		return writeReportError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(createReportResponse{
		Report: newReportResponse(reportResponseInput{
			ID:          report.ID,
			UserID:      report.UserID,
			Type:        report.Type,
			Description: report.Description,
			Latitude:    report.Latitude,
			Longitude:   report.Longitude,
			OccurredAt:  report.OccurredAt,
			CreatedAt:   report.CreatedAt,
			Source:      report.Source,
			EvidenceIDs: report.EvidenceIDs,
			TrustScore:  &report.TrustScore,
		}),
	})
}

func (h *Handler) listNearby(c *fiber.Ctx) error {
	latitude, err := parseRequiredFloatQuery(c, "lat")
	if err != nil {
		return writeReportError(c, err)
	}

	longitude, err := parseRequiredFloatQuery(c, "lng")
	if err != nil {
		return writeReportError(c, err)
	}

	radius, err := parseRequiredFloatQuery(c, "radius")
	if err != nil {
		return writeReportError(c, err)
	}

	limit, err := parseOptionalIntQuery(c, "limit", 0)
	if err != nil {
		return writeReportError(c, err)
	}

	offset, err := parseOptionalIntQuery(c, "offset", 0)
	if err != nil {
		return writeReportError(c, err)
	}

	page, err := h.service.ListNearby(c.UserContext(), NearbyReportsInput{
		Latitude:  latitude,
		Longitude: longitude,
		Radius:    radius,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		return writeReportError(c, err)
	}

	items := make([]reportResponse, 0, len(page.Reports))
	for _, report := range page.Reports {
		trustScore := report.TrustScore
		distance := report.DistanceMeters
		items = append(items, newReportResponse(reportResponseInput{
			ID:          report.ID,
			UserID:      report.UserID,
			Type:        report.Type,
			Description: report.Description,
			Latitude:    report.Latitude,
			Longitude:   report.Longitude,
			OccurredAt:  report.OccurredAt,
			CreatedAt:   report.CreatedAt,
			Source:      report.Source,
			TrustScore:  &trustScore,
			DistanceM:   &distance,
		}))
	}

	return c.Status(fiber.StatusOK).JSON(listReportsResponse{
		Reports: items,
		Count:   page.Count,
		Limit:   page.Limit,
		Offset:  page.Offset,
	})
}

func (h *Handler) listUserHistory(c *fiber.Ctx) error {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeReportError(c, ErrUnauthorizedReport)
	}

	reportsList, err := h.service.ListUserHistory(c.UserContext(), currentUser.ID)
	if err != nil {
		return writeReportError(c, err)
	}

	items := make([]historyReportResponse, len(reportsList))
	for i, report := range reportsList {
		items[i] = historyReportResponse{
			reportResponse: newReportResponse(reportResponseInput{
				ID:          report.ID,
				UserID:      report.UserID,
				Type:        report.Type,
				Description: report.Description,
				Latitude:    report.Latitude,
				Longitude:   report.Longitude,
				OccurredAt:  report.OccurredAt,
				CreatedAt:   report.CreatedAt,
				Source:      "app",
				EvidenceIDs: report.EvidenceIDs,
			}),
			Status: report.Status,
			Events: report.Events,
		}
	}

	return c.Status(fiber.StatusOK).JSON(historyResponse{
		Reports: items,
	})
}

func writeReportError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrUnauthorizedReport):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrReportNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrInvalidReportType),
		errors.Is(err, ErrUnsupportedReportType),
		errors.Is(err, ErrInvalidLatitude),
		errors.Is(err, ErrInvalidLongitude),
		errors.Is(err, ErrInvalidRadius),
		errors.Is(err, ErrInvalidLimit),
		errors.Is(err, ErrInvalidOffset),
		errors.Is(err, ErrInvalidReportID),
		errors.Is(err, ErrDescriptionTooLong):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

type reportResponseInput struct {
	ID          string
	UserID      string
	Type        string
	Description *string
	Latitude    float64
	Longitude   float64
	OccurredAt  interface{ Format(string) string }
	CreatedAt   interface{ Format(string) string }
	Source      string
	EvidenceIDs []string
	TrustScore  *float64
	DistanceM   *float64
}

func newReportResponse(input reportResponseInput) reportResponse {
	return reportResponse{
		ID:          input.ID,
		UserID:      input.UserID,
		Type:        input.Type,
		Description: input.Description,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		OccurredAt:  input.OccurredAt.Format(timeFormatRFC3339),
		CreatedAt:   input.CreatedAt.Format(timeFormatRFC3339),
		Source:      input.Source,
		EvidenceIDs: input.EvidenceIDs,
		TrustScore:  input.TrustScore,
		DistanceM:   input.DistanceM,
	}
}

func parseRequiredFloatQuery(c *fiber.Ctx, key string) (float64, error) {
	value := c.Query(key)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		switch key {
		case "lat":
			return 0, ErrInvalidLatitude
		case "lng":
			return 0, ErrInvalidLongitude
		default:
			return 0, ErrInvalidRadius
		}
	}

	return parsed, nil
}

func parseOptionalIntQuery(c *fiber.Ctx, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		if key == "offset" {
			return 0, ErrInvalidOffset
		}

		return 0, ErrInvalidLimit
	}

	return parsed, nil
}
