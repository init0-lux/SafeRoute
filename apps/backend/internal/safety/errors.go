package safety

import "errors"

var (
	ErrInvalidLatitude             = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude            = errors.New("longitude must be between -180 and 180")
	ErrInvalidRadius               = errors.New("radius must be greater than 0 and within the configured maximum")
	ErrInvalidOriginLatitude       = errors.New("origin latitude must be between -90 and 90")
	ErrInvalidOriginLongitude      = errors.New("origin longitude must be between -180 and 180")
	ErrInvalidDestinationLatitude  = errors.New("destination latitude must be between -90 and 90")
	ErrInvalidDestinationLongitude = errors.New("destination longitude must be between -180 and 180")
	ErrUnsupportedTravelMode       = errors.New("travel mode is not supported")
	ErrRouteProviderUnavailable    = errors.New("route provider unavailable")
	ErrRouteNotFound               = errors.New("route not found")
	ErrRouteProviderFailed         = errors.New("route provider request failed")
	ErrRouteTooLong                = errors.New("route exceeds the configured maximum distance")
)
