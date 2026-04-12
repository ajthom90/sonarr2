package library

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

type postgresStatsStore struct {
	q      *pggen.Queries
	helper recomputeHelper
}

func newPostgresStatsStore(pool *db.PostgresPool, episodes EpisodesStore, files EpisodeFilesStore) *postgresStatsStore {
	return &postgresStatsStore{
		q:      pggen.New(pool.Raw()),
		helper: recomputeHelper{episodes: episodes, files: files},
	}
}

func (s *postgresStatsStore) Get(ctx context.Context, seriesID int64) (SeriesStatistics, error) {
	row, err := s.q.GetSeriesStatistics(ctx, seriesID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SeriesStatistics{}, ErrNotFound
	}
	if err != nil {
		return SeriesStatistics{}, fmt.Errorf("library: get series stats: %w", err)
	}
	return SeriesStatistics{
		SeriesID:              row.SeriesID,
		EpisodeCount:          row.EpisodeCount,
		EpisodeFileCount:      row.EpisodeFileCount,
		MonitoredEpisodeCount: row.MonitoredEpisodeCount,
		SizeOnDisk:            row.SizeOnDisk,
		UpdatedAt:             row.UpdatedAt.Time,
	}, nil
}

func (s *postgresStatsStore) Recompute(ctx context.Context, seriesID int64) error {
	stats, err := s.helper.derive(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("library: recompute derive: %w", err)
	}
	if err := s.q.UpsertSeriesStatistics(ctx, pggen.UpsertSeriesStatisticsParams{
		SeriesID:              stats.SeriesID,
		EpisodeCount:          stats.EpisodeCount,
		EpisodeFileCount:      stats.EpisodeFileCount,
		MonitoredEpisodeCount: stats.MonitoredEpisodeCount,
		SizeOnDisk:            stats.SizeOnDisk,
	}); err != nil {
		return fmt.Errorf("library: upsert series stats: %w", err)
	}
	return nil
}

func (s *postgresStatsStore) Delete(ctx context.Context, seriesID int64) error {
	if err := s.q.DeleteSeriesStatistics(ctx, seriesID); err != nil {
		return fmt.Errorf("library: delete series stats: %w", err)
	}
	return nil
}
