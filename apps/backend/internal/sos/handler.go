package sos

import (
	"context"
	"errors"
	"strings"
	"time"

	"saferoute-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

const websocketUserIDKey = "sos_ws_user_id"

type Handler struct {
	service *Service
	auth    *auth.Middleware
}

type sessionEnvelope struct {
	Session sessionResponse `json:"session"`
}

type sessionResponse struct {
	ID        string        `json:"id"`
	UserID    *string       `json:"user_id,omitempty"`
	Status    SessionStatus `json:"status"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   *time.Time    `json:"ended_at"`
}

type locationPingMessage struct {
	Latitude   float64   `json:"lat"`
	Longitude  float64   `json:"lng"`
	RecordedAt time.Time `json:"ts"`
}

type locationAck struct {
	Status     string    `json:"status"`
	RecordedAt time.Time `json:"recorded_at"`
}

func NewHandler(service *Service, authMiddleware *auth.Middleware) *Handler {
	return &Handler{
		service: service,
		auth:    authMiddleware,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	sosRoutes := router.Group("/sos")
	sosRoutes.Post("/start", h.auth.VerifyUser(), h.start)
	sosRoutes.Get("/:id", h.auth.VerifyUser(), h.get)
	sosRoutes.Post("/:id/end", h.auth.VerifyUser(), h.end)
	sosRoutes.Get("/:id/stream", h.auth.VerifyUser(), h.prepareStream, websocket.New(h.stream))
}

func (h *Handler) start(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeSOSError(c, auth.ErrUnauthorized)
	}

	session, err := h.service.StartSession(c.UserContext(), user.ID)
	if err != nil {
		return writeSOSError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(sessionEnvelope{
		Session: newSessionResponse(session),
	})
}

func (h *Handler) get(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeSOSError(c, auth.ErrUnauthorized)
	}

	session, err := h.service.GetSession(c.UserContext(), c.Params("id"), user.ID)
	if err != nil {
		return writeSOSError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(sessionEnvelope{
		Session: newSessionResponse(session),
	})
}

func (h *Handler) end(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeSOSError(c, auth.ErrUnauthorized)
	}

	session, err := h.service.EndSession(c.UserContext(), c.Params("id"), user.ID)
	if err != nil {
		return writeSOSError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(sessionEnvelope{
		Session: newSessionResponse(session),
	})
}

func (h *Handler) prepareStream(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeSOSError(c, auth.ErrUnauthorized)
	}

	c.Locals(websocketUserIDKey, user.ID)
	return c.Next()
}

func (h *Handler) stream(conn *websocket.Conn) {
	userID, _ := conn.Locals(websocketUserIDKey).(string)
	sessionID := strings.TrimSpace(conn.Params("id"))

	if _, err := h.service.GetSession(context.Background(), sessionID, userID); err != nil {
		_ = conn.WriteJSON(fiber.Map{"error": err.Error()})
		_ = conn.Close()
		return
	}

	for {
		var message locationPingMessage
		if err := conn.ReadJSON(&message); err != nil {
			return
		}

		if err := h.service.RecordLocationPing(context.Background(), sessionID, userID, message.Latitude, message.Longitude, message.RecordedAt); err != nil {
			_ = conn.WriteJSON(fiber.Map{"error": err.Error()})
			if errors.Is(err, ErrSessionAlreadyEnded) || errors.Is(err, ErrSessionForbidden) || errors.Is(err, ErrSessionNotFound) {
				_ = conn.Close()
				return
			}
			continue
		}

		recordedAt := message.RecordedAt
		if recordedAt.IsZero() {
			recordedAt = time.Now().UTC()
		}

		if err := conn.WriteJSON(locationAck{
			Status:     "accepted",
			RecordedAt: recordedAt.UTC(),
		}); err != nil {
			return
		}
	}
}

func newSessionResponse(session *SOSSession) sessionResponse {
	return sessionResponse{
		ID:        session.ID,
		UserID:    session.UserID,
		Status:    session.Status,
		StartedAt: session.StartedAt,
		EndedAt:   session.EndedAt,
	}
}

func writeSOSError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, auth.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrInvalidSessionID), errors.Is(err, ErrInvalidUserID), errors.Is(err, ErrInvalidLatitude), errors.Is(err, ErrInvalidLongitude):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrSessionForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrSessionNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrActiveSessionExists), errors.Is(err, ErrSessionAlreadyEnded):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}
