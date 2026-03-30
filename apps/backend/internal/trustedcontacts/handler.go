package trustedcontacts

import (
	"errors"
	"time"

	"saferoute-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
	auth    *auth.Middleware
}

type createRequestPayload struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type acceptRequestPayload struct {
	Token string `json:"token"`
}

type trustedContactRequestResponse struct {
	Request     trustedContactRequestBody `json:"request"`
	AcceptToken string                    `json:"accept_token,omitempty"`
}

type trustedContactAcceptedResponse struct {
	Request trustedContactRequestBody `json:"request"`
	Contact trustedContactBody        `json:"contact"`
}

type trustedContactRequestBody struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	Email       *string    `json:"email,omitempty"`
	Status      string     `json:"status"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type trustedContactBody struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	RequestID  *string   `json:"request_id,omitempty"`
	Name       string    `json:"name"`
	Phone      string    `json:"phone"`
	Email      *string   `json:"email,omitempty"`
	AcceptedAt time.Time `json:"accepted_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewHandler(service *Service, authMiddleware *auth.Middleware) *Handler {
	return &Handler{
		service: service,
		auth:    authMiddleware,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/trusted-contacts")
	group.Post("/requests", h.auth.VerifyUser(), h.createRequest)
	group.Post("/requests/:id/accept", h.acceptRequest)
	group.Delete("/:id", h.auth.VerifyUser(), h.deleteTrustedContact)
}

func (h *Handler) createRequest(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustedContactError(c, ErrUnauthorized)
	}

	var payload createRequestPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	request, token, err := h.service.CreateRequest(c.UserContext(), user.ID, CreateRequestInput{
		Name:  payload.Name,
		Phone: payload.Phone,
		Email: payload.Email,
	})
	if err != nil {
		return writeTrustedContactError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(trustedContactRequestResponse{
		Request:     newTrustedContactRequestBody(request),
		AcceptToken: token,
	})
}

func (h *Handler) acceptRequest(c *fiber.Ctx) error {
	var payload acceptRequestPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	request, contact, err := h.service.AcceptRequest(c.UserContext(), c.Params("id"), AcceptRequestInput{
		Token: payload.Token,
	})
	if err != nil {
		return writeTrustedContactError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(trustedContactAcceptedResponse{
		Request: newTrustedContactRequestBody(request),
		Contact: newTrustedContactBody(contact),
	})
}

func (h *Handler) deleteTrustedContact(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustedContactError(c, ErrUnauthorized)
	}

	if err := h.service.RemoveTrustedContact(c.UserContext(), user.ID, c.Params("id")); err != nil {
		return writeTrustedContactError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "deleted",
	})
}

func writeTrustedContactError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidContactName), errors.Is(err, ErrInvalidPhone), errors.Is(err, ErrInvalidRequestID), errors.Is(err, ErrInvalidRequestToken):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrRequestNotFound), errors.Is(err, ErrTrustedContactNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrPendingRequestExists), errors.Is(err, ErrTrustedContactExists), errors.Is(err, ErrRequestAlreadyProcessed), errors.Is(err, ErrRequestExpired):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func newTrustedContactRequestBody(request *TrustedContactRequest) trustedContactRequestBody {
	return trustedContactRequestBody{
		ID:          request.ID,
		UserID:      request.UserID,
		Name:        request.Name,
		Phone:       request.Phone,
		Email:       request.Email,
		Status:      string(request.Status),
		ExpiresAt:   request.ExpiresAt,
		RespondedAt: request.RespondedAt,
		CreatedAt:   request.CreatedAt,
	}
}

func newTrustedContactBody(contact *TrustedContact) trustedContactBody {
	return trustedContactBody{
		ID:         contact.ID,
		UserID:     contact.UserID,
		RequestID:  contact.RequestID,
		Name:       contact.Name,
		Phone:      contact.Phone,
		Email:      contact.Email,
		AcceptedAt: contact.AcceptedAt,
		CreatedAt:  contact.CreatedAt,
	}
}
