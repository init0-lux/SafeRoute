package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"saferoute-backend/config"
	appcore "saferoute-backend/internal/app"
	"saferoute-backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

func TestRegisterSetsSessionCookiesAndAllowsMe(t *testing.T) {
	application := newTestApp(t)

	registerResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 99999 11111",
		"email":    "person@example.com",
		"password": "supersecret",
	}, nil)

	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResp.StatusCode)
	}

	accessCookie := findCookie(t, registerResp, "test_access")
	if accessCookie.Value == "" {
		t.Fatal("expected access cookie to be set")
	}

	refreshCookie := findCookie(t, registerResp, "test_refresh")
	if refreshCookie.Value == "" {
		t.Fatal("expected refresh cookie to be set")
	}

	body := decodeBody(t, registerResp)
	user := body["user"].(map[string]any)
	if user["phone"] != "+919999911111" {
		t.Fatalf("expected normalized phone in response, got %#v", user["phone"])
	}

	meResp := performJSONRequest(t, application, http.MethodGet, "/api/v1/auth/me", nil, []*http.Cookie{accessCookie})
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("expected me status 200, got %d", meResp.StatusCode)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	repo := newMemoryRepository()
	service := auth.NewService(repo)
	if _, err := service.Register(context.Background(), auth.RegisterInput{
		Phone:    "+919999900000",
		Password: "valid-password",
	}); err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	application := newTestAppWithRepo(t, repo)
	loginResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"phone":    "+919999900000",
		"password": "wrong-password",
	}, nil)

	if loginResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected login status 401, got %d", loginResp.StatusCode)
	}
}

func TestRefreshReissuesSessionCookies(t *testing.T) {
	application := newTestApp(t)

	registerResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+919888877777",
		"password": "supersecret",
	}, nil)

	refreshResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/auth/refresh", nil, []*http.Cookie{
		findCookie(t, registerResp, "test_refresh"),
	})

	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d", refreshResp.StatusCode)
	}

	if findCookie(t, refreshResp, "test_access").Value == "" {
		t.Fatal("expected refreshed access cookie")
	}

	if findCookie(t, refreshResp, "test_refresh").Value == "" {
		t.Fatal("expected refreshed refresh cookie")
	}
}

func TestVerifyUserMiddlewareProtectsReusableRoute(t *testing.T) {
	repo := newMemoryRepository()
	service := auth.NewService(repo)
	user, err := service.Register(context.Background(), auth.RegisterInput{
		Phone:    "+919777766666",
		Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	sessionManager, err := auth.NewSessionManager(auth.SessionConfig{
		AccessSecret:      "access-secret",
		RefreshSecret:     "refresh-secret",
		AccessTTL:         15 * time.Minute,
		RefreshTTL:        7 * 24 * time.Hour,
		AccessCookieName:  "test_access",
		RefreshCookieName: "test_refresh",
		CookieSameSite:    "Lax",
	})
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	pair, err := sessionManager.IssuePair(*user)
	if err != nil {
		t.Fatalf("failed to issue session pair: %v", err)
	}

	authMiddleware := auth.NewMiddleware(service, sessionManager)
	application := fiber.New()
	application.Get("/protected", authMiddleware.VerifyUser(), func(c *fiber.Ctx) error {
		currentUser, ok := auth.CurrentUser(c)
		if !ok {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"phone": currentUser.Phone,
		})
	})

	resp := performJSONRequest(t, application, http.MethodGet, "/protected", nil, []*http.Cookie{
		{Name: "test_access", Value: pair.AccessToken},
	})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected protected route status 200, got %d", resp.StatusCode)
	}

	body := decodeBody(t, resp)
	if body["phone"] != "+919777766666" {
		t.Fatalf("expected protected route to expose verified user, got %#v", body["phone"])
	}
}

func newTestApp(t *testing.T) *fiber.App {
	t.Helper()
	repo := newMemoryRepository()
	return buildApp(t, repo)
}

func newTestAppWithRepo(t *testing.T, repo *memoryRepository) *fiber.App {
	t.Helper()
	return buildApp(t, repo)
}

func buildApp(t *testing.T, repo *memoryRepository) *fiber.App {
	t.Helper()

	sessionManager, err := auth.NewSessionManager(auth.SessionConfig{
		AccessSecret:      "access-secret",
		RefreshSecret:     "refresh-secret",
		AccessTTL:         15 * time.Minute,
		RefreshTTL:        7 * 24 * time.Hour,
		AccessCookieName:  "test_access",
		RefreshCookieName: "test_refresh",
		CookieSameSite:    "Lax",
	})
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	handler := auth.NewHandler(auth.NewService(repo), sessionManager)
	app := appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, handler.RegisterRoutes)

	return app
}

func performJSONRequest(t *testing.T, application *fiber.App, method, path string, payload any, cookies []*http.Cookie) *http.Response {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("failed to encode request: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	return resp
}

func findCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("cookie %q not found in response", name)
	return nil
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	return body
}

type memoryRepository struct {
	mu         sync.Mutex
	nextUserID int
	byID       map[string]*auth.User
	byPhone    map[string]*auth.User
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		byID:    make(map[string]*auth.User),
		byPhone: make(map[string]*auth.User),
	}
}

func (r *memoryRepository) CreateUser(_ context.Context, user *auth.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byPhone[user.Phone]; exists {
		return auth.ErrUserAlreadyExists
	}

	r.nextUserID++
	copyUser := *user
	copyUser.ID = fmt.Sprintf("user-%d", r.nextUserID)
	r.byID[copyUser.ID] = &copyUser
	r.byPhone[copyUser.Phone] = &copyUser
	user.ID = copyUser.ID

	return nil
}

func (r *memoryRepository) GetUserByPhone(_ context.Context, phone string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byPhone[strings.Join(strings.Fields(phone), "")]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

func (r *memoryRepository) GetUserByID(_ context.Context, id string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[id]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

func (r *memoryRepository) UpdatePushToken(_ context.Context, id string, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[id]
	if !exists {
		return auth.ErrUserNotFound
	}

	user.ExpoPushToken = &token
	return nil
}
