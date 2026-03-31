package evidence

import "errors"

var (
	ErrEvidenceNotFound          = errors.New("evidence not found")
	ErrFileRequired              = errors.New("file is required")
	ErrFileTooLarge              = errors.New("file exceeds the configured size limit")
	ErrUnsupportedMediaType      = errors.New("unsupported media type")
	ErrInvalidEvidenceID         = errors.New("evidence id must be a valid UUID")
	ErrInvalidReportID           = errors.New("report id must be a valid UUID")
	ErrInvalidSessionID          = errors.New("session id must be a valid UUID")
	ErrForbiddenEvidenceAccess   = errors.New("forbidden")
	ErrAttachmentTargetRequired  = errors.New("exactly one of report_id or session_id is required")
	ErrAttachmentTargetConflict  = errors.New("exactly one of report_id or session_id is required")
	ErrParentNotFound            = errors.New("attachment target not found")
)
