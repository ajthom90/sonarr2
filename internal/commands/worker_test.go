package commands_test

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/events"
)

// waitForStatus polls the queue until the command reaches the target status or
// the deadline is exceeded.
func waitForStatus(t *testing.T, q commands.Queue, id int64, want commands.Status, timeout time.Duration) commands.Command {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cmd, err := q.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("waitForStatus: Get(%d): %v", id, err)
		}
		if cmd.Status == want {
			return cmd
		}
		time.Sleep(20 * time.Millisecond)
	}
	cmd, _ := q.Get(context.Background(), id)
	t.Fatalf("waitForStatus: command %d never reached %q (last status: %q)", id, want, cmd.Status)
	return commands.Command{}
}

// TestWorkerPoolExecutesEnqueuedCommand registers a HandlerFunc that writes to
// a channel, enqueues a command, starts a pool with 1 worker, and asserts the
// command completes.
func TestWorkerPoolExecutesEnqueuedCommand(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)
	reg := commands.NewRegistry()
	bus := events.NewNoopBus()
	log := newTestLogger(t)

	executed := make(chan struct{}, 1)
	reg.Register("Echo", commands.HandlerFunc(func(ctx context.Context, cmd commands.Command) error {
		executed <- struct{}{}
		return nil
	}))

	cmd, err := q.Enqueue(ctx, "Echo", json.RawMessage(`{}`), commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	pool := commands.NewWorkerPool(q, reg, bus, log, 1)
	pool.Start(ctx)
	t.Cleanup(pool.Stop)

	select {
	case <-executed:
		// Handler ran; verify the DB row reflects completion.
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for handler to execute")
	}

	got := waitForStatus(t, q, cmd.ID, commands.StatusCompleted, 5*time.Second)
	if got.Status != commands.StatusCompleted {
		t.Errorf("command status = %q; want completed", got.Status)
	}
}

// TestWorkerPoolHandlesPanic registers a handler that panics, enqueues a
// command, and verifies the command transitions to failed with the panic message
// captured in Exception.
func TestWorkerPoolHandlesPanic(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)
	reg := commands.NewRegistry()
	bus := events.NewNoopBus()
	log := newTestLogger(t)

	const panicMsg = "intentional test panic"
	reg.Register("Boom", commands.HandlerFunc(func(ctx context.Context, cmd commands.Command) error {
		panic(panicMsg)
	}))

	cmd, err := q.Enqueue(ctx, "Boom", json.RawMessage(`{}`), commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	pool := commands.NewWorkerPool(q, reg, bus, log, 1)
	pool.Start(ctx)
	t.Cleanup(pool.Stop)

	got := waitForStatus(t, q, cmd.ID, commands.StatusFailed, 5*time.Second)
	if got.Exception == "" {
		t.Error("expected non-empty Exception after panic, got empty string")
	}
	// The panic message should be embedded in the exception.
	if got.Exception != "panic: "+panicMsg {
		t.Errorf("Exception = %q; want %q", got.Exception, "panic: "+panicMsg)
	}
}

// TestWorkerPoolStopsGracefully starts a pool with 2 workers (no commands
// enqueued), calls Stop(), and verifies it returns within 2 seconds with no
// goroutine leak.
func TestWorkerPoolStopsGracefully(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)
	reg := commands.NewRegistry()
	bus := events.NewNoopBus()
	log := newTestLogger(t)

	goroutinesBefore := runtime.NumGoroutine()

	pool := commands.NewWorkerPool(q, reg, bus, log, 2)
	pool.Start(ctx)

	// Let the workers settle into their claim-retry loops.
	time.Sleep(50 * time.Millisecond)

	stopDone := make(chan struct{})
	go func() {
		pool.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
		// Good — Stop returned promptly.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2 seconds")
	}

	// Allow goroutines to fully terminate.
	time.Sleep(50 * time.Millisecond)

	goroutinesAfter := runtime.NumGoroutine()
	// Allow some slack for Go runtime goroutines that may appear/disappear.
	const slack = 5
	if goroutinesAfter > goroutinesBefore+slack {
		t.Errorf("possible goroutine leak: before=%d after=%d (delta=%d, slack=%d)",
			goroutinesBefore, goroutinesAfter, goroutinesAfter-goroutinesBefore, slack)
	}
}
