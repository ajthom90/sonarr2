package customformats

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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

func (s *sqliteStore) Create(ctx context.Context, cf CustomFormat) (CustomFormat, error) {
	specsJSON, err := json.Marshal(cf.Specifications)
	if err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: marshal specifications: %w", err)
	}

	var out CustomFormat
	err = s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateCustomFormat(ctx, sqlitegen.CreateCustomFormatParams{
			Name:                cf.Name,
			IncludeWhenRenaming: boolToInt64(cf.IncludeWhenRenaming),
			Specifications:      string(specsJSON),
		})
		if err != nil {
			return fmt.Errorf("customformats: create: %w", err)
		}
		var convErr error
		out, convErr = customFormatFromSQLite(row)
		return convErr
	})
	if err != nil {
		return CustomFormat{}, err
	}
	return out, nil
}

func (s *sqliteStore) GetByID(ctx context.Context, id int) (CustomFormat, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetCustomFormatByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return CustomFormat{}, ErrNotFound
	}
	if err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: get by id: %w", err)
	}
	return customFormatFromSQLite(row)
}

func (s *sqliteStore) List(ctx context.Context) ([]CustomFormat, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListCustomFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("customformats: list: %w", err)
	}
	out := make([]CustomFormat, 0, len(rows))
	for _, r := range rows {
		cf, err := customFormatFromSQLite(r)
		if err != nil {
			return nil, err
		}
		out = append(out, cf)
	}
	return out, nil
}

func (s *sqliteStore) Update(ctx context.Context, cf CustomFormat) error {
	specsJSON, err := json.Marshal(cf.Specifications)
	if err != nil {
		return fmt.Errorf("customformats: marshal specifications: %w", err)
	}

	err = s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpdateCustomFormat(ctx, sqlitegen.UpdateCustomFormatParams{
			ID:                  int64(cf.ID),
			Name:                cf.Name,
			IncludeWhenRenaming: boolToInt64(cf.IncludeWhenRenaming),
			Specifications:      string(specsJSON),
		})
	})
	if err != nil {
		return fmt.Errorf("customformats: update: %w", err)
	}
	return nil
}

func (s *sqliteStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteCustomFormat(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("customformats: delete: %w", err)
	}
	return nil
}

func customFormatFromSQLite(r sqlitegen.CustomFormat) (CustomFormat, error) {
	var specs []Specification
	if err := json.Unmarshal([]byte(r.Specifications), &specs); err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: unmarshal specifications: %w", err)
	}
	if specs == nil {
		specs = []Specification{}
	}
	return CustomFormat{
		ID:                  int(r.ID),
		Name:                r.Name,
		IncludeWhenRenaming: r.IncludeWhenRenaming != 0,
		Specifications:      specs,
	}, nil
}

// boolToInt64 converts a Go bool to the SQLite integer representation.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
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
