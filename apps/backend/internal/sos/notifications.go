package sos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"saferoute-backend/internal/notify"
	"saferoute-backend/internal/trustedcontacts"
)

type TrustedContactsReader interface {
	ListTrustedContacts(ctx context.Context, userID string) ([]trustedcontacts.TrustedContact, error)
}

type NotificationFanoutResult struct {
	ContactID   string         `json:"contact_id"`
	Channel     notify.Channel `json:"channel"`
	ViewerToken string         `json:"viewer_token"`
	ViewerURL   string         `json:"viewer_url"`
	Status      string         `json:"status"`
	Message     string         `json:"message,omitempty"`
}

type NotificationFanoutSummary struct {
	SessionID  string                     `json:"session_id"`
	UserID     string                     `json:"user_id"`
	StartedAt  time.Time                  `json:"started_at"`
	Results    []NotificationFanoutResult `json:"results"`
	Successful int                        `json:"successful"`
	Failed     int                        `json:"failed"`
}

func (s *Service) NotifyTrustedContactsForSession(
	ctx context.Context,
	contactsReader TrustedContactsReader,
	sender notify.Sender,
	sessionID string,
	userID string,
	baseViewerURL string,
) (*NotificationFanoutSummary, error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	return s.NotifyTrustedContacts(
		ctx,
		contactsReader,
		sender,
		session,
		baseViewerURL,
	)
}

func (s *Service) NotifyTrustedContacts(
	ctx context.Context,
	contactsReader TrustedContactsReader,
	sender notify.Sender,
	session *SOSSession,
	baseViewerURL string,
) (*NotificationFanoutSummary, error) {
	if session == nil {
		return nil, ErrSessionNotFound
	}
	if session.UserID == nil || strings.TrimSpace(*session.UserID) == "" {
		return nil, ErrInvalidUserID
	}
	if strings.TrimSpace(session.ID) == "" {
		return nil, ErrInvalidSessionID
	}
	if contactsReader == nil {
		return nil, fmt.Errorf("trusted contacts reader is required")
	}
	if sender == nil {
		return nil, fmt.Errorf("notification sender is required")
	}

	userID := strings.TrimSpace(*session.UserID)
	contacts, err := contactsReader.ListTrustedContacts(ctx, userID)
	if err != nil {
		return nil, err
	}

	summary := &NotificationFanoutSummary{
		SessionID: session.ID,
		UserID:    userID,
		StartedAt: session.StartedAt.UTC(),
		Results:   make([]NotificationFanoutResult, 0, len(contacts)),
	}
	latestLocation, err := s.GetLatestLocation(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	reporterIdentifier := userID
	for _, contact := range contacts {
		_, token, err := s.ReplaceViewerGrant(ctx, session.ID, userID, contact.ID)
		if err != nil {
			summary.Results = append(summary.Results, NotificationFanoutResult{
				ContactID: contact.ID,
				Status:    "failed",
				Message:   err.Error(),
			})
			summary.Failed++
			continue
		}

		channel := notificationChannelForContact(contact)
		viewerURL := BuildViewerStreamURL(baseViewerURL, token)

		message, buildErr := notify.BuildSOSStartedNotification(
			channel,
			notify.Recipient{
				TrustedContactID: contact.ID,
				Name:             contact.Name,
				Phone:            derefString(contact.Phone),
				Email:            derefOptional(contact.Email),
				PushToken:        contact.PushToken,
			},
			notify.SOSAlertPayload{
				SOSSessionID:       session.ID,
				ViewerToken:        token,
				ViewerURL:          viewerURL,
				ReporterIdentifier: reporterIdentifier,
				StartedAt:          session.StartedAt.UTC(),
				Latitude:           locationLatitude(latestLocation),
				Longitude:          locationLongitude(latestLocation),
				RecordedAt:         locationRecordedAt(latestLocation),
			},
		)
		if buildErr != nil {
			summary.Results = append(summary.Results, NotificationFanoutResult{
				ContactID:   contact.ID,
				Channel:     channel,
				ViewerToken: token,
				ViewerURL:   viewerURL,
				Status:      "failed",
				Message:     buildErr.Error(),
			})
			summary.Failed++
			continue
		}

		result, sendErr := sender.Send(ctx, message)
		if sendErr != nil {
			summary.Results = append(summary.Results, NotificationFanoutResult{
				ContactID:   contact.ID,
				Channel:     channel,
				ViewerToken: token,
				ViewerURL:   viewerURL,
				Status:      "failed",
				Message:     sendErr.Error(),
			})
			summary.Failed++
			continue
		}

		summary.Results = append(summary.Results, NotificationFanoutResult{
			ContactID:   contact.ID,
			Channel:     channel,
			ViewerToken: token,
			ViewerURL:   viewerURL,
			Status:      result.Status,
			Message:     result.Message,
		})
		summary.Successful++
	}

	return summary, nil
}

func locationLatitude(location *LocationSnapshot) *float64 {
	if location == nil {
		return nil
	}

	latitude := location.Latitude
	return &latitude
}

func locationLongitude(location *LocationSnapshot) *float64 {
	if location == nil {
		return nil
	}

	longitude := location.Longitude
	return &longitude
}

func locationRecordedAt(location *LocationSnapshot) *time.Time {
	if location == nil {
		return nil
	}

	recordedAt := location.RecordedAt.UTC()
	return &recordedAt
}

func BuildViewerStreamURL(baseViewerURL, viewerToken string) string {
	baseViewerURL = strings.TrimSpace(baseViewerURL)
	viewerToken = strings.TrimSpace(viewerToken)

	if baseViewerURL == "" {
		return fmt.Sprintf("/api/v1/sos/viewer/stream?token=%s", viewerToken)
	}

	baseViewerURL = strings.TrimRight(baseViewerURL, "/")
	if strings.Contains(baseViewerURL, "?") {
		return baseViewerURL + "&token=" + viewerToken
	}

	return baseViewerURL + "?token=" + viewerToken
}

func notificationChannelForContact(contact trustedcontacts.TrustedContact) notify.Channel {
	if strings.TrimSpace(contact.PushToken) != "" {
		return notify.ChannelPush
	}
	if strings.TrimSpace(derefOptional(contact.Email)) != "" {
		return notify.ChannelEmail
	}
	return notify.ChannelSMS
}

func derefOptional(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func derefString(value string) string {
	return strings.TrimSpace(value)
}
