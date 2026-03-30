package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"saferoute-backend/config"
)

func TestHealthRoute(t *testing.T) {
	application := New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyText := string(body)
	if !strings.Contains(bodyText, `"status":"ok"`) {
		t.Fatalf("expected health status in response body, got %s", bodyText)
	}

	if !strings.Contains(bodyText, `"environment":"test"`) {
		t.Fatalf("expected environment in response body, got %s", bodyText)
	}
}
