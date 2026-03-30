package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ExpoPushSender implements notify.Sender using Expo's Push API.
type ExpoPushSender struct {
	httpClient *http.Client
}

type expoPushMessage struct {
	To    string `json:"to"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Data  any    `json:"data,omitempty"`
}

func NewExpoPushSender() *ExpoPushSender {
	return &ExpoPushSender{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *ExpoPushSender) Send(ctx context.Context, notification Notification) (Result, error) {
	if s == nil || s.httpClient == nil {
		return Result{}, fmt.Errorf("expo push sender is not initialized")
	}

	if err := ValidateNotification(notification); err != nil {
		return Result{}, err
	}

	// We can only send push if we have a token
	pushToken := strings.TrimSpace(notification.Recipient.PushToken)
	if pushToken == "" {
		return Result{}, fmt.Errorf("recipient push token is required for expo sender")
	}

	title := "SafeRoute SOS Alert!"
	body := RenderMessage(notification)

	msg := expoPushMessage{
		To:    pushToken,
		Title: title,
		Body:  body,
		Data:  notification.SOSAlert,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return Result{}, fmt.Errorf("failed to marshal expo push message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://exp.host/--/api/v2/push/send", bytes.NewBuffer(payload))
	if err != nil {
		return Result{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("failed to send expo push request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("expo push API returned status %d", resp.StatusCode)
	}

	result := Result{
		Channel:          notification.Channel,
		Template:         notification.Template,
		TrustedContactID: strings.TrimSpace(notification.Recipient.TrustedContactID),
		RecipientHint:    recipientHint(notification),
		Status:           "sent",
		Message:          body,
		DeliveredAt:      time.Now().UTC(),
	}

	return result, nil
}
