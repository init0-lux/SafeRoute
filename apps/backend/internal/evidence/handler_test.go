package evidence_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"saferoute-backend/config"
	appcore "saferoute-backend/internal/app"
	"saferoute-backend/internal/auth"
	"saferoute-backend/internal/evidence"
	"saferoute-backend/internal/reports"

	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
)

const (
	testReportID  = "11111111-1111-1111-1111-111111111111"
	testSessionID = "22222222-2222-2222-2222-222222222222"
)

func TestUploadEvidenceRequiresAuthentication(t *testing.T) {
	application := newEvidenceTestApp(t)

	body, contentType := buildMultipartEvidenceRequest(t, "file", "photo.png", "image/png", pngFixture(), map[string]string{
		"report_id": testReportID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/upload", body)
	req.Header.Set("Content-Type", contentType)

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestUploadEvidenceToReportSucceeds(t *testing.T) {
	application := newEvidenceTestApp(t)
	accessCookie := registerEvidenceUser(t, application, "+91 99999 11111")

	body, contentType := buildMultipartEvidenceRequest(t, "file", "photo.png", "image/png", pngFixture(), map[string]string{
		"report_id": testReportID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(accessCookie)

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	payload := decodeEvidenceBody(t, resp)
	evidenceBody := payload["evidence"].(map[string]any)
	if evidenceBody["type"] != "image" {
		t.Fatalf("expected type image, got %#v", evidenceBody["type"])
	}
	if evidenceBody["media_type"] != "image/png" {
		t.Fatalf("expected media_type image/png, got %#v", evidenceBody["media_type"])
	}
}

func TestUploadEvidenceRejectsUnsupportedMime(t *testing.T) {
	application := newEvidenceTestApp(t)
	accessCookie := registerEvidenceUser(t, application, "+91 88888 11111")

	body, contentType := buildMultipartEvidenceRequest(t, "file", "note.txt", "text/plain", []byte("hello"), map[string]string{
		"report_id": testReportID,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(accessCookie)

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestUploadEvidenceRejectsMissingParent(t *testing.T) {
	application := newEvidenceTestApp(t)
	accessCookie := registerEvidenceUser(t, application, "+91 77777 11111")

	body, contentType := buildMultipartEvidenceRequest(t, "file", "photo.png", "image/png", pngFixture(), nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(accessCookie)

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestUploadEvidenceRejectsInvalidReportID(t *testing.T) {
	application := newEvidenceTestApp(t)
	accessCookie := registerEvidenceUser(t, application, "+91 77777 22222")

	body, contentType := buildMultipartEvidenceRequest(t, "file", "photo.png", "image/png", pngFixture(), map[string]string{
		"report_id": "report-123",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/upload", body)
	req.Header.Set("Content-Type", contentType)
	req.AddCookie(accessCookie)

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetEvidenceReturnsMetadataAndContent(t *testing.T) {
	application := newEvidenceTestApp(t)
	accessCookie := registerEvidenceUser(t, application, "+91 66666 11111")

	body, contentType := buildMultipartEvidenceRequest(t, "file", "photo.png", "image/png", pngFixture(), map[string]string{
		"session_id": testSessionID,
	})
	uploadReq := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/upload", body)
	uploadReq.Header.Set("Content-Type", contentType)
	uploadReq.AddCookie(accessCookie)

	uploadResp, err := application.Test(uploadReq)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if uploadResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d", uploadResp.StatusCode)
	}

	payload := decodeEvidenceBody(t, uploadResp)
	evidenceBody := payload["evidence"].(map[string]any)
	evidenceID := evidenceBody["id"].(string)

	metaReq := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/"+evidenceID, nil)
	metaReq.AddCookie(accessCookie)
	metaResp, err := application.Test(metaReq)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if metaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected metadata status 200, got %d", metaResp.StatusCode)
	}

	contentReq := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/"+evidenceID+"/content", nil)
	contentReq.AddCookie(accessCookie)
	contentResp, err := application.Test(contentReq)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if contentResp.StatusCode != http.StatusOK {
		t.Fatalf("expected content status 200, got %d", contentResp.StatusCode)
	}

	if contentResp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("expected content-type image/png, got %q", contentResp.Header.Get("Content-Type"))
	}
}

func TestGetEvidenceRejectsInvalidID(t *testing.T) {
	application := newEvidenceTestApp(t)
	accessCookie := registerEvidenceUser(t, application, "+91 66666 22222")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/evidence-123", nil)
	req.AddCookie(accessCookie)

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func newEvidenceTestApp(t *testing.T) *fiber.App {
	t.Helper()

	root := t.TempDir()
	authRepo := newMemoryEvidenceAuthRepository()
	authService := auth.NewService(authRepo)
	reportsLookup := &stubReportLookup{knownIDs: map[string]bool{testReportID: true}}
	sessionsLookup := &stubSessionLookup{knownIDs: map[string]bool{testSessionID: true}}
	evidenceHandler := evidence.NewHandler(
		evidence.NewService(
			newMemoryEvidenceRepository(),
			evidence.NewLocalStorage(root),
			reportsLookup,
			sessionsLookup,
			evidence.ServiceConfig{MaxFileSizeBytes: 1024 * 1024},
		),
		auth.NewMiddleware(authService, mustSessionManager(t)).VerifyUser(),
	)
	authHandler := auth.NewHandler(authService, mustSessionManager(t))

	return appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, authHandler.RegisterRoutes, evidenceHandler.RegisterRoutes)
}

func mustSessionManager(t *testing.T) *auth.SessionManager {
	t.Helper()
	manager, err := auth.NewSessionManager(auth.SessionConfig{
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

	return manager
}

func registerEvidenceUser(t *testing.T, application *fiber.App, phone string) *http.Cookie {
	t.Helper()
	resp := performEvidenceJSONRequest(t, application, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"phone":    phone,
		"password": "supersecret",
	}, nil)

	return findEvidenceCookie(t, resp, "test_access")
}

func performEvidenceJSONRequest(t *testing.T, application *fiber.App, method, path string, payload any, cookies []*http.Cookie) *http.Response {
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

func buildMultipartEvidenceRequest(t *testing.T, fieldName, filename, contentType string, data []byte, values map[string]string) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range values {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("failed to write field: %v", err)
		}
	}

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	_ = contentType
	return body, writer.FormDataContentType()
}

func decodeEvidenceBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	return body
}

func pngFixture() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	}
}


func findEvidenceCookie(t *testing.T, resp *http.Response, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("cookie %q not found in response", name)
	return nil
}

type stubReportLookup struct {
	knownIDs map[string]bool
}

func (s *stubReportLookup) GetByID(_ context.Context, id string) (*reports.ReportDetails, error) {
	if !s.knownIDs[id] {
		return nil, os.ErrNotExist
	}

	return &reports.ReportDetails{ID: id}, nil
}

type stubSessionLookup struct {
	knownIDs map[string]bool
}

func (s *stubSessionLookup) ExistsSession(_ context.Context, id string) (bool, error) {
	return s.knownIDs[id], nil
}

type memoryEvidenceAuthRepository struct {
	mu      sync.Mutex
	nextID  int
	byID    map[string]*auth.User
	byPhone map[string]*auth.User
}

func newMemoryEvidenceAuthRepository() *memoryEvidenceAuthRepository {
	return &memoryEvidenceAuthRepository{
		byID:    make(map[string]*auth.User),
		byPhone: make(map[string]*auth.User),
	}
}

func (r *memoryEvidenceAuthRepository) CreateUser(_ context.Context, user *auth.User) error {
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

func (r *memoryEvidenceAuthRepository) GetUserByPhone(_ context.Context, phone string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byPhone[strings.Join(strings.Fields(phone), "")]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}

type memoryEvidenceRepository struct {
	mu      sync.Mutex
	nextID  int
	records map[string]*evidence.StoredEvidence
}

func newMemoryEvidenceRepository() *memoryEvidenceRepository {
	return &memoryEvidenceRepository{
		records: make(map[string]*evidence.StoredEvidence),
	}
}

func (r *memoryEvidenceRepository) Create(_ context.Context, record *evidence.StoredEvidence) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	record.ID = uuid.NewString()
	record.CreatedAt = time.Now().UTC()
	copyRecord := *record
	r.records[record.ID] = &copyRecord
	return nil
}

func (r *memoryEvidenceRepository) GetByID(_ context.Context, id string) (*evidence.StoredEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.records[id]
	if !exists {
		return nil, evidence.ErrEvidenceNotFound
	}

	copyRecord := *record
	return &copyRecord, nil
}

func (r *memoryEvidenceAuthRepository) GetUserByID(_ context.Context, id string) (*auth.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[id]
	if !exists {
		return nil, auth.ErrUserNotFound
	}

	copyUser := *user
	return &copyUser, nil
}
