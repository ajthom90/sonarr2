# Milestone 3 — Scheduler, Command Queue, and Worker Pool

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a durable command queue (DB-backed), a scheduler that enqueues time-based commands, and a worker pool that claims and executes commands. After M3, the binary runs background work automatically — which is what M4+ domain logic (RSS sync, metadata refresh, import) needs.

**Architecture:** `internal/commands/` owns the command lifecycle: types, queue, handler registry, and worker pool. `internal/scheduler/` owns the tick loop that reads `scheduled_tasks` and enqueues due commands. `app.Run` starts both as background goroutines alongside the HTTP server.

**Tech Stack:** Existing Go 1.23 stack. No new runtime deps. Commands and scheduled_tasks are two new DB tables (migrations 00007-00008).

---

## Layout

```
internal/
├── commands/
│   ├── command.go       # Command type, Priority, Status, Trigger, Handler iface
│   ├── registry.go      # HandlerRegistry
│   ├── registry_test.go
│   ├── queue.go         # Queue interface
│   ├── queue_postgres.go
│   ├── queue_sqlite.go
│   ├── queue_test.go
│   ├── worker.go        # WorkerPool
│   └── worker_test.go
├── scheduler/
│   ├── scheduler.go     # Scheduler tick loop
│   ├── task.go          # ScheduledTask type + TaskStore iface
│   ├── task_postgres.go
│   ├── task_sqlite.go
│   └── scheduler_test.go
├── db/
│   ├── migrations/{postgres,sqlite}/00007_commands.sql
│   ├── migrations/{postgres,sqlite}/00008_scheduled_tasks.sql
│   ├── queries/{postgres,sqlite}/commands.sql
│   ├── queries/{postgres,sqlite}/scheduled_tasks.sql
│   └── gen/  (regenerated)
└── app/
    ├── app.go           # MODIFIED: starts scheduler + workers in Run
    └── app_test.go      # MODIFIED: integration test
```

---

## Task 1 — Migrations + queries + sqlc regen

Add `commands` and `scheduled_tasks` tables for both dialects, query files, and regenerate sqlc.

**Files:** 4 migration files + 4 query files + regenerated gen/ + sqlc.yaml unchanged.

### Postgres migration 00007_commands.sql

```sql
-- +goose Up
CREATE TABLE commands (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    body        JSONB NOT NULL DEFAULT '{}',
    priority    SMALLINT NOT NULL DEFAULT 2,
    status      TEXT NOT NULL DEFAULT 'queued',
    queued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at  TIMESTAMPTZ,
    ended_at    TIMESTAMPTZ,
    duration_ms BIGINT,
    exception   TEXT NOT NULL DEFAULT '',
    trigger     TEXT NOT NULL DEFAULT 'manual',
    message     TEXT NOT NULL DEFAULT '',
    result      JSONB NOT NULL DEFAULT '{}',
    worker_id   TEXT NOT NULL DEFAULT '',
    lease_until TIMESTAMPTZ,
    dedup_key   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX commands_claim_idx ON commands (priority ASC, queued_at ASC)
    WHERE status = 'queued';
CREATE INDEX commands_lease_sweep_idx ON commands (lease_until)
    WHERE status = 'running';
CREATE INDEX commands_dedup_idx ON commands (dedup_key)
    WHERE dedup_key != '' AND status IN ('queued', 'running');

-- +goose Down
DROP TABLE commands;
```

### SQLite migration 00007_commands.sql

```sql
-- +goose Up
CREATE TABLE commands (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '{}',
    priority    INTEGER NOT NULL DEFAULT 2,
    status      TEXT NOT NULL DEFAULT 'queued',
    queued_at   TEXT NOT NULL DEFAULT (datetime('now')),
    started_at  TEXT,
    ended_at    TEXT,
    duration_ms INTEGER,
    exception   TEXT NOT NULL DEFAULT '',
    trigger     TEXT NOT NULL DEFAULT 'manual',
    message     TEXT NOT NULL DEFAULT '',
    result      TEXT NOT NULL DEFAULT '{}',
    worker_id   TEXT NOT NULL DEFAULT '',
    lease_until TEXT,
    dedup_key   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX commands_claim_idx ON commands (priority ASC, queued_at ASC)
    WHERE status = 'queued';
CREATE INDEX commands_lease_sweep_idx ON commands (lease_until)
    WHERE status = 'running';
CREATE INDEX commands_dedup_idx ON commands (dedup_key)
    WHERE dedup_key != '' AND status IN ('queued', 'running');

-- +goose Down
DROP TABLE commands;
```

