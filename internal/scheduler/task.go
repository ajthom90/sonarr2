package scheduler

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by Get when no scheduled task with the given type
// name exists in the store.
var ErrNotFound = errors.New("scheduler: not found")

// ScheduledTask is the domain representation of a row in the scheduled_tasks
// table.
type ScheduledTask struct {
	TypeName      string
	IntervalSecs  int
	LastExecution *time.Time
	NextExecution time.Time
}

// TaskStore persists and queries scheduled tasks.
type TaskStore interface {
	// GetDueTasks returns all tasks whose next_execution is <= now.
	GetDueTasks(ctx context.Context) ([]ScheduledTask, error)

	// UpdateExecution sets last_execution = now and next_execution = nextExecution.
	UpdateExecution(ctx context.Context, typeName string, nextExecution time.Time) error

	// Upsert inserts or updates a scheduled task. On conflict the interval is
	// updated; next_execution is only written on INSERT.
	Upsert(ctx context.Context, task ScheduledTask) error

	// Get returns the task with the given type name, or ErrNotFound.
	Get(ctx context.Context, typeName string) (ScheduledTask, error)

	// List returns all scheduled tasks ordered by type_name.
	List(ctx context.Context) ([]ScheduledTask, error)
}
