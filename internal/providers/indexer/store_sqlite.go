package indexer

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
	settings := sqliteSettings(inst.Settings)
	var out Instance
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteIndexerExec{exec: exec})
		row, err := queries.CreateIndexer(ctx, sqlitegen.CreateIndexerParams{
			Name:                    inst.Name,
			Implementation:          inst.Implementation,
			Settings:                settings,
			EnableRss:               boolToInt(inst.EnableRss),
			EnableAutomaticSearch:   boolToInt(inst.EnableAutomaticSearch),
			EnableInteractiveSearch: boolToInt(inst.EnableInteractiveSearch),
			Priority:                int64(inst.Priority),
		})
		if err != nil {
			return fmt.Errorf("indexer: create: %w", err)
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
	queries := sqlitegen.New(sqliteIndexerQuery{q: s.pool.RawReader()})
	row, err := queries.GetIndexerByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("indexer: get by id: %w", err)
	}
	return instanceFromSQLite(row)
}

func (s *sqliteInstanceStore) List(ctx context.Context) ([]Instance, error) {
	queries := sqlitegen.New(sqliteIndexerQuery{q: s.pool.RawReader()})
	rows, err := queries.ListIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("indexer: list: %w", err)
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
	settings := sqliteSettings(inst.Settings)
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteIndexerExec{exec: exec})
		return queries.UpdateIndexer(ctx, sqlitegen.UpdateIndexerParams{
			ID:                      int64(inst.ID),
			Name:                    inst.Name,
			Implementation:          inst.Implementation,
			Settings:                settings,
			EnableRss:               boolToInt(inst.EnableRss),
			EnableAutomaticSearch:   boolToInt(inst.EnableAutomaticSearch),
			EnableInteractiveSearch: boolToInt(inst.EnableInteractiveSearch),
			Priority:                int64(inst.Priority),
		})
	})
	if err != nil {
		return fmt.Errorf("indexer: update: %w", err)
	}
	return nil
}

func (s *sqliteInstanceStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteIndexerExec{exec: exec})
		return queries.DeleteIndexer(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("indexer: delete: %w", err)
	}
	return nil
}

func instanceFromSQLite(r sqlitegen.Indexer) (Instance, error) {
	return Instance{
		ID:                      int(r.ID),
		Name:                    r.Name,
		Implementation:          r.Implementation,
		Settings:                json.RawMessage(r.Settings),
		EnableRss:               r.EnableRss != 0,
		EnableAutomaticSearch:   r.EnableAutomaticSearch != 0,
		EnableInteractiveSearch: r.EnableInteractiveSearch != 0,
		Priority:                int(r.Priority),
		Added:                   parseSqliteTime(r.Added),
	}, nil
}

// sqliteSettings converts a json.RawMessage to a string for SQLite storage.
// A nil/empty value is stored as "{}".
func sqliteSettings(s json.RawMessage) string {
	if len(s) == 0 {
		return "{}"
	}
	return string(s)
}

// boolToInt converts a Go bool to the SQLite integer representation.
func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// sqliteTimeLayout matches the format produced by SQLite's datetime('now').
const sqliteTimeLayout = "2006-01-02 15:04:05"

// parseSqliteTime parses a SQLite datetime string into a time.Time (UTC).
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

// sqliteIndexerExec adapts a db.Executor to sqlc's DBTX interface for writes.
type sqliteIndexerExec struct{ exec db.Executor }

func (a *sqliteIndexerExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteIndexerExec) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteIndexerExec: PrepareContext not supported")
}
func (a *sqliteIndexerExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteIndexerExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteIndexerQuery adapts a read-only db.Querier to sqlc's DBTX interface.
type sqliteIndexerQuery struct{ q db.Querier }

func (a sqliteIndexerQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteIndexerQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteIndexerQuery) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteIndexerQuery: PrepareContext not supported")
}
func (a sqliteIndexerQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteIndexerQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
