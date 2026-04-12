package commands_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/db"
)

// setupSQLiteQueue opens an in-memory SQLite pool, runs all migrations, and
// returns a Queue. The pool is closed via t.Cleanup.
func setupSQLiteQueue(t *testing.T) commands.Queue {
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
	return commands.NewSQLiteQueue(pool)
}

// TestQueueEnqueueAndGet verifies that a command can be enqueued and retrieved
// by ID with all fields intact.
func TestQueueEnqueueAndGet(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)

	body := json.RawMessage(`{"key":"value"}`)
	cmd, err := q.Enqueue(ctx, "TestCmd", body, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if cmd.ID == 0 {
		t.Fatal("expected non-zero ID after enqueue")
	}
	if cmd.Name != "TestCmd" {
		t.Errorf("Name = %q; want %q", cmd.Name, "TestCmd")
	}
	if cmd.Status != commands.StatusQueued {
		t.Errorf("Status = %q; want %q", cmd.Status, commands.StatusQueued)
	}
	if string(cmd.Body) != string(body) {
		t.Errorf("Body = %s; want %s", cmd.Body, body)
	}

	got, err := q.Get(ctx, cmd.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != cmd.ID {
		t.Errorf("Get ID = %d; want %d", got.ID, cmd.ID)
	}
	if got.Name != cmd.Name {
		t.Errorf("Get Name = %q; want %q", got.Name, cmd.Name)
	}
}

// TestQueueClaimReturnsHighestPriority enqueues three commands with different
// priorities and verifies they are claimed High → Normal → Low.
func TestQueueClaimReturnsHighestPriority(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)

	body := json.RawMessage(`{}`)
	if _, err := q.Enqueue(ctx, "Low", body, commands.PriorityLow, commands.TriggerManual, ""); err != nil {
		t.Fatalf("Enqueue Low: %v", err)
	}
	if _, err := q.Enqueue(ctx, "High", body, commands.PriorityHigh, commands.TriggerManual, ""); err != nil {
		t.Fatalf("Enqueue High: %v", err)
	}
	if _, err := q.Enqueue(ctx, "Normal", body, commands.PriorityNormal, commands.TriggerManual, ""); err != nil {
		t.Fatalf("Enqueue Normal: %v", err)
	}

	want := []string{"High", "Normal", "Low"}
	for i, wantName := range want {
		claimed, err := q.Claim(ctx, "w1", time.Minute)
		if err != nil {
			t.Fatalf("Claim %d: %v", i, err)
		}
		if claimed == nil {
			t.Fatalf("Claim %d: got nil, want %q", i, wantName)
		}
		if claimed.Name != wantName {
			t.Errorf("Claim %d: got %q; want %q", i, claimed.Name, wantName)
		}
	}
}

// TestQueueClaimReturnsNilWhenEmpty verifies that Claim returns nil when no
// commands are queued.
func TestQueueClaimReturnsNilWhenEmpty(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)

	claimed, err := q.Claim(ctx, "w1", time.Minute)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed != nil {
		t.Errorf("Claim on empty queue returned %+v; want nil", claimed)
	}
}

// TestQueueCompleteAndFail enqueues two commands, claims them, completes one
// and fails the other, then verifies their statuses.
func TestQueueCompleteAndFail(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)

	body := json.RawMessage(`{}`)

	// Enqueue two commands.
	c1, err := q.Enqueue(ctx, "CompleteMe", body, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue c1: %v", err)
	}
	c2, err := q.Enqueue(ctx, "FailMe", body, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue c2: %v", err)
	}

	// Claim both.
	cl1, err := q.Claim(ctx, "w1", time.Minute)
	if err != nil || cl1 == nil {
		t.Fatalf("Claim c1: err=%v claimed=%v", err, cl1)
	}
	cl2, err := q.Claim(ctx, "w1", time.Minute)
	if err != nil || cl2 == nil {
		t.Fatalf("Claim c2: err=%v claimed=%v", err, cl2)
	}

	// Complete c1, fail c2.
	result := json.RawMessage(`{"done":true}`)
	if err := q.Complete(ctx, c1.ID, 100, result, "ok"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if err := q.Fail(ctx, c2.ID, 50, "boom", "it exploded"); err != nil {
		t.Fatalf("Fail: %v", err)
	}

	got1, err := q.Get(ctx, c1.ID)
	if err != nil {
		t.Fatalf("Get c1: %v", err)
	}
	if got1.Status != commands.StatusCompleted {
		t.Errorf("c1 status = %q; want completed", got1.Status)
	}
	if got1.Message != "ok" {
		t.Errorf("c1 message = %q; want %q", got1.Message, "ok")
	}

	got2, err := q.Get(ctx, c2.ID)
	if err != nil {
		t.Fatalf("Get c2: %v", err)
	}
	if got2.Status != commands.StatusFailed {
		t.Errorf("c2 status = %q; want failed", got2.Status)
	}
	if got2.Exception != "boom" {
		t.Errorf("c2 exception = %q; want %q", got2.Exception, "boom")
	}
}

