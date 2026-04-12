package commands

import (
	"context"
	"encoding/json"
	"time"
)

type Priority int

const (
	PriorityHigh   Priority = 1
	PriorityNormal Priority = 2
	PriorityLow    Priority = 3
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

type Trigger string

const (
	TriggerManual    Trigger = "manual"
	TriggerScheduled Trigger = "scheduled"
)

type Command struct {
	ID         int64
	Name       string
	Body       json.RawMessage
	Priority   Priority
	Status     Status
	QueuedAt   time.Time
	StartedAt  *time.Time
	EndedAt    *time.Time
	DurationMs *int64
	Exception  string
	Trigger    Trigger
	Message    string
	Result     json.RawMessage
	WorkerID   string
	LeaseUntil *time.Time
	DedupKey   string
}

// Handler executes a single command. Implementations are registered with
// a HandlerRegistry keyed by command name.
type Handler interface {
	Handle(ctx context.Context, cmd Command) error
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(ctx context.Context, cmd Command) error

func (f HandlerFunc) Handle(ctx context.Context, cmd Command) error {
	return f(ctx, cmd)
}

type CommandStarted struct {
	ID   int64
	Name string
}

type CommandCompleted struct {
	ID         int64
	Name       string
	DurationMs int64
}

type CommandFailed struct {
	ID        int64
	Name      string
	Exception string
}
