package workers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"saferoute-backend/internal/evidence"
)

const defaultRelayQueueSize = 100

var txHashPattern = regexp.MustCompile(`Transaction hash is ([a-f0-9]+)`)

type EvidenceRelayer interface {
	SubmitEvidence(ctx context.Context, submission evidence.BlockchainEvidenceSubmission) (string, error)
}

type EvidenceRelayQueue struct {
	ch chan evidence.BlockchainEvidenceSubmission
}

type EvidenceRelayJob struct {
	queue   *EvidenceRelayQueue
	relayer EvidenceRelayer
}

type SorobanCLIRelayer struct {
	binary     string
	contractID string
	source     string
	network    string
	runner     CommandRunner
}

type CommandRunner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execCommandRunner struct{}

func NewEvidenceRelayQueue(size int) *EvidenceRelayQueue {
	if size <= 0 {
		size = defaultRelayQueueSize
	}

	return &EvidenceRelayQueue{
		ch: make(chan evidence.BlockchainEvidenceSubmission, size),
	}
}

func NewEvidenceRelayJob(queue *EvidenceRelayQueue, relayer EvidenceRelayer) *EvidenceRelayJob {
	return &EvidenceRelayJob{
		queue:   queue,
		relayer: relayer,
	}
}

func NewSorobanCLIRelayer(binary, contractID, source, network string) *SorobanCLIRelayer {
	return &SorobanCLIRelayer{
		binary:     strings.TrimSpace(binary),
		contractID: strings.TrimSpace(contractID),
		source:     strings.TrimSpace(source),
		network:    strings.TrimSpace(network),
		runner:     execCommandRunner{},
	}
}

func (q *EvidenceRelayQueue) QueueEvidence(ctx context.Context, submission evidence.BlockchainEvidenceSubmission) error {
	if q == nil {
		return errors.New("evidence relay queue is not configured")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- submission:
		return nil
	default:
		return errors.New("evidence relay queue is full")
	}
}

func (q *EvidenceRelayQueue) dequeue(ctx context.Context) (evidence.BlockchainEvidenceSubmission, bool) {
	if q == nil {
		return evidence.BlockchainEvidenceSubmission{}, false
	}

	select {
	case <-ctx.Done():
		return evidence.BlockchainEvidenceSubmission{}, false
	case submission := <-q.ch:
		return submission, true
	}
}

func (j *EvidenceRelayJob) Name() string {
	return "soroban_relayer"
}

func (j *EvidenceRelayJob) Run(ctx context.Context) {
	if j.queue == nil || j.relayer == nil {
		slog.Debug("soroban relayer placeholder tick")
		<-ctx.Done()
		return
	}

	for {
		submission, ok := j.queue.dequeue(ctx)
		if !ok {
			return
		}

		txHash, err := j.relayer.SubmitEvidence(ctx, submission)
		if err != nil {
			slog.Warn(
				"soroban relayer failed to submit evidence",
				"report_id", submission.ReportID,
				"error", err,
			)
			continue
		}

		slog.Info(
			"soroban relayer submitted evidence",
			"report_id", submission.ReportID,
			"tx_hash", txHash,
		)
	}
}

func (r *SorobanCLIRelayer) SubmitEvidence(ctx context.Context, submission evidence.BlockchainEvidenceSubmission) (string, error) {
	if r == nil {
		return "", errors.New("soroban relayer is not configured")
	}
	if r.binary == "" {
		return "", errors.New("soroban cli binary is not configured")
	}
	if r.contractID == "" {
		return "", errors.New("soroban contract id is not configured")
	}
	if r.source == "" {
		return "", errors.New("soroban source identity is not configured")
	}
	if r.network == "" {
		return "", errors.New("soroban network is not configured")
	}

	args := []string{
		"contract", "invoke",
		"--id", r.contractID,
		"--source", r.source,
		"--network", r.network,
		"--network-passphrase", "Test SDF Network ; September 2015",
		"--",
		"log_evidence",
		"--report-id", submission.ReportID,
		"--hash", submission.Hash,
	}

	output, err := r.runner.CombinedOutput(ctx, r.binary, args...)
	if err != nil {
		return "", fmt.Errorf("soroban invoke failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return extractTxHash(output), nil
}

func (execCommandRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func extractTxHash(output []byte) string {
	match := txHashPattern.FindSubmatch(output)
	if len(match) < 2 {
		return ""
	}

	return string(bytes.TrimSpace(match[1]))
}

func withRelayTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 30*time.Second)
}