### Postgres migration 00008_scheduled_tasks.sql

```sql
-- +goose Up
CREATE TABLE scheduled_tasks (
    type_name      TEXT PRIMARY KEY,
    interval_secs  INTEGER NOT NULL,
    last_execution TIMESTAMPTZ,
    next_execution TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE scheduled_tasks;
```

### SQLite migration 00008_scheduled_tasks.sql

```sql
-- +goose Up
CREATE TABLE scheduled_tasks (
    type_name      TEXT PRIMARY KEY,
    interval_secs  INTEGER NOT NULL,
    last_execution TEXT,
    next_execution TEXT NOT NULL
);

-- +goose Down
DROP TABLE scheduled_tasks;
```

### Postgres queries/commands.sql

```sql
-- name: EnqueueCommand :one
INSERT INTO commands (name, body, priority, trigger, dedup_key)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ClaimCommand :one
UPDATE commands
SET status = 'running',
    worker_id = $1,
    started_at = now(),
    lease_until = $2
WHERE id = (
    SELECT id FROM commands
    WHERE status = 'queued'
    ORDER BY priority ASC, queued_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: CompleteCommand :exec
UPDATE commands
SET status = 'completed',
    ended_at = now(),
    duration_ms = $2,
    result = $3,
    message = $4
WHERE id = $1;

-- name: FailCommand :exec
UPDATE commands
SET status = 'failed',
    ended_at = now(),
    duration_ms = $2,
    exception = $3,
    message = $4
WHERE id = $1;

-- name: GetCommand :one
SELECT * FROM commands WHERE id = $1;

-- name: RefreshLease :exec
UPDATE commands SET lease_until = $2 WHERE id = $1 AND status = 'running';

-- name: SweepExpiredLeases :execrows
UPDATE commands
SET status = 'queued', worker_id = '', started_at = NULL, lease_until = NULL
WHERE status = 'running' AND lease_until < now();

-- name: FindDuplicate :one
SELECT id FROM commands
WHERE dedup_key = $1 AND status IN ('queued', 'running')
LIMIT 1;

-- name: DeleteOldCompleted :execrows
DELETE FROM commands
WHERE status IN ('completed', 'failed') AND ended_at < $1;
```

### SQLite queries/commands.sql

Same queries but `?` params, `datetime('now')` instead of `now()`, no `FOR UPDATE SKIP LOCKED` on `ClaimCommand`:

```sql
-- name: EnqueueCommand :one
INSERT INTO commands (name, body, priority, trigger, dedup_key)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ClaimCommand :one
UPDATE commands
SET status = 'running',
    worker_id = ?,
    started_at = datetime('now'),
    lease_until = ?
WHERE id = (
    SELECT id FROM commands
    WHERE status = 'queued'
    ORDER BY priority ASC, queued_at ASC
    LIMIT 1
)
RETURNING *;

-- name: CompleteCommand :exec
UPDATE commands
SET status = 'completed',
    ended_at = datetime('now'),
    duration_ms = ?,
    result = ?,
    message = ?
WHERE id = ?;

-- name: FailCommand :exec
UPDATE commands
SET status = 'failed',
    ended_at = datetime('now'),
    duration_ms = ?,
    exception = ?,
    message = ?
WHERE id = ?;

-- name: GetCommand :one
SELECT * FROM commands WHERE id = ?;

-- name: RefreshLease :exec
UPDATE commands SET lease_until = ? WHERE id = ? AND status = 'running';

-- name: SweepExpiredLeases :execrows
UPDATE commands
SET status = 'queued', worker_id = '', started_at = NULL, lease_until = NULL
WHERE status = 'running' AND lease_until < datetime('now');

-- name: FindDuplicate :one
SELECT id FROM commands
WHERE dedup_key = ? AND status IN ('queued', 'running')
LIMIT 1;

-- name: DeleteOldCompleted :execrows
DELETE FROM commands
WHERE status IN ('completed', 'failed') AND ended_at < ?;
```

