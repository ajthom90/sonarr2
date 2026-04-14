package rootfolder_test

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/rootfolder"
)

// newTestPool returns an in-memory SQLite pool with all migrations applied.
// Shared between rootfolder_test.go and backfill_test.go so tests that need
// both a rootfolder.Store and a library.SeriesStore can reuse one pool.
func newTestPool(t *testing.T) *db.SQLitePool {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

// newTestStore constructs a rootfolder.Store backed by a fresh pool.
func newTestStore(t *testing.T) rootfolder.Store {
	t.Helper()
	return rootfolder.NewSQLiteStore(newTestPool(t))
}

// newTestSeriesStore constructs a library.SeriesStore backed by the given
// pool so callers can share one pool across multiple stores.
func newTestSeriesStore(t *testing.T, pool *db.SQLitePool) library.SeriesStore {
	t.Helper()
	lib, err := library.New(pool, events.NewBus(4))
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return lib.Series
}

// seedQualityProfile inserts quality_profiles(id=1, name='Any') so series
// Create calls satisfy the FK on series.quality_profile_id.
func seedQualityProfile(t *testing.T, pool *db.SQLitePool) error {
	t.Helper()
	return pool.Write(context.Background(), func(exec db.Executor) error {
		_, err := exec.ExecContext(context.Background(),
			`INSERT INTO quality_profiles (id, name) VALUES (1, 'Any')`)
		return err
	})
}
