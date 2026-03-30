package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidPhone       = errors.New("phone is required")
	ErrInvalidPassword    = errors.New("password must be at least 8 characters")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
)
