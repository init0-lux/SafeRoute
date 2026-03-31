package safety

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

type routeScoreRequest struct {
	Origin      Coordinate `json:"origin"`
	Destination Coordinate `json:"destination"`
	TravelMode  string     `json:"travel_mode"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/safety/score", h.getScore)
	router.Post("/safety/route-score", h.getRouteScore)
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

	result, err := h.service.ScorePoint(c.UserContext(), ScoreInput{
		Latitude:  latitude,
		Longitude: longitude,
		Radius:    radius,
	})
	if err != nil {
		return writeSafetyError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func (h *Handler) getRouteScore(c *fiber.Ctx) error {
	var req routeScoreRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.ScoreRoute(c.UserContext(), RouteScoreInput{
		Origin:      req.Origin,
		Destination: req.Destination,
		TravelMode:  req.TravelMode,
	})
	if err != nil {
		return writeSafetyError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func writeSafetyError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidLatitude),
		errors.Is(err, ErrInvalidLongitude),
		errors.Is(err, ErrInvalidRadius),
		errors.Is(err, ErrInvalidOriginLatitude),
		errors.Is(err, ErrInvalidOriginLongitude),
		errors.Is(err, ErrInvalidDestinationLatitude),
		errors.Is(err, ErrInvalidDestinationLongitude),
		errors.Is(err, ErrUnsupportedTravelMode),
		errors.Is(err, ErrRouteTooLong):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrRouteNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrRouteProviderUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrRouteProviderFailed):
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "route provider request failed"})
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
