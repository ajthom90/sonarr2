package scheduler_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/scheduler"
)

// setupSchedulerTest opens an in-memory SQLite pool, migrates, and returns
// both a TaskStore and a commands.Queue. The pool is closed via t.Cleanup.
func setupSchedulerTest(t *testing.T) (scheduler.TaskStore, commands.Queue) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	taskStore := scheduler.NewSQLiteTaskStore(pool)
	queue := commands.NewSQLiteQueue(pool)
	return taskStore, queue
}

// newTestLogger returns a slog.Logger that writes to t.Log.
func newTestLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(&testWriter{t: t}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

type testWriter struct{ t *testing.T }

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

// TestSchedulerEnqueuesDueTask verifies that a task whose next_execution is in
// the past gets enqueued by the scheduler.
func TestSchedulerEnqueuesDueTask(t *testing.T) {
	ctx := context.Background()
	taskStore, queue := setupSchedulerTest(t)

	// Insert a task that is already due (next_execution in the past).
	pastTask := scheduler.ScheduledTask{
		TypeName:      "DueTask",
		IntervalSecs:  60,
		NextExecution: time.Now().Add(-time.Minute),
	}
	if err := taskStore.Upsert(ctx, pastTask); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	log := newTestLogger(t)
	sched := scheduler.New(taskStore, queue, log)
	// Use a short tick interval for test speed.
	scheduler.SetInterval(sched, 50*time.Millisecond)

	sched.Start(ctx)
	defer sched.Stop()

	// Wait long enough for at least one tick.
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	// Verify the command was enqueued by claiming it.
	cmd, err := queue.Claim(ctx, "test-worker", time.Minute)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected a command to be enqueued; got nil")
	}
	if cmd.Name != "DueTask" {
		t.Errorf("command name = %q; want %q", cmd.Name, "DueTask")
	}
}

// TestSchedulerSkipsNotDueTask verifies that a task with next_execution in the
// future does not get enqueued.
func TestSchedulerSkipsNotDueTask(t *testing.T) {
	ctx := context.Background()
	taskStore, queue := setupSchedulerTest(t)

	// Insert a task that is not due yet.
	futureTask := scheduler.ScheduledTask{
		TypeName:      "FutureTask",
		IntervalSecs:  3600,
		NextExecution: time.Now().Add(time.Hour),
	}
	if err := taskStore.Upsert(ctx, futureTask); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	log := newTestLogger(t)
	sched := scheduler.New(taskStore, queue, log)
	scheduler.SetInterval(sched, 50*time.Millisecond)

	sched.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	// No command should have been enqueued.
	cmd, err := queue.Claim(ctx, "test-worker", time.Minute)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if cmd != nil {
		t.Errorf("expected no command; got %+v", cmd)
	}
}

// TestSchedulerSweepsExpiredLeases verifies that the scheduler calls
// SweepExpiredLeases on each tick, recovering commands with expired leases
// back to the queued state so they can be reclaimed.
func TestSchedulerSweepsExpiredLeases(t *testing.T) {
	ctx := context.Background()
	taskStore, queue := setupSchedulerTest(t)

	// Enqueue a command and immediately claim it with an already-expired lease.
	_, err := queue.Enqueue(ctx, "LeaseCmd", nil, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	claimed, err := queue.Claim(ctx, "worker1", -time.Second)
	if err != nil || claimed == nil {
		t.Fatalf("Claim (expired): err=%v claimed=%v", err, claimed)
	}
	if claimed.Status != commands.StatusRunning {
		t.Fatalf("expected running status; got %q", claimed.Status)
	}

	// Start the scheduler — it should sweep the expired lease on the first tick.
	log := newTestLogger(t)
	sched := scheduler.New(taskStore, queue, log)
	scheduler.SetInterval(sched, 50*time.Millisecond)

	sched.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	sched.Stop()

	// The command should be back in queued state and claimable.
	reclaimed, err := queue.Claim(ctx, "worker2", time.Minute)
	if err != nil {
		t.Fatalf("re-Claim: %v", err)
	}
	if reclaimed == nil {
		t.Fatal("expected command to be reclaimed after sweep; got nil")
	}
	if reclaimed.ID != claimed.ID {
		t.Errorf("reclaimed ID = %d; want %d", reclaimed.ID, claimed.ID)
	}
}
