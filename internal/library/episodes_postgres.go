package library

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type postgresEpisodesStore struct {
	q    *pggen.Queries
	pool *db.PostgresPool
	bus  events.Bus
}

func newPostgresEpisodesStore(pool *db.PostgresPool, bus events.Bus) *postgresEpisodesStore {
	return &postgresEpisodesStore{q: pggen.New(pool.Raw()), pool: pool, bus: bus}
}

func (s *postgresEpisodesStore) Create(ctx context.Context, in Episode) (Episode, error) {
	row, err := s.q.CreateEpisode(ctx, pggen.CreateEpisodeParams{
		SeriesID:              in.SeriesID,
		SeasonNumber:          in.SeasonNumber,
		EpisodeNumber:         in.EpisodeNumber,
		AbsoluteEpisodeNumber: pgInt4FromPtr(in.AbsoluteEpisodeNumber),
		Title:                 in.Title,
		Overview:              in.Overview,
		AirDateUtc:            pgTimestamptzFromPtr(in.AirDateUtc),
		Monitored:             in.Monitored,
		EpisodeFileID:         pgInt8FromPtr(in.EpisodeFileID),
	})
	if err != nil {
		return Episode{}, fmt.Errorf("library: create episode: %w", err)
	}
	out := episodeFromPostgres(row)
	if err := s.bus.Publish(ctx, EpisodeAdded{ID: out.ID, SeriesID: out.SeriesID}); err != nil {
		return out, fmt.Errorf("library: publish EpisodeAdded: %w", err)
	}
	return out, nil
}

func (s *postgresEpisodesStore) Get(ctx context.Context, id int64) (Episode, error) {
	row, err := s.q.GetEpisode(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Episode{}, ErrNotFound
	}
	if err != nil {
		return Episode{}, fmt.Errorf("library: get episode: %w", err)
	}
	return episodeFromPostgres(row), nil
}

func (s *postgresEpisodesStore) ListForSeries(ctx context.Context, seriesID int64) ([]Episode, error) {
	rows, err := s.q.ListEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: list episodes: %w", err)
	}
	out := make([]Episode, 0, len(rows))
	for _, r := range rows {
		out = append(out, episodeFromPostgres(r))
	}
	return out, nil
}

func (s *postgresEpisodesStore) ListAll(ctx context.Context) ([]Episode, error) {
	const q = `SELECT id, series_id, season_number, episode_number, absolute_episode_number,
	                  title, overview, air_date_utc, monitored, episode_file_id,
	                  created_at, updated_at
	           FROM episodes ORDER BY air_date_utc ASC`
	rows, err := s.pool.Raw().Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("library: list all episodes: %w", err)
	}
	defer rows.Close()

	var out []Episode
	for rows.Next() {
		var r pggen.Episode
		if err := rows.Scan(
			&r.ID, &r.SeriesID, &r.SeasonNumber, &r.EpisodeNumber,
			&r.AbsoluteEpisodeNumber, &r.Title, &r.Overview, &r.AirDateUtc,
			&r.Monitored, &r.EpisodeFileID, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("library: list all episodes scan: %w", err)
		}
		out = append(out, episodeFromPostgres(r))
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("library: list all episodes rows: %w", rows.Err())
	}
	return out, nil
}

func (s *postgresEpisodesStore) Update(ctx context.Context, in Episode) error {
	if err := s.q.UpdateEpisode(ctx, pggen.UpdateEpisodeParams{
		ID:                    in.ID,
		AbsoluteEpisodeNumber: pgInt4FromPtr(in.AbsoluteEpisodeNumber),
		Title:                 in.Title,
		Overview:              in.Overview,
		AirDateUtc:            pgTimestamptzFromPtr(in.AirDateUtc),
		Monitored:             in.Monitored,
		EpisodeFileID:         pgInt8FromPtr(in.EpisodeFileID),
	}); err != nil {
		return fmt.Errorf("library: update episode: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeUpdated{ID: in.ID, SeriesID: in.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeUpdated: %w", err)
	}
	return nil
}

func (s *postgresEpisodesStore) SetMonitored(ctx context.Context, episodeID int64, monitored bool) error {
	// Look up series_id first so we can publish EpisodeUpdated with it.
	row, err := s.q.GetEpisode(ctx, episodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("library: set monitored (lookup): %w", err)
	}
	if err := s.q.SetEpisodeMonitored(ctx, pggen.SetEpisodeMonitoredParams{
		Monitored: monitored,
		ID:        episodeID,
	}); err != nil {
		return fmt.Errorf("library: set monitored: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeUpdated{ID: episodeID, SeriesID: row.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeUpdated: %w", err)
	}
	return nil
}

func (s *postgresEpisodesStore) Delete(ctx context.Context, id int64) error {
	// Load the series_id before deleting so we can publish the event with it.
	row, err := s.q.GetEpisode(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("library: delete episode (lookup): %w", err)
	}
	if err := s.q.DeleteEpisode(ctx, id); err != nil {
		return fmt.Errorf("library: delete episode: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeDeleted{ID: id, SeriesID: row.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeDeleted: %w", err)
	}
	return nil
}

func (s *postgresEpisodesStore) CountForSeries(ctx context.Context, seriesID int64) (int, int, error) {
	row, err := s.q.CountEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return 0, 0, fmt.Errorf("library: count episodes: %w", err)
	}
	return int(row.EpisodeCount), int(row.MonitoredCount), nil
}

// Helpers to convert between Go pointer types and pgx nullable types.

func pgInt4FromPtr(p *int32) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *p, Valid: true}
}

func pgInt8FromPtr(p *int64) pgtype.Int8 {
	if p == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *p, Valid: true}
}

func pgTimestamptzFromPtr(p *time.Time) pgtype.Timestamptz {
	if p == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *p, Valid: true}
}

func ptrFromPgInt4(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	n := v.Int32
	return &n
}

func ptrFromPgInt8(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func ptrFromPgTimestamptz(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func episodeFromPostgres(r pggen.Episode) Episode {
	return Episode{
		ID:                    r.ID,
		SeriesID:              r.SeriesID,
		SeasonNumber:          r.SeasonNumber,
		EpisodeNumber:         r.EpisodeNumber,
		AbsoluteEpisodeNumber: ptrFromPgInt4(r.AbsoluteEpisodeNumber),
		Title:                 r.Title,
		Overview:              r.Overview,
		AirDateUtc:            ptrFromPgTimestamptz(r.AirDateUtc),
		Monitored:             r.Monitored,
		EpisodeFileID:         ptrFromPgInt8(r.EpisodeFileID),
		CreatedAt:             r.CreatedAt.Time,
		UpdatedAt:             r.UpdatedAt.Time,
	}
}
