package workers

import (
	"context"
	"log/slog"
	"time"
)

type SafetyRefresher interface {
	Refresh(ctx context.Context) error
}

type SafetyCacheJob struct {
	refresher SafetyRefresher
	interval  time.Duration
}

func NewSafetyCacheJob(refresher SafetyRefresher, interval time.Duration) *SafetyCacheJob {
	if interval <= 0 {
		interval = 10 * time.Minute
	}

	return &SafetyCacheJob{
		refresher: refresher,
		interval:  interval,
	}
}

func (j *SafetyCacheJob) Name() string {
	return "safety_cache"
}

func (j *SafetyCacheJob) Run(ctx context.Context) {
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

func (j *SafetyCacheJob) runOnce(ctx context.Context) {
	if j.refresher == nil {
		slog.Debug("safety cache worker placeholder tick")
		return
	}

	if err := j.refresher.Refresh(ctx); err != nil {
		slog.Warn("safety cache refresh failed", "error", err)
	}
}