// TestQueueLeaseRefreshAndSweep enqueues a command, claims it with a very
// short lease, sleeps past expiry, sweeps, and verifies it can be reclaimed.
func TestQueueLeaseRefreshAndSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping lease sweep test in short mode")
	}

	ctx := context.Background()
	q := setupSQLiteQueue(t)

	body := json.RawMessage(`{}`)
	_, err := q.Enqueue(ctx, "LeasedCmd", body, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Claim with a very short lease (-1ns so it expires immediately in SQLite).
	// We use a negative duration so the lease_until is already in the past when
	// the UPDATE executes, making the sweep hit it right away.
	claimed, err := q.Claim(ctx, "w1", -time.Second)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed == nil {
		t.Fatal("Claim returned nil; expected a command")
	}
	if claimed.Status != commands.StatusRunning {
		t.Errorf("claimed status = %q; want running", claimed.Status)
	}

	// Sweep — the lease is already expired.
	n, err := q.SweepExpiredLeases(ctx)
	if err != nil {
		t.Fatalf("SweepExpiredLeases: %v", err)
	}
	if n != 1 {
		t.Errorf("swept %d; want 1", n)
	}

	// The command should be queued again and claimable.
	reclaimed, err := q.Claim(ctx, "w2", time.Minute)
	if err != nil {
		t.Fatalf("re-Claim: %v", err)
	}
	if reclaimed == nil {
		t.Fatal("re-Claim returned nil; expected a command after sweep")
	}
	if reclaimed.ID != claimed.ID {
		t.Errorf("reclaimed ID = %d; want %d", reclaimed.ID, claimed.ID)
	}
}

// TestQueueDeduplication verifies that FindDuplicate returns the ID of an
// active command with the same dedup_key.
func TestQueueDeduplication(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)

	body := json.RawMessage(`{}`)
	const key = "unique-sync-key"

	// Before enqueue: no duplicate.
	id, found, err := q.FindDuplicate(ctx, key)
	if err != nil {
		t.Fatalf("FindDuplicate (before): %v", err)
	}
	if found {
		t.Errorf("FindDuplicate before enqueue: found=%v id=%d; want not found", found, id)
	}

	// Enqueue with a dedup key.
	cmd, err := q.Enqueue(ctx, "DedupCmd", body, commands.PriorityNormal, commands.TriggerManual, key)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// After enqueue: duplicate found.
	id, found, err = q.FindDuplicate(ctx, key)
	if err != nil {
		t.Fatalf("FindDuplicate (after): %v", err)
	}
	if !found {
		t.Fatal("FindDuplicate after enqueue: not found; want found")
	}
	if id != cmd.ID {
		t.Errorf("FindDuplicate id = %d; want %d", id, cmd.ID)
	}
}

// TestQueueDeleteOldCompleted enqueues a command, completes it, then runs
// DeleteOldCompleted with a cutoff in the future to verify it is removed.
func TestQueueDeleteOldCompleted(t *testing.T) {
	ctx := context.Background()
	q := setupSQLiteQueue(t)

	body := json.RawMessage(`{}`)
	cmd, err := q.Enqueue(ctx, "OldCmd", body, commands.PriorityNormal, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Claim and complete the command.
	claimed, err := q.Claim(ctx, "w1", time.Minute)
	if err != nil || claimed == nil {
		t.Fatalf("Claim: err=%v claimed=%v", err, claimed)
	}
	if err := q.Complete(ctx, cmd.ID, 10, json.RawMessage(`{}`), "done"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Delete with a cutoff of now+1h to catch the just-completed command.
	n, err := q.DeleteOldCompleted(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("DeleteOldCompleted: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted %d; want 1", n)
	}

	// The command should no longer exist.
	_, err = q.Get(ctx, cmd.ID)
	if err == nil {
		t.Error("Get after delete: expected error, got nil")
	}
}
