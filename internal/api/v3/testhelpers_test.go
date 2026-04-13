package v3

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// profileStores groups quality profile and definition stores.
type profileStores struct {
	profiles profiles.QualityProfileStore
	defs     profiles.QualityDefinitionStore
}

// setupTestPool returns an in-memory SQLite pool with migrations applied.
func setupTestPool(t *testing.T) (*db.SQLitePool, error) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, err
	}
	return pool, nil
}

// setupQualityProfileStore returns profile/definition stores backed by pool.
func setupQualityProfileStore(t *testing.T, pool *db.SQLitePool) profileStores {
	t.Helper()
	return profileStores{
		profiles: profiles.NewSQLiteQualityProfileStore(pool),
		defs:     profiles.NewSQLiteQualityDefinitionStore(pool),
	}
}

// setupCFStore returns a customformats.Store backed by pool.
func setupCFStore(t *testing.T, pool *db.SQLitePool) customformats.Store {
	t.Helper()
	return customformats.NewSQLiteStore(pool)
}

// setupCommandQueue returns a commands.Queue backed by pool.
func setupCommandQueue(t *testing.T, pool *db.SQLitePool) commands.Queue {
	t.Helper()
	return commands.NewSQLiteQueue(pool)
}

// setupHistoryStore returns a history.Store backed by pool.
func setupHistoryStore(t *testing.T, pool *db.SQLitePool) history.Store {
	t.Helper()
	return history.NewSQLiteStore(pool)
}

// mustStringReader wraps a string in a *bytes.Buffer for use as an HTTP body.
func mustStringReader(s string) *bytes.Buffer {
	return bytes.NewBufferString(s)
}
