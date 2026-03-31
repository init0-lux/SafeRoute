package safety

import (
	"bytes"
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
	service := NewService(&stubRepository{}, nil, ServiceConfig{
		DefaultRadiusM:       500,
		MaxRadiusM:           3000,
		RecentWindow:         6 * time.Hour,
		RouteCorridorRadiusM: 75,
		RouteSegmentLengthM:  150,
		RouteMaxDistanceM:    10000,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC))

	result, err := service.ScorePoint(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("ScorePoint returned error: %v", err)
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
		pointAggregates: &Aggregates{
			RecentReports:         4,
			HistoricalReports:     10,
			RecentTrustWeight:     2.7,
			HistoricalTrustWeight: 6.2,
		},
	}, nil, ServiceConfig{
		DefaultRadiusM:       500,
		MaxRadiusM:           3000,
		RecentWindow:         6 * time.Hour,
		RouteCorridorRadiusM: 75,
		RouteSegmentLengthM:  150,
		RouteMaxDistanceM:    10000,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC))

	result, err := service.ScorePoint(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
		Radius:    400,
	})
	if err != nil {
		t.Fatalf("ScorePoint returned error: %v", err)
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
		pointAggregates: &Aggregates{
			RecentReports:         1,
			HistoricalReports:     3,
			RecentTrustWeight:     0.7,
			HistoricalTrustWeight: 1.8,
		},
	}
	service := NewService(repo, nil, ServiceConfig{
		DefaultRadiusM:       500,
		MaxRadiusM:           3000,
		RecentWindow:         6 * time.Hour,
		RouteCorridorRadiusM: 75,
		RouteSegmentLengthM:  150,
		RouteMaxDistanceM:    10000,
	})

	setSafetyClock(service, time.Date(2026, 3, 31, 13, 0, 0, 0, time.UTC))
	dayResult, err := service.ScorePoint(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("day ScorePoint returned error: %v", err)
	}

	setSafetyClock(service, time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC))
	nightResult, err := service.ScorePoint(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("night ScorePoint returned error: %v", err)
	}

	if dayResult.Factors.TimeRisk != "low" {
		t.Fatalf("expected daytime time risk to be low, got %q", dayResult.Factors.TimeRisk)
	}
	if nightResult.Factors.TimeRisk != "high" {
		t.Fatalf("expected nighttime time risk to be high, got %q", nightResult.Factors.TimeRisk)
	}
}

func TestSafetyScoreUsesConfiguredTimeRiskHours(t *testing.T) {
	service := NewService(&stubRepository{
		pointAggregates: &Aggregates{
			RecentReports:         1,
			HistoricalReports:     2,
			RecentTrustWeight:     0.6,
			HistoricalTrustWeight: 1.2,
		},
	}, nil, ServiceConfig{
		DefaultRadiusM:       500,
		MaxRadiusM:           3000,
		RecentWindow:         6 * time.Hour,
		RouteCorridorRadiusM: 75,
		RouteSegmentLengthM:  150,
		RouteMaxDistanceM:    10000,
		TimeRisk: TimeRiskConfig{
			Timezone:         "UTC",
			HighStartHour:    20,
			HighEndHour:      6,
			MorningStartHour: 6,
			MorningEndHour:   8,
			EveningStartHour: 18,
			EveningEndHour:   20,
		},
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 21, 0, 0, 0, time.UTC))

	result, err := service.ScorePoint(context.Background(), ScoreInput{
		Latitude:  12.9716,
		Longitude: 77.5946,
	})
	if err != nil {
		t.Fatalf("ScorePoint returned error: %v", err)
	}
	if result.Factors.TimeRisk != "high" {
		t.Fatalf("expected configured time window to mark 21:00 as high risk, got %q", result.Factors.TimeRisk)
	}
}

func TestSafetyHandlerRejectsInvalidRadius(t *testing.T) {
	application := newSafetyTestApp(t, &stubRepository{}, nil)

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
		pointAggregates: &Aggregates{
			RecentReports:         2,
			HistoricalReports:     5,
			RecentTrustWeight:     1.1,
			HistoricalTrustWeight: 2.9,
		},
	}, nil)

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

