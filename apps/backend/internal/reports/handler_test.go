package reports_test

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
	"saferoute-backend/internal/reports"

	"github.com/gofiber/fiber/v2"
)

func TestCreateReportRequiresAuthentication(t *testing.T) {
	application := newReportsTestApp(t)

	resp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/reports", map[string]any{
		"type":        "harassment",
		"description": "man following me",
		"lat":         12.9716,
		"lng":         77.5946,
	}, nil)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCreateReportCreatesAuthenticatedReport(t *testing.T) {
	application := newReportsTestApp(t)

	registerResp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 99999 11111",
		"password": "supersecret",
	}, nil)

	accessCookie := findReportsCookie(t, registerResp, "test_access")

	resp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/reports", map[string]any{
		"type":        "harassment",
		"description": "man following me",
		"lat":         12.9716,
		"lng":         77.5946,
	}, []*http.Cookie{accessCookie})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	body := decodeReportsBody(t, resp)
	report := body["report"].(map[string]any)

	if report["type"] != "harassment" {
		t.Fatalf("expected report type harassment, got %#v", report["type"])
	}

	if report["lat"] != 12.9716 {
		t.Fatalf("expected latitude 12.9716, got %#v", report["lat"])
	}

	if report["lng"] != 77.5946 {
		t.Fatalf("expected longitude 77.5946, got %#v", report["lng"])
	}
}

func TestCreateReportRejectsInvalidCoordinates(t *testing.T) {
	application := newReportsTestApp(t)

	registerResp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 88888 11111",
		"password": "supersecret",
	}, nil)

	accessCookie := findReportsCookie(t, registerResp, "test_access")

	resp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/reports", map[string]any{
		"type":        "harassment",
		"description": "bad latitude",
		"lat":         123.45,
		"lng":         77.5946,
	}, []*http.Cookie{accessCookie})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func newReportsTestApp(t *testing.T) *fiber.App {
	t.Helper()

	authRepo := newMemoryAuthRepository()
	authService := auth.NewService(authRepo)

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

	authHandler := auth.NewHandler(authService, sessionManager)
	reportsHandler := reports.NewHandler(
		reports.NewService(newMemoryReportsRepository()),
		auth.NewMiddleware(authService, sessionManager).VerifyUser(),
	)

	return appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, authHandler.RegisterRoutes, reportsHandler.RegisterRoutes)
}

func performReportsJSONRequest(t *testing.T, application *fiber.App, method, path string, payload any, cookies []*http.Cookie) *http.Response {
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

func findReportsCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("cookie %q not found in response", name)
	return nil
}

func decodeReportsBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	return body
}

type memoryAuthRepository struct {
	mu      sync.Mutex
	nextID  int
	byID    map[string]*auth.User
	byPhone map[string]*auth.User
}

func newMemoryAuthRepository() *memoryAuthRepository {
	return &memoryAuthRepository{
		byID:    make(map[string]*auth.User),
		byPhone: make(map[string]*auth.User),
	}
}

func (r *memoryAuthRepository) CreateUser(_ context.Context, user *auth.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byPhone[user.Phone]; exists {
		return auth.ErrUserAlreadyExists
	}

	r.nextID++
	copyUser := *user
	copyUser.ID = fmt.Sprintf("user-%d", r.nextID)
	r.byID[copyUser.ID] = &copyUser
	r.byPhone[copyUser.Phone] = &copyUser
	user.ID = copyUser.ID

	return nil
}

func (r *memoryAuthRepository) GetUserByPhone(_ context.Context, phone string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byPhone[strings.Join(strings.Fields(phone), "")]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

func (r *memoryAuthRepository) GetUserByID(_ context.Context, id string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[id]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

type memoryReportsRepository struct {
	mu     sync.Mutex
	nextID int
}

func newMemoryReportsRepository() *memoryReportsRepository {
	return &memoryReportsRepository{}
}

func (r *memoryReportsRepository) Create(_ context.Context, input reports.CreateParams) (*reports.Report, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	reportID := fmt.Sprintf("report-%d", r.nextID)
	report := &reports.Report{
		ID:          reportID,
		UserID:      input.UserID,
		Category:    input.Category,
		Description: input.Description,
		OccurredAt:  input.OccurredAt,
		CreatedAt:   input.OccurredAt,
		Source:      input.Source,
	}

	return report, nil
}
