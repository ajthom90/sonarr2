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

type sqliteEpisodeFilesStore struct {
	pool *db.SQLitePool
	bus  events.Bus
}

func newSqliteEpisodeFilesStore(pool *db.SQLitePool, bus events.Bus) *sqliteEpisodeFilesStore {
	return &sqliteEpisodeFilesStore{pool: pool, bus: bus}
}

func (s *sqliteEpisodeFilesStore) Create(ctx context.Context, in EpisodeFile) (EpisodeFile, error) {
	var out EpisodeFile
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateEpisodeFile(ctx, sqlitegen.CreateEpisodeFileParams{
			SeriesID:     in.SeriesID,
			SeasonNumber: int64(in.SeasonNumber),
			RelativePath: in.RelativePath,
			Size:         in.Size,
			ReleaseGroup: in.ReleaseGroup,
			QualityName:  in.QualityName,
		})
		if err != nil {
			return fmt.Errorf("library: create episode file: %w", err)
		}
		out = episodeFileFromSqlite(row)
		return nil
	})
	if err != nil {
		return EpisodeFile{}, err
	}
	if err := s.bus.Publish(ctx, EpisodeFileAdded{
		ID: out.ID, SeriesID: out.SeriesID, Size: out.Size,
	}); err != nil {
		return out, fmt.Errorf("library: publish EpisodeFileAdded: %w", err)
	}
	return out, nil
}

func (s *sqliteEpisodeFilesStore) Get(ctx context.Context, id int64) (EpisodeFile, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetEpisodeFile(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return EpisodeFile{}, ErrNotFound
	}
	if err != nil {
		return EpisodeFile{}, fmt.Errorf("library: get episode file: %w", err)
	}
	return episodeFileFromSqlite(row), nil
}

func (s *sqliteEpisodeFilesStore) ListForSeries(ctx context.Context, seriesID int64) ([]EpisodeFile, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListEpisodeFilesForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: list episode files: %w", err)
	}
	out := make([]EpisodeFile, 0, len(rows))
	for _, r := range rows {
		out = append(out, episodeFileFromSqlite(r))
	}
	return out, nil
}

func (s *sqliteEpisodeFilesStore) Delete(ctx context.Context, id int64) error {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetEpisodeFile(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("library: delete episode file (lookup): %w", err)
	}
	err = s.pool.Write(ctx, func(exec db.Executor) error {
		wq := sqlitegen.New(&sqliteExec{exec: exec})
		return wq.DeleteEpisodeFile(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("library: delete episode file: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeFileDeleted{ID: id, SeriesID: row.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeFileDeleted: %w", err)
	}
	return nil
}

func (s *sqliteEpisodeFilesStore) SumSizesForSeries(ctx context.Context, seriesID int64) (int, int64, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.SumEpisodeFileSizesForSeries(ctx, seriesID)
	if err != nil {
		return 0, 0, fmt.Errorf("library: sum episode file sizes: %w", err)
	}
	// sqlc's handling of COALESCE(SUM(...), 0) in SQLite produces interface{} —
	// unwrap via the same defensive switch as CountEpisodesForSeries.
	count := int(row.FileCount)
	var size int64
	switch v := row.SizeOnDisk.(type) {
	case int64:
		size = v
	case float64:
		size = int64(v)
	case sql.NullInt64:
		if v.Valid {
			size = v.Int64
		}
	case sql.NullFloat64:
		if v.Valid {
			size = int64(v.Float64)
		}
	}
	return count, size, nil
}

func episodeFileFromSqlite(r sqlitegen.EpisodeFile) EpisodeFile {
	return EpisodeFile{
		ID:           r.ID,
		SeriesID:     r.SeriesID,
		SeasonNumber: int32(r.SeasonNumber),
		RelativePath: r.RelativePath,
		Size:         r.Size,
		DateAdded:    parseSqliteTime(r.DateAdded),
		ReleaseGroup: r.ReleaseGroup,
		QualityName:  r.QualityName,
		CreatedAt:    parseSqliteTime(r.CreatedAt),
		UpdatedAt:    parseSqliteTime(r.UpdatedAt),
	}
}
