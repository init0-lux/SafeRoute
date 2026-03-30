package workers

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestManagerStartsJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job1 := &stubJob{name: "job1", started: make(chan struct{}, 1)}
	job2 := &stubJob{name: "job2", started: make(chan struct{}, 1)}
	manager := NewManager(job1, job2)

	manager.Start(ctx)

	waitForWorkerStart(t, job1.started)
	waitForWorkerStart(t, job2.started)
}

type stubJob struct {
	name    string
	started chan struct{}
	once    sync.Once
}

func (j *stubJob) Name() string {
	return j.name
}

func (j *stubJob) Run(ctx context.Context) {
	j.once.Do(func() {
		j.started <- struct{}{}
	})
	<-ctx.Done()
}

func waitForWorkerStart(t *testing.T, started <-chan struct{}) {
	t.Helper()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start in time")
	}
}