func TestRouteScoreRejectsUnsupportedTravelMode(t *testing.T) {
	application := newSafetyTestApp(t, &stubRepository{}, &stubRouteProvider{
		route: testComputedRoute(),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/safety/route-score", mustJSONBody(t, map[string]any{
		"origin": map[string]any{
			"lat": 12.9716,
			"lng": 77.5946,
		},
		"destination": map[string]any{
			"lat": 12.9352,
			"lng": 77.6245,
		},
		"travel_mode": "driving",
	}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestRouteScoreReturnsProviderUnavailableWhenMissingKey(t *testing.T) {
	service := NewService(&stubRepository{}, NewGoogleRoutesProvider(GoogleRoutesConfig{}), ServiceConfig{
		DefaultRadiusM:       500,
		MaxRadiusM:           3000,
		RecentWindow:         6 * time.Hour,
		RouteCorridorRadiusM: 75,
		RouteSegmentLengthM:  150,
		RouteMaxDistanceM:    10000,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC))
	application := appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, NewHandler(service).RegisterRoutes)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/safety/route-score", mustJSONBody(t, map[string]any{
		"origin": map[string]any{
			"lat": 12.9716,
			"lng": 77.5946,
		},
		"destination": map[string]any{
			"lat": 12.9352,
			"lng": 77.6245,
		},
	}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", resp.StatusCode)
	}
}

func TestRouteScoreReturnsProviderFailure(t *testing.T) {
	application := newSafetyTestApp(t, &stubRepository{}, &stubRouteProvider{
		err: ErrRouteProviderFailed,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/safety/route-score", mustJSONBody(t, map[string]any{
		"origin": map[string]any{
			"lat": 12.9716,
			"lng": 77.5946,
		},
		"destination": map[string]any{
			"lat": 12.9352,
			"lng": 77.6245,
		},
	}))
	req.Header.Set("Content-Type", "application/json")

	resp, err := application.Test(req)
	if err != nil {
		t.Fatalf("app.Test returned error: %v", err)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", resp.StatusCode)
	}
}

func TestRouteScoreReturnsSegments(t *testing.T) {
	application := newSafetyTestApp(t, &stubRepository{
		routeSignals: []RouteSignal{
			{Fraction: 0.05, IsRecent: false, TrustWeight: 0.4},
			{Fraction: 0.35, IsRecent: true, TrustWeight: 0.9},
			{Fraction: 0.60, IsRecent: true, TrustWeight: 0.8},
			{Fraction: 0.95, IsRecent: false, TrustWeight: 0.3},
		},
	}, &stubRouteProvider{
		route: testComputedRoute(),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/safety/route-score", mustJSONBody(t, map[string]any{
		"origin": map[string]any{
			"lat": 12.9716,
			"lng": 77.5946,
		},
		"destination": map[string]any{
			"lat": 12.9352,
			"lng": 77.6245,
		},
	}))
	req.Header.Set("Content-Type", "application/json")

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

	if _, ok := body["route"]; !ok {
		t.Fatalf("expected route in response, got %#v", body)
	}
	segments, ok := body["segments"].([]any)
	if !ok || len(segments) < 3 {
		t.Fatalf("expected segment breakdown in response, got %#v", body["segments"])
	}
	summary := body["summary"].(map[string]any)
	if _, ok := summary["highest_risk_segment_index"]; !ok {
		t.Fatalf("expected highest_risk_segment_index, got %#v", summary)
	}
}

