package sos_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"saferoute-backend/internal/sos"
)

func TestStartSessionRejectsSecondActiveSession(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	firstSession, err := service.StartSession(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected first start to succeed, got %v", err)
	}

	if firstSession.Status != sos.SessionStatusActive {
		t.Fatalf("expected active session, got %s", firstSession.Status)
	}

	if _, err := service.StartSession(context.Background(), "user-1"); err == nil {
		t.Fatal("expected second active session to fail")
	} else if err != sos.ErrActiveSessionExists {
		t.Fatalf("expected ErrActiveSessionExists, got %v", err)
	}
}

func TestStartSessionWithInputPersistsInitialLocation(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	latitude := 12.9716
	longitude := 77.5946
	recordedAt := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	session, err := service.StartSessionWithInput(context.Background(), "user-location", sos.StartSessionInput{
		Latitude:   &latitude,
		Longitude:  &longitude,
		RecordedAt: recordedAt,
	})
	if err != nil {
		t.Fatalf("expected start with input to succeed, got %v", err)
	}

	latestLocation, err := service.GetLatestLocation(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("expected latest location lookup to succeed, got %v", err)
	}
	if latestLocation == nil {
		t.Fatal("expected initial location to be persisted")
	}
	if latestLocation.Latitude != latitude || latestLocation.Longitude != longitude {
		t.Fatalf("expected initial location %.4f, %.4f; got %.4f, %.4f", latitude, longitude, latestLocation.Latitude, latestLocation.Longitude)
	}
	if !latestLocation.RecordedAt.Equal(recordedAt) {
		t.Fatalf("expected recorded_at %s, got %s", recordedAt, latestLocation.RecordedAt)
	}
}

func TestEndSessionTransitionsToEnded(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	endedSession, err := service.EndSession(context.Background(), session.ID, "user-2")
	if err != nil {
		t.Fatalf("failed to end session: %v", err)
	}

	if endedSession.Status != sos.SessionStatusEnded {
		t.Fatalf("expected ended status, got %s", endedSession.Status)
	}

	if endedSession.EndedAt == nil {
		t.Fatal("expected ended_at to be set")
	}
}

func TestEndSessionIsIdempotent(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-2b")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	firstEnd, err := service.EndSession(context.Background(), session.ID, "user-2b")
	if err != nil {
		t.Fatalf("failed to end session the first time: %v", err)
	}

	secondEnd, err := service.EndSession(context.Background(), session.ID, "user-2b")
	if err != nil {
		t.Fatalf("expected second end to succeed, got %v", err)
	}

	if secondEnd.Status != sos.SessionStatusEnded {
		t.Fatalf("expected ended status, got %s", secondEnd.Status)
	}

	if secondEnd.EndedAt == nil {
		t.Fatal("expected ended_at to stay populated")
	}

	if firstEnd.EndedAt == nil || !secondEnd.EndedAt.Equal(*firstEnd.EndedAt) {
		t.Fatalf("expected ended_at to remain stable across repeated end calls")
	}
}

func TestStartSessionWithInputCleansUpOnInitialLocationFailure(t *testing.T) {
	repo := newMemorySOSRepository()
	repo.SetCreateLocationPingErr(errors.New("persist ping failed"))
	service := sos.NewService(repo)

	latitude := 12.9716
	longitude := 77.5946

	if _, err := service.StartSessionWithInput(context.Background(), "user-location-failure", sos.StartSessionInput{
		Latitude:  &latitude,
		Longitude: &longitude,
	}); err == nil {
		t.Fatal("expected start with failing initial location to return an error")
	}

	if _, err := service.GetActiveSession(context.Background(), "user-location-failure"); err == nil {
		t.Fatal("expected no active session after failed startup cleanup")
	} else if err != sos.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after cleanup, got %v", err)
	}
}

func TestRecordLocationPingPersistsReporterLocation(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-3")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	recordedAt := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	if err := service.RecordLocationPing(context.Background(), session.ID, "user-3", 12.9716, 77.5946, recordedAt); err != nil {
		t.Fatalf("failed to record location ping: %v", err)
	}

	if got := repo.LocationPingCount(session.ID); got != 1 {
		t.Fatalf("expected 1 location ping, got %d", got)
	}
}

func TestCreateViewerGrantRequiresOwnedTrustedContact(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-4")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	repo.AddTrustedContact("user-4", "contact-1")

	grant, token, err := service.CreateViewerGrant(context.Background(), session.ID, "user-4", sos.CreateViewerGrantInput{
		TrustedContactID: "contact-1",
	})
	if err != nil {
		t.Fatalf("failed to create viewer grant: %v", err)
	}

	if grant.TrustedContactID != "contact-1" {
		t.Fatalf("expected trusted contact id contact-1, got %s", grant.TrustedContactID)
	}

	if token == "" {
		t.Fatal("expected viewer token to be returned")
	}
}

