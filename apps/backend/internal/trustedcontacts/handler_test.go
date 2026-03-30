package trustedcontacts_test

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
	"saferoute-backend/internal/trustedcontacts"

	"github.com/gofiber/fiber/v2"
)

func TestTrustedContactRequestCanBeAcceptedAndRemoved(t *testing.T) {
	application, accessCookie := newTrustedContactsApp(t)

	createResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/trusted-contacts/requests", map[string]string{
		"name":  "Emergency Contact",
		"phone": "+91 88888 22222",
		"email": "HELPER@EXAMPLE.COM",
	}, []*http.Cookie{accessCookie})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected request create status 201, got %d", createResp.StatusCode)
	}

	createBody := decodeBody(t, createResp)
	requestBody := createBody["request"].(map[string]any)
	requestID := requestBody["id"].(string)
	acceptToken := createBody["accept_token"].(string)
	if requestBody["status"] != "pending" {
		t.Fatalf("expected request status pending, got %#v", requestBody["status"])
	}

	acceptResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/trusted-contacts/requests/"+requestID+"/accept", map[string]string{
		"token": acceptToken,
	}, nil)
	if acceptResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected request accept status 201, got %d", acceptResp.StatusCode)
	}

	acceptBody := decodeBody(t, acceptResp)
	contact := acceptBody["contact"].(map[string]any)
	if contact["phone"] != "+918888822222" {
		t.Fatalf("expected normalized trusted contact phone, got %#v", contact["phone"])
	}

	deleteResp := performJSONRequest(t, application, http.MethodDelete, "/api/v1/trusted-contacts/"+contact["id"].(string), nil, []*http.Cookie{accessCookie})
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected trusted contact delete status 200, got %d", deleteResp.StatusCode)
	}
}

func TestTrustedContactRequestRejectsDuplicatePendingRequest(t *testing.T) {
	application, accessCookie := newTrustedContactsApp(t)

	payload := map[string]string{
		"name":  "Primary Contact",
		"phone": "+91 88888 22222",
	}

	firstResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/trusted-contacts/requests", payload, []*http.Cookie{accessCookie})
	if firstResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected first request create status 201, got %d", firstResp.StatusCode)
	}

	secondResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/trusted-contacts/requests", payload, []*http.Cookie{accessCookie})
	if secondResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate request status 409, got %d", secondResp.StatusCode)
	}
}

func TestTrustedContactRequestRejectsInvalidAcceptToken(t *testing.T) {
	application, accessCookie := newTrustedContactsApp(t)

	createResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/trusted-contacts/requests", map[string]string{
		"name":  "Emergency Contact",
		"phone": "+91 88888 22222",
	}, []*http.Cookie{accessCookie})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected request create status 201, got %d", createResp.StatusCode)
	}

	requestID := decodeBody(t, createResp)["request"].(map[string]any)["id"].(string)
	acceptResp := performJSONRequest(t, application, http.MethodPost, "/api/v1/trusted-contacts/requests/"+requestID+"/accept", map[string]string{
		"token": "wrong-token",
	}, nil)
	if acceptResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid token status 400, got %d", acceptResp.StatusCode)
	}
}

func newTrustedContactsApp(t *testing.T) (*fiber.App, *http.Cookie) {
	t.Helper()

	authRepo := newMemoryAuthRepository()
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
	handler := trustedcontacts.NewHandler(trustedcontacts.NewService(newMemoryRepository()), authMiddleware)
	application := appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, handler.RegisterRoutes)

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

	resp, err := application.Test(req)
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

type memoryRepository struct {
	mu              sync.Mutex
	nextRequestID   int
	nextContactID   int
	requestsByID    map[string]*trustedcontacts.TrustedContactRequest
	contactsByID    map[string]*trustedcontacts.TrustedContact
	contactsByPhone map[string]*trustedcontacts.TrustedContact
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		requestsByID:    make(map[string]*trustedcontacts.TrustedContactRequest),
		contactsByID:    make(map[string]*trustedcontacts.TrustedContact),
		contactsByPhone: make(map[string]*trustedcontacts.TrustedContact),
	}
}