### Postgres queries/scheduled_tasks.sql

```sql
-- name: GetDueTasks :many
SELECT * FROM scheduled_tasks
WHERE next_execution <= now()
ORDER BY next_execution ASC;

-- name: UpdateTaskExecution :exec
UPDATE scheduled_tasks
SET last_execution = now(),
    next_execution = $2
WHERE type_name = $1;

-- name: UpsertScheduledTask :exec
INSERT INTO scheduled_tasks (type_name, interval_secs, next_execution)
VALUES ($1, $2, $3)
ON CONFLICT (type_name) DO UPDATE
SET interval_secs = EXCLUDED.interval_secs;

-- name: GetScheduledTask :one
SELECT * FROM scheduled_tasks WHERE type_name = $1;

-- name: ListScheduledTasks :many
SELECT * FROM scheduled_tasks ORDER BY type_name;
```

### SQLite queries/scheduled_tasks.sql

```sql
-- name: GetDueTasks :many
SELECT * FROM scheduled_tasks
WHERE next_execution <= datetime('now')
ORDER BY next_execution ASC;

-- name: UpdateTaskExecution :exec
UPDATE scheduled_tasks
SET last_execution = datetime('now'),
    next_execution = ?
WHERE type_name = ?;

-- name: UpsertScheduledTask :exec
INSERT INTO scheduled_tasks (type_name, interval_secs, next_execution)
VALUES (?, ?, ?)
ON CONFLICT (type_name) DO UPDATE
SET interval_secs = excluded.interval_secs;

-- name: GetScheduledTask :one
SELECT * FROM scheduled_tasks WHERE type_name = ?;

-- name: ListScheduledTasks :many
SELECT * FROM scheduled_tasks ORDER BY type_name;
```

### Steps

- [ ] Create 4 migration files with exact content
- [ ] Create 4 query files with exact content
- [ ] Run `go test -race -timeout 60s -short ./internal/db/...` — migrations pass
- [ ] Run `sqlc generate` — verify generated code in gen/{postgres,sqlite}/
- [ ] Run `go build ./internal/db/gen/...` — compiles
- [ ] Commit: `feat(db): add commands and scheduled_tasks tables with queries`

**NOTE on ClaimCommand for SQLite:** sqlc may reject the subquery-in-UPDATE pattern. If it does, split into two queries: `SelectNextQueuedCommand :one` (SELECT) and `MarkCommandRunning :exec` (UPDATE by id). The SQLite single-writer discipline ensures no race between the two. Report the issue and adapt.

---

## Task 2 — commands package: types + registry

Define the core types (`Command`, `Priority`, `Status`, `Trigger`) and the `Handler` interface. Add a `HandlerRegistry` that maps command names to handlers.

**Files:**
- Create: `internal/commands/command.go`
- Create: `internal/commands/registry.go`
- Create: `internal/commands/registry_test.go`

### command.go

```go
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
```

### registry.go

```go
package commands

import "fmt"

// Registry maps command names to Handlers.
type Registry struct {
    handlers map[string]Handler
}

func NewRegistry() *Registry {
    return &Registry{handlers: make(map[string]Handler)}
}

func (r *Registry) Register(name string, h Handler) {
    r.handlers[name] = h
}

func (r *Registry) Get(name string) (Handler, error) {
    h, ok := r.handlers[name]
    if !ok {
        return nil, fmt.Errorf("commands: no handler registered for %q", name)
    }
    return h, nil
}
```

### Tests

- `TestRegistryRegisterAndGet` — register a handler, get it back, call it
- `TestRegistryGetMissing` — returns error for unregistered name
- `TestHandlerFuncAdapter` — wraps a plain function

### Steps

- [ ] Write failing tests
- [ ] Implement types + registry
- [ ] Verify all tests pass
- [ ] Commit: `feat(commands): add Command types and HandlerRegistry`

---

## Task 3 — CommandQueue interface + implementations

Define the `Queue` interface and implement it for both dialects.

