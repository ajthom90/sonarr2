package hostconfig

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

// SQLiteStore implements Store against a SQLite database using the
// sqlc-generated queries in internal/db/gen/sqlite. All writes funnel
// through the pool's writer goroutine; reads go through the read-only
// pool.
type SQLiteStore struct {
	pool *db.SQLitePool
}

// NewSQLiteStore returns a Store backed by the given SQLite pool.
func NewSQLiteStore(pool *db.SQLitePool) *SQLiteStore {
	return &SQLiteStore{pool: pool}
}

// Get implements Store.
func (s *SQLiteStore) Get(ctx context.Context) (HostConfig, error) {
	// The sqlc-generated SQLite Queries type accepts any DBTX that satisfies
	// database/sql's interface. Our SQLitePool.RawReader() is a *sql.DB, which
	// satisfies this — reads go through the read-only connection.
	queries := sqlitegen.New(s.pool.RawReader())
	row, err := queries.GetHostConfig(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return HostConfig{}, ErrNotFound
	}
	if err != nil {
		return HostConfig{}, fmt.Errorf("hostconfig: sqlite get: %w", err)
	}
	createdAt, _ := time.Parse("2006-01-02 15:04:05", row.CreatedAt)
	updatedAt, _ := time.Parse("2006-01-02 15:04:05", row.UpdatedAt)
	return HostConfig{
		APIKey:                row.ApiKey,
		AuthMode:              row.AuthMode,
		MigrationState:        row.MigrationState,
		TvdbApiKey:            row.TvdbApiKey,
		RecycleBin:            row.RecycleBin,
		RecycleBinCleanupDays: int(row.RecycleBinCleanupDays),
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	}, nil
}

// Upsert implements Store.
func (s *SQLiteStore) Upsert(ctx context.Context, hc HostConfig) error {
	return s.pool.Write(ctx, func(exec db.Executor) error {
		// Wrap exec in an adapter that satisfies sqlc's DBTX interface for writes.
		queries := sqlitegen.New(execAdapter{exec: exec})
		return queries.UpsertHostConfig(ctx, sqlitegen.UpsertHostConfigParams{
			ApiKey:                hc.APIKey,
			AuthMode:              hc.AuthMode,
			MigrationState:        hc.MigrationState,
			TvdbApiKey:            hc.TvdbApiKey,
			RecycleBin:            hc.RecycleBin,
			RecycleBinCleanupDays: int64(hc.RecycleBinCleanupDays),
		})
	})
}

// execAdapter adapts a db.Executor to sqlc's DBTX interface for writes.
// The PrepareContext method is not implemented because sqlc's generated code
// for SQLite does not use prepared statements (`emit_prepared_queries: false`
// in sqlc.yaml). If that setting ever changes, this adapter must grow.
type execAdapter struct{ exec db.Executor }

func (a execAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a execAdapter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("execAdapter: PrepareContext not supported")
}
func (a execAdapter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a execAdapter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}
