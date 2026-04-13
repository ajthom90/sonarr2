package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
	"github.com/ajthom90/sonarr2/internal/events"
)

type sqliteEpisodesStore struct {
	pool *db.SQLitePool
	bus  events.Bus
}

func newSqliteEpisodesStore(pool *db.SQLitePool, bus events.Bus) *sqliteEpisodesStore {
	return &sqliteEpisodesStore{pool: pool, bus: bus}
}

func (s *sqliteEpisodesStore) Create(ctx context.Context, in Episode) (Episode, error) {
	var out Episode
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateEpisode(ctx, sqlitegen.CreateEpisodeParams{
			SeriesID:              in.SeriesID,
			SeasonNumber:          int64(in.SeasonNumber),
			EpisodeNumber:         int64(in.EpisodeNumber),
			AbsoluteEpisodeNumber: nullInt64FromPtr32(in.AbsoluteEpisodeNumber),
			Title:                 in.Title,
			Overview:              in.Overview,
			AirDateUtc:            nullStringFromTimePtr(in.AirDateUtc),
			Monitored:             boolToInt64(in.Monitored),
			EpisodeFileID:         nullInt64FromPtr64(in.EpisodeFileID),
		})
		if err != nil {
			return fmt.Errorf("library: create episode: %w", err)
		}
		out = episodeFromSqlite(row)
		return nil
	})
	if err != nil {
		return Episode{}, err
	}
	if err := s.bus.Publish(ctx, EpisodeAdded{ID: out.ID, SeriesID: out.SeriesID}); err != nil {
		return out, fmt.Errorf("library: publish EpisodeAdded: %w", err)
	}
	return out, nil
}

func (s *sqliteEpisodesStore) Get(ctx context.Context, id int64) (Episode, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetEpisode(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Episode{}, ErrNotFound
	}
	if err != nil {
		return Episode{}, fmt.Errorf("library: get episode: %w", err)
	}
	return episodeFromSqlite(row), nil
}

func (s *sqliteEpisodesStore) ListForSeries(ctx context.Context, seriesID int64) ([]Episode, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: list episodes: %w", err)
	}
	out := make([]Episode, 0, len(rows))
	for _, r := range rows {
		out = append(out, episodeFromSqlite(r))
	}
	return out, nil
}

func (s *sqliteEpisodesStore) ListAll(ctx context.Context) ([]Episode, error) {
	const q = `SELECT id, series_id, season_number, episode_number, absolute_episode_number,
	                  title, overview, air_date_utc, monitored, episode_file_id,
	                  created_at, updated_at
	           FROM episodes ORDER BY air_date_utc ASC`
	rows, err := s.pool.RawReader().QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("library: list all episodes: %w", err)
	}
	defer rows.Close()

	var out []Episode
	for rows.Next() {
		var r sqlitegen.Episode
		if err := rows.Scan(
			&r.ID, &r.SeriesID, &r.SeasonNumber, &r.EpisodeNumber,
			&r.AbsoluteEpisodeNumber, &r.Title, &r.Overview, &r.AirDateUtc,
			&r.Monitored, &r.EpisodeFileID, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("library: list all episodes scan: %w", err)
		}
		out = append(out, episodeFromSqlite(r))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("library: list all episodes rows: %w", err)
	}
	return out, nil
}

func (s *sqliteEpisodesStore) Update(ctx context.Context, in Episode) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpdateEpisode(ctx, sqlitegen.UpdateEpisodeParams{
			ID:                    in.ID,
			AbsoluteEpisodeNumber: nullInt64FromPtr32(in.AbsoluteEpisodeNumber),
			Title:                 in.Title,
			Overview:              in.Overview,
			AirDateUtc:            nullStringFromTimePtr(in.AirDateUtc),
			Monitored:             boolToInt64(in.Monitored),
			EpisodeFileID:         nullInt64FromPtr64(in.EpisodeFileID),
		})
	})
	if err != nil {
		return fmt.Errorf("library: update episode: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeUpdated{ID: in.ID, SeriesID: in.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeUpdated: %w", err)
	}
	return nil
}

func (s *sqliteEpisodesStore) Delete(ctx context.Context, id int64) error {
	// Look up series_id first so we can publish the event.
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetEpisode(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("library: delete episode (lookup): %w", err)
	}
	err = s.pool.Write(ctx, func(exec db.Executor) error {
		wq := sqlitegen.New(&sqliteExec{exec: exec})
		return wq.DeleteEpisode(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("library: delete episode: %w", err)
	}
	if err := s.bus.Publish(ctx, EpisodeDeleted{ID: id, SeriesID: row.SeriesID}); err != nil {
		return fmt.Errorf("library: publish EpisodeDeleted: %w", err)
	}
	return nil
}

func (s *sqliteEpisodesStore) CountForSeries(ctx context.Context, seriesID int64) (int, int, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.CountEpisodesForSeries(ctx, seriesID)
	if err != nil {
		return 0, 0, fmt.Errorf("library: count episodes: %w", err)
	}
	// row.EpisodeCount is int64; row.MonitoredCount type varies by sqlc version.
	// Defensive type switch handles all known possibilities.
	total := int(row.EpisodeCount)
	monitored := 0
	switch v := any(row.MonitoredCount).(type) {
	case int64:
		monitored = int(v)
	case float64:
		monitored = int(v)
	case sql.NullInt64:
		if v.Valid {
			monitored = int(v.Int64)
		}
	case sql.NullFloat64:
		if v.Valid {
			monitored = int(v.Float64)
		}
	}
	return total, monitored, nil
}

func nullInt64FromPtr32(p *int32) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

func nullInt64FromPtr64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullStringFromTimePtr(p *time.Time) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: p.UTC().Format(sqliteTimeLayout), Valid: true}
}

func ptrFromNullInt32(v sql.NullInt64) *int32 {
	if !v.Valid {
		return nil
	}
	n := int32(v.Int64)
	return &n
}

func ptrFromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	n := v.Int64
	return &n
}

func ptrFromNullString(v sql.NullString) *time.Time {
	if !v.Valid {
		return nil
	}
	t, err := time.Parse(sqliteTimeLayout, v.String)
	if err != nil {
		return nil
	}
	return &t
}

func episodeFromSqlite(r sqlitegen.Episode) Episode {
	return Episode{
		ID:                    r.ID,
		SeriesID:              r.SeriesID,
		SeasonNumber:          int32(r.SeasonNumber),
		EpisodeNumber:         int32(r.EpisodeNumber),
		AbsoluteEpisodeNumber: ptrFromNullInt32(r.AbsoluteEpisodeNumber),
		Title:                 r.Title,
		Overview:              r.Overview,
		AirDateUtc:            ptrFromNullString(r.AirDateUtc),
		Monitored:             r.Monitored != 0,
		EpisodeFileID:         ptrFromNullInt64(r.EpisodeFileID),
		CreatedAt:             parseSqliteTime(r.CreatedAt),
		UpdatedAt:             parseSqliteTime(r.UpdatedAt),
	}
}
