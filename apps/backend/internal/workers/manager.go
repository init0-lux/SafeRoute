package workers

import (
	"context"
	"log/slog"
	"sync"
)

type Job interface {
	Name() string
	Run(ctx context.Context)
}

type Manager struct {
	jobs []Job
}

func NewManager(jobs ...Job) *Manager {
	filtered := make([]Job, 0, len(jobs))
	for _, job := range jobs {
		if job != nil {
			filtered = append(filtered, job)
		}
	}

	return &Manager{jobs: filtered}
}

func (m *Manager) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, job := range m.jobs {
		job := job
		wg.Add(1)

		go func() {
			defer wg.Done()
			slog.Info("worker started", "job", job.Name())
			job.Run(ctx)
			slog.Info("worker stopped", "job", job.Name())
		}()
	}

	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
}
