package releaseprofile

import (
	"context"
	"encoding/json"
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

func (s *postgresStore) Create(ctx context.Context, p Profile) (Profile, error) {
	req, _ := json.Marshal(nonNilStr(p.Required))
	ign, _ := json.Marshal(nonNilStr(p.Ignored))
	tags, _ := json.Marshal(nonNilInt(p.Tags))
	row, err := s.q.CreateReleaseProfile(ctx, pggen.CreateReleaseProfileParams{
		Name:      p.Name,
		Enabled:   p.Enabled,
		Required:  req,
		Ignored:   ign,
		IndexerID: int32(p.IndexerID),
		Tags:      tags,
	})
	if err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: create: %w", err)
	}
	return fromPG(row)
}

func (s *postgresStore) GetByID(ctx context.Context, id int) (Profile, error) {
	row, err := s.q.GetReleaseProfileByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: get: %w", err)
	}
	return fromPG(row)
}

func (s *postgresStore) List(ctx context.Context) ([]Profile, error) {
	rows, err := s.q.ListReleaseProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("releaseprofile: list: %w", err)
	}
	out := make([]Profile, 0, len(rows))
	for _, r := range rows {
		p, err := fromPG(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *postgresStore) Update(ctx context.Context, p Profile) error {
	req, _ := json.Marshal(nonNilStr(p.Required))
	ign, _ := json.Marshal(nonNilStr(p.Ignored))
	tags, _ := json.Marshal(nonNilInt(p.Tags))
	if err := s.q.UpdateReleaseProfile(ctx, pggen.UpdateReleaseProfileParams{
		ID:        int32(p.ID),
		Name:      p.Name,
		Enabled:   p.Enabled,
		Required:  req,
		Ignored:   ign,
		IndexerID: int32(p.IndexerID),
		Tags:      tags,
	}); err != nil {
		return fmt.Errorf("releaseprofile: update: %w", err)
	}
	return nil
}

func (s *postgresStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteReleaseProfile(ctx, int32(id)); err != nil {
		return fmt.Errorf("releaseprofile: delete: %w", err)
	}
	return nil
}

func fromPG(r pggen.ReleaseProfile) (Profile, error) {
	var req, ign []string
	var tags []int
	if err := json.Unmarshal(r.Required, &req); err != nil {
		return Profile{}, err
	}
	if err := json.Unmarshal(r.Ignored, &ign); err != nil {
		return Profile{}, err
	}
	if err := json.Unmarshal(r.Tags, &tags); err != nil {
		return Profile{}, err
	}
	return Profile{
		ID:        int(r.ID),
		Name:      r.Name,
		Enabled:   r.Enabled,
		Required:  nonNilStr(req),
		Ignored:   nonNilStr(ign),
		IndexerID: int(r.IndexerID),
		Tags:      nonNilInt(tags),
	}, nil
}
