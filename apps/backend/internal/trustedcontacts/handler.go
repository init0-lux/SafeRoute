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

// pendingRequestBody is for incoming requests to the current user
type pendingRequestBody struct {
	ID             string     `json:"id"`
	RequesterID    string     `json:"requester_id"`
	RequesterName  string     `json:"requester_name"`
	RequesterPhone string     `json:"requester_phone"`
	Name           string     `json:"name"`
	Phone          string     `json:"phone"`
	Status         string     `json:"status"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	AcceptToken    string     `json:"accept_token,omitempty"`
}

func NewHandler(service *Service, authMiddleware *auth.Middleware) *Handler {
	return &Handler{
		service: service,
		auth:    authMiddleware,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/trusted-contacts")
	group.Get("/", h.auth.VerifyUser(), h.listContacts)
	group.Post("/requests", h.auth.VerifyUser(), h.createRequest)
	group.Get("/requests/pending", h.auth.VerifyUser(), h.listPendingRequests)
	group.Get("/requests/outgoing", h.auth.VerifyUser(), h.listOutgoingRequests)
	group.Post("/requests/:id/accept", h.auth.OptionalUser(), h.acceptRequest)
	group.Post("/requests/:id/reject", h.auth.VerifyUser(), h.rejectRequest)
	group.Delete("/:id", h.auth.VerifyUser(), h.deleteTrustedContact)
}

func (h *Handler) listContacts(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustedContactError(c, ErrUnauthorized)
	}

	contacts, err := h.service.ListTrustedContacts(c.UserContext(), user.ID)
	if err != nil {
		return writeTrustedContactError(c, err)
	}

	response := make([]trustedContactBody, len(contacts))
	for i, contact := range contacts {
		response[i] = newTrustedContactBody(&contact)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"contacts": response,
	})
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
	// Try authenticated accept first (if user is logged in and it's their request)
	if user, ok := auth.CurrentUser(c); ok {
		request, contact, err := h.service.AcceptRequestByPhone(c.UserContext(), c.Params("id"), user.Phone)
		if err == nil {
			return c.Status(fiber.StatusCreated).JSON(trustedContactAcceptedResponse{
				Request: newTrustedContactRequestBody(request),
				Contact: newTrustedContactBody(contact),
			})
		}
		// For authenticated users, return the error directly
		// (ErrUnauthorized = not their request, other errors = request state issues)
		return writeTrustedContactError(c, err)
	}

	// Token-based accept for unauthenticated users
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

func (h *Handler) listPendingRequests(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustedContactError(c, ErrUnauthorized)
	}

	requests, err := h.service.ListPendingRequestsForUser(c.UserContext(), user.Phone)
	if err != nil {
		return writeTrustedContactError(c, err)
	}

	response := make([]pendingRequestBody, len(requests))
	for i, req := range requests {
		response[i] = newPendingRequestBody(&req)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"requests": response,
	})
}

func (h *Handler) listOutgoingRequests(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustedContactError(c, ErrUnauthorized)
	}

	requests, err := h.service.ListOutgoingRequests(c.UserContext(), user.ID)
	if err != nil {
		return writeTrustedContactError(c, err)
	}

	response := make([]trustedContactRequestBody, len(requests))
	for i, req := range requests {
		response[i] = newTrustedContactRequestBody(&req)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"requests": response,
	})
}

func (h *Handler) rejectRequest(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return writeTrustedContactError(c, ErrUnauthorized)
	}

	request, err := h.service.RejectRequest(c.UserContext(), c.Params("id"), user.Phone)
	if err != nil {
		return writeTrustedContactError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"request": newTrustedContactRequestBody(request),
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
	case errors.Is(err, ErrInvalidContactName), errors.Is(err, ErrInvalidPhone), errors.Is(err, ErrInvalidRequestID), errors.Is(err, ErrInvalidRequestToken), errors.Is(err, ErrContactNotRegistered):
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

func newPendingRequestBody(request *TrustedContactRequest) pendingRequestBody {
	requesterName := request.Name // fallback to target name if User not loaded
	if request.User.Username != "" {
		requesterName = request.User.Username
	}
	
	return pendingRequestBody{
		ID:             request.ID,
		RequesterID:    request.UserID,
		RequesterName:  requesterName,
		RequesterPhone: "",            // We don't expose requester's phone for privacy
		Name:           request.Name,  // Target user's username
		Phone:          request.Phone, // Target user's phone
		Status:         string(request.Status),
		ExpiresAt:      request.ExpiresAt,
		CreatedAt:      request.CreatedAt,
	}
}
