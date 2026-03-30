package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

const currentUserKey = "current_user"

type Handler struct {
	service  *Service
	sessions *SessionManager
}

type authRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	User userResponse `json:"user"`
}

type userResponse struct {
	ID         string  `json:"id"`
	Phone      string  `json:"phone"`
	Email      *string `json:"email,omitempty"`
	TrustScore float64 `json:"trust_score"`
	Verified   bool    `json:"verified"`
}

func NewHandler(service *Service, sessions *SessionManager) *Handler {
	return &Handler{
		service:  service,
		sessions: sessions,
	}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	authRoutes := router.Group("/auth")
	authRoutes.Post("/register", h.register)
	authRoutes.Post("/login", h.login)
	authRoutes.Post("/refresh", h.refresh)
	authRoutes.Post("/logout", h.logout)
	authRoutes.Get("/me", h.requireAuth, h.me)
}

func (h *Handler) register(c *fiber.Ctx) error {
	var req authRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	user, err := h.service.Register(c.UserContext(), RegisterInput{
		Phone:    req.Phone,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return h.writeAuthError(c, err)
	}

	return h.signIn(c, user, fiber.StatusCreated)
}

func (h *Handler) login(c *fiber.Ctx) error {
	var req authRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	user, err := h.service.Login(c.UserContext(), LoginInput{
		Phone:    req.Phone,
		Password: req.Password,
	})
	if err != nil {
		return h.writeAuthError(c, err)
	}

	return h.signIn(c, user, fiber.StatusOK)
}

func (h *Handler) refresh(c *fiber.Ctx) error {
	refreshToken, err := h.sessions.RefreshTokenFromCookies(c)
	if err != nil {
		return h.writeAuthError(c, err)
	}

	claims, err := h.sessions.ParseRefreshToken(refreshToken)
	if err != nil {
		h.sessions.ClearSessionCookies(c)
		return h.writeAuthError(c, err)
	}

	user, err := h.service.GetUserByID(c.UserContext(), claims.Subject)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.sessions.ClearSessionCookies(c)
		}

		return h.writeAuthError(c, err)
	}

	return h.signIn(c, user, fiber.StatusOK)
}

func (h *Handler) logout(c *fiber.Ctx) error {
	h.sessions.ClearSessionCookies(c)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "logged_out",
	})
}

func (h *Handler) me(c *fiber.Ctx) error {
	user, ok := c.Locals(currentUserKey).(*User)
	if !ok || user == nil {
		return h.writeAuthError(c, ErrUnauthorized)
	}

	return c.Status(fiber.StatusOK).JSON(authResponse{
		User: newUserResponse(user),
	})
}

func (h *Handler) requireAuth(c *fiber.Ctx) error {
	accessToken, err := h.sessions.AccessTokenFromCookies(c)
	if err != nil {
		return h.writeAuthError(c, err)
	}

	claims, err := h.sessions.ParseAccessToken(accessToken)
	if err != nil {
		return h.writeAuthError(c, err)
	}

	user, err := h.service.GetUserByID(c.UserContext(), claims.Subject)
	if err != nil {
		return h.writeAuthError(c, err)
	}

	c.Locals(currentUserKey, user)
	return c.Next()
}

func (h *Handler) signIn(c *fiber.Ctx, user *User, status int) error {
	pair, err := h.sessions.IssuePair(*user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to issue auth session",
		})
	}

	h.sessions.SetSessionCookies(c, pair)

	return c.Status(status).JSON(authResponse{
		User: newUserResponse(user),
	})
}

func (h *Handler) writeAuthError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidPhone), errors.Is(err, ErrInvalidPassword):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrUnauthorized), errors.Is(err, ErrUserNotFound):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrUserAlreadyExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func newUserResponse(user *User) userResponse {
	return userResponse{
		ID:         user.ID,
		Phone:      user.Phone,
		Email:      user.Email,
		TrustScore: user.TrustScore,
		Verified:   user.Verified,
	}
}
