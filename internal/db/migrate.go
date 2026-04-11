package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/postgres/*.sql migrations/sqlite/*.sql
var migrationsFS embed.FS

// Migrate runs all pending migrations for the given pool's dialect. It is
// safe to call repeatedly; already-applied migrations are skipped.
func Migrate(ctx context.Context, pool Pool) error {
	switch p := pool.(type) {
	case *PostgresPool:
		return migratePostgres(ctx, p)
	case *SQLitePool:
		return migrateSQLite(ctx, p)
	default:
		return fmt.Errorf("db: unsupported pool type %T", pool)
	}
}

func migratePostgres(ctx context.Context, p *PostgresPool) error {
	// goose operates on *sql.DB, not pgxpool. Wrap via pgx's stdlib.
	conn := stdlib.OpenDBFromPool(p.pool)
	defer conn.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose set dialect: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations/postgres")
	if err != nil {
		return fmt.Errorf("db: subfs postgres migrations: %w", err)
	}
	goose.SetBaseFS(sub)
	defer goose.SetBaseFS(nil)

	if err := goose.UpContext(ctx, conn, "."); err != nil {
		return fmt.Errorf("db: postgres migrate up: %w", err)
	}
	return nil
}

func migrateSQLite(ctx context.Context, p *SQLitePool) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("db: goose set dialect: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations/sqlite")
	if err != nil {
		return fmt.Errorf("db: subfs sqlite migrations: %w", err)
	}
	goose.SetBaseFS(sub)
	defer goose.SetBaseFS(nil)

	if err := goose.UpContext(ctx, p.writer, "."); err != nil {
		return fmt.Errorf("db: sqlite migrate up: %w", err)
	}
	return nil
}
