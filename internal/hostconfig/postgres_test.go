package hostconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgresForTest(t *testing.T) *db.PostgresPool {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Postgres in -short mode")
	}
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("sonarr2_test"),
		tcpostgres.WithUsername("sonarr2"),
		tcpostgres.WithPassword("sonarr2"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("postgres container unavailable: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	pool, err := db.OpenPostgres(ctx, db.PostgresOptions{
		DSN:          dsn,
		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func TestPostgresStoreGetReturnsNotFoundWhenEmpty(t *testing.T) {
	pool := setupPostgresForTest(t)
	store := NewPostgresStore(pool)
	_, err := store.Get(context.Background())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestPostgresStoreUpsertAndGet(t *testing.T) {
	pool := setupPostgresForTest(t)
	store := NewPostgresStore(pool)

	want := HostConfig{
		APIKey:         "test-api-key",
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
	if got.MigrationState != want.MigrationState {
		t.Errorf("MigrationState = %q, want %q", got.MigrationState, want.MigrationState)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestPostgresNewAPIKeyRoundtrip(t *testing.T) {
	pool := setupPostgresForTest(t)
	store := NewPostgresStore(pool)

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
	if time.Since(got.CreatedAt) > time.Minute {
		t.Errorf("CreatedAt too old: %v", got.CreatedAt)
	}
}
