package sos_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"saferoute-backend/config"
	appcore "saferoute-backend/internal/app"
	"saferoute-backend/internal/auth"
	"saferoute-backend/internal/sos"

	wsclient "github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
)

func TestSOSLifecycleRoutes(t *testing.T) {
	authRepo := newMemoryAuthRepository()
	sosRepo := newMemorySOSRepository()
	application, accessCookie := newSOSApp(t, authRepo, sosRepo)

	startResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/sos/start", map[string]any{}, []*http.Cookie{accessCookie})
	if startResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected start status 201, got %d", startResp.StatusCode)
	}

	startBody := decodeBody(t, startResp)
	session := startBody["session"].(map[string]any)
	sessionID := session["id"].(string)

	getResp := performJSONRequest(t, application, http.MethodGet, "/api/v1/sos/"+sessionID, nil, []*http.Cookie{accessCookie})
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getResp.StatusCode)
	}

	endResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/sos/"+sessionID+"/end", map[string]any{}, []*http.Cookie{accessCookie})
	if endResp.StatusCode != http.StatusOK {
		t.Fatalf("expected end status 200, got %d", endResp.StatusCode)
	}

	endBody := decodeBody(t, endResp)
	endedSession := endBody["session"].(map[string]any)
	if endedSession["status"] != "ended" {
		t.Fatalf("expected ended status, got %#v", endedSession["status"])
	}
}

func TestSOSStartRejectsSecondActiveSession(t *testing.T) {
	authRepo := newMemoryAuthRepository()
	sosRepo := newMemorySOSRepository()
	application, accessCookie := newSOSApp(t, authRepo, sosRepo)

	firstResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/sos/start", map[string]any{}, []*http.Cookie{accessCookie})
	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected first start status 201, got %d", firstResp.StatusCode)
	}

	secondResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/sos/start", map[string]any{}, []*http.Cookie{accessCookie})
	if secondResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected second start status 409, got %d", secondResp.StatusCode)
	}
}

func TestSOSWebSocketPersistsLocationPing(t *testing.T) {
	authRepo := newMemoryAuthRepository()
	sosRepo := newMemorySOSRepository()
	application, accessCookie := newSOSApp(t, authRepo, sosRepo)

	startResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/sos/start", map[string]any{}, []*http.Cookie{accessCookie})
	if startResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected start status 201, got %d", startResp.StatusCode)
	}

	startBody := decodeBody(t, startResp)
	session := startBody["session"].(map[string]any)
	sessionID := session["id"].(string)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- application.Listener(listener)
	}()
	defer application.Shutdown()

	wsURL := fmt.Sprintf("ws://%s/api/v1/sos/%s/stream", listener.Addr().String(), sessionID)
	header := http.Header{}
	header.Add("Cookie", accessCookie.String())

	conn, resp, err := wsclient.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			t.Fatalf("failed to dial websocket: %v (status %d)", err, resp.StatusCode)
		}
		t.Fatalf("failed to dial websocket: %v", err)
	}
	defer conn.Close()

	recordedAt := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	if err := conn.WriteJSON(map[string]any{
		"lat": 12.9716,
		"lng": 77.5946,
		"ts":  recordedAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("failed to write websocket payload: %v", err)
	}

	var ack map[string]any
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatalf("failed to read websocket ack: %v", err)
	}

	if ack["status"] != "accepted" {
		t.Fatalf("expected accepted ack, got %#v", ack["status"])
	}

	if sosRepo.LocationPingCount(sessionID) != 1 {
		t.Fatalf("expected one persisted location ping, got %d", sosRepo.LocationPingCount(sessionID))
	}

	select {
	case err := <-serverErrors:
		if err != nil {
			t.Fatalf("fiber listener exited unexpectedly: %v", err)
		}
	default:
	}
}

func newSOSApp(t *testing.T, authRepo *memoryAuthRepository, sosRepo *memorySOSRepository) (*fiber.App, *http.Cookie) {
	t.Helper()

	authService := auth.NewService(authRepo)
	user, err := authService.Register(context.Background(), auth.RegisterInput{
		Phone:    "+919999911111",
		Password: "supersecret",
	})
	if err != nil {
		t.Fatalf("failed to seed auth user: %v", err)
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
		t.Fatalf("failed to issue auth session pair: %v", err)
	}

	authMiddleware := auth.NewMiddleware(authService, sessionManager)
	sosHandler := sos.NewHandler(sos.NewService(sosRepo), authMiddleware)
	application := appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, sosHandler.RegisterRoutes)

	return application, &http.Cookie{
		Name:  "test_access",
		Value: pair.AccessToken,
		Path:  "/",
	}
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

	resp, err := application.Test(req, -1)
	if err != nil {
		t.Fatalf("application.Test returned error: %v", err)
	}

	return resp
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

	user, exists := r.byPhone[phone]
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

func (r *memoryAuthRepository) UpdatePushToken(_ context.Context, id string, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.byID[id]
	if !exists {
		return auth.ErrUserNotFound
	}

	user.ExpoPushToken = &token
	return nil
}

