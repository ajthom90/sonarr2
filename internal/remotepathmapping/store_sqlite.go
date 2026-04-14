package remotepathmapping

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

type sqliteStore struct{ pool *db.SQLitePool }

// NewSQLiteStore returns a Store backed by SQLite.
func NewSQLiteStore(pool *db.SQLitePool) Store {
	return &sqliteStore{pool: pool}
}

func (s *sqliteStore) Create(ctx context.Context, m Mapping) (Mapping, error) {
	var out Mapping
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		q := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := q.CreateRemotePathMapping(ctx, sqlitegen.CreateRemotePathMappingParams{
			Host:       m.Host,
			RemotePath: m.RemotePath,
			LocalPath:  m.LocalPath,
		})
		if err != nil {
			return err
		}
		out = Mapping{ID: int(row.ID), Host: row.Host, RemotePath: row.RemotePath, LocalPath: row.LocalPath}
		return nil
	})
	if err != nil {
		return Mapping{}, fmt.Errorf("remotepathmapping: create: %w", err)
	}
	return out, nil
}

func (s *sqliteStore) GetByID(ctx context.Context, id int) (Mapping, error) {
	q := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := q.GetRemotePathMappingByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Mapping{}, ErrNotFound
	}
	if err != nil {
		return Mapping{}, fmt.Errorf("remotepathmapping: get: %w", err)
	}
	return Mapping{ID: int(row.ID), Host: row.Host, RemotePath: row.RemotePath, LocalPath: row.LocalPath}, nil
}

func (s *sqliteStore) List(ctx context.Context) ([]Mapping, error) {
	q := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := q.ListRemotePathMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("remotepathmapping: list: %w", err)
	}
	out := make([]Mapping, 0, len(rows))
	for _, r := range rows {
		out = append(out, Mapping{ID: int(r.ID), Host: r.Host, RemotePath: r.RemotePath, LocalPath: r.LocalPath})
	}
	return out, nil
}

func (s *sqliteStore) Update(ctx context.Context, m Mapping) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		q := sqlitegen.New(&sqliteExec{exec: exec})
		return q.UpdateRemotePathMapping(ctx, sqlitegen.UpdateRemotePathMappingParams{
			ID:         int64(m.ID),
			Host:       m.Host,
			RemotePath: m.RemotePath,
			LocalPath:  m.LocalPath,
		})
	})
	if err != nil {
		return fmt.Errorf("remotepathmapping: update: %w", err)
	}
	return nil
}

func (s *sqliteStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		q := sqlitegen.New(&sqliteExec{exec: exec})
		return q.DeleteRemotePathMapping(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("remotepathmapping: delete: %w", err)
	}
	return nil
}

// sqliteExec adapts db.Executor to sqlc's DBTX.
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

type sqliteQuery struct{ q db.Querier }

func (a sqliteQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteQuery: ExecContext not allowed")
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
