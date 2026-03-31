package safety

import "context"

type RouteProvider interface {
	ComputeRoute(ctx context.Context, input RouteRequest) (*ComputedRoute, error)
}

type RouteRequest struct {
	Origin      Coordinate
	Destination Coordinate
	TravelMode  string
}

type ComputedRoute struct {
	DistanceMeters  int64
	DurationSeconds int64
	EncodedPolyline string
	Points          []Coordinate
}
