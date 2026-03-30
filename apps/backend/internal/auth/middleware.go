package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type Middleware struct {
	service  *Service
	sessions *SessionManager
}

func NewMiddleware(service *Service, sessions *SessionManager) *Middleware {
	return &Middleware{
		service:  service,
		sessions: sessions,
	}
}

func (m *Middleware) VerifyUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessToken, err := m.sessions.AccessTokenFromCookies(c)
		if err != nil {
			return writeAuthError(c, err)
		}

		claims, err := m.sessions.ParseAccessToken(accessToken)
		if err != nil {
			return writeAuthError(c, err)
		}

		user, err := m.service.GetUserByID(c.UserContext(), claims.Subject)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return writeAuthError(c, ErrUnauthorized)
			}

			return writeAuthError(c, err)
		}

		c.Locals(currentUserKey, user)
		return c.Next()
	}
}

func CurrentUser(c *fiber.Ctx) (*User, bool) {
	user, ok := c.Locals(currentUserKey).(*User)
	if !ok || user == nil {
		return nil, false
	}

	return user, true
}
