package downloadclient

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
	settings := dcSQLiteSettings(inst.Settings)
	var out Instance
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteDCExec{exec: exec})
		row, err := queries.CreateDownloadClient(ctx, sqlitegen.CreateDownloadClientParams{
			Name:                     inst.Name,
			Implementation:           inst.Implementation,
			Settings:                 settings,
			Enable:                   dcBoolToInt(inst.Enable),
			Priority:                 int64(inst.Priority),
			RemoveCompletedDownloads: dcBoolToInt(inst.RemoveCompletedDownloads),
			RemoveFailedDownloads:    dcBoolToInt(inst.RemoveFailedDownloads),
		})
		if err != nil {
			return fmt.Errorf("downloadclient: create: %w", err)
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
	queries := sqlitegen.New(sqliteDCQuery{q: s.pool.RawReader()})
	row, err := queries.GetDownloadClientByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("downloadclient: get by id: %w", err)
	}
	return instanceFromSQLite(row)
}

func (s *sqliteInstanceStore) List(ctx context.Context) ([]Instance, error) {
	queries := sqlitegen.New(sqliteDCQuery{q: s.pool.RawReader()})
	rows, err := queries.ListDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("downloadclient: list: %w", err)
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
	settings := dcSQLiteSettings(inst.Settings)
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteDCExec{exec: exec})
		return queries.UpdateDownloadClient(ctx, sqlitegen.UpdateDownloadClientParams{
			ID:                       int64(inst.ID),
			Name:                     inst.Name,
			Implementation:           inst.Implementation,
			Settings:                 settings,
			Enable:                   dcBoolToInt(inst.Enable),
			Priority:                 int64(inst.Priority),
			RemoveCompletedDownloads: dcBoolToInt(inst.RemoveCompletedDownloads),
			RemoveFailedDownloads:    dcBoolToInt(inst.RemoveFailedDownloads),
		})
	})
	if err != nil {
		return fmt.Errorf("downloadclient: update: %w", err)
	}
	return nil
}

func (s *sqliteInstanceStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteDCExec{exec: exec})
		return queries.DeleteDownloadClient(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("downloadclient: delete: %w", err)
	}
	return nil
}

func instanceFromSQLite(r sqlitegen.DownloadClient) (Instance, error) {
	return Instance{
		ID:                       int(r.ID),
		Name:                     r.Name,
		Implementation:           r.Implementation,
		Settings:                 json.RawMessage(r.Settings),
		Enable:                   r.Enable != 0,
		Priority:                 int(r.Priority),
		RemoveCompletedDownloads: r.RemoveCompletedDownloads != 0,
		RemoveFailedDownloads:    r.RemoveFailedDownloads != 0,
		Added:                    parseDCSqliteTime(r.Added),
	}, nil
}

// dcSQLiteSettings converts a json.RawMessage to a string for SQLite storage.
// A nil/empty value is stored as "{}".
func dcSQLiteSettings(s json.RawMessage) string {
	if len(s) == 0 {
		return "{}"
	}
	return string(s)
}

// dcBoolToInt converts a Go bool to the SQLite integer representation.
func dcBoolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// dcSqliteTimeLayout matches the format produced by SQLite's datetime('now').
const dcSqliteTimeLayout = "2006-01-02 15:04:05"

// parseDCSqliteTime parses a SQLite datetime string into a time.Time (UTC).
func parseDCSqliteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(dcSqliteTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// sqliteDCExec adapts a db.Executor to sqlc's DBTX interface for writes.
type sqliteDCExec struct{ exec db.Executor }

func (a *sqliteDCExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteDCExec) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteDCExec: PrepareContext not supported")
}
func (a *sqliteDCExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteDCExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteDCQuery adapts a read-only db.Querier to sqlc's DBTX interface.
type sqliteDCQuery struct{ q db.Querier }

func (a sqliteDCQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteDCQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteDCQuery) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteDCQuery: PrepareContext not supported")
}
func (a sqliteDCQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteDCQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
