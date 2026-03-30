package workers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupJobRemovesOnlyOldTempFiles(t *testing.T) {
	root := t.TempDir()
	oldTemp := filepath.Join(root, "stale.tmp")
	newTemp := filepath.Join(root, "fresh.tmp")
	keepFile := filepath.Join(root, "evidence.bin")

	for _, path := range []string{oldTemp, newTemp, keepFile} {
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
	}

	oldModTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldTemp, oldModTime, oldModTime); err != nil {
		t.Fatalf("failed to age stale temp file: %v", err)
	}

	job := NewCleanupJob(root, time.Hour, 24*time.Hour)
	job.runOnce(nil)

	if _, err := os.Stat(oldTemp); !os.IsNotExist(err) {
		t.Fatalf("expected old temp file to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(newTemp); err != nil {
		t.Fatalf("expected fresh temp file to remain, stat err=%v", err)
	}
	if _, err := os.Stat(keepFile); err != nil {
		t.Fatalf("expected non-temp file to remain, stat err=%v", err)
	}
}
