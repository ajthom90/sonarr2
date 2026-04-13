package history

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
)

type postgresStore struct {
	q *pggen.Queries
}

// NewPostgresStore returns a Store backed by a Postgres pool.
func NewPostgresStore(pool *db.PostgresPool) Store {
	return &postgresStore{q: pggen.New(pool.Raw())}
}

func (s *postgresStore) Create(ctx context.Context, entry Entry) (Entry, error) {
	data := historyDataPostgres(entry.Data)
	row, err := s.q.CreateHistoryEntry(ctx, pggen.CreateHistoryEntryParams{
		EpisodeID:   entry.EpisodeID,
		SeriesID:    entry.SeriesID,
		SourceTitle: entry.SourceTitle,
		QualityName: entry.QualityName,
		EventType:   string(entry.EventType),
		DownloadID:  entry.DownloadID,
		Data:        data,
	})
	if err != nil {
		return Entry{}, fmt.Errorf("history: create: %w", err)
	}
	return historyFromPostgres(row)
}

func (s *postgresStore) ListForSeries(ctx context.Context, seriesID int64) ([]Entry, error) {
	rows, err := s.q.ListForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("history: list for series: %w", err)
	}
	return historySliceFromPostgres(rows)
}

func (s *postgresStore) ListForEpisode(ctx context.Context, episodeID int64) ([]Entry, error) {
	rows, err := s.q.ListForEpisode(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("history: list for episode: %w", err)
	}
	return historySliceFromPostgres(rows)
}

func (s *postgresStore) FindByDownloadID(ctx context.Context, downloadID string) ([]Entry, error) {
	rows, err := s.q.FindByDownloadID(ctx, downloadID)
	if err != nil {
		return nil, fmt.Errorf("history: find by download id: %w", err)
	}
	return historySliceFromPostgres(rows)
}

func (s *postgresStore) DeleteForSeries(ctx context.Context, seriesID int64) error {
	if err := s.q.DeleteForSeries(ctx, seriesID); err != nil {
		return fmt.Errorf("history: delete for series: %w", err)
	}
	return nil
}

// historyFromPostgres converts a sqlc-generated postgres.History row to Entry.
func historyFromPostgres(r pggen.History) (Entry, error) {
	data := json.RawMessage(r.Data)
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	return Entry{
		ID:          r.ID,
		EpisodeID:   r.EpisodeID,
		SeriesID:    r.SeriesID,
		SourceTitle: r.SourceTitle,
		QualityName: r.QualityName,
		EventType:   EventType(r.EventType),
		Date:        r.Date.Time,
		DownloadID:  r.DownloadID,
		Data:        data,
	}, nil
}

// historySliceFromPostgres converts a slice of postgres.History rows.
func historySliceFromPostgres(rows []pggen.History) ([]Entry, error) {
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		e, err := historyFromPostgres(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// historyDataPostgres ensures a nil/empty Data blob becomes {}.
func historyDataPostgres(d json.RawMessage) []byte {
	if len(d) == 0 {
		return []byte("{}")
	}
	return d
}
