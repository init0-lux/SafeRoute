package safety

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

type scoreResponse struct {
	Score     int          `json:"score"`
	RiskLevel string       `json:"risk_level"`
	Factors   ScoreFactors `json:"factors"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/safety/score", h.getScore)
}

func (h *Handler) getScore(c *fiber.Ctx) error {
	latitude, err := parseRequiredFloatQuery(c, "lat")
	if err != nil {
		return writeSafetyError(c, err)
	}

	longitude, err := parseRequiredFloatQuery(c, "lng")
	if err != nil {
		return writeSafetyError(c, err)
	}

	radius, err := parseOptionalFloatQuery(c, "radius", 0)
	if err != nil {
		return writeSafetyError(c, err)
	}

	result, err := h.service.Score(c.UserContext(), ScoreInput{
		Latitude:  latitude,
		Longitude: longitude,
		Radius:    radius,
	})
	if err != nil {
		return writeSafetyError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(scoreResponse{
		Score:     result.Score,
		RiskLevel: result.RiskLevel,
		Factors:   result.Factors,
	})
}

func writeSafetyError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidLatitude), errors.Is(err, ErrInvalidLongitude), errors.Is(err, ErrInvalidRadius):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func parseRequiredFloatQuery(c *fiber.Ctx, key string) (float64, error) {
	value := c.Query(key)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		switch key {
		case "lat":
			return 0, ErrInvalidLatitude
		default:
			return 0, ErrInvalidLongitude
		}
	}
	return parsed, nil
}

func parseOptionalFloatQuery(c *fiber.Ctx, key string, fallback float64) (float64, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, ErrInvalidRadius
	}

	return parsed, nil
}
