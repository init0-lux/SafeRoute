package safety

import "errors"

var (
	ErrInvalidLatitude  = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
	ErrInvalidRadius    = errors.New("radius must be greater than 0 and within the configured maximum")
)
