package delayprofile

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
	tags, _ := json.Marshal(nonNilInt(p.Tags))
	if p.PreferredProtocol == "" {
		p.PreferredProtocol = ProtocolUsenet
	}
	row, err := s.q.CreateDelayProfile(ctx, pggen.CreateDelayProfileParams{
		EnableUsenet:                     p.EnableUsenet,
		EnableTorrent:                    p.EnableTorrent,
		PreferredProtocol:                string(p.PreferredProtocol),
		UsenetDelay:                      int32(p.UsenetDelay),
		TorrentDelay:                     int32(p.TorrentDelay),
		SortOrder:                        int32(p.Order),
		BypassIfHighestQuality:           p.BypassIfHighestQuality,
		BypassIfAboveCustomFormatScore:   p.BypassIfAboveCustomFormatScore,
		MinimumCustomFormatScore:         int32(p.MinimumCustomFormatScore),
		Tags:                             tags,
	})
	if err != nil {
		return Profile{}, fmt.Errorf("delayprofile: create: %w", err)
	}
	return fromPG(row)
}

func (s *postgresStore) GetByID(ctx context.Context, id int) (Profile, error) {
	row, err := s.q.GetDelayProfileByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("delayprofile: get: %w", err)
	}
	return fromPG(row)
}

func (s *postgresStore) List(ctx context.Context) ([]Profile, error) {
	rows, err := s.q.ListDelayProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("delayprofile: list: %w", err)
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
	tags, _ := json.Marshal(nonNilInt(p.Tags))
	if err := s.q.UpdateDelayProfile(ctx, pggen.UpdateDelayProfileParams{
		ID:                               int32(p.ID),
		EnableUsenet:                     p.EnableUsenet,
		EnableTorrent:                    p.EnableTorrent,
		PreferredProtocol:                string(p.PreferredProtocol),
		UsenetDelay:                      int32(p.UsenetDelay),
		TorrentDelay:                     int32(p.TorrentDelay),
		SortOrder:                        int32(p.Order),
		BypassIfHighestQuality:           p.BypassIfHighestQuality,
		BypassIfAboveCustomFormatScore:   p.BypassIfAboveCustomFormatScore,
		MinimumCustomFormatScore:         int32(p.MinimumCustomFormatScore),
		Tags:                             tags,
	}); err != nil {
		return fmt.Errorf("delayprofile: update: %w", err)
	}
	return nil
}

func (s *postgresStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteDelayProfile(ctx, int32(id)); err != nil {
		return fmt.Errorf("delayprofile: delete: %w", err)
	}
	return nil
}

func fromPG(r pggen.DelayProfile) (Profile, error) {
	var tags []int
	if err := json.Unmarshal(r.Tags, &tags); err != nil {
		return Profile{}, err
	}
	return Profile{
		ID:                              int(r.ID),
		EnableUsenet:                    r.EnableUsenet,
		EnableTorrent:                   r.EnableTorrent,
		PreferredProtocol:               Protocol(r.PreferredProtocol),
		UsenetDelay:                     int(r.UsenetDelay),
		TorrentDelay:                    int(r.TorrentDelay),
		Order:                           int(r.SortOrder),
		BypassIfHighestQuality:          r.BypassIfHighestQuality,
		BypassIfAboveCustomFormatScore:  r.BypassIfAboveCustomFormatScore,
		MinimumCustomFormatScore:        int(r.MinimumCustomFormatScore),
		Tags:                            nonNilInt(tags),
	}, nil
}
