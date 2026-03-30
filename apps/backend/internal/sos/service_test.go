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
