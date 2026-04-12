package downloadclient

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
	settingsJSON := marshalDCSettings(inst.Settings)
	row, err := s.q.CreateDownloadClient(ctx, pggen.CreateDownloadClientParams{
		Name:                     inst.Name,
		Implementation:           inst.Implementation,
		Settings:                 settingsJSON,
		Enable:                   inst.Enable,
		Priority:                 int32(inst.Priority),
		RemoveCompletedDownloads: inst.RemoveCompletedDownloads,
		RemoveFailedDownloads:    inst.RemoveFailedDownloads,
	})
	if err != nil {
		return Instance{}, fmt.Errorf("downloadclient: create: %w", err)
	}
	return instanceFromPostgres(row)
}

func (s *postgresInstanceStore) GetByID(ctx context.Context, id int) (Instance, error) {
	row, err := s.q.GetDownloadClientByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("downloadclient: get by id: %w", err)
	}
	return instanceFromPostgres(row)
}

func (s *postgresInstanceStore) List(ctx context.Context) ([]Instance, error) {
	rows, err := s.q.ListDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("downloadclient: list: %w", err)
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
	settingsJSON := marshalDCSettings(inst.Settings)
	if err := s.q.UpdateDownloadClient(ctx, pggen.UpdateDownloadClientParams{
		ID:                       int32(inst.ID),
		Name:                     inst.Name,
		Implementation:           inst.Implementation,
		Settings:                 settingsJSON,
		Enable:                   inst.Enable,
		Priority:                 int32(inst.Priority),
		RemoveCompletedDownloads: inst.RemoveCompletedDownloads,
		RemoveFailedDownloads:    inst.RemoveFailedDownloads,
	}); err != nil {
		return fmt.Errorf("downloadclient: update: %w", err)
	}
	return nil
}

func (s *postgresInstanceStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteDownloadClient(ctx, int32(id)); err != nil {
		return fmt.Errorf("downloadclient: delete: %w", err)
	}
	return nil
}

func instanceFromPostgres(r pggen.DownloadClient) (Instance, error) {
	return Instance{
		ID:                       int(r.ID),
		Name:                     r.Name,
		Implementation:           r.Implementation,
		Settings:                 json.RawMessage(r.Settings),
		Enable:                   r.Enable,
		Priority:                 int(r.Priority),
		RemoveCompletedDownloads: r.RemoveCompletedDownloads,
		RemoveFailedDownloads:    r.RemoveFailedDownloads,
		Added:                    r.Added.Time,
	}, nil
}

// marshalDCSettings ensures a nil/empty Settings blob becomes {}.
func marshalDCSettings(s json.RawMessage) []byte {
	if len(s) == 0 {
		return []byte("{}")
	}
	return s
}
