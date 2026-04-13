package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

type sqliteStore struct {
	pool *db.SQLitePool
}

// NewSQLiteStore returns a Store backed by a SQLite pool.
func NewSQLiteStore(pool *db.SQLitePool) Store {
	return &sqliteStore{pool: pool}
}

func (s *sqliteStore) Create(ctx context.Context, entry Entry) (Entry, error) {
	data := historyDataSQLite(entry.Data)
	var out Entry
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteHistoryExec{exec: exec})
		row, err := queries.CreateHistoryEntry(ctx, sqlitegen.CreateHistoryEntryParams{
			EpisodeID:   entry.EpisodeID,
			SeriesID:    entry.SeriesID,
			SourceTitle: entry.SourceTitle,
			QualityName: entry.QualityName,
			EventType:   string(entry.EventType),
			DownloadID:  entry.DownloadID,
			Data:        data,
		})
		if err != nil {
			return fmt.Errorf("history: create: %w", err)
		}
		var convErr error
		out, convErr = historyFromSQLite(row)
		return convErr
	})
	if err != nil {
		return Entry{}, err
	}
	return out, nil
}

func (s *sqliteStore) ListForSeries(ctx context.Context, seriesID int64) ([]Entry, error) {
	queries := sqlitegen.New(sqliteHistoryQuery{q: s.pool.RawReader()})
	rows, err := queries.ListForSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("history: list for series: %w", err)
	}
	return historySliceFromSQLite(rows)
}

func (s *sqliteStore) ListForEpisode(ctx context.Context, episodeID int64) ([]Entry, error) {
	queries := sqlitegen.New(sqliteHistoryQuery{q: s.pool.RawReader()})
	rows, err := queries.ListForEpisode(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("history: list for episode: %w", err)
	}
	return historySliceFromSQLite(rows)
}

func (s *sqliteStore) FindByDownloadID(ctx context.Context, downloadID string) ([]Entry, error) {
	queries := sqlitegen.New(sqliteHistoryQuery{q: s.pool.RawReader()})
	rows, err := queries.FindByDownloadID(ctx, downloadID)
	if err != nil {
		return nil, fmt.Errorf("history: find by download id: %w", err)
	}
	return historySliceFromSQLite(rows)
}

func (s *sqliteStore) DeleteForSeries(ctx context.Context, seriesID int64) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteHistoryExec{exec: exec})
		return queries.DeleteForSeries(ctx, seriesID)
	})
	if err != nil {
		return fmt.Errorf("history: delete for series: %w", err)
	}
	return nil
}

// historyFromSQLite converts a sqlc-generated sqlite.History row to Entry.
func historyFromSQLite(r sqlitegen.History) (Entry, error) {
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
		Date:        parseSQLiteHistoryTime(r.Date),
		DownloadID:  r.DownloadID,
		Data:        data,
	}, nil
}

// historySliceFromSQLite converts a slice of sqlite.History rows.
func historySliceFromSQLite(rows []sqlitegen.History) ([]Entry, error) {
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		e, err := historyFromSQLite(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// historyDataSQLite converts a json.RawMessage to a string for SQLite storage.
// A nil/empty value is stored as "{}".
func historyDataSQLite(d json.RawMessage) string {
	if len(d) == 0 {
		return "{}"
	}
	return string(d)
}

// sqliteHistoryTimeLayout matches the format produced by SQLite's datetime('now').
const sqliteHistoryTimeLayout = "2006-01-02 15:04:05"

// parseSQLiteHistoryTime parses a SQLite datetime string into a time.Time (UTC).
func parseSQLiteHistoryTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(sqliteHistoryTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// sqliteHistoryExec adapts a db.Executor to sqlc's DBTX interface for writes.
type sqliteHistoryExec struct{ exec db.Executor }

func (a *sqliteHistoryExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteHistoryExec) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteHistoryExec: PrepareContext not supported")
}
func (a *sqliteHistoryExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteHistoryExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteHistoryQuery adapts a read-only db.Querier to sqlc's DBTX interface.
type sqliteHistoryQuery struct{ q db.Querier }

func (a sqliteHistoryQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteHistoryQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteHistoryQuery) PrepareContext(_ context.Context, _ string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteHistoryQuery: PrepareContext not supported")
}
func (a sqliteHistoryQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteHistoryQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
