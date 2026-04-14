package blocklist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type postgresStore struct {
	q *pggen.Queries
}

// NewPostgresStore returns a Store backed by Postgres.
func NewPostgresStore(pool *db.PostgresPool) Store {
	return &postgresStore{q: pggen.New(pool.Raw())}
}

func (s *postgresStore) Create(ctx context.Context, e Entry) (Entry, error) {
	epIDs, err := json.Marshal(intSlice(e.EpisodeIDs))
	if err != nil {
		return Entry{}, fmt.Errorf("blocklist: marshal episodeIds: %w", err)
	}
	row, err := s.q.CreateBlocklist(ctx, pggen.CreateBlocklistParams{
		SeriesID:        int32(e.SeriesID),
		EpisodeIds:      epIDs,
		SourceTitle:     e.SourceTitle,
		Quality:         jsonBytes(e.Quality, "{}"),
		Languages:       jsonBytes(e.Languages, "[]"),
		Date:            pgtype.Timestamptz{Time: e.Date, Valid: true},
		PublishedDate:   pgTimeFromPtr(e.PublishedDate),
		Size:            pgInt8FromPtr(e.Size),
		Protocol:        string(e.Protocol),
		Indexer:         e.Indexer,
		IndexerFlags:    int32(e.IndexerFlags),
		ReleaseType:     e.ReleaseType,
		Message:         e.Message,
		TorrentInfoHash: pgTextFromStr(e.TorrentInfoHash),
	})
	if err != nil {
		return Entry{}, fmt.Errorf("blocklist: create: %w", err)
	}
	return entryFromPG(row)
}

func (s *postgresStore) GetByID(ctx context.Context, id int) (Entry, error) {
	row, err := s.q.GetBlocklistByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("blocklist: get by id: %w", err)
	}
	return entryFromPG(row)
}

func (s *postgresStore) List(ctx context.Context, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	rows, err := s.q.ListBlocklist(ctx, pggen.ListBlocklistParams{
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
	})
	if err != nil {
		return Page{}, fmt.Errorf("blocklist: list: %w", err)
	}
	total, err := s.q.CountBlocklist(ctx)
	if err != nil {
		return Page{}, fmt.Errorf("blocklist: count: %w", err)
	}
	out := Page{Page: page, PageSize: pageSize, TotalRecords: int(total)}
	out.Records = make([]Entry, 0, len(rows))
	for _, r := range rows {
		e, err := entryFromPG(r)
		if err != nil {
			return Page{}, err
		}
		out.Records = append(out.Records, e)
	}
	return out, nil
}

func (s *postgresStore) ListBySeries(ctx context.Context, seriesID int) ([]Entry, error) {
	rows, err := s.q.ListBlocklistBySeries(ctx, int32(seriesID))
	if err != nil {
		return nil, fmt.Errorf("blocklist: list by series: %w", err)
	}
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		e, err := entryFromPG(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *postgresStore) Delete(ctx context.Context, id int) error {
	return s.q.DeleteBlocklist(ctx, int32(id))
}

func (s *postgresStore) DeleteMany(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := s.q.DeleteBlocklist(ctx, int32(id)); err != nil {
			return err
		}
	}
	return nil
}

func (s *postgresStore) DeleteBySeries(ctx context.Context, seriesID int) error {
	return s.q.DeleteBlocklistBySeries(ctx, int32(seriesID))
}

func (s *postgresStore) Clear(ctx context.Context) error {
	return s.q.ClearBlocklist(ctx)
}

func entryFromPG(r pggen.Blocklist) (Entry, error) {
	var epIDs []int
	if err := json.Unmarshal(r.EpisodeIds, &epIDs); err != nil {
		return Entry{}, fmt.Errorf("blocklist: unmarshal episodeIds: %w", err)
	}
	e := Entry{
		ID:           int(r.ID),
		SeriesID:     int(r.SeriesID),
		EpisodeIDs:   epIDs,
		SourceTitle:  r.SourceTitle,
		Quality:      r.Quality,
		Languages:    r.Languages,
		Date:         r.Date.Time,
		Protocol:     Protocol(r.Protocol),
		Indexer:      r.Indexer,
		IndexerFlags: int(r.IndexerFlags),
		ReleaseType:  r.ReleaseType,
		Message:      r.Message,
	}
	if r.PublishedDate.Valid {
		t := r.PublishedDate.Time
		e.PublishedDate = &t
	}
	if r.Size.Valid {
		sz := r.Size.Int64
		e.Size = &sz
	}
	if r.TorrentInfoHash.Valid {
		e.TorrentInfoHash = r.TorrentInfoHash.String
	}
	return e, nil
}

func jsonBytes(b []byte, fallback string) []byte {
	if len(b) == 0 {
		return []byte(fallback)
	}
	return b
}

func pgTimeFromPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func pgInt8FromPtr(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

func pgTextFromStr(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
