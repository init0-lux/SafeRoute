package sos

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) StartSession(ctx context.Context, userID string) (*SOSSession, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	if _, err := s.repo.GetActiveSessionByUserID(ctx, userID); err == nil {
		return nil, ErrActiveSessionExists
	} else if !errors.Is(err, ErrSessionNotFound) {
		return nil, err
	}

	session := &SOSSession{
		UserID: &userID,
		Status: SessionStatusActive,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) GetSession(ctx context.Context, sessionID, userID string) (*SOSSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	userID = strings.TrimSpace(userID)
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	if userID == "" {
		return nil, ErrInvalidUserID
	}

	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.UserID == nil || *session.UserID != userID {
		return nil, ErrSessionForbidden
	}

	return session, nil
}

func (s *Service) EndSession(ctx context.Context, sessionID, userID string) (*SOSSession, error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	if session.Status == SessionStatusEnded {
		return nil, ErrSessionAlreadyEnded
	}

	endedAt := time.Now().UTC()
	session.Status = SessionStatusEnded
	session.EndedAt = &endedAt

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) RecordLocationPing(ctx context.Context, sessionID, userID string, latitude, longitude float64, recordedAt time.Time) error {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	if session.Status == SessionStatusEnded {
		return ErrSessionAlreadyEnded
	}

	if latitude < -90 || latitude > 90 {
		return ErrInvalidLatitude
	}

	if longitude < -180 || longitude > 180 {
		return ErrInvalidLongitude
	}

	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	return s.repo.CreateLocationPing(ctx, sessionID, latitude, longitude, recordedAt.UTC())
}
