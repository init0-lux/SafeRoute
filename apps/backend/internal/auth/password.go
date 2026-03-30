package auth

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

func HashPassword(password string) (string, error) {
	if len(password) < minPasswordLength || strings.TrimSpace(password) == "" {
		return "", ErrInvalidPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