**Files:**
- Create: `internal/commands/queue.go`
- Create: `internal/commands/queue_postgres.go`
- Create: `internal/commands/queue_sqlite.go`
- Create: `internal/commands/queue_test.go`

### Queue interface

```go
type Queue interface {
    Enqueue(ctx context.Context, name string, body json.RawMessage, priority Priority, trigger Trigger, dedupKey string) (Command, error)
    Claim(ctx context.Context, workerID string, leaseDuration time.Duration) (*Command, error)  // nil = nothing available
    Complete(ctx context.Context, id int64, durationMs int64, result json.RawMessage, message string) error
    Fail(ctx context.Context, id int64, durationMs int64, exception string, message string) error
    RefreshLease(ctx context.Context, id int64, leaseDuration time.Duration) error
    SweepExpiredLeases(ctx context.Context) (int64, error)
    FindDuplicate(ctx context.Context, dedupKey string) (int64, bool, error)
    DeleteOldCompleted(ctx context.Context, olderThan time.Time) (int64, error)
    Get(ctx context.Context, id int64) (Command, error)
}
```

### Implementation notes

**Postgres:** each method wraps the corresponding sqlc-generated query. `Claim` uses `FOR UPDATE SKIP LOCKED` via the generated `ClaimCommand` query. Returns `nil, nil` when no row is available (pgx.ErrNoRows from the Claim query → no command → return nil, nil).

**SQLite:** `Claim` goes through `pool.Write()` since it's a write operation. If sqlc rejected the subquery-in-UPDATE, the implementer should split into: `SELECT id ... WHERE status='queued' ORDER BY priority, queued_at LIMIT 1` + `UPDATE ... WHERE id = ?` inside the same `pool.Write` callback. This is safe because the single-writer serializes all writes.

**`FindDuplicate`**: returns `(id, true, nil)` if a duplicate exists, `(0, false, nil)` if not. Translates `pgx.ErrNoRows`/`sql.ErrNoRows` to `(0, false, nil)`.

### Tests

Run against SQLite in-memory (same pattern as M2). Key tests:
- `TestQueueEnqueueAndGet` — round-trip
- `TestQueueClaimReturnsHighestPriority` — enqueue 3 commands with different priorities, claim 3 times, assert order
- `TestQueueClaimReturnsNilWhenEmpty` — claim from empty queue returns nil
- `TestQueueCompleteAndFail` — enqueue, claim, complete or fail, verify status
- `TestQueueLeaseRefreshAndSweep` — enqueue, claim with short lease, let it expire, sweep, claim again
- `TestQueueDeduplication` — enqueue with dedup_key, find duplicate returns true
- `TestQueueDeleteOldCompleted` — enqueue, complete, delete old, verify gone

### Steps

- [ ] Verify sqlc-generated query method signatures via `go doc`
- [ ] Write failing tests
- [ ] Implement Queue interface + both dialects (sqlite first since tests run against it)
- [ ] Run tests under `-race`
- [ ] Commit: `feat(commands): add CommandQueue with claim, lease, dedup, sweep`

---

## Task 4 — WorkerPool

A pool of N goroutines that claim-dispatch-release commands in a loop. Handles panics, lease refreshes, and graceful shutdown.

**Files:**
- Create: `internal/commands/worker.go`
- Create: `internal/commands/worker_test.go`

### WorkerPool

```go
type WorkerPool struct {
    queue     Queue
    registry  *Registry
    bus       events.Bus
    log       *slog.Logger
    workers   int
    leaseDur  time.Duration
    cancel    context.CancelFunc
    wg        sync.WaitGroup
}

func NewWorkerPool(queue Queue, registry *Registry, bus events.Bus, log *slog.Logger, workers int) *WorkerPool

func (wp *WorkerPool) Start(ctx context.Context)
func (wp *WorkerPool) Stop()
```

Each worker goroutine loop:
1. `queue.Claim(ctx, workerID, leaseDuration)` — if nil, sleep 200ms and retry
2. Look up handler via `registry.Get(cmd.Name)`
3. Start a lease-refresh ticker (every leaseDuration/2)
4. Call `handler.Handle(ctx, cmd)` in a panic-recovering wrapper
5. On success: `queue.Complete(...)`
6. On error: `queue.Fail(...)`
7. On panic: `queue.Fail(...)` with the panic message
8. Stop the lease-refresh ticker
9. Publish `CommandCompleted` or `CommandFailed` event
10. Repeat

