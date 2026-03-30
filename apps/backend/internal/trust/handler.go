package trust

import (
	"errors"

	"saferoute-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service   *Service
	authGuard fiber.Handler
}

type verifyRequest struct {
	Verified bool `json:"verified"`
}

type response struct {
	Score              float64        `json:"score"`
	ReportsCount       int            `json:"reports_count"`
	CorroborationCount int            `json:"corroboration_count"`
	Verified           bool           `json:"verified"`
	UpdatedAt          string         `json:"updated_at"`
	Breakdown          TrustBreakdown `json:"breakdown"`
}

func NewHandler(service *Service, authGuard fiber.Handler) *Handler {
	return &Handler{
		service:   service,
		authGuard: authGuard,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/trust/me", h.authGuard, h.me)
	router.Post("/trust/verify", h.authGuard, h.verify)
}

func (h *Handler) me(c *fiber.Ctx) error {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustError(c, auth.ErrUnauthorized)
	}

	snapshot, err := h.service.GetByUserID(c.UserContext(), currentUser.ID)
	if err != nil {
		return writeTrustError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newResponse(snapshot))
}

func (h *Handler) verify(c *fiber.Ctx) error {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustError(c, auth.ErrUnauthorized)
	}

	var req verifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	snapshot, err := h.service.SetVerification(c.UserContext(), currentUser.ID, req.Verified)
	if err != nil {
		return writeTrustError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(newResponse(snapshot))
}

func writeTrustError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidUserID):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, auth.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, auth.ErrUserNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func newResponse(snapshot *Snapshot) response {
	return response{
		Score:              snapshot.Score,
		ReportsCount:       snapshot.ReportsCount,
		CorroborationCount: snapshot.CorroborationCount,
		Verified:           snapshot.Verified,
		UpdatedAt:          snapshot.UpdatedAt.Format(timeFormatRFC3339),
		Breakdown:          snapshot.Breakdown,
	}
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"
