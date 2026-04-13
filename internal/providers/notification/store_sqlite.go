package notification

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

type sqliteInstanceStore struct {
	pool *db.SQLitePool
}

// NewSQLiteInstanceStore returns an InstanceStore backed by a SQLite pool.
func NewSQLiteInstanceStore(pool *db.SQLitePool) InstanceStore {
	return &sqliteInstanceStore{pool: pool}
}

func (s *sqliteInstanceStore) Create(ctx context.Context, inst Instance) (Instance, error) {
	settings := notifSQLiteStr(inst.Settings, "{}")
	tags := notifSQLiteStr(inst.Tags, "[]")
	var out Instance
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteNotifExec{exec: exec})
		row, err := queries.CreateNotification(ctx, sqlitegen.CreateNotificationParams{
			Name:           inst.Name,
			Implementation: inst.Implementation,
			Settings:       settings,
			OnGrab:         notifBoolToInt(inst.OnGrab),
			OnDownload:     notifBoolToInt(inst.OnDownload),
			OnHealthIssue:  notifBoolToInt(inst.OnHealthIssue),
			Tags:           tags,
		})
		if err != nil {
			return fmt.Errorf("notification: create: %w", err)
		}
		var convErr error
		out, convErr = instanceFromSQLite(row)
		return convErr
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

func (s *sqliteInstanceStore) GetByID(ctx context.Context, id int) (Instance, error) {
	queries := sqlitegen.New(sqliteNotifQuery{q: s.pool.RawReader()})
	row, err := queries.GetNotificationByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("notification: get by id: %w", err)
	}
	return instanceFromSQLite(row)
}

func (s *sqliteInstanceStore) List(ctx context.Context) ([]Instance, error) {
	queries := sqlitegen.New(sqliteNotifQuery{q: s.pool.RawReader()})
	rows, err := queries.ListNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("notification: list: %w", err)
	}
	out := make([]Instance, 0, len(rows))
	for _, r := range rows {
		inst, err := instanceFromSQLite(r)
		if err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, nil
}

func (s *sqliteInstanceStore) Update(ctx context.Context, inst Instance) error {
	settings := notifSQLiteStr(inst.Settings, "{}")
	tags := notifSQLiteStr(inst.Tags, "[]")
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteNotifExec{exec: exec})
		return queries.UpdateNotification(ctx, sqlitegen.UpdateNotificationParams{
			ID:             int64(inst.ID),
			Name:           inst.Name,
			Implementation: inst.Implementation,
			Settings:       settings,
			OnGrab:         notifBoolToInt(inst.OnGrab),
			OnDownload:     notifBoolToInt(inst.OnDownload),
			OnHealthIssue:  notifBoolToInt(inst.OnHealthIssue),
			Tags:           tags,
		})
	})
	if err != nil {
		return fmt.Errorf("notification: update: %w", err)
	}
	return nil
}

func (s *sqliteInstanceStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteNotifExec{exec: exec})
		return queries.DeleteNotification(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("notification: delete: %w", err)
	}
	return nil
}

func instanceFromSQLite(r sqlitegen.Notification) (Instance, error) {
	return Instance{
		ID:             int(r.ID),
		Name:           r.Name,
		Implementation: r.Implementation,
		Settings:       json.RawMessage(r.Settings),
		OnGrab:         r.OnGrab != 0,
		OnDownload:     r.OnDownload != 0,
		OnHealthIssue:  r.OnHealthIssue != 0,
		Tags:           json.RawMessage(r.Tags),
		Added:          parseNotifSQLiteTime(r.Added),
	}, nil
}

// notifSQLiteStr converts a json.RawMessage to a string for SQLite storage.
// A nil/empty value is stored as the provided fallback (e.g. "{}" or "[]").
func notifSQLiteStr(s json.RawMessage, fallback string) string {
	if len(s) == 0 {
		return fallback
	}
	return string(s)
}

// notifBoolToInt converts a Go bool to the SQLite integer representation.
func notifBoolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// notifSQLiteTimeLayout matches the format produced by SQLite's datetime('now').
const notifSQLiteTimeLayout = "2006-01-02 15:04:05"

// parseNotifSQLiteTime parses a SQLite datetime string into a time.Time (UTC).
func parseNotifSQLiteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(notifSQLiteTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// sqliteNotifExec adapts a db.Executor to sqlc's DBTX interface for writes.
type sqliteNotifExec struct{ exec db.Executor }

func (a *sqliteNotifExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteNotifExec) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteNotifExec: PrepareContext not supported")
}
func (a *sqliteNotifExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteNotifExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteNotifQuery adapts a read-only db.Querier to sqlc's DBTX interface.
type sqliteNotifQuery struct{ q db.Querier }

func (a sqliteNotifQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteNotifQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteNotifQuery) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteNotifQuery: PrepareContext not supported")
}
func (a sqliteNotifQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteNotifQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
