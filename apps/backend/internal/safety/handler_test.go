package safety

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"saferoute-backend/config"
	appcore "saferoute-backend/internal/app"

	"github.com/gofiber/fiber/v2"
)

func TestSafetyScoreReturnsNeutralFallbackForNoData(t *testing.T) {
	service := NewService(&stubRepository{}, ServiceConfig{
		DefaultRadiusM: 500,
		MaxRadiusM:     3000,
		RecentWindow:   6 * time.Hour,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC))

	result, err := service.Score(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score != 75 {
		t.Fatalf("expected neutral fallback score 75, got %d", result.Score)
	}
	if result.Factors.Confidence != "low" {
		t.Fatalf("expected low confidence, got %q", result.Factors.Confidence)
	}
}

func TestSafetyScoreReflectsRecentClusterRisk(t *testing.T) {
	service := NewService(&stubRepository{
		aggregates: &Aggregates{
			RecentReports:         4,
			HistoricalReports:     10,
			RecentTrustWeight:     2.7,
			HistoricalTrustWeight: 6.2,
		},
	}, ServiceConfig{
		DefaultRadiusM: 500,
		MaxRadiusM:     3000,
		RecentWindow:   6 * time.Hour,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC))

	result, err := service.Score(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
		Radius:    400,
	})
	if err != nil {
		t.Fatalf("Score returned error: %v", err)
	}

	if result.Score >= 60 {
		t.Fatalf("expected higher-risk score below 60, got %d", result.Score)
	}
	if result.RiskLevel != "high" {
		t.Fatalf("expected high risk level, got %q", result.RiskLevel)
	}
}

func TestSafetyScoreReflectsTimeOfDayShift(t *testing.T) {
	repo := &stubRepository{
		aggregates: &Aggregates{
			RecentReports:         1,
			HistoricalReports:     3,
			RecentTrustWeight:     0.7,
			HistoricalTrustWeight: 1.8,
		},
	}
	service := NewService(repo, ServiceConfig{
		DefaultRadiusM: 500,
		MaxRadiusM:     3000,
		RecentWindow:   6 * time.Hour,
	})

	setSafetyClock(service, time.Date(2026, 3, 31, 13, 0, 0, 0, time.UTC))
	dayResult, err := service.Score(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("day Score returned error: %v", err)
	}

	setSafetyClock(service, time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC))
	nightResult, err := service.Score(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("night Score returned error: %v", err)
	}

	if nightResult.Score >= dayResult.Score {
		t.Fatalf("expected night score to be lower than day score, got day=%d night=%d", dayResult.Score, nightResult.Score)
	}
}

func TestSafetyHandlerRejectsInvalidRadius(t *testing.T) {
	application := newSafetyTestApp(t, &stubRepository{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/safety/score?lat=12.9716&lng=77.5946&radius=999999", nil)
	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestSafetyHandlerReturnsScorePayload(t *testing.T) {
	application := newSafetyTestApp(t, &stubRepository{
		aggregates: &Aggregates{
			RecentReports:         2,
			HistoricalReports:     5,
			RecentTrustWeight:     1.1,
			HistoricalTrustWeight: 2.9,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/safety/score?lat=12.9716&lng=77.5946&radius=500", nil)
	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if _, ok := body["score"]; !ok {
		t.Fatalf("expected score in response, got %#v", body)
	}
	if _, ok := body["factors"]; !ok {
		t.Fatalf("expected factors in response, got %#v", body)
	}
}

func newSafetyTestApp(t *testing.T, repo Repository) *fiber.App {
	t.Helper()

	service := NewService(repo, ServiceConfig{
		DefaultRadiusM: 500,
		MaxRadiusM:     3000,
		RecentWindow:   6 * time.Hour,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC))
	handler := NewHandler(service)

	return appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, handler.RegisterRoutes)
}

type stubRepository struct {
	aggregates *Aggregates
}

func (s *stubRepository) GetAggregates(_ context.Context, _ AggregateParams) (*Aggregates, error) {
	if s.aggregates == nil {
		return &Aggregates{}, nil
	}

	copyAggregates := *s.aggregates
	return &copyAggregates, nil
}

func setSafetyClock(service *Service, now time.Time) {
	serviceValueTestNow(service, now)
}
