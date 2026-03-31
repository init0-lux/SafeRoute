package workers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"saferoute-backend/internal/evidence"
)

func TestSorobanCLIRelayerSubmitEvidenceBuildsExpectedCommand(t *testing.T) {
	t.Parallel()

	runner := &stubCommandRunner{
		output: []byte("ℹ️  Transaction hash is abc123\n"),
	}
	relayer := NewSorobanCLIRelayer("stellar", "CABC", "backend-relayer", "testnet")
	relayer.runner = runner

	txHash, err := relayer.SubmitEvidence(context.Background(), evidence.BlockchainEvidenceSubmission{
		ReportID: "report-123",
		Hash:     strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatalf("SubmitEvidence returned error: %v", err)
	}
	if txHash != "abc123" {
		t.Fatalf("expected tx hash abc123, got %q", txHash)
	}

	expectedArgs := []string{
		"contract", "invoke",
		"--id", "CABC",
		"--source", "backend-relayer",
		"--network", "testnet",
		"--",
		"log_evidence",
		"--report-id", "report-123",
		"--hash", strings.Repeat("a", 64),
	}
	if runner.name != "stellar" {
		t.Fatalf("expected binary stellar, got %q", runner.name)
	}
	if strings.Join(runner.args, " ") != strings.Join(expectedArgs, " ") {
		t.Fatalf("unexpected args: %#v", runner.args)
	}
}

func TestEvidenceRelayQueueAndJobProcessSubmission(t *testing.T) {
	t.Parallel()

	queue := NewEvidenceRelayQueue(1)
	relayer := &stubEvidenceRelayer{done: make(chan evidence.BlockchainEvidenceSubmission, 1)}
	job := NewEvidenceRelayJob(queue, relayer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go job.Run(ctx)

	if err := queue.QueueEvidence(ctx, evidence.BlockchainEvidenceSubmission{
		ReportID: "report-123",
		Hash:     strings.Repeat("b", 64),
	}); err != nil {
		t.Fatalf("QueueEvidence returned error: %v", err)
	}

	select {
	case submission := <-relayer.done:
		if submission.ReportID != "report-123" {
			t.Fatalf("expected report-123, got %q", submission.ReportID)
		}
	case <-time.After(time.Second):
		t.Fatal("relayer job did not process queued submission")
	}
}

func TestEvidenceRelayQueueReturnsErrorWhenFull(t *testing.T) {
	t.Parallel()

	queue := NewEvidenceRelayQueue(1)
	ctx := context.Background()

	if err := queue.QueueEvidence(ctx, evidence.BlockchainEvidenceSubmission{ReportID: "r1"}); err != nil {
		t.Fatalf("unexpected error queueing first submission: %v", err)
	}
	if err := queue.QueueEvidence(ctx, evidence.BlockchainEvidenceSubmission{ReportID: "r2"}); err == nil {
		t.Fatal("expected queue full error")
	}
}

type stubCommandRunner struct {
	name   string
	args   []string
	output []byte
	err    error
}

func (s *stubCommandRunner) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	s.name = name
	s.args = append([]string(nil), args...)
	return s.output, s.err
}

type stubEvidenceRelayer struct {
	err  error
	done chan evidence.BlockchainEvidenceSubmission
}

func (s *stubEvidenceRelayer) SubmitEvidence(_ context.Context, submission evidence.BlockchainEvidenceSubmission) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.done != nil {
		s.done <- submission
	}
	return "tx-123", nil
}

func TestSorobanCLIRelayerReturnsRunnerErrors(t *testing.T) {
	t.Parallel()

	runner := &stubCommandRunner{err: errors.New("boom"), output: []byte("bad output")}
	relayer := NewSorobanCLIRelayer("stellar", "CABC", "backend-relayer", "testnet")
	relayer.runner = runner

	if _, err := relayer.SubmitEvidence(context.Background(), evidence.BlockchainEvidenceSubmission{
		ReportID: "report-123",
		Hash:     strings.Repeat("a", 64),
	}); err == nil {
		t.Fatal("expected submit error")
	}
}
