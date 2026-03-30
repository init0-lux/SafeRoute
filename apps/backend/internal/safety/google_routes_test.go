package safety

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGoogleRoutesProviderRejectsMissingAPIKey(t *testing.T) {
	provider := NewGoogleRoutesProvider(GoogleRoutesConfig{})

	_, err := provider.ComputeRoute(context.Background(), RouteRequest{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9352, Longitude: 77.6245},
		TravelMode:  defaultRouteTravelMode,
	})
	if err != ErrRouteProviderUnavailable {
		t.Fatalf("expected ErrRouteProviderUnavailable, got %v", err)
	}
}

func TestGoogleRoutesProviderParsesResponse(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("expected POST request, got %s", req.Method)
			}
			if req.Header.Get("X-Goog-Api-Key") != "test-key" {
				t.Fatalf("expected API key header, got %q", req.Header.Get("X-Goog-Api-Key"))
			}
			if req.Header.Get("X-Goog-FieldMask") != googleRoutesFieldMask {
				t.Fatalf("unexpected field mask: %q", req.Header.Get("X-Goog-FieldMask"))
			}

			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if body["travelMode"] != "WALK" {
				t.Fatalf("expected WALK travel mode, got %#v", body["travelMode"])
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader("{\n" +
					"  \"routes\": [\n" +
					"    {\n" +
					"      \"distanceMeters\": 4200,\n" +
					"      \"duration\": \"3120s\",\n" +
					"      \"polyline\": {\n" +
					"        \"encodedPolyline\": \"_p~iF~ps|U_ulLnnqC_mqNvxq`@\"\n" +
					"      }\n" +
					"    }\n" +
					"  ]\n" +
					"}")),
			}, nil
		}),
	}

	provider := NewGoogleRoutesProvider(GoogleRoutesConfig{
		APIKey:     "test-key",
		BaseURL:    "https://example.test/computeRoutes",
		HTTPClient: client,
	})

	route, err := provider.ComputeRoute(context.Background(), RouteRequest{
		Origin:      Coordinate{Latitude: 12.9716, Longitude: 77.5946},
		Destination: Coordinate{Latitude: 12.9352, Longitude: 77.6245},
		TravelMode:  defaultRouteTravelMode,
	})
	if err != nil {
		t.Fatalf("ComputeRoute returned error: %v", err)
	}

	if route.DistanceMeters != 4200 {
		t.Fatalf("expected distance 4200, got %d", route.DistanceMeters)
	}
	if route.DurationSeconds != 3120 {
		t.Fatalf("expected duration 3120, got %d", route.DurationSeconds)
	}
	if len(route.Points) != 3 {
		t.Fatalf("expected decoded polyline points, got %d", len(route.Points))
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
