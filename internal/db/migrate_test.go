package db

import (
	"context"
	"testing"
	"time"
)

func TestMigrateSQLiteCreatesHostConfig(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// The host_config table should exist after migrating.
	err = pool.Read(context.Background(), func(q Querier) error {
		row := q.QueryRowContext(context.Background(),
			`SELECT name FROM sqlite_master WHERE type='table' AND name='host_config'`)
		var name string
		return row.Scan(&name)
	})
	if err != nil {
		t.Errorf("host_config table missing: %v", err)
	}
}

func TestMigrateSQLiteIsIdempotent(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestMigratePostgresCreatesHostConfig(t *testing.T) {
	dsn := postgresContainer(t)

	pool, err := OpenPostgres(context.Background(), PostgresOptions{
		DSN:          dsn,
		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// The host_config table should exist.
	var exists bool
	err = pool.Raw().QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'host_config')`,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("check host_config existence: %v", err)
	}
	if !exists {
		t.Error("host_config table missing")
	}
}
