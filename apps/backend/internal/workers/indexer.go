package workers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"saferoute-backend/internal/reports"

	"github.com/stellar/go/xdr"
	"gorm.io/gorm"
)

const (
	defaultIndexerPollInterval    = 10 * time.Second
	defaultIndexerLookbackLedgers = 120
	sorobanRPCLatestLedgerMethod  = "getLatestLedger"
	sorobanRPCGetEventsMethod     = "getEvents"
)

type SorobanEventClient interface {
	LatestLedger(ctx context.Context) (uint32, error)
	GetEvents(ctx context.Context, query SorobanEventsQuery) (*SorobanEventsPage, error)
}

type EvidenceChainStore interface {
	MarkReportEvidenceVerified(ctx context.Context, reportID, txHash string, verifiedAt time.Time) (int64, error)
}

type IndexerJob struct {
	client          SorobanEventClient
	store           EvidenceChainStore
	contractID      string
	interval        time.Duration
	lookbackLedgers uint32
	cursor          string
}

type SorobanEventsQuery struct {
	ContractID  string
	Cursor      string
	StartLedger uint32
	Limit       int
}

type SorobanEventsPage struct {
	Events       []SorobanEvent
	LatestLedger uint32
	Cursor       string
}

type SorobanEvent struct {
	ID             string
	ContractID     string
	TxHash         string
	LedgerClosedAt string
	Topic          []string
}

type SorobanRPCClient struct {
	endpoint   string
	httpClient *http.Client
}

type GormEvidenceChainStore struct {
	db *gorm.DB
}

type latestLedgerRPCResponse struct {
	Result struct {
		Sequence uint32 `json:"sequence"`
	} `json:"result"`
	Error *rpcError `json:"error"`
}

type getEventsRPCResponse struct {
	Result struct {
		LatestLedger uint32 `json:"latestLedger"`
		Cursor       string `json:"cursor"`
		Events       []struct {
			ID             string   `json:"id"`
			ContractID     string   `json:"contractId"`
			TxHash         string   `json:"txHash"`
			LedgerClosedAt string   `json:"ledgerClosedAt"`
			Topic          []string `json:"topic"`
		} `json:"events"`
	} `json:"result"`
	Error *rpcError `json:"error"`
}

func NewIndexerJob(client SorobanEventClient, store EvidenceChainStore, contractID string, interval time.Duration, lookbackLedgers uint32) *IndexerJob {
	if interval <= 0 {
		interval = defaultIndexerPollInterval
	}
	if lookbackLedgers == 0 {
		lookbackLedgers = defaultIndexerLookbackLedgers
	}

	return &IndexerJob{
		client:          client,
		store:           store,
		contractID:      strings.TrimSpace(contractID),
		interval:        interval,
		lookbackLedgers: lookbackLedgers,
	}
}

func NewSorobanRPCClient(endpoint string, httpClient *http.Client) *SorobanRPCClient {
	endpoint = strings.TrimSpace(endpoint)
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &SorobanRPCClient{
		endpoint:   endpoint,
		httpClient: httpClient,
	}
}

func NewGormEvidenceChainStore(db *gorm.DB) *GormEvidenceChainStore {
	return &GormEvidenceChainStore{db: db}
}

func (j *IndexerJob) Name() string {
	return "soroban_indexer"
}