### Command events

Define in `command.go`:
```go
type CommandStarted struct { ID int64; Name string }
type CommandCompleted struct { ID int64; Name string; DurationMs int64 }
type CommandFailed struct { ID int64; Name string; Exception string }
```

### Tests

- `TestWorkerPoolExecutesEnqueuedCommand` — register a handler that sets a flag, enqueue a command, start pool with 1 worker, wait for the flag
- `TestWorkerPoolHandlesPanic` — register a handler that panics, enqueue, verify command transitions to `failed`
- `TestWorkerPoolStopsGracefully` — start pool, stop it, verify no goroutine leak

### Steps

- [ ] Write failing tests
- [ ] Implement WorkerPool
- [ ] Run tests under `-race`
- [ ] Commit: `feat(commands): add WorkerPool with panic recovery and lease refresh`

---

## Task 5 — Scheduler

Tick loop that reads `scheduled_tasks` for due entries and enqueues commands. Also sweeps expired leases on each tick.

**Files:**
- Create: `internal/scheduler/task.go` (ScheduledTask type + TaskStore iface)
- Create: `internal/scheduler/task_postgres.go`
- Create: `internal/scheduler/task_sqlite.go`
- Create: `internal/scheduler/scheduler.go`
- Create: `internal/scheduler/scheduler_test.go`

### TaskStore interface

```go
type TaskStore interface {
    GetDueTasks(ctx context.Context) ([]ScheduledTask, error)
    UpdateExecution(ctx context.Context, typeName string, nextExecution time.Time) error
    Upsert(ctx context.Context, task ScheduledTask) error
    Get(ctx context.Context, typeName string) (ScheduledTask, error)
    List(ctx context.Context) ([]ScheduledTask, error)
}
```

### Scheduler

```go
type Scheduler struct {
    taskStore TaskStore
    queue     commands.Queue
    log       *slog.Logger
    interval  time.Duration  // tick interval, default 500ms
    cancel    context.CancelFunc
    wg        sync.WaitGroup
}

func New(taskStore TaskStore, queue commands.Queue, log *slog.Logger) *Scheduler
func (s *Scheduler) Start(ctx context.Context)
func (s *Scheduler) Stop()
```

Tick loop:
1. Query `taskStore.GetDueTasks(ctx)` for tasks where `next_execution <= now()`
2. For each due task: enqueue a command with `Name=task.TypeName, Priority=Normal, Trigger=Scheduled, DedupKey=task.TypeName` (so duplicate scheduled runs are rejected)
3. Update `taskStore.UpdateExecution(ctx, task.TypeName, now + interval_secs)`
4. Call `queue.SweepExpiredLeases(ctx)` — recover commands from crashed workers
5. Sleep for `s.interval`

### Tests

- `TestSchedulerEnqueuesDueTask` — insert a task with next_execution in the past, start scheduler, wait briefly, verify a command was enqueued
- `TestSchedulerSkipsNotDueTask` — task with next_execution in the future, scheduler tick does not enqueue
- `TestSchedulerSweepsExpiredLeases` — create a command with expired lease, scheduler sweep recovers it

### Steps

- [ ] Write failing tests
- [ ] Implement TaskStore for both dialects
- [ ] Implement Scheduler
- [ ] Run tests under `-race`
- [ ] Commit: `feat(scheduler): add tick-based scheduler with due-task dispatch and lease sweep`

---

## Task 6 — Wire into app.New and app.Run

