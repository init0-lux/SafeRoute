package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

var (
	ErrMissingRecipient          = errors.New("notification recipient is required")
	ErrMissingChannel            = errors.New("notification channel is required")
	ErrMissingTemplate           = errors.New("notification template is required")
	ErrMissingSOSSessionID       = errors.New("sos session id is required")
	ErrMissingViewerToken        = errors.New("viewer token is required")
	ErrMissingViewerURL          = errors.New("viewer url is required")
	ErrMissingReporterIdentifier = errors.New("reporter identifier is required")
)

type Channel string

const (
	ChannelSMS   Channel = "sms"
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
)

type Template string

const (
	TemplateSOSStarted Template = "sos_started"
)

type Recipient struct {
	TrustedContactID string
	Name             string
	Phone            string
	Email            string
	PushToken        string
}

type SOSAlertPayload struct {
	Type               string     `json:"type"`
	SOSSessionID       string     `json:"sos_session_id"`
	ViewerToken        string     `json:"viewer_token"`
	ViewerURL          string     `json:"viewer_url"`
	ReporterIdentifier string     `json:"reporter_identifier"`
	StartedAt          time.Time  `json:"started_at"`
	Latitude           *float64   `json:"lat,omitempty"`
	Longitude          *float64   `json:"lng,omitempty"`
	RecordedAt         *time.Time `json:"recorded_at,omitempty"`
	Message            string     `json:"message,omitempty"`
}

type Notification struct {
	Channel    Channel
	Template   Template
	Recipient  Recipient
	SOSAlert   *SOSAlertPayload
	Metadata   map[string]string
	OccurredAt time.Time
}

type Result struct {
	Channel          Channel           `json:"channel"`
	Template         Template          `json:"template"`
	TrustedContactID string            `json:"trusted_contact_id"`
	RecipientHint    string            `json:"recipient_hint"`
	Status           string            `json:"status"`
	Message          string            `json:"message"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	DeliveredAt      time.Time         `json:"delivered_at"`
}

type Sender interface {
	Send(ctx context.Context, notification Notification) (Result, error)
}

type MultiSender struct {
	senders map[Channel]Sender
}

type DevSender struct {
	logger *slog.Logger
}

func NewMultiSender() *MultiSender {
	return &MultiSender{
		senders: make(map[Channel]Sender),
	}
}

func (m *MultiSender) Register(channel Channel, sender Sender) {
	if m == nil || sender == nil {
		return
	}
	m.senders[channel] = sender
}

func (m *MultiSender) Send(ctx context.Context, notification Notification) (Result, error) {
	if err := ValidateNotification(notification); err != nil {
		return Result{}, err
	}

	sender, ok := m.senders[notification.Channel]
	if !ok {
		return Result{}, fmt.Errorf("no sender registered for channel %q", notification.Channel)
	}

	return sender.Send(ctx, notification)
}

func NewDevSender(logger *slog.Logger) *DevSender {
	if logger == nil {
		logger = slog.Default()
	}

	return &DevSender{
		logger: logger,
	}
}

func (s *DevSender) Send(ctx context.Context, notification Notification) (Result, error) {
	if err := ValidateNotification(notification); err != nil {
		return Result{}, err
	}

	result := Result{
		Channel:          notification.Channel,
		Template:         notification.Template,
		TrustedContactID: strings.TrimSpace(notification.Recipient.TrustedContactID),
		RecipientHint:    recipientHint(notification),
		Status:           "sent",
		Message:          RenderMessage(notification),
		Metadata:         cloneMetadata(notification.Metadata),
		DeliveredAt:      time.Now().UTC(),
	}

	s.logger.InfoContext(
		ctx,
		"dev notification sent",
		"channel", result.Channel,
		"template", result.Template,
		"trusted_contact_id", result.TrustedContactID,
		"recipient", result.RecipientHint,
		"sos_session_id", notification.SOSAlert.SOSSessionID,
		"viewer_url", notification.SOSAlert.ViewerURL,
	)

	return result, nil
}

func ValidateNotification(notification Notification) error {
	if strings.TrimSpace(notification.Recipient.TrustedContactID) == "" &&
		strings.TrimSpace(notification.Recipient.Phone) == "" &&
		strings.TrimSpace(notification.Recipient.Email) == "" {
		return ErrMissingRecipient
	}

	if strings.TrimSpace(string(notification.Channel)) == "" {
		return ErrMissingChannel
	}

	if strings.TrimSpace(string(notification.Template)) == "" {
		return ErrMissingTemplate
	}

	switch notification.Template {
	case TemplateSOSStarted:
		if notification.SOSAlert == nil {
			return ErrMissingSOSSessionID
		}
		if strings.TrimSpace(notification.SOSAlert.SOSSessionID) == "" {
			return ErrMissingSOSSessionID
		}
		if strings.TrimSpace(notification.SOSAlert.ViewerToken) == "" {
			return ErrMissingViewerToken
		}
		if strings.TrimSpace(notification.SOSAlert.ViewerURL) == "" {
			return ErrMissingViewerURL
		}
		if strings.TrimSpace(notification.SOSAlert.ReporterIdentifier) == "" {
			return ErrMissingReporterIdentifier
		}
	}

	return nil
}

func RenderMessage(notification Notification) string {
	switch notification.Template {
	case TemplateSOSStarted:
		payload := notification.SOSAlert
		startedAt := payload.StartedAt.UTC().Format(time.RFC3339)
		customMessage := strings.TrimSpace(payload.Message)
		if customMessage == "" {
			customMessage = fmt.Sprintf(
				"%s triggered SOS at %s. Open the live session: %s",
				payload.ReporterIdentifier,
				startedAt,
				payload.ViewerURL,
			)
		}

		return customMessage
	default:
		return "SafeRoute notification"
	}
}

func BuildSOSStartedNotification(channel Channel, recipient Recipient, payload SOSAlertPayload) (Notification, error) {
	notification := Notification{
		Channel:    channel,
		Template:   TemplateSOSStarted,
		Recipient:  recipient,
		SOSAlert:   &payload,
		OccurredAt: time.Now().UTC(),
	}

	if notification.SOSAlert.StartedAt.IsZero() {
		notification.SOSAlert.StartedAt = time.Now().UTC()
	}
	if strings.TrimSpace(notification.SOSAlert.Type) == "" {
		notification.SOSAlert.Type = string(TemplateSOSStarted)
	}

	if err := ValidateNotification(notification); err != nil {
		return Notification{}, err
	}

	return notification, nil
}

func recipientHint(notification Notification) string {
	if phone := strings.TrimSpace(notification.Recipient.Phone); phone != "" {
		return phone
	}
	if email := strings.TrimSpace(notification.Recipient.Email); email != "" {
		return email
	}
	return strings.TrimSpace(notification.Recipient.Name)
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}
