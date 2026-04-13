package commands

import (
	"context"
	"encoding/json"
	"time"
)

// Queue is the interface for durable command storage. A command moves through
// the lifecycle: queued → running → completed|failed.
//
// Claim returns nil when no queued command is available (not an error).
// FindDuplicate returns (id, true, nil) when a matching active command exists,
// or (0, false, nil) when none is found.
type Queue interface {
	Enqueue(ctx context.Context, name string, body json.RawMessage, priority Priority, trigger Trigger, dedupKey string) (Command, error)
	Claim(ctx context.Context, workerID string, leaseDuration time.Duration) (*Command, error)
	Complete(ctx context.Context, id int64, durationMs int64, result json.RawMessage, message string) error
	Fail(ctx context.Context, id int64, durationMs int64, exception string, message string) error
	RefreshLease(ctx context.Context, id int64, leaseDuration time.Duration) error
	SweepExpiredLeases(ctx context.Context) (int64, error)
	FindDuplicate(ctx context.Context, dedupKey string) (int64, bool, error)
	DeleteOldCompleted(ctx context.Context, olderThan time.Time) (int64, error)
	Get(ctx context.Context, id int64) (Command, error)
	// ListRecent returns up to limit commands, ordered by queued_at descending.
	// Pass 0 for limit to return all.
	ListRecent(ctx context.Context, limit int) ([]Command, error)
}
