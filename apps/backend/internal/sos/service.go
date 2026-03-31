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

type LocationSnapshot struct {
	Latitude   float64   `json:"lat"`
	Longitude  float64   `json:"lng"`
	RecordedAt time.Time `json:"recorded_at"`
}

type StartSessionInput struct {
	Latitude   *float64
	Longitude  *float64
	RecordedAt time.Time
}

type ActiveTrustedContactAlert struct {
	SessionID        string     `json:"session_id"`
	TrustedContactID string     `json:"trusted_contact_id"`
	ViewerToken      string     `json:"viewer_token"`
	ReporterName     string     `json:"reporter_name"`
	ReporterPhone    string     `json:"reporter_phone"`
	StartedAt        time.Time  `json:"started_at"`
	Latitude         *float64   `json:"lat,omitempty"`
	Longitude        *float64   `json:"lng,omitempty"`
	RecordedAt       *time.Time `json:"recorded_at,omitempty"`
}

type SSEBroadcaster struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan ViewerEvent]struct{}
}

type Service struct {
	repo                      Repository
	broadcaster               *SSEBroadcaster
	notifyTrustedContactsFunc func(ctx context.Context, session *SOSSession) error
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

func (s *Service) SetNotifyTrustedContactsFunc(fn func(ctx context.Context, session *SOSSession) error) {
	s.notifyTrustedContactsFunc = fn
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
	return s.StartSessionWithInput(ctx, userID, StartSessionInput{})
}

func (s *Service) StartSessionWithInput(ctx context.Context, userID string, input StartSessionInput) (_ *SOSSession, err error) {
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

	defer func() {
		if err == nil {
			return
		}

		if cleanupErr := s.markSessionEnded(ctx, session); cleanupErr != nil {
			err = fmt.Errorf("%w: failed to clean up session %s: %v", err, session.ID, cleanupErr)
		}
	}()

	if input.Latitude != nil || input.Longitude != nil {
		if input.Latitude == nil {
			return nil, ErrInvalidLatitude
		}
		if input.Longitude == nil {
			return nil, ErrInvalidLongitude
		}

		recordedAt := input.RecordedAt
		if recordedAt.IsZero() {
			recordedAt = session.StartedAt
		}
		if err := s.RecordLocationPing(ctx, session.ID, userID, *input.Latitude, *input.Longitude, recordedAt); err != nil {
			return nil, err
		}
	}

	if s.notifyTrustedContactsFunc != nil {
		if err := s.notifyTrustedContactsFunc(ctx, session); err != nil {
			return nil, err
		}
	}

	return session, nil
}

func (s *Service) GetActiveSession(ctx context.Context, userID string) (*SOSSession, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	return s.repo.GetActiveSessionByUserID(ctx, userID)
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

	return s.endLoadedSession(ctx, session)
}

func (s *Service) EndActiveSession(ctx context.Context, userID string) (*SOSSession, error) {
	session, err := s.GetActiveSession(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.endLoadedSession(ctx, session)
}

func (s *Service) endLoadedSession(ctx context.Context, session *SOSSession) (*SOSSession, error) {
	if session.Status == SessionStatusEnded {
		return session, nil
	}

	endedAt := time.Now().UTC()
	session.Status = SessionStatusEnded
	session.EndedAt = &endedAt

	if err := s.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) markSessionEnded(ctx context.Context, session *SOSSession) error {
	if session == nil || strings.TrimSpace(session.ID) == "" {
		return nil
	}
	if session.Status == SessionStatusEnded && session.EndedAt != nil {
		return nil
	}

	endedAt := time.Now().UTC()
	session.Status = SessionStatusEnded
	session.EndedAt = &endedAt

	return s.repo.UpdateSession(ctx, session)
}

func (s *Service) CreateViewerGrant(ctx context.Context, sessionID, userID string, input CreateViewerGrantInput) (*SOSViewerGrant, string, error) {
	return s.issueViewerGrant(ctx, sessionID, userID, input.TrustedContactID, false)
}

func (s *Service) ReplaceViewerGrant(ctx context.Context, sessionID, userID, trustedContactID string) (*SOSViewerGrant, string, error) {
	return s.issueViewerGrant(ctx, sessionID, userID, trustedContactID, true)
}

func (s *Service) issueViewerGrant(ctx context.Context, sessionID, userID, trustedContactID string, replaceExisting bool) (*SOSViewerGrant, string, error) {
	session, err := s.GetSession(ctx, sessionID, userID)
	if err != nil {
		return nil, "", err
	}

	if session.Status == SessionStatusEnded {
		return nil, "", ErrSessionAlreadyEnded
	}

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
	if replaceExisting {
		if err := s.repo.RevokeActiveViewerGrantBySessionContact(ctx, sessionID, trustedContactID, now); err != nil {
			return nil, "", err
		}
	} else {
		if _, err := s.repo.GetActiveViewerGrantBySessionContact(ctx, sessionID, trustedContactID, now); err == nil {
			return nil, "", ErrViewerGrantConflict
		} else if !errors.Is(err, ErrViewerGrantNotFound) {
			return nil, "", err
		}
	}

	token, tokenHash, err := GenerateViewerToken()
	if err != nil {
		return nil, "", err
	}

	grant := &SOSViewerGrant{
		SessionID:        sessionID,
		UserID:           userID,
		TrustedContactID: trustedContactID,
		Token:            token,
		TokenHash:        tokenHash,
		ExpiresAt:        now.Add(defaultViewerGrantTTL),
	}

	if err := s.repo.CreateViewerGrant(ctx, grant); err != nil {
		return nil, "", err
	}

	return grant, token, nil
}

func (s *Service) GetLatestLocation(ctx context.Context, sessionID string) (*LocationSnapshot, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrInvalidSessionID
	}

	return s.repo.GetLatestLocationPing(ctx, sessionID)
}

func (s *Service) ListActiveAlerts(ctx context.Context, viewerPhone string) ([]ActiveTrustedContactAlert, error) {
	viewerPhone = strings.TrimSpace(viewerPhone)
	if viewerPhone == "" {
		return nil, ErrInvalidUserID
	}

	alerts, err := s.repo.ListActiveSessionAlertsByViewerPhone(ctx, viewerPhone)
	if err != nil {
		return nil, err
	}

	results := make([]ActiveTrustedContactAlert, 0, len(alerts))
	now := time.Now().UTC()
	for _, alert := range alerts {
		var token string

		grant, grantErr := s.repo.GetActiveViewerGrantBySessionContact(ctx, alert.SessionID, alert.TrustedContactID, now)
		switch {
		case grantErr == nil && strings.TrimSpace(grant.Token) != "":
			token = grant.Token
		case grantErr == nil:
			_, token, grantErr = s.ReplaceViewerGrant(ctx, alert.SessionID, alert.UserID, alert.TrustedContactID)
		case errors.Is(grantErr, ErrViewerGrantNotFound):
			_, token, grantErr = s.CreateViewerGrant(ctx, alert.SessionID, alert.UserID, CreateViewerGrantInput{
				TrustedContactID: alert.TrustedContactID,
			})
		}
		if grantErr != nil {
			continue
		}

		results = append(results, ActiveTrustedContactAlert{
			SessionID:        alert.SessionID,
			TrustedContactID: alert.TrustedContactID,
			ViewerToken:      token,
			ReporterName:     alert.ReporterName,
			ReporterPhone:    alert.ReporterPhone,
			StartedAt:        alert.StartedAt,
			Latitude:         alert.Latitude,
			Longitude:        alert.Longitude,
			RecordedAt:       alert.RecordedAt,
		})
	}

	return results, nil
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
