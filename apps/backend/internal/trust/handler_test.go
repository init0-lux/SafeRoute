package trust_test

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
	"saferoute-backend/internal/trust"

	"github.com/gofiber/fiber/v2"
)

func TestTrustMeRequiresAuthentication(t *testing.T) {
	application := newTrustTestApp(t)

	resp := performTrustJSONRequest(t, application, http.MethodGet, "/api/v1/trust/me", nil, nil)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestTrustMeReturnsBreakdown(t *testing.T) {
	application := newTrustTestApp(t)

	registerResp := performTrustJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 99999 11111",
		"password": "supersecret",
	}, nil)

	accessCookie := findTrustCookie(t, registerResp, "test_access")

	resp := performTrustJSONRequest(t, application, http.MethodGet, "/api/v1/trust/me", nil, []*http.Cookie{accessCookie})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := decodeTrustBody(t, resp)
	if body["score"] != 0.3 {
		t.Fatalf("expected score 0.3, got %#v", body["score"])
	}

	breakdown := body["breakdown"].(map[string]any)
	if breakdown["base"] != 0.3 {
		t.Fatalf("expected base 0.3, got %#v", breakdown["base"])
	}
}

func TestTrustVerifyUpdatesVerifiedFlag(t *testing.T) {
	application := newTrustTestApp(t)

	registerResp := performTrustJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 88888 11111",
		"password": "supersecret",
	}, nil)

	accessCookie := findTrustCookie(t, registerResp, "test_access")

	resp := performTrustJSONRequest(t, application, http.MethodPost, "/api/v1/trust/verify", map[string]bool{
		"verified": true,
	}, []*http.Cookie{accessCookie})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := decodeTrustBody(t, resp)
	if body["verified"] != true {
		t.Fatalf("expected verified true, got %#v", body["verified"])
	}
	if body["score"] != 0.45 {
		t.Fatalf("expected score 0.45, got %#v", body["score"])
	}
}

func TestReportCreationUpdatesTrustCounts(t *testing.T) {
	application := newTrustTestApp(t)

	registerResp := performTrustJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 77777 11111",
		"password": "supersecret",
	}, nil)

	accessCookie := findTrustCookie(t, registerResp, "test_access")

	createResp := performTrustJSONRequest(t, application, http.MethodPost, "/api/v1/reports", map[string]any{
		"type":        "harassment",
		"description": "report for trust hook",
		"lat":         12.9716,
		"lng":         77.5946,
	}, []*http.Cookie{accessCookie})

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected report create status 201, got %d", createResp.StatusCode)
	}

	trustResp := performTrustJSONRequest(t, application, http.MethodGet, "/api/v1/trust/me", nil, []*http.Cookie{accessCookie})
	if trustResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trust me status 200, got %d", trustResp.StatusCode)
	}

	body := decodeTrustBody(t, trustResp)
	if body["reports_count"] != float64(1) {
		t.Fatalf("expected reports_count 1, got %#v", body["reports_count"])
	}
	if body["score"] != 0.3 {
		t.Fatalf("expected score to remain 0.3 after first report, got %#v", body["score"])
	}

	breakdown := body["breakdown"].(map[string]any)
	if _, exists := breakdown["reports_bonus"]; exists {
		t.Fatalf("expected reports_bonus to be absent, got %#v", breakdown["reports_bonus"])
	}
}

func newTrustTestApp(t *testing.T) *fiber.App {
	t.Helper()

	authRepo := newMemoryTrustAuthRepository()
	authService := auth.NewService(authRepo)
	trustService := trust.NewService(authRepo)

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
	authMiddleware := auth.NewMiddleware(authService, sessionManager)
	reportsHandler := reports.NewHandler(
		reports.NewService(newMemoryTrustReportsRepository(), reports.ServiceConfig{
			DefaultNearbyLimit: 20,
			MaxNearbyLimit:     50,
			MaxNearbyRadiusM:   5000,
		}, trustService),
		authMiddleware.VerifyUser(),
	)
	trustHandler := trust.NewHandler(trustService, authMiddleware.VerifyUser())

	return appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, authHandler.RegisterRoutes, reportsHandler.RegisterRoutes, trustHandler.RegisterRoutes)
}

func performTrustJSONRequest(t *testing.T, application *fiber.App, method, path string, payload any, cookies []*http.Cookie) *http.Response {
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

func findTrustCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("cookie %q not found in response", name)
	return nil
}

func decodeTrustBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	return body
}

type memoryTrustAuthRepository struct {
	mu      sync.Mutex
	nextID  int
	byID    map[string]*auth.User
	byPhone map[string]*auth.User
}

func newMemoryTrustAuthRepository() *memoryTrustAuthRepository {
	return &memoryTrustAuthRepository{
		byID:    make(map[string]*auth.User),
		byPhone: make(map[string]*auth.User),
	}
}

func (r *memoryTrustAuthRepository) CreateUser(_ context.Context, user *auth.User) error {
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

func (r *memoryTrustAuthRepository) GetUserByPhone(_ context.Context, phone string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byPhone[strings.Join(strings.Fields(phone), "")]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

func (r *memoryTrustAuthRepository) GetUserByID(_ context.Context, id string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[id]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

func (r *memoryTrustAuthRepository) GetByUserID(ctx context.Context, id string) (*auth.User, error) {
	return r.GetUserByID(ctx, id)
}

func (r *memoryTrustAuthRepository) IncrementReportCount(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[userID]
	if !exists {
		return auth.ErrUserNotFound
	}

	user.ReportCount++
	return nil
}

func (r *memoryTrustAuthRepository) SetVerificationStatus(_ context.Context, userID string, verified bool, verifiedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[userID]
	if !exists {
		return auth.ErrUserNotFound
	}

	user.Verified = verified
	user.VerifiedAt = verifiedAt
	return nil
}

func (r *memoryTrustAuthRepository) UpdateTrustScore(_ context.Context, userID string, score float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[userID]
	if !exists {
		return auth.ErrUserNotFound
	}

	user.TrustScore = score
	return nil
}

type memoryTrustReportsRepository struct {
	mu     sync.Mutex
	nextID int
}

func newMemoryTrustReportsRepository() *memoryTrustReportsRepository {
	return &memoryTrustReportsRepository{}
}

func (r *memoryTrustReportsRepository) Create(_ context.Context, input reports.CreateParams) (*reports.StoredReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	reportID := fmt.Sprintf("report-%d", r.nextID)
	report := reports.StoredReport{
		ID:          reportID,
		UserID:      input.UserID,
		Category:    input.Category,
		Description: input.Description,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		OccurredAt:  input.OccurredAt,
		CreatedAt:   input.OccurredAt,
		Source:      input.Source,
		TrustScore:  0.3,
	}

	return &report, nil
}

func (r *memoryTrustReportsRepository) GetByID(_ context.Context, _ string) (*reports.StoredReport, error) {
	return nil, reports.ErrReportNotFound
}

func (r *memoryTrustReportsRepository) ListEvidenceIDs(_ context.Context, _ string) ([]string, error) {
	return []string{}, nil
}

func (r *memoryTrustReportsRepository) ListNearby(_ context.Context, _ reports.NearbyParams) ([]reports.NearbyReportRow, error) {
	return []reports.NearbyReportRow{}, nil
}

func (r *memoryTrustReportsRepository) CountNearby(_ context.Context, _ reports.NearbyParams) (int64, error) {
	return 0, nil
}
