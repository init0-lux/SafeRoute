package reports

import "errors"

var (
	ErrInvalidReportType   = errors.New("report type is required")
	ErrInvalidLatitude     = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude    = errors.New("longitude must be between -180 and 180")
	ErrDescriptionTooLong  = errors.New("description must be 1000 characters or fewer")
	ErrUnauthorizedReport  = errors.New("unauthorized")
)
