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
		// Try cookie first, then fall back to Authorization: Bearer header.
		accessToken, err := m.sessions.AccessTokenFromCookies(c)
		if err != nil {
			accessToken, err = m.sessions.AccessTokenFromHeader(c)
			if err != nil {
				return writeAuthError(c, err)
			}
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

// OptionalUser tries to authenticate the user but doesn't fail if no auth is present
func (m *Middleware) OptionalUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try cookie first, then fall back to Authorization: Bearer header.
		accessToken, err := m.sessions.AccessTokenFromCookies(c)
		if err != nil {
			accessToken, err = m.sessions.AccessTokenFromHeader(c)
			if err != nil {
				// No auth present, continue without user
				return c.Next()
			}
		}

		claims, err := m.sessions.ParseAccessToken(accessToken)
		if err != nil {
			// Invalid token, continue without user
			return c.Next()
		}

		user, err := m.service.GetUserByID(c.UserContext(), claims.Subject)
		if err == nil {
			// Valid user, set in context
			c.Locals(currentUserKey, user)
		}

		// Continue regardless of user status
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
