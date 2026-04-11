package db

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/config"
)

func TestOpenFromConfigSQLite(t *testing.T) {
	pool, err := OpenFromConfig(context.Background(), config.DBConfig{
		Dialect:     "sqlite",
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenFromConfig: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if pool.Dialect() != DialectSQLite {
		t.Errorf("Dialect() = %q, want sqlite", pool.Dialect())
	}
}

func TestOpenFromConfigUnknownDialect(t *testing.T) {
	_, err := OpenFromConfig(context.Background(), config.DBConfig{
		Dialect: "mysql",
		DSN:     "mysql://",
	})
	if err == nil {
		t.Fatal("expected error for unknown dialect")
	}
}
