package workers

import (
	"context"
	"log/slog"
	"time"
)

type IPFSSyncer interface {
	SyncPending(ctx context.Context) error
}

type IPFSUploadJob struct {
	syncer   IPFSSyncer
	interval time.Duration
}

func NewIPFSUploadJob(syncer IPFSSyncer, interval time.Duration) *IPFSUploadJob {
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	return &IPFSUploadJob{
		syncer:   syncer,
		interval: interval,
	}
}

func (j *IPFSUploadJob) Name() string {
	return "ipfs_upload"
}

func (j *IPFSUploadJob) Run(ctx context.Context) {
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

func (j *IPFSUploadJob) runOnce(ctx context.Context) {
	if j.syncer == nil {
		slog.Debug("ipfs worker placeholder tick")
		return
	}

	if err := j.syncer.SyncPending(ctx); err != nil {
		slog.Warn("ipfs sync failed", "error", err)
	}
}
