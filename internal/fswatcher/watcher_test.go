package fswatcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDebouncerCoalesces verifies that 5 rapid triggers for the same path
// result in exactly one fire call after the debounce delay.
func TestDebouncerCoalesces(t *testing.T) {
	const debounce = 100 * time.Millisecond

	var callCount int32
	deb := newDebouncer(debounce, func(path string) {
		atomic.AddInt32(&callCount, 1)
	})

	const path = "/series/show1"
	for i := 0; i < 5; i++ {
		deb.Trigger(path)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait long enough for the debounce timer to fire.
	time.Sleep(debounce + 50*time.Millisecond)

	got := atomic.LoadInt32(&callCount)
	if got != 1 {
		t.Errorf("fire count = %d, want 1", got)
	}
}

// TestDebouncerDifferentPaths verifies that two different paths each fire
// independently.
func TestDebouncerDifferentPaths(t *testing.T) {
	const debounce = 80 * time.Millisecond

	fired := make(map[string]int)
	var mu sync.Mutex

	deb := newDebouncer(debounce, func(path string) {
		mu.Lock()
		fired[path]++
		mu.Unlock()
	})

	pathA := "/series/show-a/file.mkv"
	pathB := "/series/show-b/file.mkv"

	deb.Trigger(pathA)
	deb.Trigger(pathB)

	time.Sleep(debounce + 50*time.Millisecond)

	mu.Lock()
	countA := fired[pathA]
	countB := fired[pathB]
	mu.Unlock()

	if countA != 1 {
		t.Errorf("pathA fire count = %d, want 1", countA)
	}
	if countB != 1 {
		t.Errorf("pathB fire count = %d, want 1", countB)
	}
}

// stubResolver is a SeriesResolver that returns a fixed seriesID for any path.
type stubResolver struct {
	seriesID int64
}

func (s *stubResolver) ResolveSeriesID(_ string) (int64, bool) {
	return s.seriesID, true
}

// recordingEnqueuer records Enqueue calls for inspection.
type recordingEnqueuer struct {
	mu       sync.Mutex
	commands []enqueuedCmd
}

type enqueuedCmd struct {
	name string
	body []byte
}

func (r *recordingEnqueuer) Enqueue(_ context.Context, name string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = append(r.commands, enqueuedCmd{name: name, body: body})
	return nil
}

func (r *recordingEnqueuer) Commands() []enqueuedCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]enqueuedCmd, len(r.commands))
	copy(out, r.commands)
	return out
}

// TestWatcherDetectsNewFile creates a temp directory, adds it as a root,
// writes a file, and verifies that a ScanSeriesFolder command was enqueued
// with the correct seriesId.
func TestWatcherDetectsNewFile(t *testing.T) {
	dir := t.TempDir()

	resolver := &stubResolver{seriesID: 42}
	enqueuer := &recordingEnqueuer{}

	w := New(resolver, enqueuer, noopLogger(t))
	w.debounce = 100 * time.Millisecond // speed up the test

	if err := w.AddRoot(dir); err != nil {
		t.Fatalf("AddRoot: %v", err)
	}
	t.Cleanup(w.Stop)

	// Write a file into the watched directory to trigger an event.
	path := filepath.Join(dir, "Show.S01E01.1080p.WEB-DL.mkv")
	if err := os.WriteFile(path, []byte("fake media"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Wait for debounce + processing.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cmds := enqueuer.Commands()
		if len(cmds) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cmds := enqueuer.Commands()
	if len(cmds) == 0 {
		t.Fatal("expected at least one enqueued command; got none")
	}
	if cmds[0].name != "ScanSeriesFolder" {
		t.Errorf("command name = %q, want ScanSeriesFolder", cmds[0].name)
	}

	// Verify the body contains the correct seriesId.
	body := string(cmds[0].body)
	const wantSubstr = `"seriesId":42`
	if !contains(body, wantSubstr) {
		t.Errorf("body = %q, want it to contain %q", body, wantSubstr)
	}
}

// contains reports whether substr is in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// noopLogger returns a slog.Logger that discards all output.
func noopLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}
