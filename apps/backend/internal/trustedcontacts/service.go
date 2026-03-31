package trustedcontacts

import (
	"context"
	"errors"
	"strings"
	"time"

	"saferoute-backend/internal/auth"

	"github.com/nyaruka/phonenumbers"
)

const defaultRequestTTL = 7 * 24 * time.Hour

type Service struct {
	repo     Repository
	authRepo auth.Repository
}

type CreateRequestInput struct {
	Phone string
	Email string
}

type AcceptRequestInput struct {
	Token string
}

type ListTrustedContactsOutput struct {
	Contacts []TrustedContact
}

func NewService(repo Repository, authRepo auth.Repository) *Service {
	return &Service{
		repo:     repo,
		authRepo: authRepo,
	}
}

func (s *Service) CreateRequest(ctx context.Context, userID string, input CreateRequestInput) (*TrustedContactRequest, string, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, "", ErrUnauthorized
	}

	phone := normalizePhone(input.Phone)
	if phone == "" {
		return nil, "", ErrInvalidPhone
	}

	email := normalizeEmail(input.Email)

	// Validate if the contact is a registered user on the platform and get their username
	var targetUser *auth.User
	var err error
	if phone != "" {
		targetUser, err = s.authRepo.GetUserByPhone(ctx, phone)
		if err == nil {
			// Found by phone
		} else if email != "" {
			targetUser, err = s.authRepo.GetUserByEmail(ctx, email)
		}
	} else if email != "" {
		targetUser, err = s.authRepo.GetUserByEmail(ctx, email)
	}

	if targetUser == nil || err != nil {
		return nil, "", ErrContactNotRegistered
	}

	// Use the target user's username as the name
	name := targetUser.Username

	if _, err := s.repo.GetTrustedContactByUserPhone(ctx, userID, phone); err == nil {
		return nil, "", ErrTrustedContactExists
	} else if !errors.Is(err, ErrTrustedContactNotFound) {
		return nil, "", err
	}

	now := time.Now().UTC()
	if _, err := s.repo.GetActiveRequestByUserPhone(ctx, userID, phone, now); err == nil {
		return nil, "", ErrPendingRequestExists
	} else if !errors.Is(err, ErrRequestNotFound) {
		return nil, "", err
	}

	token, tokenHash, err := generateInviteToken()
	if err != nil {
		return nil, "", err
	}

	request := &TrustedContactRequest{
		UserID:          userID,
		Name:            name,
		Phone:           phone,
		Status:          RequestStatusPending,
		InviteTokenHash: tokenHash,
		ExpiresAt:       now.Add(defaultRequestTTL),
	}

	if email != "" {
		request.Email = &email
	}

	if err := s.repo.CreateRequest(ctx, request); err != nil {
		return nil, "", err
	}

	return request, token, nil
}

func (s *Service) AcceptRequest(ctx context.Context, requestID string, input AcceptRequestInput) (*TrustedContactRequest, *TrustedContact, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, nil, ErrInvalidRequestID
	}

	token := strings.TrimSpace(input.Token)
	if token == "" {
		return nil, nil, ErrInvalidRequestToken
	}

	request, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, nil, err
	}

	if request.Status != RequestStatusPending {
		return nil, nil, ErrRequestAlreadyProcessed
	}

	now := time.Now().UTC()
	if request.ExpiresAt.Before(now) {
		request.Status = RequestStatusExpired
		request.RespondedAt = &now
		if err := s.repo.UpdateRequestState(ctx, request.ID, RequestStatusExpired, request.RespondedAt); err != nil {
			return nil, nil, err
		}

		return nil, nil, ErrRequestExpired
	}

	if !compareInviteToken(request.InviteTokenHash, token) {
		return nil, nil, ErrInvalidRequestToken
	}

	if _, err := s.repo.GetTrustedContactByUserPhone(ctx, request.UserID, request.Phone); err == nil {
		return nil, nil, ErrTrustedContactExists
	} else if !errors.Is(err, ErrTrustedContactNotFound) {
		return nil, nil, err
	}

	contact := &TrustedContact{
		UserID:     request.UserID,
		RequestID:  &request.ID,
		Name:       request.Name,
		Phone:      request.Phone,
		Email:      request.Email,
		AcceptedAt: now,
	}

	request.Status = RequestStatusAccepted
	request.RespondedAt = &now

	if err := s.repo.CompleteRequestAcceptance(ctx, request, contact); err != nil {
		return nil, nil, err
	}

	request.AcceptedContactID = &contact.ID

	return request, contact, nil
}

