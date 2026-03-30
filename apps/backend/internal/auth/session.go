package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type SessionConfig struct {
	AccessSecret      string
	RefreshSecret     string
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
	AccessCookieName  string
	RefreshCookieName string
	CookieDomain      string
	CookieSameSite    string
	CookieSecure      bool
}

type SessionClaims struct {
	TokenType string `json:"token_type"`
	Phone     string `json:"phone,omitempty"`
	jwt.RegisteredClaims
}

type SessionPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type SessionManager struct {
	accessSecret      []byte
	refreshSecret     []byte
	accessTTL         time.Duration
	refreshTTL        time.Duration
	accessCookieName  string
	refreshCookieName string
	cookieDomain      string
	cookieSameSite    string
	cookieSecure      bool
}

func NewSessionManager(cfg SessionConfig) (*SessionManager, error) {
	if strings.TrimSpace(cfg.AccessSecret) == "" || strings.TrimSpace(cfg.RefreshSecret) == "" {
		return nil, errors.New("jwt secrets are required")
	}

	if cfg.AccessTTL <= 0 || cfg.RefreshTTL <= 0 {
		return nil, errors.New("jwt ttl values must be greater than zero")
	}

	return &SessionManager{
		accessSecret:      []byte(cfg.AccessSecret),
		refreshSecret:     []byte(cfg.RefreshSecret),
		accessTTL:         cfg.AccessTTL,
		refreshTTL:        cfg.RefreshTTL,
		accessCookieName:  cfg.AccessCookieName,
		refreshCookieName: cfg.RefreshCookieName,
		cookieDomain:      cfg.CookieDomain,
		cookieSameSite:    cfg.CookieSameSite,
		cookieSecure:      cfg.CookieSecure,
	}, nil
}

func (m *SessionManager) IssuePair(user User) (SessionPair, error) {
	accessToken, accessExpiresAt, err := m.issueToken(user, tokenTypeAccess, m.accessTTL, m.accessSecret)
	if err != nil {
		return SessionPair{}, err
	}

	refreshToken, refreshExpiresAt, err := m.issueToken(user, tokenTypeRefresh, m.refreshTTL, m.refreshSecret)
	if err != nil {
		return SessionPair{}, err
	}

	return SessionPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (m *SessionManager) ParseAccessToken(token string) (*SessionClaims, error) {
	return m.parseToken(token, tokenTypeAccess, m.accessSecret)
}

func (m *SessionManager) ParseRefreshToken(token string) (*SessionClaims, error) {
	return m.parseToken(token, tokenTypeRefresh, m.refreshSecret)
}

func (m *SessionManager) SetSessionCookies(c *fiber.Ctx, pair SessionPair) {
	c.Cookie(m.newCookie(m.accessCookieName, pair.AccessToken, pair.AccessExpiresAt))
	c.Cookie(m.newCookie(m.refreshCookieName, pair.RefreshToken, pair.RefreshExpiresAt))
}

func (m *SessionManager) ClearSessionCookies(c *fiber.Ctx) {
	expiredAt := time.Unix(0, 0).UTC()
	c.Cookie(m.newCookie(m.accessCookieName, "", expiredAt))
	c.Cookie(m.newCookie(m.refreshCookieName, "", expiredAt))
}

func (m *SessionManager) AccessTokenFromCookies(c *fiber.Ctx) (string, error) {
	token := strings.TrimSpace(c.Cookies(m.accessCookieName))
	if token == "" {
		return "", ErrUnauthorized
	}

	return token, nil
}

func (m *SessionManager) RefreshTokenFromCookies(c *fiber.Ctx) (string, error) {
	token := strings.TrimSpace(c.Cookies(m.refreshCookieName))
	if token == "" {
		return "", ErrUnauthorized
	}

	return token, nil
}

func (m *SessionManager) issueToken(user User, tokenType string, ttl time.Duration, secret []byte) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	claims := SessionClaims{
		TokenType: tokenType,
		Phone:     user.Phone,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func (m *SessionManager) parseToken(tokenString, expectedType string, secret []byte) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnauthorized
		}

		return secret, nil
	})
	if err != nil {
		return nil, ErrUnauthorized
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return nil, ErrUnauthorized
	}

	if claims.TokenType != expectedType || strings.TrimSpace(claims.Subject) == "" {
		return nil, ErrUnauthorized
	}

	return claims, nil
}

func (m *SessionManager) newCookie(name, value string, expiresAt time.Time) *fiber.Cookie {
	return &fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   m.cookieDomain,
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   m.cookieSecure,
		SameSite: m.cookieSameSite,
	}
}
