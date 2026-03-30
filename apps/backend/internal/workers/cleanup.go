package workers

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CleanupJob struct {
	root     string
	interval time.Duration
	maxAge   time.Duration
}

func NewCleanupJob(root string, interval, maxAge time.Duration) *CleanupJob {
	if interval <= 0 {
		interval = time.Hour
	}
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}

	return &CleanupJob{
		root:     root,
		interval: interval,
		maxAge:   maxAge,
	}
}

func (j *CleanupJob) Name() string {
	return "cleanup"
}

func (j *CleanupJob) Run(ctx context.Context) {
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

func (j *CleanupJob) runOnce(_ context.Context) {
	root := strings.TrimSpace(j.root)
	if root == "" {
		return
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}

		slog.Warn("cleanup worker stat failed", "root", root, "error", err)
		return
	}
	if !info.IsDir() {
		return
	}

	cutoff := time.Now().UTC().Add(-j.maxAge)
	removed := 0

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".tmp") {
			return nil
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if fileInfo.ModTime().After(cutoff) {
			return nil
		}

		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		removed++
		return nil
	})
	if walkErr != nil {
		slog.Warn("cleanup worker sweep failed", "root", root, "error", walkErr)
		return
	}

	if removed > 0 {
		slog.Info("cleanup worker removed stale temp files", "count", removed, "root", root)
	}
}
