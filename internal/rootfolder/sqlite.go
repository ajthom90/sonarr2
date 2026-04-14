package rootfolder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

type sqliteStore struct {
	pool *db.SQLitePool
}

// NewSQLiteStore returns a Store backed by a SQLite pool.
func NewSQLiteStore(pool *db.SQLitePool) Store {
	return &sqliteStore{pool: pool}
}

func (s *sqliteStore) Create(ctx context.Context, path string) (RootFolder, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return RootFolder{}, errors.New("rootfolder: path is required")
	}
	var out RootFolder
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateRootFolder(ctx, path)
		if err != nil {
			if isSQLiteUniqueErr(err) {
				return ErrAlreadyExists
			}
			return fmt.Errorf("rootfolder: create: %w", err)
		}
		out = rowToDomain(row)
		return nil
	})
	if err != nil {
		return RootFolder{}, err
	}
	return out, nil
}

func (s *sqliteStore) Get(ctx context.Context, id int64) (RootFolder, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetRootFolder(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return RootFolder{}, ErrNotFound
	}
	if err != nil {
		return RootFolder{}, fmt.Errorf("rootfolder: get: %w", err)
	}
	return rowToDomain(row), nil
}

func (s *sqliteStore) GetByPath(ctx context.Context, path string) (RootFolder, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetRootFolderByPath(ctx, strings.TrimSpace(path))
	if errors.Is(err, sql.ErrNoRows) {
		return RootFolder{}, ErrNotFound
	}
	if err != nil {
		return RootFolder{}, fmt.Errorf("rootfolder: get by path: %w", err)
	}
	return rowToDomain(row), nil
}

func (s *sqliteStore) List(ctx context.Context) ([]RootFolder, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListRootFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("rootfolder: list: %w", err)
	}
	out := make([]RootFolder, 0, len(rows))
	for _, r := range rows {
		out = append(out, rowToDomain(r))
	}
	return out, nil
}

func (s *sqliteStore) Delete(ctx context.Context, id int64) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteRootFolder(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("rootfolder: delete: %w", err)
	}
	return nil
}

// rowToDomain converts a sqlc-generated RootFolder row into the package's
// domain type. SQLite stores created_at as TEXT via datetime('now') in UTC;
// parse with the canonical "2006-01-02 15:04:05" layout.
func rowToDomain(row sqlitegen.RootFolder) RootFolder {
	createdAt, _ := time.Parse("2006-01-02 15:04:05", row.CreatedAt)
	return RootFolder{
		ID:        row.ID,
		Path:      row.Path,
		CreatedAt: createdAt,
	}
}

// isSQLiteUniqueErr reports whether err indicates a UNIQUE constraint violation
// on SQLite. modernc.org/sqlite surfaces these as "constraint failed: UNIQUE".
func isSQLiteUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
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