func TestCreateViewerGrantRejectsUnownedTrustedContact(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-5")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	if _, _, err := service.CreateViewerGrant(context.Background(), session.ID, "user-5", sos.CreateViewerGrantInput{
		TrustedContactID: "contact-missing",
	}); err == nil {
		t.Fatal("expected viewer grant creation to fail")
	} else if err != sos.ErrSessionForbidden {
		t.Fatalf("expected ErrSessionForbidden, got %v", err)
	}
}

func TestAuthorizeViewerReturnsGrantAndSession(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-6")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	repo.AddTrustedContact("user-6", "contact-2")

	grant, token, err := service.CreateViewerGrant(context.Background(), session.ID, "user-6", sos.CreateViewerGrantInput{
		TrustedContactID: "contact-2",
	})
	if err != nil {
		t.Fatalf("failed to create viewer grant: %v", err)
	}

	authorizedGrant, authorizedSession, err := service.AuthorizeViewer(context.Background(), token)
	if err != nil {
		t.Fatalf("failed to authorize viewer: %v", err)
	}

	if authorizedGrant.ID != grant.ID {
		t.Fatalf("expected grant id %s, got %s", grant.ID, authorizedGrant.ID)
	}

	if authorizedSession.ID != session.ID {
		t.Fatalf("expected session id %s, got %s", session.ID, authorizedSession.ID)
	}
}

func TestAuthorizeViewerRejectsInvalidToken(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	if _, _, err := service.AuthorizeViewer(context.Background(), "invalid-token"); err == nil {
		t.Fatal("expected invalid viewer token to fail")
	} else if err != sos.ErrViewerGrantNotFound {
		t.Fatalf("expected ErrViewerGrantNotFound, got %v", err)
	}
}

func TestListActiveAlertsReturnsViewerTokenAndLatestLocation(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-7")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	repo.AddTrustedContactWithPhone("user-7", "contact-3", "+919999911111")

	recordedAt := time.Date(2026, 3, 31, 12, 5, 0, 0, time.UTC)
	if err := service.RecordLocationPing(context.Background(), session.ID, "user-7", 13.0827, 80.2707, recordedAt); err != nil {
		t.Fatalf("failed to record location ping: %v", err)
	}

	alerts, err := service.ListActiveAlerts(context.Background(), "+919999911111")
	if err != nil {
		t.Fatalf("expected active alerts lookup to succeed, got %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 active alert, got %d", len(alerts))
	}

	alert := alerts[0]
	if alert.SessionID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, alert.SessionID)
	}
	if alert.ViewerToken == "" {
		t.Fatal("expected viewer token to be generated for active alert")
	}
	if alert.Latitude == nil || alert.Longitude == nil {
		t.Fatal("expected active alert to include latest location")
	}
	if *alert.Latitude != 13.0827 || *alert.Longitude != 80.2707 {
		t.Fatalf("expected latest location 13.0827, 80.2707; got %.4f, %.4f", *alert.Latitude, *alert.Longitude)
	}
}

func TestListActiveAlertsReusesExistingViewerToken(t *testing.T) {
	repo := newMemorySOSRepository()
	service := sos.NewService(repo)

	session, err := service.StartSession(context.Background(), "user-8")
	if err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	repo.AddTrustedContactWithPhone("user-8", "contact-4", "+919999922222")

	firstAlerts, err := service.ListActiveAlerts(context.Background(), "+919999922222")
	if err != nil {
		t.Fatalf("expected first active alerts lookup to succeed, got %v", err)
	}
	if len(firstAlerts) != 1 {
		t.Fatalf("expected 1 first alert, got %d", len(firstAlerts))
	}

	secondAlerts, err := service.ListActiveAlerts(context.Background(), "+919999922222")
	if err != nil {
		t.Fatalf("expected second active alerts lookup to succeed, got %v", err)
	}
	if len(secondAlerts) != 1 {
		t.Fatalf("expected 1 second alert, got %d", len(secondAlerts))
	}

	firstToken := firstAlerts[0].ViewerToken
	secondToken := secondAlerts[0].ViewerToken
	if firstToken == "" || secondToken == "" {
		t.Fatal("expected viewer tokens to be populated")
	}
	if firstToken != secondToken {
		t.Fatalf("expected viewer token to be reused, got %q and %q", firstToken, secondToken)
	}

	grant, authorizedSession, err := service.AuthorizeViewer(context.Background(), firstToken)
	if err != nil {
		t.Fatalf("expected original viewer token to remain authorized, got %v", err)
	}
	if grant.TrustedContactID != "contact-4" {
		t.Fatalf("expected trusted contact id contact-4, got %s", grant.TrustedContactID)
	}
	if authorizedSession.ID != session.ID {
		t.Fatalf("expected session id %q, got %q", session.ID, authorizedSession.ID)
	}
}