type memorySOSRepository struct {
	mu              sync.Mutex
	nextID          int
	sessions        map[string]*sos.SOSSession
	pings           map[string][]storedPing
	viewerGrants    map[string]*sos.SOSViewerGrant
	trustedContacts map[string]map[string]struct{}
}

type storedPing struct {
	Latitude   float64
	Longitude  float64
	RecordedAt time.Time
}

func newMemorySOSRepository() *memorySOSRepository {
	return &memorySOSRepository{
		sessions:        make(map[string]*sos.SOSSession),
		pings:           make(map[string][]storedPing),
		viewerGrants:    make(map[string]*sos.SOSViewerGrant),
		trustedContacts: make(map[string]map[string]struct{}),
	}
}

func (r *memorySOSRepository) CreateSession(_ context.Context, session *sos.SOSSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	clone := cloneSession(session)
	clone.ID = fmt.Sprintf("sos-%d", r.nextID)
	if clone.StartedAt.IsZero() {
		clone.StartedAt = time.Now().UTC()
	}
	r.sessions[clone.ID] = clone
	session.ID = clone.ID
	session.StartedAt = clone.StartedAt

	return nil
}

func (r *memorySOSRepository) ExistsSession(_ context.Context, sessionID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.sessions[sessionID]
	return exists, nil
}

func (r *memorySOSRepository) GetSessionByID(_ context.Context, sessionID string) (*sos.SOSSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists {
		return nil, sos.ErrSessionNotFound
	}

	return cloneSession(session), nil
}

func (r *memorySOSRepository) GetActiveSessionByUserID(_ context.Context, userID string) (*sos.SOSSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, session := range r.sessions {
		if session.UserID != nil && *session.UserID == userID && session.Status == sos.SessionStatusActive {
			return cloneSession(session), nil
		}
	}

	return nil, sos.ErrSessionNotFound
}

func (r *memorySOSRepository) UpdateSession(_ context.Context, session *sos.SOSSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.ID] = cloneSession(session)
	return nil
}

func (r *memorySOSRepository) CreateLocationPing(_ context.Context, sessionID string, latitude, longitude float64, recordedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pings[sessionID] = append(r.pings[sessionID], storedPing{
		Latitude:   latitude,
		Longitude:  longitude,
		RecordedAt: recordedAt,
	})

	return nil
}

func (r *memorySOSRepository) CreateViewerGrant(_ context.Context, grant *sos.SOSViewerGrant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.viewerGrants {
		if existing.SessionID == grant.SessionID &&
			existing.TrustedContactID == grant.TrustedContactID &&
			existing.RevokedAt == nil &&
			existing.ExpiresAt.After(time.Now().UTC()) {
			return sos.ErrViewerGrantConflict
		}
	}

	r.nextID++
	clone := cloneViewerGrant(grant)
	clone.ID = fmt.Sprintf("viewer-%d", r.nextID)
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = time.Now().UTC()
	}
	r.viewerGrants[clone.ID] = clone
	grant.ID = clone.ID
	grant.CreatedAt = clone.CreatedAt

	return nil
}

func (r *memorySOSRepository) GetActiveViewerGrantBySessionContact(_ context.Context, sessionID, trustedContactID string, now time.Time) (*sos.SOSViewerGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, grant := range r.viewerGrants {
		if grant.SessionID == sessionID &&
			grant.TrustedContactID == trustedContactID &&
			grant.RevokedAt == nil &&
			grant.ExpiresAt.After(now) {
			return cloneViewerGrant(grant), nil
		}
	}

	return nil, sos.ErrViewerGrantNotFound
}

func (r *memorySOSRepository) GetViewerGrantByToken(_ context.Context, tokenHash string) (*sos.SOSViewerGrant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, grant := range r.viewerGrants {
		if grant.TokenHash == tokenHash {
			return cloneViewerGrant(grant), nil
		}
	}

	return nil, sos.ErrViewerGrantNotFound
}

func (r *memorySOSRepository) IsTrustedContactOwnedByUser(_ context.Context, userID, trustedContactID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	contacts := r.trustedContacts[userID]
	if contacts == nil {
		return false, nil
	}

	_, exists := contacts[trustedContactID]
	return exists, nil
}

func (r *memorySOSRepository) AddTrustedContact(userID, trustedContactID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.trustedContacts[userID]; !exists {
		r.trustedContacts[userID] = make(map[string]struct{})
	}
	r.trustedContacts[userID][trustedContactID] = struct{}{}
}

func (r *memorySOSRepository) LocationPingCount(sessionID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.pings[sessionID])
}

func cloneSession(session *sos.SOSSession) *sos.SOSSession {
	clone := *session
	if session.UserID != nil {
		userID := *session.UserID
		clone.UserID = &userID
	}

	if session.EndedAt != nil {
		endedAt := *session.EndedAt
		clone.EndedAt = &endedAt
	}

	return &clone
}

func cloneViewerGrant(grant *sos.SOSViewerGrant) *sos.SOSViewerGrant {
	clone := *grant
	if grant.RevokedAt != nil {
		revokedAt := *grant.RevokedAt
		clone.RevokedAt = &revokedAt
	}

	return &clone
}
