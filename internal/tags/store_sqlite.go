package tags

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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

func (s *sqliteStore) Create(ctx context.Context, label string) (Tag, error) {
	label = NormalizeLabel(label)
	if label == "" {
		return Tag{}, errors.New("tags: label is required")
	}
	var out Tag
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateTag(ctx, label)
		if err != nil {
			if isSQLiteUniqueErr(err) {
				return ErrDuplicateLabel
			}
			return fmt.Errorf("tags: create: %w", err)
		}
		out = Tag{ID: int(row.ID), Label: row.Label}
		return nil
	})
	if err != nil {
		return Tag{}, err
	}
	return out, nil
}

func (s *sqliteStore) GetByID(ctx context.Context, id int) (Tag, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetTagByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	if err != nil {
		return Tag{}, fmt.Errorf("tags: get by id: %w", err)
	}
	return Tag{ID: int(row.ID), Label: row.Label}, nil
}

func (s *sqliteStore) GetByLabel(ctx context.Context, label string) (Tag, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetTagByLabel(ctx, NormalizeLabel(label))
	if errors.Is(err, sql.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	if err != nil {
		return Tag{}, fmt.Errorf("tags: get by label: %w", err)
	}
	return Tag{ID: int(row.ID), Label: row.Label}, nil
}

func (s *sqliteStore) List(ctx context.Context) ([]Tag, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("tags: list: %w", err)
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, Tag{ID: int(r.ID), Label: r.Label})
	}
	return out, nil
}

func (s *sqliteStore) Update(ctx context.Context, t Tag) error {
	label := NormalizeLabel(t.Label)
	if label == "" {
		return errors.New("tags: label is required")
	}
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		if err := queries.UpdateTag(ctx, sqlitegen.UpdateTagParams{
			Label: label,
			ID:    int64(t.ID),
		}); err != nil {
			if isSQLiteUniqueErr(err) {
				return ErrDuplicateLabel
			}
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("tags: update: %w", err)
	}
	return nil
}

func (s *sqliteStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteTag(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("tags: delete: %w", err)
	}
	return nil
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
