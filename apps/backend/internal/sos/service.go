package sos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultViewerGrantTTL = 2 * time.Hour

type ViewerEvent struct {
	SessionID   string    `json:"session_id"`
	Latitude    float64   `json:"lat"`
	Longitude   float64   `json:"lng"`
	RecordedAt  time.Time `json:"recorded_at"`
	PublishedAt time.Time `json:"published_at"`
}

type SSEBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan ViewerEvent]struct{}
}

type Service struct {
	repo        Repository
	broadcaster *SSEBroadcaster
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:        repo,
		broadcaster: NewSSEBroadcaster(),
	}
}

func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		subscribers: make(map[string]map[chan ViewerEvent]struct{}),
	}
}

func (b *SSEBroadcaster) Subscribe(sessionID string) (<-chan ViewerEvent, func()) {
	ch := make(chan ViewerEvent, 16)

	b.mu.Lock()
	if _, ok := b.subscribers[sessionID]; !ok {
		b.subscribers[sessionID] = make(map[chan ViewerEvent]struct{})
	}
	b.subscribers[sessionID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		if subscribers, ok := b.subscribers[sessionID]; ok {
			if _, exists := subscribers[ch]; exists {
				delete(subscribers, ch)
				close(ch)
			}
			if len(subscribers) == 0 {
				delete(b.subscribers, sessionID)
			}
		}
		b.mu.Unlock()
	}

	return ch, unsubscribe
}

func (b *SSEBroadcaster) Publish(event ViewerEvent) {
	b.mu.RLock()
	subscribers := b.subscribers[event.SessionID]
	channels := make([]chan ViewerEvent, 0, len(subscribers))
	for ch := range subscribers {
		channels = append(channels, ch)
	}
	b.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}

type CreateViewerGrantInput struct {
	TrustedContactID string
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

func (s *Service) CreateViewerGrant(ctx context.Context, sessionID, userID string, input CreateViewerGrantInput) (*SOSViewerGrant, string, error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, "", err
	}

	if session.Status == SessionStatusEnded {
		return nil, "", ErrSessionAlreadyEnded
	}

	trustedContactID := strings.TrimSpace(input.TrustedContactID)
	if trustedContactID == "" {
		return nil, "", ErrInvalidTrustedContactID
	}

	owned, err := s.repo.IsTrustedContactOwnedByUser(ctx, userID, trustedContactID)
	if err != nil {
		return nil, "", err
	}
	if !owned {
		return nil, "", ErrSessionForbidden
	}

	now := time.Now().UTC()
	if _, err := s.repo.GetActiveViewerGrantBySessionContact(ctx, sessionID, trustedContactID, now); err == nil {
		return nil, "", ErrViewerGrantConflict
	} else if !errors.Is(err, ErrViewerGrantNotFound) {
		return nil, "", err
	}

	token, tokenHash, err := GenerateViewerToken()
	if err != nil {
		return nil, "", err
	}

	grant := &SOSViewerGrant{
		SessionID:        sessionID,
		UserID:           userID,
		TrustedContactID: trustedContactID,
		TokenHash:        tokenHash,
		ExpiresAt:        now.Add(defaultViewerGrantTTL),
	}

	if err := s.repo.CreateViewerGrant(ctx, grant); err != nil {
		return nil, "", err
	}

	return grant, token, nil
}

func (s *Service) AuthorizeViewer(ctx context.Context, token string) (*SOSViewerGrant, *SOSSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil, ErrInvalidViewerToken
	}

	grant, err := s.repo.GetViewerGrantByToken(ctx, HashViewerToken(token))
	if err != nil {
		return nil, nil, err
	}

	if !CompareViewerToken(grant.TokenHash, token) {
		return nil, nil, ErrInvalidViewerToken
	}

	now := time.Now().UTC()
	if grant.RevokedAt != nil {
		return nil, nil, ErrViewerGrantRevoked
	}

	if !grant.ExpiresAt.After(now) {
		return nil, nil, ErrViewerGrantExpired
	}

	session, err := s.repo.GetSessionByID(ctx, grant.SessionID)
	if err != nil {
		return nil, nil, err
	}

	return grant, session, nil
}

func (s *Service) SubscribeViewer(sessionID string) (<-chan ViewerEvent, func(), error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, nil, ErrInvalidSessionID
	}

	ch, unsubscribe := s.broadcaster.Subscribe(sessionID)
	return ch, unsubscribe, nil
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

	recordedAt = recordedAt.UTC()
	if err := s.repo.CreateLocationPing(ctx, sessionID, latitude, longitude, recordedAt); err != nil {
		return err
	}

	s.broadcaster.Publish(ViewerEvent{
		SessionID:   sessionID,
		Latitude:    latitude,
		Longitude:   longitude,
		RecordedAt:  recordedAt,
		PublishedAt: time.Now().UTC(),
	})

	return nil
}

func FormatSSEEvent(event string, data string) string {
	if event == "" {
		return fmt.Sprintf("data: %s\n\n", data)
	}

	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, data)
}