func (s *Service) RemoveTrustedContact(ctx context.Context, userID, contactID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrUnauthorized
	}

	if strings.TrimSpace(contactID) == "" {
		return ErrTrustedContactNotFound
	}

	return s.repo.DeleteTrustedContact(ctx, userID, contactID)
}

func (s *Service) ListTrustedContacts(ctx context.Context, userID string) ([]TrustedContact, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	return s.repo.ListTrustedContactsByUserID(ctx, userID)
}

func (s *Service) ListPendingRequestsForUser(ctx context.Context, userPhone string) ([]TrustedContactRequest, error) {
	phone := normalizePhone(userPhone)
	if phone == "" {
		return nil, ErrInvalidPhone
	}

	now := time.Now().UTC()
	return s.repo.ListPendingRequestsForPhone(ctx, phone, now)
}

func (s *Service) ListOutgoingRequests(ctx context.Context, userID string) ([]TrustedContactRequest, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	return s.repo.ListOutgoingRequestsByUserID(ctx, userID)
}

func (s *Service) RejectRequest(ctx context.Context, requestID string, userPhone string) (*TrustedContactRequest, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, ErrInvalidRequestID
	}

	phone := normalizePhone(userPhone)
	if phone == "" {
		return nil, ErrUnauthorized
	}

	request, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	// Verify the request is for this user's phone
	if request.Phone != phone {
		return nil, ErrUnauthorized
	}

	if request.Status != RequestStatusPending {
		return nil, ErrRequestAlreadyProcessed
	}

	now := time.Now().UTC()
	request.Status = RequestStatusRejected
	request.RespondedAt = &now

	if err := s.repo.UpdateRequestState(ctx, request.ID, RequestStatusRejected, request.RespondedAt); err != nil {
		return nil, err
	}

	return request, nil
}

// AcceptRequestByPhone allows an authenticated user to accept a request sent to their phone
// without needing the original invite token
func (s *Service) AcceptRequestByPhone(ctx context.Context, requestID string, userPhone string) (*TrustedContactRequest, *TrustedContact, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, nil, ErrInvalidRequestID
	}

	phone := normalizePhone(userPhone)
	if phone == "" {
		return nil, nil, ErrUnauthorized
	}

	request, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, nil, err
	}

	// Verify the request is for this user's phone
	if request.Phone != phone {
		return nil, nil, ErrUnauthorized
	}

	if request.Status != RequestStatusPending {
		return nil, nil, ErrRequestAlreadyProcessed
	}

	now := time.Now().UTC()
	if request.ExpiresAt.Before(now) {
		request.Status = RequestStatusExpired
		request.RespondedAt = &now
		if err := s.repo.UpdateRequestState(ctx, request.ID, RequestStatusExpired, request.RespondedAt); err != nil {
			return nil, nil, err
		}
		return nil, nil, ErrRequestExpired
	}

	// Check if contact already exists
	if _, err := s.repo.GetTrustedContactByUserPhone(ctx, request.UserID, request.Phone); err == nil {
		return nil, nil, ErrTrustedContactExists
	} else if !errors.Is(err, ErrTrustedContactNotFound) {
		return nil, nil, err
	}

	contact := &TrustedContact{
		UserID:     request.UserID,
		RequestID:  &request.ID,
		Name:       request.Name,
		Phone:      request.Phone,
		Email:      request.Email,
		AcceptedAt: now,
	}

	request.Status = RequestStatusAccepted
	request.RespondedAt = &now

	if err := s.repo.CompleteRequestAcceptance(ctx, request, contact); err != nil {
		return nil, nil, err
	}

	request.AcceptedContactID = &contact.ID

	return request, contact, nil
}

func normalizePhone(phone string) string {
	num, err := phonenumbers.Parse(phone, "IN") // default to India if no code provided
	if err != nil {
		return ""
	}

	if !phonenumbers.IsValidNumber(num) {
		return ""
	}

	return phonenumbers.Format(num, phonenumbers.E164)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
