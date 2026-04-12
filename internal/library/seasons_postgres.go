package library

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/jackc/pgx/v5"
)

type postgresSeasonsStore struct {
	q   *pggen.Queries
	bus events.Bus
}

func newPostgresSeasonsStore(pool *db.PostgresPool, bus events.Bus) *postgresSeasonsStore {
	return &postgresSeasonsStore{q: pggen.New(pool.Raw()), bus: bus}
}

func (s *postgresSeasonsStore) Upsert(ctx context.Context, in Season) error {
	if err := s.q.UpsertSeason(ctx, pggen.UpsertSeasonParams{
		SeriesID:     in.SeriesID,
		SeasonNumber: in.SeasonNumber,
		Monitored:    in.Monitored,
	}); err != nil {
		return fmt.Errorf("library: upsert season: %w", err)
	}
	if err := s.bus.Publish(ctx, SeasonUpdated{
		SeriesID:     in.SeriesID,
		SeasonNumber: in.SeasonNumber,
		Monitored:    in.Monitored,
	}); err != nil {
		return fmt.Errorf("library: publish SeasonUpdated: %w", err)
	}
	return nil
}

func (s *postgresSeasonsStore) Get(ctx context.Context, seriesID int64, seasonNumber int32) (Season, error) {
	row, err := s.q.GetSeason(ctx, pggen.GetSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Season{}, ErrNotFound
	}
	if err != nil {
		return Season{}, fmt.Errorf("library: get season: %w", err)
	}
	return Season{
		SeriesID:     row.SeriesID,
		SeasonNumber: row.SeasonNumber,
		Monitored:    row.Monitored,
	}, nil
}

func (s *postgresSeasonsStore) ListForSeries(ctx context.Context, seriesID int64) ([]Season, error) {
	rows, err := s.q.ListSeasonsForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: list seasons: %w", err)
	}
	out := make([]Season, 0, len(rows))
	for _, r := range rows {
		out = append(out, Season{
			SeriesID:     r.SeriesID,
			SeasonNumber: r.SeasonNumber,
			Monitored:    r.Monitored,
		})
	}
	return out, nil
}

func (s *postgresSeasonsStore) Delete(ctx context.Context, seriesID int64, seasonNumber int32) error {
	if err := s.q.DeleteSeason(ctx, pggen.DeleteSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	}); err != nil {
		return fmt.Errorf("library: delete season: %w", err)
	}
	return nil
}