func (j *IndexerJob) Run(ctx context.Context) {
	j.runOnce(ctx)

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

func (j *IndexerJob) SyncPending(ctx context.Context) error {
	if j.client == nil || j.store == nil || j.contractID == "" {
		return nil
	}

	query, err := j.nextQuery(ctx)
	if err != nil {
		return err
	}

	for {
		page, err := j.client.GetEvents(ctx, query)
		if err != nil {
			return err
		}

		for _, event := range page.Events {
			reportID, ok, err := evidenceLoggedReportID(event)
			if err != nil {
				slog.Debug("skipping soroban event with unreadable topics", "event_id", event.ID, "error", err)
				continue
			}
			if !ok {
				continue
			}

			verifiedAt := parseLedgerClosedAt(event.LedgerClosedAt)
			rows, err := j.store.MarkReportEvidenceVerified(ctx, reportID, event.TxHash, verifiedAt)
			if err != nil {
				return err
			}

			slog.Info(
				"indexed soroban evidence event",
				"event_id", event.ID,
				"report_id", reportID,
				"tx_hash", event.TxHash,
				"rows_updated", rows,
			)
		}

		if page.Cursor == "" || page.Cursor == j.cursor {
			return nil
		}

		j.cursor = page.Cursor
		if len(page.Events) == 0 {
			return nil
		}

		query = SorobanEventsQuery{
			ContractID: j.contractID,
			Cursor:     j.cursor,
			Limit:      query.Limit,
		}
	}
}

func (j *IndexerJob) runOnce(ctx context.Context) {
	if j.client == nil || j.store == nil || j.contractID == "" {
		slog.Debug("soroban indexer placeholder tick")
		return
	}

	if err := j.SyncPending(ctx); err != nil {
		slog.Warn("soroban indexer sync failed", "error", err)
	}
}

func (j *IndexerJob) nextQuery(ctx context.Context) (SorobanEventsQuery, error) {
	if j.cursor != "" {
		return SorobanEventsQuery{
			ContractID: j.contractID,
			Cursor:     j.cursor,
			Limit:      100,
		}, nil
	}

	latest, err := j.client.LatestLedger(ctx)
	if err != nil {
		return SorobanEventsQuery{}, err
	}

	startLedger := latest
	if latest > j.lookbackLedgers {
		startLedger = latest - j.lookbackLedgers + 1
	}
	if startLedger == 0 {
		startLedger = 1
	}

	return SorobanEventsQuery{
		ContractID:  j.contractID,
		StartLedger: startLedger,
		Limit:       100,
	}, nil
}

func (s *SorobanRPCClient) LatestLedger(ctx context.Context) (uint32, error) {
	if s == nil || s.endpoint == "" {
		return 0, errors.New("soroban rpc endpoint is not configured")
	}

	var response latestLedgerRPCResponse

	if err := s.call(ctx, sorobanRPCLatestLedgerMethod, nil, &response); err != nil {
		return 0, err
	}

	return response.Result.Sequence, nil
}

func (s *SorobanRPCClient) GetEvents(ctx context.Context, query SorobanEventsQuery) (*SorobanEventsPage, error) {
	if s == nil || s.endpoint == "" {
		return nil, errors.New("soroban rpc endpoint is not configured")
	}

	params := map[string]any{
		"filters": []map[string]any{
			{
				"type":        "contract",
				"contractIds": []string{query.ContractID},
			},
		},
		"pagination": map[string]any{
			"limit": clampEventLimit(query.Limit),
		},
		"xdrFormat": "base64",
	}
	if query.Cursor != "" {
		params["pagination"].(map[string]any)["cursor"] = query.Cursor
	} else if query.StartLedger > 0 {
		params["startLedger"] = query.StartLedger
	}

	var response getEventsRPCResponse

	if err := s.call(ctx, sorobanRPCGetEventsMethod, params, &response); err != nil {
		return nil, err
	}

	events := make([]SorobanEvent, 0, len(response.Result.Events))
	for _, item := range response.Result.Events {
		events = append(events, SorobanEvent{
			ID:             item.ID,
			ContractID:     item.ContractID,
			TxHash:         item.TxHash,
			LedgerClosedAt: item.LedgerClosedAt,
			Topic:          append([]string(nil), item.Topic...),
		})
	}

	return &SorobanEventsPage{
		Events:       events,
		LatestLedger: response.Result.LatestLedger,
		Cursor:       response.Result.Cursor,
	}, nil
}

func (s *SorobanRPCClient) call(ctx context.Context, method string, params any, out any) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      method,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("soroban rpc returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}

	switch value := out.(type) {
	case *latestLedgerRPCResponse:
		if value.Error != nil {
			return value.Error
		}
	case *getEventsRPCResponse:
		if value.Error != nil {
			return value.Error
		}
	}

	return nil
}

func (s *GormEvidenceChainStore) MarkReportEvidenceVerified(ctx context.Context, reportID, txHash string, verifiedAt time.Time) (int64, error) {
	updates := map[string]any{
		"on_chain_tx":         txHash,
		"on_chain_verified":   true,
		"on_chain_verified_at": verifiedAt.UTC(),
	}

	result := s.db.WithContext(ctx).
		Model(&reports.Evidence{}).
		Where("report_id = ? AND on_chain_verified = ?", reportID, false).
		Updates(updates)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("soroban rpc error %d: %s", e.Code, e.Message)
}

func evidenceLoggedReportID(event SorobanEvent) (string, bool, error) {
	if len(event.Topic) < 3 {
		return "", false, nil
	}

	kind, err := decodeSCValStringLike(event.Topic[0])
	if err != nil {
		return "", false, err
	}
	action, err := decodeSCValStringLike(event.Topic[1])
	if err != nil {
		return "", false, err
	}
	if kind != "evidence" || action != "logged" {
		return "", false, nil
	}

	reportID, err := decodeSCValStringLike(event.Topic[2])
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(reportID) == "" {
		return "", false, errors.New("empty report id in soroban event")
	}

	return reportID, true, nil
}

func decodeSCValStringLike(base64XDR string) (string, error) {
	var value xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(base64XDR, &value); err != nil {
		return "", err
	}

	switch value.Type {
	case xdr.ScValTypeScvString:
		if value.Str == nil {
			return "", errors.New("scv string missing payload")
		}
		return string(*value.Str), nil
	case xdr.ScValTypeScvSymbol:
		if value.Sym == nil {
			return "", errors.New("scv symbol missing payload")
		}
		return string(*value.Sym), nil
	default:
		return "", fmt.Errorf("unsupported scval type %s", value.Type)
	}
}

func parseLedgerClosedAt(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC()
	}

	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UTC()
	}

	return time.Now().UTC()
}

func clampEventLimit(limit int) int {
	switch {
	case limit <= 0:
		return 100
	case limit > 10_000:
		return 10_000
	default:
		return limit
	}
}
