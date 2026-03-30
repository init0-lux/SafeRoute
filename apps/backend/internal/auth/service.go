package auth

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	repo Repository
}

type RegisterInput struct {
	Phone    string
	Email    string
	Password string
}

type LoginInput struct {
	Phone    string
	Password string
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*User, error) {
	phone := normalizePhone(input.Phone)
	if phone == "" {
		return nil, ErrInvalidPhone
	}

	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Phone:        phone,
		PasswordHash: passwordHash,
	}

	if email := normalizeEmail(input.Email); email != "" {
		user.Email = &email
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*User, error) {
	phone := normalizePhone(input.Phone)
	if phone == "" || input.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if err := ComparePassword(user.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrUnauthorized
	}

	return s.repo.GetUserByID(ctx, id)
}

func normalizePhone(phone string) string {
	return strings.Join(strings.Fields(phone), "")
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
