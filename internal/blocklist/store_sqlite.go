package blocklist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

type sqliteStore struct {
	pool *db.SQLitePool
}

// NewSQLiteStore returns a Store backed by SQLite.
func NewSQLiteStore(pool *db.SQLitePool) Store {
	return &sqliteStore{pool: pool}
}

func (s *sqliteStore) Create(ctx context.Context, e Entry) (Entry, error) {
	epIDs, err := json.Marshal(intSlice(e.EpisodeIDs))
	if err != nil {
		return Entry{}, fmt.Errorf("blocklist: marshal episodeIds: %w", err)
	}
	params := sqlitegen.CreateBlocklistParams{
		SeriesID:        int64(e.SeriesID),
		EpisodeIds:      string(epIDs),
		SourceTitle:     e.SourceTitle,
		Quality:         jsonOrEmpty(e.Quality, "{}"),
		Languages:       jsonOrEmpty(e.Languages, "[]"),
		Date:            e.Date,
		PublishedDate:   nullTime(e.PublishedDate),
		Size:            nullInt64(e.Size),
		Protocol:        string(e.Protocol),
		Indexer:         e.Indexer,
		IndexerFlags:    int64(e.IndexerFlags),
		ReleaseType:     e.ReleaseType,
		Message:         e.Message,
		TorrentInfoHash: nullString(e.TorrentInfoHash),
	}
	var out Entry
	err = s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateBlocklist(ctx, params)
		if err != nil {
			return fmt.Errorf("blocklist: create: %w", err)
		}
		var convErr error
		out, convErr = entryFromSQLite(row)
		return convErr
	})
	if err != nil {
		return Entry{}, err
	}
	return out, nil
}

func (s *sqliteStore) GetByID(ctx context.Context, id int) (Entry, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetBlocklistByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("blocklist: get by id: %w", err)
	}
	return entryFromSQLite(row)
}

func (s *sqliteStore) List(ctx context.Context, page, pageSize int) (Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListBlocklist(ctx, sqlitegen.ListBlocklistParams{
		Limit:  int64(pageSize),
		Offset: int64((page - 1) * pageSize),
	})
	if err != nil {
		return Page{}, fmt.Errorf("blocklist: list: %w", err)
	}
	total, err := queries.CountBlocklist(ctx)
	if err != nil {
		return Page{}, fmt.Errorf("blocklist: count: %w", err)
	}
	out := Page{Page: page, PageSize: pageSize, TotalRecords: int(total)}
	out.Records = make([]Entry, 0, len(rows))
	for _, r := range rows {
		e, err := entryFromSQLite(r)
		if err != nil {
			return Page{}, err
		}
		out.Records = append(out.Records, e)
	}
	return out, nil
}

func (s *sqliteStore) ListBySeries(ctx context.Context, seriesID int) ([]Entry, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListBlocklistBySeries(ctx, int64(seriesID))
	if err != nil {
		return nil, fmt.Errorf("blocklist: list by series: %w", err)
	}
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		e, err := entryFromSQLite(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *sqliteStore) Delete(ctx context.Context, id int) error {
	return s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteBlocklist(ctx, int64(id))
	})
}

func (s *sqliteStore) DeleteMany(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	return s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		for _, id := range ids {
			if err := queries.DeleteBlocklist(ctx, int64(id)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *sqliteStore) DeleteBySeries(ctx context.Context, seriesID int) error {
	return s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteBlocklistBySeries(ctx, int64(seriesID))
	})
}

func (s *sqliteStore) Clear(ctx context.Context) error {
	return s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.ClearBlocklist(ctx)
	})
}

func entryFromSQLite(r sqlitegen.Blocklist) (Entry, error) {
	var epIDs []int
	if err := json.Unmarshal([]byte(r.EpisodeIds), &epIDs); err != nil {
		return Entry{}, fmt.Errorf("blocklist: unmarshal episodeIds: %w", err)
	}
	e := Entry{
		ID:           int(r.ID),
		SeriesID:     int(r.SeriesID),
		EpisodeIDs:   epIDs,
		SourceTitle:  r.SourceTitle,
		Quality:      []byte(r.Quality),
		Languages:    []byte(r.Languages),
		Date:         r.Date,
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

func intSlice(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

func jsonOrEmpty(b []byte, fallback string) string {
	if len(b) == 0 || strings.TrimSpace(string(b)) == "" {
		return fallback
	}
	return string(b)
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// sqliteExec adapts db.Executor to sqlc's DBTX for writes.
type sqliteExec struct{ exec db.Executor }

func (a *sqliteExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteExec) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteExec: PrepareContext not supported")
}
func (a *sqliteExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteQuery adapts a read-only Querier to sqlc's DBTX.
type sqliteQuery struct{ q db.Querier }

func (a sqliteQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteQuery) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteQuery: PrepareContext not supported")
}
func (a sqliteQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
