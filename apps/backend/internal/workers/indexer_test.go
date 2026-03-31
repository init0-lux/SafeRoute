package workers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stellar/go/xdr"
)

func TestIndexerSyncPendingMarksEvidenceVerifiedFromContractEvents(t *testing.T) {
	t.Parallel()

	client := &stubSorobanEventClient{
		latestLedger: 200,
		pages: []*SorobanEventsPage{
			{
				Cursor: "cursor-1",
				Events: []SorobanEvent{
					{
						ID:             "evt-1",
						ContractID:     "contract-1",
						TxHash:         "abc123",
						LedgerClosedAt: "2026-03-31T12:34:56Z",
						Topic: []string{
							mustMarshalScSymbol(t, "evidence"),
							mustMarshalScSymbol(t, "logged"),
							mustMarshalScString(t, "REP123"),
						},
					},
					{
						ID:             "evt-2",
						ContractID:     "contract-1",
						TxHash:         "ignore-me",
						LedgerClosedAt: "2026-03-31T12:35:00Z",
						Topic: []string{
							mustMarshalScSymbol(t, "trust"),
							mustMarshalScSymbol(t, "updated"),
							mustMarshalScString(t, "USER42"),
						},
					},
				},
			},
			{
				Cursor: "cursor-2",
				Events: []SorobanEvent{},
			},
		},
	}
	store := &stubEvidenceChainStore{}
	job := NewIndexerJob(client, store, "contract-1", time.Second, 25)

	if err := job.SyncPending(context.Background()); err != nil {
		t.Fatalf("SyncPending returned error: %v", err)
	}

	if len(store.calls) != 1 {
		t.Fatalf("expected 1 verification update, got %d", len(store.calls))
	}

	call := store.calls[0]
	if call.ReportID != "REP123" {
		t.Fatalf("expected report ID REP123, got %q", call.ReportID)
	}
	if call.TxHash != "abc123" {
		t.Fatalf("expected tx hash abc123, got %q", call.TxHash)
	}
	if call.VerifiedAt.Format(time.RFC3339) != "2026-03-31T12:34:56Z" {
		t.Fatalf("expected ledger close time to be persisted, got %s", call.VerifiedAt.Format(time.RFC3339))
	}

	if client.latestCalls != 1 {
		t.Fatalf("expected latest ledger to be requested once, got %d", client.latestCalls)
	}
	if len(client.queries) != 2 {
		t.Fatalf("expected 2 getEvents queries, got %d", len(client.queries))
	}
	if client.queries[0].StartLedger != 176 {
		t.Fatalf("expected initial start ledger 176, got %d", client.queries[0].StartLedger)
	}
	if client.queries[1].Cursor != "cursor-1" {
		t.Fatalf("expected cursor follow-up query, got %#v", client.queries[1])
	}
}

func TestIndexerSyncPendingUsesCursorOnSubsequentRuns(t *testing.T) {
	t.Parallel()

	client := &stubSorobanEventClient{
		latestLedger: 50,
		pages: []*SorobanEventsPage{
			{Cursor: "cursor-1"},
			{Cursor: "cursor-2"},
		},
	}
	job := NewIndexerJob(client, &stubEvidenceChainStore{}, "contract-1", time.Second, 10)

	if err := job.SyncPending(context.Background()); err != nil {
		t.Fatalf("first SyncPending returned error: %v", err)
	}
	if err := job.SyncPending(context.Background()); err != nil {
		t.Fatalf("second SyncPending returned error: %v", err)
	}

	if len(client.queries) != 2 {
		t.Fatalf("expected 2 getEvents queries, got %d", len(client.queries))
	}
	if client.queries[1].Cursor != "cursor-1" {
		t.Fatalf("expected second sync to resume from cursor-1, got %#v", client.queries[1])
	}
}

func TestEvidenceLoggedReportIDRejectsMalformedTopics(t *testing.T) {
	t.Parallel()

	_, ok, err := evidenceLoggedReportID(SorobanEvent{
		Topic: []string{mustMarshalScSymbol(t, "evidence")},
	})
	if err != nil {
		t.Fatalf("expected short topic list to be ignored without error, got %v", err)
	}
	if ok {
		t.Fatal("expected short topic list to be ignored")
	}

	_, _, err = evidenceLoggedReportID(SorobanEvent{
		Topic: []string{"not-base64", "still-not-base64", "bad"},
	})
	if err == nil {
		t.Fatal("expected malformed xdr topics to return an error")
	}
}

func mustMarshalScSymbol(t *testing.T, value string) string {
	t.Helper()

	symbol := xdr.ScSymbol(value)
	scVal, err := xdr.NewScVal(xdr.ScValTypeScvSymbol, symbol)
	if err != nil {
		t.Fatalf("failed to build symbol scval: %v", err)
	}

	encoded, err := xdr.MarshalBase64(scVal)
	if err != nil {
		t.Fatalf("failed to marshal symbol scval: %v", err)
	}

	return encoded
}

func mustMarshalScString(t *testing.T, value string) string {
	t.Helper()

	scString := xdr.ScString(value)
	scVal, err := xdr.NewScVal(xdr.ScValTypeScvString, scString)
	if err != nil {
		t.Fatalf("failed to build string scval: %v", err)
	}

	encoded, err := xdr.MarshalBase64(scVal)
	if err != nil {
		t.Fatalf("failed to marshal string scval: %v", err)
	}

	return encoded
}

type stubSorobanEventClient struct {
	latestLedger uint32
	latestErr    error
	pages        []*SorobanEventsPage
	pageErr      error
	latestCalls  int
	queries      []SorobanEventsQuery
}

func (s *stubSorobanEventClient) LatestLedger(context.Context) (uint32, error) {
	s.latestCalls++
	if s.latestErr != nil {
		return 0, s.latestErr
	}

	return s.latestLedger, nil
}

func (s *stubSorobanEventClient) GetEvents(_ context.Context, query SorobanEventsQuery) (*SorobanEventsPage, error) {
	s.queries = append(s.queries, query)
	if s.pageErr != nil {
		return nil, s.pageErr
	}
	if len(s.pages) == 0 {
		return &SorobanEventsPage{}, nil
	}

	page := s.pages[0]
	s.pages = s.pages[1:]
	return page, nil
}

type stubEvidenceChainStore struct {
	err   error
	calls []stubEvidenceChainCall
}

type stubEvidenceChainCall struct {
	ReportID   string
	TxHash     string
	VerifiedAt time.Time
}

func (s *stubEvidenceChainStore) MarkReportEvidenceVerified(_ context.Context, reportID, txHash string, verifiedAt time.Time) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}

	s.calls = append(s.calls, stubEvidenceChainCall{
		ReportID:   reportID,
		TxHash:     txHash,
		VerifiedAt: verifiedAt,
	})
	return 1, nil
}

func TestIndexerSyncPendingPropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	client := &stubSorobanEventClient{
		latestLedger: 10,
		pages: []*SorobanEventsPage{
			{
				Cursor: "cursor-1",
				Events: []SorobanEvent{
					{
						ID:             "evt-1",
						TxHash:         "abc123",
						LedgerClosedAt: "2026-03-31T12:34:56Z",
						Topic: []string{
							mustMarshalScSymbol(t, "evidence"),
							mustMarshalScSymbol(t, "logged"),
							mustMarshalScString(t, "REP123"),
						},
					},
				},
			},
		},
	}
	store := &stubEvidenceChainStore{err: errors.New("boom")}
	job := NewIndexerJob(client, store, "contract-1", time.Second, 5)

	if err := job.SyncPending(context.Background()); err == nil {
		t.Fatal("expected store error to be returned")
	}
}
