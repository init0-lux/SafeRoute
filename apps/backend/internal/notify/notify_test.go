package notify

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestBuildSOSStartedNotificationDefaultsStartedAt(t *testing.T) {
	notification, err := BuildSOSStartedNotification(
		ChannelSMS,
		Recipient{
			TrustedContactID: "contact-1",
			Name:             "Emergency Contact",
			Phone:            "+919999911111",
		},
		SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ViewerURL:          "http://localhost:8080/api/v1/sos/viewer/stream?token=viewer-token",
			ReporterIdentifier: "+918888822222",
		},
	)
	if err != nil {
		t.Fatalf("expected notification build to succeed, got %v", err)
	}

	if notification.Template != TemplateSOSStarted {
		t.Fatalf("expected template %q, got %q", TemplateSOSStarted, notification.Template)
	}

	if notification.SOSAlert == nil {
		t.Fatal("expected sos alert payload to be present")
	}

	if notification.SOSAlert.StartedAt.IsZero() {
		t.Fatal("expected started_at to be defaulted")
	}
}

func TestValidateNotificationRequiresViewerURL(t *testing.T) {
	err := ValidateNotification(Notification{
		Channel:  ChannelSMS,
		Template: TemplateSOSStarted,
		Recipient: Recipient{
			TrustedContactID: "contact-1",
			Phone:            "+919999911111",
		},
		SOSAlert: &SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ReporterIdentifier: "+918888822222",
		},
	})
	if err != ErrMissingViewerURL {
		t.Fatalf("expected ErrMissingViewerURL, got %v", err)
	}
}

func TestRenderMessageUsesCustomMessageWhenProvided(t *testing.T) {
	message := RenderMessage(Notification{
		Channel:  ChannelSMS,
		Template: TemplateSOSStarted,
		Recipient: Recipient{
			TrustedContactID: "contact-1",
			Phone:            "+919999911111",
		},
		SOSAlert: &SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ViewerURL:          "http://localhost/view",
			ReporterIdentifier: "+918888822222",
			Message:            "Custom emergency message",
			StartedAt:          time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		},
	})

	if message != "Custom emergency message" {
		t.Fatalf("expected custom message to be used, got %q", message)
	}
}

func TestRenderMessageBuildsDefaultSOSMessage(t *testing.T) {
	message := RenderMessage(Notification{
		Channel:  ChannelSMS,
		Template: TemplateSOSStarted,
		Recipient: Recipient{
			TrustedContactID: "contact-1",
			Phone:            "+919999911111",
		},
		SOSAlert: &SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ViewerURL:          "http://localhost/view",
			ReporterIdentifier: "+918888822222",
			StartedAt:          time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		},
	})

	expected := "+918888822222 triggered SOS at 2026-03-31T12:00:00Z. Open the live session: http://localhost/view"
	if message != expected {
		t.Fatalf("expected %q, got %q", expected, message)
	}
}

func TestMultiSenderDelegatesToRegisteredSender(t *testing.T) {
	sender := NewMultiSender()
	expectedResult := Result{
		Channel:          ChannelSMS,
		Template:         TemplateSOSStarted,
		TrustedContactID: "contact-1",
		RecipientHint:    "+919999911111",
		Status:           "sent",
		Message:          "ok",
		DeliveredAt:      time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
	}
	sender.Register(ChannelSMS, stubSender{
		result: expectedResult,
	})

	notification, err := BuildSOSStartedNotification(
		ChannelSMS,
		Recipient{
			TrustedContactID: "contact-1",
			Phone:            "+919999911111",
		},
		SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ViewerURL:          "http://localhost/view",
			ReporterIdentifier: "+918888822222",
		},
	)
	if err != nil {
		t.Fatalf("expected notification build to succeed, got %v", err)
	}

	result, err := sender.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected multi sender send to succeed, got %v", err)
	}

	if result.Channel != expectedResult.Channel ||
		result.Template != expectedResult.Template ||
		result.TrustedContactID != expectedResult.TrustedContactID ||
		result.RecipientHint != expectedResult.RecipientHint ||
		result.Status != expectedResult.Status ||
		result.Message != expectedResult.Message ||
		!result.DeliveredAt.Equal(expectedResult.DeliveredAt) {
		t.Fatalf("expected result %#v, got %#v", expectedResult, result)
	}
}

func TestMultiSenderReturnsErrorWhenChannelIsNotRegistered(t *testing.T) {
	sender := NewMultiSender()

	notification, err := BuildSOSStartedNotification(
		ChannelSMS,
		Recipient{
			TrustedContactID: "contact-1",
			Phone:            "+919999911111",
		},
		SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ViewerURL:          "http://localhost/view",
			ReporterIdentifier: "+918888822222",
		},
	)
	if err != nil {
		t.Fatalf("expected notification build to succeed, got %v", err)
	}

	if _, err := sender.Send(context.Background(), notification); err == nil {
		t.Fatal("expected missing sender registration to fail")
	}
}

func TestDevSenderSendsRenderedMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sender := NewDevSender(logger)

	notification, err := BuildSOSStartedNotification(
		ChannelSMS,
		Recipient{
			TrustedContactID: "contact-1",
			Name:             "Emergency Contact",
			Phone:            "+919999911111",
		},
		SOSAlertPayload{
			SOSSessionID:       "sos-1",
			ViewerToken:        "viewer-token",
			ViewerURL:          "http://localhost/view",
			ReporterIdentifier: "+918888822222",
			StartedAt:          time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		},
	)
	if err != nil {
		t.Fatalf("expected notification build to succeed, got %v", err)
	}

	result, err := sender.Send(context.Background(), notification)
	if err != nil {
		t.Fatalf("expected dev sender to succeed, got %v", err)
	}

	if result.Status != "sent" {
		t.Fatalf("expected status sent, got %q", result.Status)
	}

	if result.TrustedContactID != "contact-1" {
		t.Fatalf("expected trusted contact id contact-1, got %q", result.TrustedContactID)
	}

	expectedMessage := "+918888822222 triggered SOS at 2026-03-31T12:00:00Z. Open the live session: http://localhost/view"
	if result.Message != expectedMessage {
		t.Fatalf("expected %q, got %q", expectedMessage, result.Message)
	}
}

func TestRecipientHintPrefersPhoneThenEmailThenName(t *testing.T) {
	withPhone := recipientHint(Notification{
		Recipient: Recipient{
			Name:  "Primary Contact",
			Phone: "+919999911111",
			Email: "helper@example.com",
		},
	})
	if withPhone != "+919999911111" {
		t.Fatalf("expected phone recipient hint, got %q", withPhone)
	}

	withEmail := recipientHint(Notification{
		Recipient: Recipient{
			Name:  "Primary Contact",
			Email: "helper@example.com",
		},
	})
	if withEmail != "helper@example.com" {
		t.Fatalf("expected email recipient hint, got %q", withEmail)
	}

	withName := recipientHint(Notification{
		Recipient: Recipient{
			Name: "Primary Contact",
		},
	})
	if withName != "Primary Contact" {
		t.Fatalf("expected name recipient hint, got %q", withName)
	}
}

type stubSender struct {
	result Result
	err    error
}

func (s stubSender) Send(_ context.Context, _ Notification) (Result, error) {
	return s.result, s.err
}
