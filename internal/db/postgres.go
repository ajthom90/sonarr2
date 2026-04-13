package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresOptions controls the Postgres connection pool.
type PostgresOptions struct {
	DSN             string
	MaxOpenConns    int
	MinOpenConns    int
	ConnMaxLifetime time.Duration
}

// PostgresPool is the Postgres implementation of Pool. It embeds a
// *pgxpool.Pool and exposes it via Raw for use by sqlc-generated code.
type PostgresPool struct {
	pool *pgxpool.Pool
}

// Raw returns the underlying pgxpool.Pool so sqlc-generated code can use it
// as a DBTX. Callers should NOT retain this pointer past the owning Pool's
// lifetime.
func (p *PostgresPool) Raw() *pgxpool.Pool {
	return p.pool
}

// Dialect implements Pool.
func (p *PostgresPool) Dialect() Dialect { return DialectPostgres }

// Ping implements Pool.
func (p *PostgresPool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close implements Pool.
func (p *PostgresPool) Close() error {
	p.pool.Close()
	return nil
}

// Vacuum is a no-op for Postgres — autovacuum handles compaction.
func (p *PostgresPool) Vacuum(_ context.Context) error {
	return nil
}

// OpenPostgres parses opts.DSN, applies pool sizing, and returns a
// connected *PostgresPool. It returns an error if the DSN is invalid or the
// initial Ping fails.
func OpenPostgres(ctx context.Context, opts PostgresOptions) (*PostgresPool, error) {
	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse postgres DSN: %w", err)
	}
	if opts.MaxOpenConns > 0 {
		cfg.MaxConns = int32(opts.MaxOpenConns)
	}
	if opts.MinOpenConns > 0 {
		cfg.MinConns = int32(opts.MinOpenConns)
	}
	if opts.ConnMaxLifetime > 0 {
		cfg.MaxConnLifetime = opts.ConnMaxLifetime
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open postgres pool: %w", err)
	}

	// Verify the connection is actually usable.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: postgres ping: %w", err)
	}

	return &PostgresPool{pool: pool}, nil
}
