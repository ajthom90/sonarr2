package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

type postgresInstanceStore struct {
	q *pggen.Queries
}

// NewPostgresInstanceStore returns an InstanceStore backed by a Postgres pool.
func NewPostgresInstanceStore(pool *db.PostgresPool) InstanceStore {
	return &postgresInstanceStore{q: pggen.New(pool.Raw())}
}

func (s *postgresInstanceStore) Create(ctx context.Context, inst Instance) (Instance, error) {
	settingsJSON, err := marshalSettings(inst.Settings)
	if err != nil {
		return Instance{}, fmt.Errorf("indexer: marshal settings: %w", err)
	}
	row, err := s.q.CreateIndexer(ctx, pggen.CreateIndexerParams{
		Name:                    inst.Name,
		Implementation:          inst.Implementation,
		Settings:                settingsJSON,
		EnableRss:               inst.EnableRss,
		EnableAutomaticSearch:   inst.EnableAutomaticSearch,
		EnableInteractiveSearch: inst.EnableInteractiveSearch,
		Priority:                int32(inst.Priority),
	})
	if err != nil {
		return Instance{}, fmt.Errorf("indexer: create: %w", err)
	}
	return instanceFromPostgres(row)
}

func (s *postgresInstanceStore) GetByID(ctx context.Context, id int) (Instance, error) {
	row, err := s.q.GetIndexerByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("indexer: get by id: %w", err)
	}
	return instanceFromPostgres(row)
}

func (s *postgresInstanceStore) List(ctx context.Context) ([]Instance, error) {
	rows, err := s.q.ListIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("indexer: list: %w", err)
	}
	out := make([]Instance, 0, len(rows))
	for _, r := range rows {
		inst, err := instanceFromPostgres(r)
		if err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, nil
}

func (s *postgresInstanceStore) Update(ctx context.Context, inst Instance) error {
	settingsJSON, err := marshalSettings(inst.Settings)
	if err != nil {
		return fmt.Errorf("indexer: marshal settings: %w", err)
	}
	if err := s.q.UpdateIndexer(ctx, pggen.UpdateIndexerParams{
		ID:                      int32(inst.ID),
		Name:                    inst.Name,
		Implementation:          inst.Implementation,
		Settings:                settingsJSON,
		EnableRss:               inst.EnableRss,
		EnableAutomaticSearch:   inst.EnableAutomaticSearch,
		EnableInteractiveSearch: inst.EnableInteractiveSearch,
		Priority:                int32(inst.Priority),
	}); err != nil {
		return fmt.Errorf("indexer: update: %w", err)
	}
	return nil
}

func (s *postgresInstanceStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteIndexer(ctx, int32(id)); err != nil {
		return fmt.Errorf("indexer: delete: %w", err)
	}
	return nil
}

func instanceFromPostgres(r pggen.Indexer) (Instance, error) {
	return Instance{
		ID:                      int(r.ID),
		Name:                    r.Name,
		Implementation:          r.Implementation,
		Settings:                json.RawMessage(r.Settings),
		EnableRss:               r.EnableRss,
		EnableAutomaticSearch:   r.EnableAutomaticSearch,
		EnableInteractiveSearch: r.EnableInteractiveSearch,
		Priority:                int(r.Priority),
		Added:                   r.Added.Time,
	}, nil
}

// marshalSettings ensures a nil/empty Settings blob becomes {}.
func marshalSettings(s json.RawMessage) ([]byte, error) {
	if len(s) == 0 {
		return []byte("{}"), nil
	}
	return s, nil
}
