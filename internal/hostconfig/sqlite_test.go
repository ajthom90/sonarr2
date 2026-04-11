package hostconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
)

func setupSQLiteForTest(t *testing.T) *db.SQLitePool {
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

func TestSQLiteStoreGetReturnsNotFoundWhenEmpty(t *testing.T) {
	pool := setupSQLiteForTest(t)
	store := NewSQLiteStore(pool)
	_, err := store.Get(context.Background())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStoreUpsertAndGet(t *testing.T) {
	pool := setupSQLiteForTest(t)
	store := NewSQLiteStore(pool)

	want := HostConfig{
		APIKey:         "sqlite-test-key",
		AuthMode:       "forms",
		MigrationState: "clean",
	}
	if err := store.Upsert(context.Background(), want); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if got.AuthMode != want.AuthMode {
		t.Errorf("AuthMode = %q, want %q", got.AuthMode, want.AuthMode)
	}
}

func TestSQLiteNewAPIKeyRoundtrip(t *testing.T) {
	pool := setupSQLiteForTest(t)
	store := NewSQLiteStore(pool)

	key := NewAPIKey()
	err := store.Upsert(context.Background(), HostConfig{
		APIKey:         key,
		AuthMode:       "forms",
		MigrationState: "clean",
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.APIKey != key {
		t.Errorf("APIKey roundtrip: got %q, want %q", got.APIKey, key)
	}
}
