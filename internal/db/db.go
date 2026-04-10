// Package db provides database connection pooling, migration running, and
// typed query code for sonarr2's Postgres and SQLite backends.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Dialect identifies which database backend is in use.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// ErrNoRows is returned by repository methods when a lookup finds nothing.
// Store implementations translate driver-specific no-row errors into this
// sentinel so callers can use errors.Is without importing driver packages.
var ErrNoRows = errors.New("db: no rows")

// ParseDialect normalizes a dialect string. Accepts any casing of
// "postgres" or "sqlite".
func ParseDialect(s string) (Dialect, error) {
	switch strings.ToLower(s) {
	case "postgres":
		return DialectPostgres, nil
	case "sqlite":
		return DialectSQLite, nil
	default:
		return "", fmt.Errorf("db: unknown dialect %q", s)
	}
}

// Pool is a high-level abstraction over the backend connection pool. It
// exposes the minimum surface the rest of the application needs: query
// execution via the dialect-specific generated code (Queries accessor),
// liveness checks, and graceful close. The concrete types are returned by
// OpenPostgres and OpenSQLite.
type Pool interface {
	// Dialect reports which backend this pool uses.
	Dialect() Dialect

	// Ping verifies the database is reachable. Returns nil if the
	// connection is healthy.
	Ping(ctx context.Context) error

	// Close releases all resources held by the pool. Must be called
	// during shutdown.
	Close() error
}