func (r *memoryRepository) CreateRequest(_ context.Context, request *trustedcontacts.TrustedContactRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextRequestID++
	copyRequest := *request
	copyRequest.ID = fmt.Sprintf("request-%d", r.nextRequestID)
	if copyRequest.CreatedAt.IsZero() {
		copyRequest.CreatedAt = time.Now().UTC()
	}
	r.requestsByID[copyRequest.ID] = &copyRequest
	request.ID = copyRequest.ID
	request.CreatedAt = copyRequest.CreatedAt

	return nil
}

func (r *memoryRepository) GetRequestByID(_ context.Context, id string) (*trustedcontacts.TrustedContactRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	request, exists := r.requestsByID[id]
	if !exists {
		return nil, trustedcontacts.ErrRequestNotFound
	}

	copyRequest := *request
	return &copyRequest, nil
}

func (r *memoryRepository) GetActiveRequestByUserPhone(_ context.Context, userID, phone string, now time.Time) (*trustedcontacts.TrustedContactRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, request := range r.requestsByID {
		if request.UserID == userID && request.Phone == phone && request.Status == trustedcontacts.RequestStatusPending && request.ExpiresAt.After(now) {
			copyRequest := *request
			return &copyRequest, nil
		}
	}

	return nil, trustedcontacts.ErrRequestNotFound
}

func (r *memoryRepository) GetTrustedContactByUserPhone(_ context.Context, userID, phone string) (*trustedcontacts.TrustedContact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	contact, exists := r.contactsByPhone[userPhoneKey(userID, phone)]
	if !exists {
		return nil, trustedcontacts.ErrTrustedContactNotFound
	}

	copyContact := *contact
	return &copyContact, nil
}

func (r *memoryRepository) CompleteRequestAcceptance(_ context.Context, request *trustedcontacts.TrustedContactRequest, contact *trustedcontacts.TrustedContact) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.contactsByPhone[userPhoneKey(contact.UserID, contact.Phone)]; exists {
		return trustedcontacts.ErrTrustedContactExists
	}

	storedRequest, exists := r.requestsByID[request.ID]
	if !exists {
		return trustedcontacts.ErrRequestNotFound
	}
	if storedRequest.Status != trustedcontacts.RequestStatusPending {
		return trustedcontacts.ErrRequestAlreadyProcessed
	}

	r.nextContactID++
	copyContact := *contact
	copyContact.ID = fmt.Sprintf("contact-%d", r.nextContactID)
	if copyContact.CreatedAt.IsZero() {
		copyContact.CreatedAt = time.Now().UTC()
	}
	r.contactsByID[copyContact.ID] = &copyContact
	r.contactsByPhone[userPhoneKey(copyContact.UserID, copyContact.Phone)] = &copyContact
	contact.ID = copyContact.ID
	contact.CreatedAt = copyContact.CreatedAt

	storedRequest.Status = trustedcontacts.RequestStatusAccepted
	storedRequest.RespondedAt = request.RespondedAt
	storedRequest.AcceptedContactID = &copyContact.ID

	return nil
}

func (r *memoryRepository) UpdateRequestState(_ context.Context, requestID string, status trustedcontacts.RequestStatus, respondedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	request, exists := r.requestsByID[requestID]
	if !exists {
		return trustedcontacts.ErrRequestNotFound
	}

	request.Status = status
	request.RespondedAt = respondedAt

	return nil
}

func (r *memoryRepository) DeleteTrustedContact(_ context.Context, userID, contactID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	contact, exists := r.contactsByID[contactID]
	if !exists || contact.UserID != userID {
		return trustedcontacts.ErrTrustedContactNotFound
	}

	delete(r.contactsByID, contactID)
	delete(r.contactsByPhone, userPhoneKey(contact.UserID, contact.Phone))

	return nil
}

func userPhoneKey(userID, phone string) string {
	return userID + "|" + phone
}
