package trustedcontacts

import "errors"

var (
	ErrInvalidContactName      = errors.New("trusted contact name is required")
	ErrInvalidPhone            = errors.New("phone is required")
	ErrInvalidRequestID        = errors.New("trusted contact request id is required")
	ErrInvalidRequestToken     = errors.New("trusted contact request token is invalid")
	ErrPendingRequestExists    = errors.New("trusted contact request already pending")
	ErrRequestAlreadyProcessed = errors.New("trusted contact request already processed")
	ErrRequestExpired          = errors.New("trusted contact request has expired")
	ErrRequestNotFound         = errors.New("trusted contact request not found")
	ErrTrustedContactExists    = errors.New("trusted contact already exists")
	ErrTrustedContactNotFound  = errors.New("trusted contact not found")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrContactNotRegistered    = errors.New("the contact is not registered on the SafeRoute platform")
)
