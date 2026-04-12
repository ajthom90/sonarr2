package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
	"github.com/ajthom90/sonarr2/internal/events"
)

type sqliteSeasonsStore struct {
	pool *db.SQLitePool
	bus  events.Bus
}

func newSqliteSeasonsStore(pool *db.SQLitePool, bus events.Bus) *sqliteSeasonsStore {
	return &sqliteSeasonsStore{pool: pool, bus: bus}
}

func (s *sqliteSeasonsStore) Upsert(ctx context.Context, in Season) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpsertSeason(ctx, sqlitegen.UpsertSeasonParams{
			SeriesID:     in.SeriesID,
			SeasonNumber: int64(in.SeasonNumber),
			Monitored:    boolToInt64(in.Monitored),
		})
	})
	if err != nil {
		return fmt.Errorf("library: upsert season: %w", err)
	}
	if err := s.bus.Publish(ctx, SeasonUpdated(in)); err != nil {
		return fmt.Errorf("library: publish SeasonUpdated: %w", err)
	}
	return nil
}

func (s *sqliteSeasonsStore) Get(ctx context.Context, seriesID int64, seasonNumber int32) (Season, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetSeason(ctx, sqlitegen.GetSeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Season{}, ErrNotFound
	}
	if err != nil {
		return Season{}, fmt.Errorf("library: get season: %w", err)
	}
	return Season{
		SeriesID:     row.SeriesID,
		SeasonNumber: int32(row.SeasonNumber),
		Monitored:    row.Monitored != 0,
	}, nil
}

func (s *sqliteSeasonsStore) ListForSeries(ctx context.Context, seriesID int64) ([]Season, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListSeasonsForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: list seasons: %w", err)
	}
	out := make([]Season, 0, len(rows))
	for _, r := range rows {
		out = append(out, Season{
			SeriesID:     r.SeriesID,
			SeasonNumber: int32(r.SeasonNumber),
			Monitored:    r.Monitored != 0,
		})
	}
	return out, nil
}

func (s *sqliteSeasonsStore) Delete(ctx context.Context, seriesID int64, seasonNumber int32) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteSeason(ctx, sqlitegen.DeleteSeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: int64(seasonNumber),
		})
	})
	if err != nil {
		return fmt.Errorf("library: delete season: %w", err)
	}
	return nil
}
