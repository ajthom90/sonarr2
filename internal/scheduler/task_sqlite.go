package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

// sqliteTaskStore implements TaskStore against a SQLite pool.
type sqliteTaskStore struct {
	pool *db.SQLitePool
}

// NewSQLiteTaskStore returns a TaskStore backed by pool.
func NewSQLiteTaskStore(pool *db.SQLitePool) TaskStore {
	return &sqliteTaskStore{pool: pool}
}

func (s *sqliteTaskStore) GetDueTasks(ctx context.Context) ([]ScheduledTask, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.GetDueTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("scheduler: get due tasks: %w", err)
	}
	tasks := make([]ScheduledTask, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, scheduledTaskFromSqlite(r))
	}
	return tasks, nil
}

func (s *sqliteTaskStore) UpdateExecution(ctx context.Context, typeName string, nextExecution time.Time) error {
	next := nextExecution.UTC().Format(sqliteTimeLayout)
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpdateTaskExecution(ctx, sqlitegen.UpdateTaskExecutionParams{
			NextExecution: next,
			TypeName:      typeName,
		})
	})
	if err != nil {
		return fmt.Errorf("scheduler: update execution: %w", err)
	}
	return nil
}

func (s *sqliteTaskStore) Upsert(ctx context.Context, task ScheduledTask) error {
	next := task.NextExecution.UTC().Format(sqliteTimeLayout)
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpsertScheduledTask(ctx, sqlitegen.UpsertScheduledTaskParams{
			TypeName:      task.TypeName,
			IntervalSecs:  int64(task.IntervalSecs),
			NextExecution: next,
		})
	})
	if err != nil {
		return fmt.Errorf("scheduler: upsert: %w", err)
	}
	return nil
}

func (s *sqliteTaskStore) Get(ctx context.Context, typeName string) (ScheduledTask, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetScheduledTask(ctx, typeName)
	if errors.Is(err, sql.ErrNoRows) {
		return ScheduledTask{}, fmt.Errorf("scheduler: get %q: %w", typeName, ErrNotFound)
	}
	if err != nil {
		return ScheduledTask{}, fmt.Errorf("scheduler: get: %w", err)
	}
	return scheduledTaskFromSqlite(row), nil
}

func (s *sqliteTaskStore) List(ctx context.Context) ([]ScheduledTask, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListScheduledTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("scheduler: list: %w", err)
	}
	tasks := make([]ScheduledTask, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, scheduledTaskFromSqlite(r))
	}
	return tasks, nil
}

// scheduledTaskFromSqlite converts a sqlc-generated sqlitegen.ScheduledTask
// row to the canonical scheduler.ScheduledTask struct.
func scheduledTaskFromSqlite(r sqlitegen.ScheduledTask) ScheduledTask {
	return ScheduledTask{
		TypeName:      r.TypeName,
		IntervalSecs:  int(r.IntervalSecs),
		LastExecution: ptrFromNullString(r.LastExecution),
		NextExecution: parseSqliteTime(r.NextExecution),
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

// sqliteExec adapts a db.Executor to sqlc's DBTX interface for writes.
type sqliteExec struct{ exec db.Executor }

func (a *sqliteExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteExec) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
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

func (a sqliteQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteQuery) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteQuery: PrepareContext not supported")
}
func (a sqliteQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
