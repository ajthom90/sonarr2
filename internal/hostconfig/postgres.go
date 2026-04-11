package hostconfig

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

// PostgresStore implements Store against a Postgres database using the
// sqlc-generated queries in internal/db/gen/postgres.
type PostgresStore struct {
	q *pggen.Queries
}

// NewPostgresStore returns a Store backed by the given Postgres pool.
func NewPostgresStore(pool *db.PostgresPool) *PostgresStore {
	return &PostgresStore{q: pggen.New(pool.Raw())}
}

// Get implements Store.
func (s *PostgresStore) Get(ctx context.Context) (HostConfig, error) {
	row, err := s.q.GetHostConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return HostConfig{}, ErrNotFound
	}
	if err != nil {
		return HostConfig{}, fmt.Errorf("hostconfig: postgres get: %w", err)
	}
	return HostConfig{
		APIKey:         row.ApiKey,
		AuthMode:       row.AuthMode,
		MigrationState: row.MigrationState,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}, nil
}

// Upsert implements Store.
func (s *PostgresStore) Upsert(ctx context.Context, hc HostConfig) error {
	if err := s.q.UpsertHostConfig(ctx, pggen.UpsertHostConfigParams{
		ApiKey:         hc.APIKey,
		AuthMode:       hc.AuthMode,
		MigrationState: hc.MigrationState,
	}); err != nil {
		return fmt.Errorf("hostconfig: postgres upsert: %w", err)
	}
	return nil
}
