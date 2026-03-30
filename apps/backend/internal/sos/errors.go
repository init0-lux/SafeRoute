package sos

import "errors"

var (
	ErrInvalidSessionID        = errors.New("session id is required")
	ErrInvalidUserID           = errors.New("user id is required")
	ErrSessionNotFound         = errors.New("sos session not found")
	ErrSessionForbidden        = errors.New("forbidden")
	ErrActiveSessionExists     = errors.New("active sos session already exists")
	ErrSessionAlreadyEnded     = errors.New("sos session already ended")
	ErrInvalidLatitude         = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude        = errors.New("longitude must be between -180 and 180")
	ErrInvalidViewerToken      = errors.New("viewer token is invalid")
	ErrViewerGrantNotFound     = errors.New("sos viewer grant not found")
	ErrViewerGrantExpired      = errors.New("sos viewer grant expired")
	ErrViewerGrantRevoked      = errors.New("sos viewer grant revoked")
	ErrViewerGrantConflict     = errors.New("active sos viewer grant already exists for trusted contact")
	ErrInvalidTrustedContactID = errors.New("trusted contact id is required")
)
