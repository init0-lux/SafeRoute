package reports

import (
	"errors"

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
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Type        string  `json:"type"`
	Description *string `json:"description,omitempty"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lng"`
	OccurredAt  string  `json:"occurred_at"`
	CreatedAt   string  `json:"created_at"`
	Source      string  `json:"source"`
}

func NewHandler(service *Service, authGuard fiber.Handler) *Handler {
	return &Handler{
		service:   service,
		authGuard: authGuard,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/reports", h.authGuard, h.create)
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
		Report: reportResponse{
			ID:          report.ID,
			UserID:      report.UserID,
			Type:        report.Type,
			Description: report.Description,
			Latitude:    report.Latitude,
			Longitude:   report.Longitude,
			OccurredAt:  report.OccurredAt.Format(timeFormatRFC3339),
			CreatedAt:   report.CreatedAt.Format(timeFormatRFC3339),
			Source:      report.Source,
		},
	})
}

func writeReportError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrUnauthorizedReport):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrInvalidReportType),
		errors.Is(err, ErrInvalidLatitude),
		errors.Is(err, ErrInvalidLongitude),
		errors.Is(err, ErrDescriptionTooLong):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"
