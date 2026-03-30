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

	if report["trust_score"] != 0.3 {
		t.Fatalf("expected trust_score 0.3, got %#v", report["trust_score"])
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

func TestCreateReportRejectsUnsupportedType(t *testing.T) {
	application := newReportsTestApp(t)

	registerResp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    "+91 77777 11111",
		"password": "supersecret",
	}, nil)

	accessCookie := findReportsCookie(t, registerResp, "test_access")

	resp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/reports", map[string]any{
		"type":        "noise_complaint",
		"description": "unsupported type",
		"lat":         12.9716,
		"lng":         77.5946,
	}, []*http.Cookie{accessCookie})

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetReportByIDReturnsEvidenceIDs(t *testing.T) {
	application := newReportsTestApp(t)
	reportID := seedReportForTest(t, application, "harassment", 12.9716, 77.5946)

	resp := performReportsJSONRequest(t, application, http.MethodGet, "/api/v1/reports/"+reportID, nil, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := decodeReportsBody(t, resp)
	report := body["report"].(map[string]any)
	evidenceIDs := report["evidence_ids"].([]any)
	if len(evidenceIDs) != 1 || evidenceIDs[0] != "evidence-1" {
		t.Fatalf("expected evidence_ids to contain seeded evidence, got %#v", report["evidence_ids"])
	}

	if report["trust_score"] != 0.3 {
		t.Fatalf("expected trust_score 0.3, got %#v", report["trust_score"])
	}
}

func TestGetReportByIDReturnsNotFound(t *testing.T) {
	application := newReportsTestApp(t)

	resp := performReportsJSONRequest(t, application, http.MethodGet, "/api/v1/reports/missing-report", nil, nil)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestListNearbyReportsReturnsPaginatedResults(t *testing.T) {
	application := newReportsTestApp(t)

	seedReportForTest(t, application, "harassment", 12.9716, 77.5946)
	seedReportForTest(t, application, "unsafe_area", 12.9717, 77.5947)

	resp := performReportsJSONRequest(t, application, http.MethodGet, "/api/v1/reports?lat=12.9716&lng=77.5946&radius=500&limit=1&offset=0", nil, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := decodeReportsBody(t, resp)
	reportsBody := body["reports"].([]any)
	if len(reportsBody) != 1 {
		t.Fatalf("expected 1 paginated report, got %d", len(reportsBody))
	}

	if body["count"] != float64(2) {
		t.Fatalf("expected total count 2, got %#v", body["count"])
	}
}

func TestListNearbyReportsRejectsInvalidRadius(t *testing.T) {
	application := newReportsTestApp(t)

	resp := performReportsJSONRequest(t, application, http.MethodGet, "/api/v1/reports?lat=12.9716&lng=77.5946&radius=0", nil, nil)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestListNearbyReportsUsesConfiguredDefaultLimit(t *testing.T) {
	application := newReportsTestApp(t)

	seedReportForTest(t, application, "harassment", 12.9716, 77.5946)

	resp := performReportsJSONRequest(t, application, http.MethodGet, "/api/v1/reports?lat=12.9716&lng=77.5946&radius=500", nil, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body := decodeReportsBody(t, resp)
	if body["limit"] != float64(20) {
		t.Fatalf("expected default limit 20, got %#v", body["limit"])
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
		reports.NewService(newMemoryReportsRepository(), reports.ServiceConfig{
			DefaultNearbyLimit: 20,
			MaxNearbyLimit:     50,
			MaxNearbyRadiusM:   5000,
		}, nil),
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
	mu         sync.Mutex
	nextID     int
	reports    map[string]reports.StoredReport
	evidenceBy map[string][]string
}

func newMemoryReportsRepository() *memoryReportsRepository {
	return &memoryReportsRepository{
		reports:    make(map[string]reports.StoredReport),
		evidenceBy: make(map[string][]string),
	}
}

func (r *memoryReportsRepository) Create(_ context.Context, input reports.CreateParams) (*reports.StoredReport, error) {
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
	r.reports[reportID] = report

	if len(r.reports) == 1 {
		r.evidenceBy[reportID] = []string{"evidence-1"}
	}

	copyReport := report
	return &copyReport, nil
}

func (r *memoryReportsRepository) GetByID(_ context.Context, id string) (*reports.StoredReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	report, exists := r.reports[id]
	if !exists {
		return nil, reports.ErrReportNotFound
	}

	copyReport := report
	return &copyReport, nil
}

func (r *memoryReportsRepository) ListEvidenceIDs(_ context.Context, reportID string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	evidenceIDs := append([]string(nil), r.evidenceBy[reportID]...)
	return evidenceIDs, nil
}

func (r *memoryReportsRepository) ListNearby(_ context.Context, input reports.NearbyParams) ([]reports.NearbyReportRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows := make([]reports.NearbyReportRow, 0)
	for _, report := range r.reports {
		if distanceMeters(report.Latitude, report.Longitude, input.Latitude, input.Longitude) > input.Radius {
			continue
		}

		rows = append(rows, reports.NearbyReportRow{
			StoredReport:   report,
			DistanceMeters: distanceMeters(report.Latitude, report.Longitude, input.Latitude, input.Longitude),
		})
	}

	if input.Offset >= len(rows) {
		return []reports.NearbyReportRow{}, nil
	}

	end := input.Offset + input.Limit
	if end > len(rows) {
		end = len(rows)
	}

	return append([]reports.NearbyReportRow(nil), rows[input.Offset:end]...), nil
}

func (r *memoryReportsRepository) CountNearby(_ context.Context, input reports.NearbyParams) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int64
	for _, report := range r.reports {
		if distanceMeters(report.Latitude, report.Longitude, input.Latitude, input.Longitude) <= input.Radius {
			count++
		}
	}

	return count, nil
}

func seedReportForTest(t *testing.T, application *fiber.App, reportType string, lat, lng float64) string {
	t.Helper()

	registerResp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    fmt.Sprintf("+91 90000 %05d", time.Now().UnixNano()%100000),
		"password": "supersecret",
	}, nil)

	accessCookie := findReportsCookie(t, registerResp, "test_access")

	resp := performReportsJSONRequest(t, application, http.MethodPost, "/api/v1/reports", map[string]any{
		"type":        reportType,
		"description": "seed report",
		"lat":         lat,
		"lng":         lng,
	}, []*http.Cookie{accessCookie})

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected seeded report status 201, got %d", resp.StatusCode)
	}

	body := decodeReportsBody(t, resp)
	report := body["report"].(map[string]any)
	return report["id"].(string)
}

func distanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const metersPerDegree = 111320.0

	dLat := lat1 - lat2
	dLng := lng1 - lng2
	return metersPerDegree * sqrt(dLat*dLat+dLng*dLng)
}

func sqrt(value float64) float64 {
	if value == 0 {
		return 0
	}

	z := value
	for i := 0; i < 10; i++ {
		z -= (z*z - value) / (2 * z)
	}

	return z
}
