package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

type sqliteStatsStore struct {
	pool   *db.SQLitePool
	helper recomputeHelper
}

func newSqliteStatsStore(pool *db.SQLitePool, episodes EpisodesStore, files EpisodeFilesStore) *sqliteStatsStore {
	return &sqliteStatsStore{
		pool:   pool,
		helper: recomputeHelper{episodes: episodes, files: files},
	}
}

func (s *sqliteStatsStore) Get(ctx context.Context, seriesID int64) (SeriesStatistics, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetSeriesStatistics(ctx, seriesID)
	if errors.Is(err, sql.ErrNoRows) {
		return SeriesStatistics{}, ErrNotFound
	}
	if err != nil {
		return SeriesStatistics{}, fmt.Errorf("library: get series stats: %w", err)
	}
	return SeriesStatistics{
		SeriesID:              row.SeriesID,
		EpisodeCount:          int32(row.EpisodeCount),
		EpisodeFileCount:      int32(row.EpisodeFileCount),
		MonitoredEpisodeCount: int32(row.MonitoredEpisodeCount),
		SizeOnDisk:            row.SizeOnDisk,
		UpdatedAt:             parseSqliteTime(row.UpdatedAt),
	}, nil
}

func (s *sqliteStatsStore) Recompute(ctx context.Context, seriesID int64) error {
	stats, err := s.helper.derive(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("library: recompute derive: %w", err)
	}
	err = s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpsertSeriesStatistics(ctx, sqlitegen.UpsertSeriesStatisticsParams{
			SeriesID:              stats.SeriesID,
			EpisodeCount:          int64(stats.EpisodeCount),
			EpisodeFileCount:      int64(stats.EpisodeFileCount),
			MonitoredEpisodeCount: int64(stats.MonitoredEpisodeCount),
			SizeOnDisk:            stats.SizeOnDisk,
		})
	})
	if err != nil {
		return fmt.Errorf("library: upsert series stats: %w", err)
	}
	return nil
}

func (s *sqliteStatsStore) Delete(ctx context.Context, seriesID int64) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteSeriesStatistics(ctx, seriesID)
	})
	if err != nil {
		return fmt.Errorf("library: delete series stats: %w", err)
	}
	return nil
}
