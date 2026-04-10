package db

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// postgresContainer starts a fresh Postgres instance via testcontainers-go
// and returns a connection DSN plus a cleanup function. Tests that need
// Postgres call this helper once at the top of the test.
func postgresContainer(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Postgres container in -short mode")
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
		t.Skipf("postgres container failed to start (Docker unavailable?): %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Terminate(context.Background()); err != nil {
			t.Logf("container terminate: %v", err)
		}
	})

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return dsn
}

func TestOpenPostgresConnectsAndPings(t *testing.T) {
	dsn := postgresContainer(t)

	pool, err := OpenPostgres(context.Background(), PostgresOptions{
		DSN:             dsn,
		MaxOpenConns:    4,
		MinOpenConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if pool.Dialect() != DialectPostgres {
		t.Errorf("Dialect() = %q, want postgres", pool.Dialect())
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestOpenPostgresRejectsInvalidDSN(t *testing.T) {
	_, err := OpenPostgres(context.Background(), PostgresOptions{
		DSN:          "::::invalid::::",
		MaxOpenConns: 1,
	})
	if err == nil {
		t.Fatal("expected error for invalid DSN")
	}
}