Create the queue, registry, worker pool, and scheduler in `app.New`. Start them in `app.Run` alongside the HTTP server. Stop them during graceful shutdown.

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`

### Changes to App

```go
type App struct {
    log        *slog.Logger
    server     *api.Server
    pool       db.Pool
    bus        events.Bus
    library    *library.Library
    cmdQueue   commands.Queue       // NEW
    registry   *commands.Registry   // NEW
    workers    *commands.WorkerPool // NEW
    scheduler  *scheduler.Scheduler // NEW
}
```

`New` creates queue + registry + pool + scheduler:
```go
cmdQueue := newCommandQueue(pool)      // dialect dispatch helper
registry := commands.NewRegistry()
workers := commands.NewWorkerPool(cmdQueue, registry, bus, log, min(runtime.NumCPU(), 4))
taskStore := newTaskStore(pool)        // dialect dispatch helper
sched := scheduler.New(taskStore, cmdQueue, log)
```

`Run` starts workers + scheduler in background, stops them during shutdown:
```go
func (a *App) Run(ctx context.Context) error {
    // ... existing startup ...
    a.workers.Start(ctx)
    a.scheduler.Start(ctx)

    // ... existing select on ctx.Done() or errCh ...

    // shutdown:
    a.scheduler.Stop()
    a.workers.Stop()
    // ... existing server.Shutdown, pool.Close ...
}
```

### Integration test

`TestAppCommandExecution` — register a simple handler that sets a flag, enqueue a command via `app.cmdQueue`, start the app (in-memory SQLite), wait for the flag with a timeout, verify the command status is `completed`.

### Steps

- [ ] Implement the wiring in app.go
- [ ] Add the integration test
- [ ] Run `go test -race -timeout 60s -short ./...` — all pass
- [ ] Binary smoke test with persistent SQLite — verify startup log shows scheduler starting, shut down cleanly
- [ ] Commit: `feat(app): wire command queue, worker pool, and scheduler into app lifecycle`

---

## Task 7 — First handler: MessagingCleanup

A simple handler that deletes completed/failed commands older than 7 days. Proves the full pipeline works end-to-end: scheduler enqueues it → worker claims it → handler runs → command completes.

**Files:**
- Create: `internal/commands/handlers/cleanup.go`
- Create: `internal/commands/handlers/cleanup_test.go`
- Modify: `internal/app/app.go` (register the handler and its scheduled task)

### Handler

```go
package handlers

import (
    "context"
    "time"

    "github.com/ajthom90/sonarr2/internal/commands"
)

type CleanupHandler struct {
    queue commands.Queue
}

func NewCleanupHandler(queue commands.Queue) *CleanupHandler {
    return &CleanupHandler{queue: queue}
}

func (h *CleanupHandler) Handle(ctx context.Context, cmd commands.Command) error {
    cutoff := time.Now().Add(-7 * 24 * time.Hour)
    deleted, err := h.queue.DeleteOldCompleted(ctx, cutoff)
    if err != nil {
        return err
    }
    // cmd.Message or result could log deleted count — keep it simple for M3.
    _ = deleted
    return nil
}
```

### Registration in app.New

```go
cleanup := handlers.NewCleanupHandler(cmdQueue)
registry.Register("MessagingCleanup", cleanup)

// Register the scheduled task (1 hour interval)
taskStore.Upsert(ctx, scheduler.ScheduledTask{
    TypeName:     "MessagingCleanup",
    IntervalSecs: 3600,
    NextExecution: time.Now().Add(time.Hour),
})
```

### Test

- `TestCleanupHandlerDeletesOldCommands` — enqueue 3 commands, complete them, backdate their `ended_at`, run the handler, verify they're deleted

### Steps

- [ ] Write the handler + tests
- [ ] Register in app.New
- [ ] Run full test suite
- [ ] Commit: `feat(commands): add MessagingCleanup handler with 1-hour schedule`

---

## Task 8 — Final verification + push

- [ ] `go mod tidy` — commit if changed
- [ ] `make lint` — must pass (including staticcheck + golangci-lint in CI)
- [ ] `go test -race -count=1 -timeout 120s -short ./...` — all packages pass
- [ ] `make clean && make build` — binary builds
- [ ] Smoke test: start binary with SQLite, verify it starts, scheduler log messages appear, shut down cleanly
- [ ] `git log --oneline db80758..HEAD` — verify all M3 commits present
- [ ] `git push origin main`
- [ ] Watch CI: `gh run watch --exit-status` for both test and lint workflows
- [ ] Commit: `N/A (verification only)`

---

## Done

After Task 8 the binary has a background execution engine: commands enqueued → workers claim and run → scheduler dispatches periodic work → cleanup keeps the queue tidy. M4 (parsing) and M5+ (decision engine, providers) hang their domain commands off this substrate.
