package sos_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"saferoute-backend/internal/notify"
	"saferoute-backend/internal/sos"
	"saferoute-backend/internal/trustedcontacts"
)

func TestBuildViewerStreamURLUsesDefaultPathWhenBaseIsEmpty(t *testing.T) {
	got := sos.BuildViewerStreamURL("", "viewer-token")

	want := "/api/v1/sos/viewer/stream?token=viewer-token"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildViewerStreamURLAppendsTokenToBaseURL(t *testing.T) {
	got := sos.BuildViewerStreamURL("https://example.com/api/v1/sos/viewer/stream", "viewer-token")

	want := "https://example.com/api/v1/sos/viewer/stream?token=viewer-token"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildViewerStreamURLAppendsTokenToExistingQuery(t *testing.T) {
	got := sos.BuildViewerStreamURL("https://example.com/api/v1/sos/viewer/stream?source=sms", "viewer-token")

	want := "https://example.com/api/v1/sos/viewer/stream?source=sms&token=viewer-token"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNotifyTrustedContactsFailsWhenSessionIsNil(t *testing.T) {
	service := sos.NewService(newMemorySOSRepository())
	contactsReader := &stubTrustedContactsReader{}
	sender := notify.NewDevSender(nil)

	if _, err := service.NotifyTrustedContacts(context.Background(), contactsReader, sender, nil, ""); err == nil {
		t.Fatal("expected nil session to fail")
	} else if err != sos.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestNotifyTrustedContactsFailsWhenSessionHasNoUserID(t *testing.T) {
	service := sos.NewService(newMemorySOSRepository())
	contactsReader := &stubTrustedContactsReader{}
	sender := notify.NewDevSender(nil)

	session := &sos.SOSSession{
		ID:        "sos-1",
		Status:    sos.SessionStatusActive,
		StartedAt: time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
	}

	if _, err := service.NotifyTrustedContacts(context.Background(), contactsReader, sender, session, ""); err == nil {
		t.Fatal("expected missing user id to fail")
	} else if err != sos.ErrInvalidUserID {
		t.Fatalf("expected ErrInvalidUserID, got %v", err)
	}
}

func TestNotifyTrustedContactsFailsWhenContactsReaderIsNil(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	sender := notify.NewDevSender(nil)
	userID := "user-1"

	session := mustStartSession(t, service, userID)

	if _, err := service.NotifyTrustedContacts(context.Background(), nil, sender, session, ""); err == nil {
		t.Fatal("expected nil trusted contacts reader to fail")
	}
}

func TestNotifyTrustedContactsFailsWhenSenderIsNil(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	userID := "user-1"

	session := mustStartSession(t, service, userID)
	contactsReader := &stubTrustedContactsReader{}

	if _, err := service.NotifyTrustedContacts(context.Background(), contactsReader, nil, session, ""); err == nil {
		t.Fatal("expected nil sender to fail")
	}
}

func TestNotifyTrustedContactsForSessionReturnsSessionError(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	contactsReader := &stubTrustedContactsReader{}
	sender := notify.NewDevSender(nil)

	if _, err := service.NotifyTrustedContactsForSession(
		context.Background(),
		contactsReader,
		sender,
		"missing-session",
		"user-1",
		"",
	); err == nil {
		t.Fatal("expected missing session to fail")
	} else if err != sos.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestNotifyTrustedContactsReturnsReaderError(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	sender := notify.NewDevSender(nil)
	userID := "user-1"

	session := mustStartSession(t, service, userID)
	contactsReader := &stubTrustedContactsReader{
		err: errors.New("trusted contacts unavailable"),
	}

	if _, err := service.NotifyTrustedContacts(context.Background(), contactsReader, sender, session, ""); err == nil {
		t.Fatal("expected reader error to fail")
	} else if err.Error() != "trusted contacts unavailable" {
		t.Fatalf("expected trusted contacts unavailable error, got %v", err)
	}
}

func TestNotifyTrustedContactsCreatesViewerGrantAndSendsNotifications(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	userID := "user-1"

	session := mustStartSession(t, service, userID)
	repo.AddTrustedContact(userID, "contact-1")
	repo.AddTrustedContact(userID, "contact-2")

	email := "helper@example.com"
	contactsReader := &stubTrustedContactsReader{
		contacts: []trustedcontacts.TrustedContact{
			{
				ID:     "contact-1",
				UserID: userID,
				Name:   "Primary Contact",
				Phone:  "+919999911111",
			},
			{
				ID:     "contact-2",
				UserID: userID,
				Name:   "Email Contact",
				Phone:  "+918888822222",
				Email:  &email,
			},
		},
	}
	sender := &capturingSender{}

	summary, err := service.NotifyTrustedContacts(
		context.Background(),
		contactsReader,
		sender,
		session,
		"https://example.com/api/v1/sos/viewer/stream",
	)
	if err != nil {
		t.Fatalf("expected notify trusted contacts to succeed, got %v", err)
	}

	if summary.SessionID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, summary.SessionID)
	}

	if summary.UserID != userID {
		t.Fatalf("expected user id %q, got %q", userID, summary.UserID)
	}

	if summary.Successful != 2 {
		t.Fatalf("expected 2 successful notifications, got %d", summary.Successful)
	}

	if summary.Failed != 0 {
		t.Fatalf("expected 0 failed notifications, got %d", summary.Failed)
	}

	if len(summary.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(summary.Results))
	}

	if len(sender.notifications) != 2 {
		t.Fatalf("expected 2 sent notifications, got %d", len(sender.notifications))
	}

	for i, n := range sender.notifications {
		if n.Template != notify.TemplateSOSStarted {
			t.Fatalf("notification %d: expected template %q, got %q", i, notify.TemplateSOSStarted, n.Template)
		}

		if n.SOSAlert == nil {
			t.Fatalf("notification %d: expected SOS alert payload", i)
		}

		if n.SOSAlert.SOSSessionID != session.ID {
			t.Fatalf("notification %d: expected session id %q, got %q", i, session.ID, n.SOSAlert.SOSSessionID)
		}

		if n.SOSAlert.ViewerToken == "" {
			t.Fatalf("notification %d: expected viewer token", i)
		}

		if n.SOSAlert.ViewerURL == "" {
			t.Fatalf("notification %d: expected viewer url", i)
		}
	}

	if sender.notifications[0].Channel != notify.ChannelSMS {
		t.Fatalf("expected first notification channel sms, got %q", sender.notifications[0].Channel)
	}

	if sender.notifications[1].Channel != notify.ChannelEmail {
		t.Fatalf("expected second notification channel email, got %q", sender.notifications[1].Channel)
	}

	if sender.notifications[0].SOSAlert.ViewerToken == sender.notifications[1].SOSAlert.ViewerToken {
		t.Fatal("expected distinct viewer tokens per trusted contact")
	}

	for _, result := range summary.Results {
		if result.Status != "sent" {
			t.Fatalf("expected result status sent, got %q", result.Status)
		}

		if result.ViewerToken == "" {
			t.Fatal("expected result viewer token to be populated")
		}

		if result.ViewerURL == "" {
			t.Fatal("expected result viewer url to be populated")
		}
	}
}

func TestNotifyTrustedContactsMarksCreateViewerGrantFailures(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	userID := "user-1"

	session := mustStartSession(t, service, userID)

	contactsReader := &stubTrustedContactsReader{
		contacts: []trustedcontacts.TrustedContact{
			{
				ID:     "contact-missing",
				UserID: userID,
				Name:   "Unknown Contact",
				Phone:  "+919999911111",
			},
		},
	}
	sender := &capturingSender{}

	summary, err := service.NotifyTrustedContacts(
		context.Background(),
		contactsReader,
		sender,
		session,
		"https://example.com/api/v1/sos/viewer/stream",
	)
	if err != nil {
		t.Fatalf("expected notify trusted contacts to succeed with summary, got %v", err)
	}

	if summary.Successful != 0 {
		t.Fatalf("expected 0 successful notifications, got %d", summary.Successful)
	}

	if summary.Failed != 1 {
		t.Fatalf("expected 1 failed notification, got %d", summary.Failed)
	}

	if len(summary.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(summary.Results))
	}

	if summary.Results[0].Status != "failed" {
		t.Fatalf("expected failed result status, got %q", summary.Results[0].Status)
	}

	if summary.Results[0].Message != sos.ErrSessionForbidden.Error() {
		t.Fatalf("expected failure message %q, got %q", sos.ErrSessionForbidden.Error(), summary.Results[0].Message)
	}

	if len(sender.notifications) != 0 {
		t.Fatalf("expected 0 notifications to be sent, got %d", len(sender.notifications))
	}
}

func TestNotifyTrustedContactsMarksSendFailures(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	userID := "user-1"

	session := mustStartSession(t, service, userID)
	repo.AddTrustedContact(userID, "contact-1")

	contactsReader := &stubTrustedContactsReader{
		contacts: []trustedcontacts.TrustedContact{
			{
				ID:     "contact-1",
				UserID: userID,
				Name:   "Primary Contact",
				Phone:  "+919999911111",
			},
		},
	}
	sender := &capturingSender{
		err: errors.New("delivery failed"),
	}

	summary, err := service.NotifyTrustedContacts(
		context.Background(),
		contactsReader,
		sender,
		session,
		"https://example.com/api/v1/sos/viewer/stream",
	)
	if err != nil {
		t.Fatalf("expected notify trusted contacts to return summary, got %v", err)
	}

	if summary.Successful != 0 {
		t.Fatalf("expected 0 successful notifications, got %d", summary.Successful)
	}

	if summary.Failed != 1 {
		t.Fatalf("expected 1 failed notification, got %d", summary.Failed)
	}

	if len(summary.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(summary.Results))
	}

	if summary.Results[0].Status != "failed" {
		t.Fatalf("expected failed result status, got %q", summary.Results[0].Status)
	}

	if summary.Results[0].Message != "delivery failed" {
		t.Fatalf("expected failure message %q, got %q", "delivery failed", summary.Results[0].Message)
	}

	if len(sender.notifications) != 1 {
		t.Fatalf("expected sender to receive 1 notification, got %d", len(sender.notifications))
	}
}

