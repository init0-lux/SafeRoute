package sos_test

import (
	"context"
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
