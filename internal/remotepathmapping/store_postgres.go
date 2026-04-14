package remotepathmapping

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

type postgresStore struct{ q *pggen.Queries }

// NewPostgresStore returns a Store backed by Postgres.
func NewPostgresStore(pool *db.PostgresPool) Store {
	return &postgresStore{q: pggen.New(pool.Raw())}
}

func (s *postgresStore) Create(ctx context.Context, m Mapping) (Mapping, error) {
	row, err := s.q.CreateRemotePathMapping(ctx, pggen.CreateRemotePathMappingParams{
		Host:       m.Host,
		RemotePath: m.RemotePath,
		LocalPath:  m.LocalPath,
	})
	if err != nil {
		return Mapping{}, fmt.Errorf("remotepathmapping: create: %w", err)
	}
	return Mapping{ID: int(row.ID), Host: row.Host, RemotePath: row.RemotePath, LocalPath: row.LocalPath}, nil
}

func (s *postgresStore) GetByID(ctx context.Context, id int) (Mapping, error) {
	row, err := s.q.GetRemotePathMappingByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Mapping{}, ErrNotFound
	}
	if err != nil {
		return Mapping{}, fmt.Errorf("remotepathmapping: get: %w", err)
	}
	return Mapping{ID: int(row.ID), Host: row.Host, RemotePath: row.RemotePath, LocalPath: row.LocalPath}, nil
}

func (s *postgresStore) List(ctx context.Context) ([]Mapping, error) {
	rows, err := s.q.ListRemotePathMappings(ctx)
	if err != nil {
		return nil, fmt.Errorf("remotepathmapping: list: %w", err)
	}
	out := make([]Mapping, 0, len(rows))
	for _, r := range rows {
		out = append(out, Mapping{ID: int(r.ID), Host: r.Host, RemotePath: r.RemotePath, LocalPath: r.LocalPath})
	}
	return out, nil
}

func (s *postgresStore) Update(ctx context.Context, m Mapping) error {
	if err := s.q.UpdateRemotePathMapping(ctx, pggen.UpdateRemotePathMappingParams{
		ID:         int32(m.ID),
		Host:       m.Host,
		RemotePath: m.RemotePath,
		LocalPath:  m.LocalPath,
	}); err != nil {
		return fmt.Errorf("remotepathmapping: update: %w", err)
	}
	return nil
}

func (s *postgresStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteRemotePathMapping(ctx, int32(id)); err != nil {
		return fmt.Errorf("remotepathmapping: delete: %w", err)
	}
	return nil
}
