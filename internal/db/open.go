package db

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/config"
)

// OpenFromConfig returns a Pool for the configured dialect. It is the
// single entry point used by app.New — callers do not need to know which
// concrete type they get back.
func OpenFromConfig(ctx context.Context, cfg config.DBConfig) (Pool, error) {
	dialect, err := ParseDialect(cfg.Dialect)
	if err != nil {
		return nil, err
	}

	switch dialect {
	case DialectPostgres:
		return OpenPostgres(ctx, PostgresOptions{
			DSN:             cfg.DSN,
			MaxOpenConns:    cfg.MaxOpenConns,
			MinOpenConns:    cfg.MaxIdleConns, // reuse as min
			ConnMaxLifetime: cfg.ConnMaxLifetime,
		})
	case DialectSQLite:
		return OpenSQLite(ctx, SQLiteOptions{
			DSN:         cfg.DSN,
			BusyTimeout: cfg.BusyTimeout,
		})
	default:
		return nil, fmt.Errorf("db: unsupported dialect %q", dialect)
	}
}
