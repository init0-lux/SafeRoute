package safety

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultGoogleRoutesBaseURL = "https://routes.googleapis.com/directions/v2:computeRoutes"
	googleRoutesFieldMask      = "routes.distanceMeters,routes.duration,routes.polyline.encodedPolyline"
)

type GoogleRoutesConfig struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

type GoogleRoutesProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewGoogleRoutesProvider(cfg GoogleRoutesConfig) *GoogleRoutesProvider {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = defaultGoogleRoutesBaseURL
	}

	return &GoogleRoutesProvider{
		apiKey:  strings.TrimSpace(cfg.APIKey),
		baseURL: baseURL,
		client:  client,
	}
}

func (p *GoogleRoutesProvider) ComputeRoute(ctx context.Context, input RouteRequest) (*ComputedRoute, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, ErrRouteProviderUnavailable
	}

	payload := googleRoutesRequest{
		Origin:      newGoogleWaypoint(input.Origin),
		Destination: newGoogleWaypoint(input.Destination),
		TravelMode:  "WALK",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRouteProviderFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRouteProviderFailed, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", p.apiKey)
	req.Header.Set("X-Goog-FieldMask", googleRoutesFieldMask)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRouteProviderFailed, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrRouteProviderUnavailable
	case http.StatusNotFound:
		return nil, ErrRouteNotFound
	default:
		return nil, fmt.Errorf("%w: status %d", ErrRouteProviderFailed, resp.StatusCode)
	}

	var response googleRoutesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRouteProviderFailed, err)
	}
	if len(response.Routes) == 0 {
		return nil, ErrRouteNotFound
	}

	route := response.Routes[0]
	if strings.TrimSpace(route.Polyline.EncodedPolyline) == "" {
		return nil, fmt.Errorf("%w: missing route polyline", ErrRouteProviderFailed)
	}

	points, err := decodePolyline(route.Polyline.EncodedPolyline)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRouteProviderFailed, err)
	}
	if len(points) < 2 {
		return nil, fmt.Errorf("%w: invalid route geometry", ErrRouteProviderFailed)
	}

	duration, err := time.ParseDuration(route.Duration)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid route duration", ErrRouteProviderFailed)
	}

	return &ComputedRoute{
		DistanceMeters:  route.DistanceMeters,
		DurationSeconds: int64(duration.Seconds()),
		EncodedPolyline: route.Polyline.EncodedPolyline,
		Points:          points,
	}, nil
}

type googleRoutesRequest struct {
	Origin      googleWaypoint `json:"origin"`
	Destination googleWaypoint `json:"destination"`
	TravelMode  string         `json:"travelMode"`
}

type googleWaypoint struct {
	Location googleLocation `json:"location"`
}

type googleLocation struct {
	LatLng googleLatLng `json:"latLng"`
}

type googleLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googleRoutesResponse struct {
	Routes []googleRoute `json:"routes"`
}

type googleRoute struct {
	DistanceMeters int64               `json:"distanceMeters"`
	Duration       string              `json:"duration"`
	Polyline       googleRoutePolyline `json:"polyline"`
}

type googleRoutePolyline struct {
	EncodedPolyline string `json:"encodedPolyline"`
}

func newGoogleWaypoint(point Coordinate) googleWaypoint {
	return googleWaypoint{
		Location: googleLocation{
			LatLng: googleLatLng{
				Latitude:  point.Latitude,
				Longitude: point.Longitude,
			},
		},
	}
}

func decodePolyline(encoded string) ([]Coordinate, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, errors.New("empty polyline")
	}

	points := make([]Coordinate, 0)
	var lat, lng int

	for index := 0; index < len(encoded); {
		deltaLat, nextIndex, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = nextIndex

		deltaLng, nextIndex, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = nextIndex

		lat += deltaLat
		lng += deltaLng
		points = append(points, Coordinate{
			Latitude:  float64(lat) / 1e5,
			Longitude: float64(lng) / 1e5,
		})
	}

	return points, nil
}

func decodePolylineValue(encoded string, start int) (int, int, error) {
	result := 0
	shift := 0

	for index := start; index < len(encoded); index++ {
		value := int(encoded[index]) - 63
		if value < 0 {
			return 0, 0, io.ErrUnexpectedEOF
		}

		result |= (value & 0x1f) << shift
		shift += 5
		if value < 0x20 {
			delta := result >> 1
			if result&1 != 0 {
				delta = ^delta
			}
			return delta, index + 1, nil
		}
	}

	return 0, 0, io.ErrUnexpectedEOF
}
