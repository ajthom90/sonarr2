package v3_test

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/rootfolder"
)

// testHarness bundles the stores that v3 endpoint tests commonly need,
// along with the shared pool so tests can seed cross-cutting rows (e.g.
// quality profiles referenced by series FKs).
type testHarness struct {
	rootFolder rootfolder.Store
	series     library.SeriesStore
	hostConfig hostconfig.Store
	pool       *db.SQLitePool
}

// newTestHarness constructs an in-memory SQLite pool with all migrations
// applied, seeds the default "Any" quality profile, and returns a harness
// with every store wired up. The pool is closed when the test ends.
func newTestHarness(t *testing.T) *testHarness {
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
	// Seed the "Any" quality profile so series Create calls don't fail on FK.
	if err := pool.Write(ctx, func(exec db.Executor) error {
		_, err := exec.ExecContext(ctx, `INSERT INTO quality_profiles (id, name) VALUES (1, 'Any')`)
		return err
	}); err != nil {
		t.Fatalf("seed quality profile: %v", err)
	}
	lib, err := library.New(pool, events.NewBus(4))
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}
	return &testHarness{
		rootFolder: rootfolder.NewSQLiteStore(pool),
		series:     lib.Series,
		hostConfig: hostconfig.NewSQLiteStore(pool),
		pool:       pool,
	}
}
