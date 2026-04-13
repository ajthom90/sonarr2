package commands

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

// sqliteQueue implements Queue against a SQLite pool.
type sqliteQueue struct {
	pool *db.SQLitePool
}

// NewSQLiteQueue returns a Queue backed by pool.
func NewSQLiteQueue(pool *db.SQLitePool) Queue {
	return &sqliteQueue{pool: pool}
}

func (s *sqliteQueue) Enqueue(ctx context.Context, name string, body json.RawMessage, priority Priority, trigger Trigger, dedupKey string) (Command, error) {
	var out Command
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.EnqueueCommand(ctx, sqlitegen.EnqueueCommandParams{
			Name:     name,
			Body:     string(body),
			Priority: int64(priority),
			Trigger:  string(trigger),
			DedupKey: dedupKey,
		})
		if err != nil {
			return fmt.Errorf("commands: enqueue: %w", err)
		}
		out = commandFromSqlite(row)
		return nil
	})
	if err != nil {
		return Command{}, err
	}
	return out, nil
}

// Claim claims the highest-priority queued command using a two-step pattern
// (SelectNextQueuedCommand + MarkCommandRunning) inside a single Write
// callback. Returns nil when no command is available.
func (s *sqliteQueue) Claim(ctx context.Context, workerID string, leaseDuration time.Duration) (*Command, error) {
	var out *Command
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})

		// Step 1: select the next queued command's ID.
		id, err := queries.SelectNextQueuedCommand(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			out = nil
			return nil
		}
		if err != nil {
			return fmt.Errorf("commands: claim select: %w", err)
		}

		// Step 2: mark it running.
		leaseUntil := time.Now().Add(leaseDuration).UTC().Format(sqliteTimeLayout)
		if err := queries.MarkCommandRunning(ctx, sqlitegen.MarkCommandRunningParams{
			WorkerID:   workerID,
			LeaseUntil: sql.NullString{String: leaseUntil, Valid: true},
			ID:         id,
		}); err != nil {
			return fmt.Errorf("commands: claim mark running: %w", err)
		}

		// Step 3: load the full row to return.
		row, err := queries.GetCommand(ctx, id)
		if err != nil {
			return fmt.Errorf("commands: claim get: %w", err)
		}
		cmd := commandFromSqlite(row)
		out = &cmd
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *sqliteQueue) Complete(ctx context.Context, id int64, durationMs int64, result json.RawMessage, message string) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.CompleteCommand(ctx, sqlitegen.CompleteCommandParams{
			ID:         id,
			DurationMs: sql.NullInt64{Int64: durationMs, Valid: true},
			Result:     string(result),
			Message:    message,
		})
	})
	if err != nil {
		return fmt.Errorf("commands: complete: %w", err)
	}
	return nil
}

func (s *sqliteQueue) Fail(ctx context.Context, id int64, durationMs int64, exception string, message string) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.FailCommand(ctx, sqlitegen.FailCommandParams{
			ID:         id,
			DurationMs: sql.NullInt64{Int64: durationMs, Valid: true},
			Exception:  exception,
			Message:    message,
		})
	})
	if err != nil {
		return fmt.Errorf("commands: fail: %w", err)
	}
	return nil
}

func (s *sqliteQueue) RefreshLease(ctx context.Context, id int64, leaseDuration time.Duration) error {
	leaseUntil := time.Now().Add(leaseDuration).UTC().Format(sqliteTimeLayout)
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.RefreshLease(ctx, sqlitegen.RefreshLeaseParams{
			ID:         id,
			LeaseUntil: sql.NullString{String: leaseUntil, Valid: true},
		})
	})
	if err != nil {
		return fmt.Errorf("commands: refresh lease: %w", err)
	}
	return nil
}

func (s *sqliteQueue) SweepExpiredLeases(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		var err error
		n, err = queries.SweepExpiredLeases(ctx)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("commands: sweep expired leases: %w", err)
	}
	return n, nil
}

func (s *sqliteQueue) FindDuplicate(ctx context.Context, dedupKey string) (int64, bool, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	id, err := queries.FindDuplicate(ctx, dedupKey)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("commands: find duplicate: %w", err)
	}
	return id, true, nil
}

