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

type postgresEpisodeFilesStore struct {
	q   *pggen.Queries
	bus events.Bus
}

func newPostgresEpisodeFilesStore(pool *db.PostgresPool, bus events.Bus) *postgresEpisodeFilesStore {
	return &postgresEpisodeFilesStore{q: pggen.New(pool.Raw()), bus: bus}
}

func (s *postgresEpisodeFilesStore) Create(ctx context.Context, in EpisodeFile) (EpisodeFile, error) {
	row, err := s.q.CreateEpisodeFile(ctx, pggen.CreateEpisodeFileParams{
		SeriesID:     in.SeriesID,
		SeasonNumber: in.SeasonNumber,
		RelativePath: in.RelativePath,
		Size:         in.Size,
		ReleaseGroup: in.ReleaseGroup,
		QualityName:  in.QualityName,
	})
	if err != nil {
		return EpisodeFile{}, fmt.Errorf("library: create episode file: %w", err)
	}
	out := episodeFileFromPostgres(row)
	if err := s.bus.Publish(ctx, EpisodeFileAdded{
		ID: out.ID, SeriesID: out.SeriesID, Size: out.Size,
	}); err != nil {
		return out, fmt.Errorf("library: publish EpisodeFileAdded: %w", err)
	}
	return out, nil
}

func (s *postgresEpisodeFilesStore) Get(ctx context.Context, id int64) (EpisodeFile, error) {
	row, err := s.q.GetEpisodeFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return EpisodeFile{}, ErrNotFound
	}
	if err != nil {
		return EpisodeFile{}, fmt.Errorf("library: get episode file: %w", err)
	}
	return episodeFileFromPostgres(row), nil
}

func (s *postgresEpisodeFilesStore) ListForSeries(ctx context.Context, seriesID int64) ([]EpisodeFile, error) {
	rows, err := s.q.ListEpisodeFilesForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: list episode files: %w", err)
	}
	out := make([]EpisodeFile, 0, len(rows))
	for _, r := range rows {
		out = append(out, episodeFileFromPostgres(r))
	}
	return out, nil
}

func (s *postgresEpisodeFilesStore) Delete(ctx context.Context, id int64) error {
	// Load the row to get series_id before deleting.
	row, err := s.q.GetEpisodeFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("library: delete episode file (lookup): %w", err)
	}
	if err := s.q.DeleteEpisodeFile(ctx, id); err != nil {
		return fmt.Errorf("library: delete episode file: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeFileDeleted{ID: id, SeriesID: row.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeFileDeleted: %w", err)
	}
	return nil
}

func (s *postgresEpisodeFilesStore) SumSizesForSeries(ctx context.Context, seriesID int64) (int, int64, error) {
	row, err := s.q.SumEpisodeFileSizesForSeries(ctx, seriesID)
	if err != nil {
		return 0, 0, fmt.Errorf("library: sum episode file sizes: %w", err)
	}
	return int(row.FileCount), row.SizeOnDisk, nil
}

func episodeFileFromPostgres(r pggen.EpisodeFile) EpisodeFile {
	return EpisodeFile{
		ID:           r.ID,
		SeriesID:     r.SeriesID,
		SeasonNumber: r.SeasonNumber,
		RelativePath: r.RelativePath,
		Size:         r.Size,
		DateAdded:    r.DateAdded.Time,
		ReleaseGroup: r.ReleaseGroup,
		QualityName:  r.QualityName,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}
