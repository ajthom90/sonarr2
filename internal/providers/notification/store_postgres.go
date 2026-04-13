package notification

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
	settingsJSON := notifMarshalSettings(inst.Settings)
	tagsJSON := notifMarshalTags(inst.Tags)
	row, err := s.q.CreateNotification(ctx, pggen.CreateNotificationParams{
		Name:           inst.Name,
		Implementation: inst.Implementation,
		Settings:       settingsJSON,
		OnGrab:         inst.OnGrab,
		OnDownload:     inst.OnDownload,
		OnHealthIssue:  inst.OnHealthIssue,
		Tags:           tagsJSON,
	})
	if err != nil {
		return Instance{}, fmt.Errorf("notification: create: %w", err)
	}
	return instanceFromPostgres(row)
}

func (s *postgresInstanceStore) GetByID(ctx context.Context, id int) (Instance, error) {
	row, err := s.q.GetNotificationByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Instance{}, ErrNotFound
	}
	if err != nil {
		return Instance{}, fmt.Errorf("notification: get by id: %w", err)
	}
	return instanceFromPostgres(row)
}

func (s *postgresInstanceStore) List(ctx context.Context) ([]Instance, error) {
	rows, err := s.q.ListNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("notification: list: %w", err)
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
	settingsJSON := notifMarshalSettings(inst.Settings)
	tagsJSON := notifMarshalTags(inst.Tags)
	if err := s.q.UpdateNotification(ctx, pggen.UpdateNotificationParams{
		ID:             int32(inst.ID),
		Name:           inst.Name,
		Implementation: inst.Implementation,
		Settings:       settingsJSON,
		OnGrab:         inst.OnGrab,
		OnDownload:     inst.OnDownload,
		OnHealthIssue:  inst.OnHealthIssue,
		Tags:           tagsJSON,
	}); err != nil {
		return fmt.Errorf("notification: update: %w", err)
	}
	return nil
}

func (s *postgresInstanceStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteNotification(ctx, int32(id)); err != nil {
		return fmt.Errorf("notification: delete: %w", err)
	}
	return nil
}

func instanceFromPostgres(r pggen.Notification) (Instance, error) {
	return Instance{
		ID:             int(r.ID),
		Name:           r.Name,
		Implementation: r.Implementation,
		Settings:       json.RawMessage(r.Settings),
		OnGrab:         r.OnGrab,
		OnDownload:     r.OnDownload,
		OnHealthIssue:  r.OnHealthIssue,
		Tags:           json.RawMessage(r.Tags),
		Added:          r.Added.Time,
	}, nil
}

// notifMarshalSettings ensures a nil/empty Settings blob becomes {}.
func notifMarshalSettings(s json.RawMessage) []byte {
	if len(s) == 0 {
		return []byte("{}")
	}
	return s
}

// notifMarshalTags ensures a nil/empty Tags blob becomes [].
func notifMarshalTags(t json.RawMessage) []byte {
	if len(t) == 0 {
		return []byte("[]")
	}
	return t
}
