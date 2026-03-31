package trustedcontacts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	CreateRequest(ctx context.Context, request *TrustedContactRequest) error
	GetRequestByID(ctx context.Context, id string) (*TrustedContactRequest, error)
	GetActiveRequestByUserPhone(ctx context.Context, userID, phone string, now time.Time) (*TrustedContactRequest, error)
	GetTrustedContactByUserPhone(ctx context.Context, userID, phone string) (*TrustedContact, error)
	ListTrustedContactsByUserID(ctx context.Context, userID string) ([]TrustedContact, error)
	ListPendingRequestsForPhone(ctx context.Context, phone string, now time.Time) ([]TrustedContactRequest, error)
	ListOutgoingRequestsByUserID(ctx context.Context, userID string) ([]TrustedContactRequest, error)
	CompleteRequestAcceptance(ctx context.Context, request *TrustedContactRequest, contact *TrustedContact) error
	UpdateRequestState(ctx context.Context, requestID string, status RequestStatus, respondedAt *time.Time) error
	DeleteTrustedContact(ctx context.Context, userID, contactID string) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) CreateRequest(ctx context.Context, request *TrustedContactRequest) error {
	if err := r.db.WithContext(ctx).Create(request).Error; err != nil {
		return err
	}

	return nil
}

func (r *GormRepository) GetRequestByID(ctx context.Context, id string) (*TrustedContactRequest, error) {
	var request TrustedContactRequest
	if err := r.db.WithContext(ctx).First(&request, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRequestNotFound
		}

		return nil, err
	}

	return &request, nil
}

func (r *GormRepository) GetActiveRequestByUserPhone(ctx context.Context, userID, phone string, now time.Time) (*TrustedContactRequest, error) {
	var request TrustedContactRequest
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND phone = ? AND status = ? AND expires_at > ?", userID, phone, RequestStatusPending, now).
		Order("created_at DESC").
		First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRequestNotFound
		}

		return nil, err
	}

	return &request, nil
}

func (r *GormRepository) GetTrustedContactByUserPhone(ctx context.Context, userID, phone string) (*TrustedContact, error) {
	var contact TrustedContact
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND phone = ?", userID, phone).
		First(&contact).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTrustedContactNotFound
		}

		return nil, err
	}

	return &contact, nil
}

func (r *GormRepository) ListTrustedContactsByUserID(ctx context.Context, userID string) ([]TrustedContact, error) {
	var contacts []TrustedContact
	if err := r.db.WithContext(ctx).
		Select("trusted_contacts.*, users.expo_push_token AS push_token").
		Joins("LEFT JOIN users ON users.phone = trusted_contacts.phone").
		Where("trusted_contacts.user_id = ?", userID).
		Order("trusted_contacts.created_at ASC").
		Find(&contacts).Error; err != nil {
		return nil, err
	}

	return contacts, nil
}

func (r *GormRepository) ListPendingRequestsForPhone(ctx context.Context, phone string, now time.Time) ([]TrustedContactRequest, error) {
	var requests []TrustedContactRequest
	if err := r.db.WithContext(ctx).
		Where("phone = ? AND status = ? AND expires_at > ?", phone, RequestStatusPending, now).
		Order("created_at DESC").
		Find(&requests).Error; err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *GormRepository) ListOutgoingRequestsByUserID(ctx context.Context, userID string) ([]TrustedContactRequest, error) {
	var requests []TrustedContactRequest
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, RequestStatusPending).
		Order("created_at DESC").
		Find(&requests).Error; err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *GormRepository) CompleteRequestAcceptance(ctx context.Context, request *TrustedContactRequest, contact *TrustedContact) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(contact).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrTrustedContactExists
			}

			return err
		}

		updates := map[string]any{
			"status":              RequestStatusAccepted,
			"accepted_contact_id": contact.ID,
			"responded_at":        request.RespondedAt,
		}

		result := tx.Model(&TrustedContactRequest{}).
			Where("id = ? AND status = ?", request.ID, RequestStatusPending).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return ErrRequestAlreadyProcessed
		}

		return nil
	})
}

func (r *GormRepository) UpdateRequestState(ctx context.Context, requestID string, status RequestStatus, respondedAt *time.Time) error {
	updates := map[string]any{
		"status": status,
	}
	if respondedAt != nil {
		updates["responded_at"] = respondedAt
	}

	result := r.db.WithContext(ctx).
		Model(&TrustedContactRequest{}).
		Where("id = ?", requestID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrRequestNotFound
	}

	return nil
}

func (r *GormRepository) DeleteTrustedContact(ctx context.Context, userID, contactID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", contactID, userID).
		Delete(&TrustedContact{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrTrustedContactNotFound
	}

	return nil
}