func (s *sqliteQueue) DeleteOldCompleted(ctx context.Context, olderThan time.Time) (int64, error) {
	var n int64
	ts := olderThan.UTC().Format(sqliteTimeLayout)
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		var err error
		n, err = queries.DeleteOldCompleted(ctx, sql.NullString{String: ts, Valid: true})
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("commands: delete old completed: %w", err)
	}
	return n, nil
}

func (s *sqliteQueue) Get(ctx context.Context, id int64) (Command, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetCommand(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, fmt.Errorf("commands: get %d: %w", id, ErrNotFound)
	}
	if err != nil {
		return Command{}, fmt.Errorf("commands: get: %w", err)
	}
	return commandFromSqlite(row), nil
}

// ListRecent returns up to limit commands ordered by queued_at DESC.
// Pass 0 for no limit.
func (s *sqliteQueue) ListRecent(ctx context.Context, limit int) ([]Command, error) {
	q := `SELECT id, name, body, priority, status, queued_at, started_at, ended_at,
	             duration_ms, exception, trigger, message, result, worker_id,
	             lease_until, dedup_key
	      FROM commands ORDER BY queued_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := s.pool.RawReader().QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("commands: list recent: %w", err)
	}
	defer rows.Close()

	var out []Command
	for rows.Next() {
		var r sqlitegen.Command
		var startedAt, endedAt, leaseUntil sql.NullString
		var durationMs sql.NullInt64
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Body, &r.Priority, &r.Status, &r.QueuedAt,
			&startedAt, &endedAt, &durationMs, &r.Exception,
			&r.Trigger, &r.Message, &r.Result, &r.WorkerID, &leaseUntil, &r.DedupKey,
		); err != nil {
			return nil, fmt.Errorf("commands: list recent scan: %w", err)
		}
		r.StartedAt = startedAt
		r.EndedAt = endedAt
		r.DurationMs = durationMs
		r.LeaseUntil = leaseUntil
		out = append(out, commandFromSqlite(r))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("commands: list recent rows: %w", err)
	}
	return out, nil
}

// commandFromSqlite converts a sqlc-generated sqlitegen.Command row to the
// canonical commands.Command struct.
func commandFromSqlite(r sqlitegen.Command) Command {
	return Command{
		ID:         r.ID,
		Name:       r.Name,
		Body:       json.RawMessage(r.Body),
		Priority:   Priority(r.Priority),
		Status:     Status(r.Status),
		QueuedAt:   parseSqliteTime(r.QueuedAt),
		StartedAt:  ptrFromNullString(r.StartedAt),
		EndedAt:    ptrFromNullString(r.EndedAt),
		DurationMs: ptrFromNullInt64(r.DurationMs),
		Exception:  r.Exception,
		Trigger:    Trigger(r.Trigger),
		Message:    r.Message,
		Result:     json.RawMessage(r.Result),
		WorkerID:   r.WorkerID,
		LeaseUntil: ptrFromNullString(r.LeaseUntil),
		DedupKey:   r.DedupKey,
	}
}

// sqliteTimeLayout matches the format produced by SQLite's datetime('now').
const sqliteTimeLayout = "2006-01-02 15:04:05"

// parseSqliteTime parses a SQLite datetime string into a time.Time.
func parseSqliteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(sqliteTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// ptrFromNullString converts a sql.NullString datetime to a *time.Time.
func ptrFromNullString(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, err := time.Parse(sqliteTimeLayout, v.String)
	if err != nil {
		return nil
	}
	return &t
}

// ptrFromNullInt64 converts a sql.NullInt64 to a *int64.
func ptrFromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

// sqliteExec adapts a db.Executor to sqlc's DBTX interface for writes.
type sqliteExec struct{ exec db.Executor }

func (a *sqliteExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteExec) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteExec: PrepareContext not supported")
}
func (a *sqliteExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteQuery adapts a read-only db.Querier to sqlc's DBTX interface.
type sqliteQuery struct{ q db.Querier }

func (a sqliteQuery) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errors.New("sqliteQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteQuery) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteQuery: PrepareContext not supported")
}
func (a sqliteQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