func TestScoreRouteUsesMinimumSegments(t *testing.T) {
	service := newSafetyServiceForRouteTests(&stubRepository{}, &stubRouteProvider{
		route: &ComputedRoute{
			DistanceMeters:  100,
			DurationSeconds: 90,
			EncodedPolyline: "abc",
			Points: []Coordinate{
				{Latitude: 12.9716, Longitude: 77.5946},
				{Latitude: 12.9720, Longitude: 77.5950},
			},
		},
	})

	result, err := service.ScoreRoute(context.Background(), RouteScoreInput{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9720, Longitude: 77.5950},
	})
	if err != nil {
		t.Fatalf("ScoreRoute returned error: %v", err)
	}
	if len(result.Segments) != 3 {
		t.Fatalf("expected minimum 3 segments, got %d", len(result.Segments))
	}
}

func TestScoreRouteCapsSegmentCount(t *testing.T) {
	service := newSafetyServiceForRouteTests(&stubRepository{}, &stubRouteProvider{
		route: &ComputedRoute{
			DistanceMeters:  10000,
			DurationSeconds: 6000,
			EncodedPolyline: "abc",
			Points: []Coordinate{
				{Latitude: 12.9716, Longitude: 77.5946},
				{Latitude: 12.9816, Longitude: 77.6046},
			},
		},
	})

	result, err := service.ScoreRoute(context.Background(), RouteScoreInput{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9816, Longitude: 77.6046},
	})
	if err != nil {
		t.Fatalf("ScoreRoute returned error: %v", err)
	}
	if len(result.Segments) != 20 {
		t.Fatalf("expected max 20 segments, got %d", len(result.Segments))
	}
}

func TestScoreRouteBucketsSignalsByFraction(t *testing.T) {
	service := newSafetyServiceForRouteTests(&stubRepository{
		routeSignals: []RouteSignal{
			{Fraction: 0.10, IsRecent: true, TrustWeight: 1.0},
			{Fraction: 0.70, IsRecent: false, TrustWeight: 0.5},
		},
	}, &stubRouteProvider{
		route: testComputedRoute(),
	})

	result, err := service.ScoreRoute(context.Background(), RouteScoreInput{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9352, Longitude: 77.6245},
	})
	if err != nil {
		t.Fatalf("ScoreRoute returned error: %v", err)
	}

	if result.Segments[0].Factors.RecentReports != 1 {
		t.Fatalf("expected first segment to contain recent signal, got %#v", result.Segments[0].Factors)
	}
	if result.Segments[2].Factors.HistoricalReports != 1 {
		t.Fatalf("expected later segment to contain historical signal, got %#v", result.Segments[2].Factors)
	}
}

func TestScoreRouteAppliesHotspotPenalty(t *testing.T) {
	service := newSafetyServiceForRouteTests(&stubRepository{
		routeSignals: []RouteSignal{
			{Fraction: 0.10, IsRecent: false, TrustWeight: 0.3},
			{Fraction: 0.45, IsRecent: true, TrustWeight: 2.5},
			{Fraction: 0.50, IsRecent: true, TrustWeight: 1.2},
			{Fraction: 0.90, IsRecent: false, TrustWeight: 0.4},
		},
	}, &stubRouteProvider{
		route: testComputedRoute(),
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC))

	result, err := service.ScoreRoute(context.Background(), RouteScoreInput{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9352, Longitude: 77.6245},
	})
	if err != nil {
		t.Fatalf("ScoreRoute returned error: %v", err)
	}

	worstScore := result.Segments[result.Summary.HighestRiskSegmentIndex].Score
	if result.Score > worstScore+25 {
		t.Fatalf("expected hotspot penalty to keep overall score near the dangerous segment, got overall=%d worst=%d", result.Score, worstScore)
	}
}