func TestNotifyTrustedContactsForSessionUsesStoredSession(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)
	userID := "user-1"

	session := mustStartSession(t, service, userID)
	repo.AddTrustedContact(userID, "contact-1")

	contactsReader := &stubTrustedContactsReader{
		contacts: []trustedcontacts.TrustedContact{
			{
				ID:     "contact-1",
				UserID: userID,
				Name:   "Primary Contact",
				Phone:  "+919999911111",
			},
		},
	}
	sender := &capturingSender{}

	summary, err := service.NotifyTrustedContactsForSession(
		context.Background(),
		contactsReader,
		sender,
		session.ID,
		userID,
		"https://example.com/api/v1/sos/viewer/stream",
	)
	if err != nil {
		t.Fatalf("expected NotifyTrustedContactsForSession to succeed, got %v", err)
	}

	if summary.SessionID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, summary.SessionID)
	}

	if summary.Successful != 1 {
		t.Fatalf("expected 1 successful notification, got %d", summary.Successful)
	}
}

func mustStartSession(t *testing.T, service *sos.Service, userID string) *sos.SOSSession {
	t.Helper()

	session, err := service.StartSession(context.Background(), userID)
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	return session
}

type stubTrustedContactsReader struct {
	contacts []trustedcontacts.TrustedContact
	err      error
}

func (s *stubTrustedContactsReader) ListTrustedContacts(_ context.Context, userID string) ([]trustedcontacts.TrustedContact, error) {
	if s.err != nil {
		return nil, s.err
	}

	contacts := make([]trustedcontacts.TrustedContact, 0, len(s.contacts))
	for _, contact := range s.contacts {
		if contact.UserID != "" && userID != "" && contact.UserID != userID {
			continue
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

type capturingSender struct {
	notifications []notify.Notification
	err           error
}

func (s *capturingSender) Send(_ context.Context, n notify.Notification) (notify.Result, error) {
	s.notifications = append(s.notifications, n)

	if s.err != nil {
		return notify.Result{}, s.err
	}

	return notify.Result{
		Channel:          n.Channel,
		Template:         n.Template,
		TrustedContactID: n.Recipient.TrustedContactID,
		RecipientHint:    n.Recipient.Phone,
		Status:           "sent",
		Message:          notify.RenderMessage(n),
		DeliveredAt:      time.Now().UTC(),
	}, nil
}
