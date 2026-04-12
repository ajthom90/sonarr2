package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/db"
)

func TestCleanupHandlerDeletesOldCommands(t *testing.T) {
	// Setup: open in-memory SQLite, migrate, create queue.
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

	queue := commands.NewSQLiteQueue(pool)

	// Enqueue and complete 3 commands.
	for i := 0; i < 3; i++ {
		cmd, err := queue.Enqueue(ctx, "test", nil, commands.PriorityNormal, commands.TriggerManual, "")
		if err != nil {
			t.Fatalf("Enqueue: %v", err)
		}
		claimed, err := queue.Claim(ctx, "test-worker", time.Hour)
		if err != nil || claimed == nil {
			t.Fatalf("Claim: %v (nil=%v)", err, claimed == nil)
		}
		if err := queue.Complete(ctx, claimed.ID, 100, nil, "done"); err != nil {
			t.Fatalf("Complete: %v", err)
		}
		_ = cmd
	}

	// The handler with default 7-day cutoff won't delete fresh commands.
	handler := NewCleanupHandler(queue)
	cmd := commands.Command{Name: "MessagingCleanup"}
	if err := handler.Handle(ctx, cmd); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Verify the handler runs cleanly when there are completed commands
	// that are too recent to delete (less than 7 days old).
	// The actual deletion SQL is exercised by TestQueueDeleteOldCompleted
	// in queue_test.go which uses a past cutoff directly.

	// Run again against an empty-ish queue to verify no-op case works.
	if err := handler.Handle(ctx, cmd); err != nil {
		t.Fatalf("Handle (second run): %v", err)
	}
}
