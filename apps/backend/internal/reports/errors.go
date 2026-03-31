package reports

import "errors"

var (
	ErrInvalidReportType   = errors.New("report type is required")
	ErrUnsupportedReportType = errors.New("report type is not supported")
	ErrInvalidLatitude     = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude    = errors.New("longitude must be between -180 and 180")
	ErrInvalidRadius       = errors.New("radius must be greater than 0 and within the configured maximum")
	ErrInvalidLimit        = errors.New("limit must be between 1 and 50")
	ErrInvalidOffset       = errors.New("offset must be 0 or greater")
	ErrInvalidReportID     = errors.New("report id must be a valid UUID")
	ErrDescriptionTooLong  = errors.New("description must be 1000 characters or fewer")
	ErrUnauthorizedReport  = errors.New("unauthorized")
	ErrReportNotFound      = errors.New("report not found")
)