func TestScoreRouteNighttimeLowersScore(t *testing.T) {
	repo := &stubRepository{
		routeSignals: []RouteSignal{
			{Fraction: 0.25, IsRecent: true, TrustWeight: 0.8},
			{Fraction: 0.75, IsRecent: false, TrustWeight: 0.6},
		},
	}
	provider := &stubRouteProvider{route: testComputedRoute()}
	service := newSafetyServiceForRouteTests(repo, provider)

	setSafetyClock(service, time.Date(2026, 3, 31, 13, 0, 0, 0, time.UTC))
	dayResult, err := service.ScoreRoute(context.Background(), RouteScoreInput{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9352, Longitude: 77.6245},
	})
	if err != nil {
		t.Fatalf("day ScoreRoute returned error: %v", err)
	}

	setSafetyClock(service, time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC))
	nightResult, err := service.ScoreRoute(context.Background(), RouteScoreInput{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9352, Longitude: 77.6245},
	})
	if err != nil {
		t.Fatalf("night ScoreRoute returned error: %v", err)
	}

	if dayResult.Segments[0].Factors.TimeRisk != "low" {
		t.Fatalf("expected daytime route segment to be low risk by time, got %q", dayResult.Segments[0].Factors.TimeRisk)
	}
	if nightResult.Segments[0].Factors.TimeRisk != "high" {
		t.Fatalf("expected nighttime route segment to be high risk by time, got %q", nightResult.Segments[0].Factors.TimeRisk)
	}
}

func newSafetyTestApp(t *testing.T, repo Repository, provider RouteProvider) *fiber.App {
	t.Helper()

	service := newSafetyServiceForRouteTests(repo, provider)
	handler := NewHandler(service)

	return appcore.New(config.Config{
		AppName:     "SafeRoute Backend",
		Environment: "test",
		Port:        "8080",
	}, handler.RegisterRoutes)
}

func newSafetyServiceForRouteTests(repo Repository, provider RouteProvider) *Service {
	service := NewService(repo, provider, ServiceConfig{
		DefaultRadiusM:       500,
		MaxRadiusM:           3000,
		RecentWindow:         6 * time.Hour,
		RouteCorridorRadiusM: 75,
		RouteSegmentLengthM:  150,
		RouteMaxDistanceM:    10000,
	})
	setSafetyClock(service, time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC))
	return service
}

type stubRepository struct {
	pointAggregates *Aggregates
	routeSignals    []RouteSignal
}

func (s *stubRepository) GetPointAggregates(_ context.Context, _ PointAggregateParams) (*Aggregates, error) {
	if s.pointAggregates == nil {
		return &Aggregates{}, nil
	}

	copyAggregates := *s.pointAggregates
	return &copyAggregates, nil
}

func (s *stubRepository) GetRouteSignals(_ context.Context, _ RouteSignalParams) ([]RouteSignal, error) {
	if len(s.routeSignals) == 0 {
		return []RouteSignal{}, nil
	}

	signals := make([]RouteSignal, len(s.routeSignals))
	copy(signals, s.routeSignals)
	return signals, nil
}

type stubRouteProvider struct {
	route *ComputedRoute
	err   error
}

func (s *stubRouteProvider) ComputeRoute(_ context.Context, _ RouteRequest) (*ComputedRoute, error) {
	if s.err != nil {
		return nil, s.err
	}

	if s.route == nil {
		return nil, ErrRouteNotFound
	}

	copyRoute := *s.route
	copyRoute.Points = append([]Coordinate(nil), s.route.Points...)
	return &copyRoute, nil
}

func mustJSONBody(t *testing.T, payload any) *jsonBodyBuffer {
	t.Helper()

	return newJSONBodyBuffer(t, payload)
}

type jsonBodyBuffer = bytes.Reader

func newJSONBodyBuffer(t *testing.T, payload any) *jsonBodyBuffer {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	return bytes.NewReader(data)
}

func testComputedRoute() *ComputedRoute {
	return &ComputedRoute{
		DistanceMeters:  600,
		DurationSeconds: 480,
		EncodedPolyline: "_p~iF~ps|U_ulLnnqC_mqNvxq`@",
		Points: []Coordinate{
			{Latitude: 12.9716, Longitude: 77.5946},
			{Latitude: 12.9650, Longitude: 77.6100},
			{Latitude: 12.9352, Longitude: 77.6245},
		},
	}
}

func setSafetyClock(service *Service, now time.Time) {
	serviceValueTestNow(service, now)
}
