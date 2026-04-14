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

// sqliteSeriesStore implements SeriesStore against a SQLite pool.
type sqliteSeriesStore struct {
	pool *db.SQLitePool
	bus  events.Bus
}

func newSqliteSeriesStore(pool *db.SQLitePool, bus events.Bus) *sqliteSeriesStore {
	return &sqliteSeriesStore{pool: pool, bus: bus}
}

// sqliteTimeLayout matches the format produced by SQLite's datetime('now').
const sqliteTimeLayout = "2006-01-02 15:04:05"

// parseSqliteTime parses a SQLite datetime string into a time.Time. Returns
// zero time on parse errors so callers see an obvious "uninitialized" value
// rather than a runtime error.
func parseSqliteTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(sqliteTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (s *sqliteSeriesStore) Create(ctx context.Context, in Series) (Series, error) {
	monitorNewItems := in.MonitorNewItems
	if monitorNewItems == "" {
		monitorNewItems = "all"
	}
	var out Series
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateSeries(ctx, sqlitegen.CreateSeriesParams{
			TvdbID:           in.TvdbID,
			Title:            in.Title,
			Slug:             in.Slug,
			Status:           in.Status,
			SeriesType:       in.SeriesType,
			Path:             in.Path,
			Monitored:        boolToInt64(in.Monitored),
			QualityProfileID: sql.NullInt64{Int64: in.QualityProfileID, Valid: in.QualityProfileID != 0},
			SeasonFolder:     boolToInt64(in.SeasonFolder),
			MonitorNewItems:  monitorNewItems,
		})
		if err != nil {
			return fmt.Errorf("library: create series: %w", err)
		}
		out = seriesFromSqlite(row)
		return nil
	})
	if err != nil {
		return Series{}, err
	}
	if err := s.bus.Publish(ctx, SeriesAdded{
		ID:     out.ID,
		TvdbID: out.TvdbID,
		Title:  out.Title,
	}); err != nil {
		return out, fmt.Errorf("library: publish SeriesAdded: %w", err)
	}
	return out, nil
}

func (s *sqliteSeriesStore) Get(ctx context.Context, id int64) (Series, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetSeries(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	if err != nil {
		return Series{}, fmt.Errorf("library: get series: %w", err)
	}
	return seriesFromSqlite(row), nil
}

func (s *sqliteSeriesStore) GetByTvdbID(ctx context.Context, tvdbID int64) (Series, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetSeriesByTvdbID(ctx, tvdbID)
	if errors.Is(err, sql.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	if err != nil {
		return Series{}, fmt.Errorf("library: get series by tvdb_id: %w", err)
	}
	return seriesFromSqlite(row), nil
}

func (s *sqliteSeriesStore) GetBySlug(ctx context.Context, slug string) (Series, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetSeriesBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	if err != nil {
		return Series{}, fmt.Errorf("library: get series by slug: %w", err)
	}
	return seriesFromSqlite(row), nil
}

func (s *sqliteSeriesStore) List(ctx context.Context) ([]Series, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("library: list series: %w", err)
	}
	out := make([]Series, 0, len(rows))
	for _, r := range rows {
		out = append(out, seriesFromSqlite(r))
	}
	return out, nil
}

func (s *sqliteSeriesStore) Update(ctx context.Context, in Series) error {
	monitorNewItems := in.MonitorNewItems
	if monitorNewItems == "" {
		monitorNewItems = "all"
	}
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpdateSeries(ctx, sqlitegen.UpdateSeriesParams{
			ID:               in.ID,
			TvdbID:           in.TvdbID,
			Title:            in.Title,
			Slug:             in.Slug,
			Status:           in.Status,
			SeriesType:       in.SeriesType,
			Path:             in.Path,
			Monitored:        boolToInt64(in.Monitored),
			QualityProfileID: sql.NullInt64{Int64: in.QualityProfileID, Valid: in.QualityProfileID != 0},
			SeasonFolder:     boolToInt64(in.SeasonFolder),
			MonitorNewItems:  monitorNewItems,
		})
	})
	if err != nil {
		return fmt.Errorf("library: update series: %w", err)
	}
	if err := s.bus.Publish(ctx, SeriesUpdated{ID: in.ID, TvdbID: in.TvdbID}); err != nil {
		return fmt.Errorf("library: publish SeriesUpdated: %w", err)
	}
	return nil
}

func (s *sqliteSeriesStore) Delete(ctx context.Context, id int64) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteSeries(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("library: delete series: %w", err)
	}
	if err := s.bus.Publish(ctx, SeriesDeleted{ID: id}); err != nil {
		return fmt.Errorf("library: publish SeriesDeleted: %w", err)
	}
	return nil
}

func seriesFromSqlite(r sqlitegen.Series) Series {
	qpID := int64(0)
	if r.QualityProfileID.Valid {
		qpID = r.QualityProfileID.Int64
	}
	return Series{
		ID:               r.ID,
		TvdbID:           r.TvdbID,
		Title:            r.Title,
		Slug:             r.Slug,
		Status:           r.Status,
		SeriesType:       r.SeriesType,
		Path:             r.Path,
		Monitored:        r.Monitored != 0,
		QualityProfileID: qpID,
		SeasonFolder:     r.SeasonFolder != 0,
		MonitorNewItems:  r.MonitorNewItems,
		Added:            parseSqliteTime(r.Added),
		CreatedAt:        parseSqliteTime(r.CreatedAt),
		UpdatedAt:        parseSqliteTime(r.UpdatedAt),
	}
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// sqliteExec adapts a db.Executor to sqlc's DBTX interface for writes.
// Same shape as internal/hostconfig/sqlite.go — we duplicate rather than
// share because the two packages have independent test surfaces.
type sqliteExec struct{ exec db.Executor }

func (a *sqliteExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteExec) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteExec: PrepareContext not supported")
}
func (a *sqliteExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}

// sqliteQuery adapts a read-only db.Querier to sqlc's DBTX interface.
// PrepareContext and ExecContext are not used by read queries, but the
// generated code references them; returning errors/nil is safe because
// sqlc's read-only call paths never invoke them.
type sqliteQuery struct{ q db.Querier }

func (a sqliteQuery) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errors.New("sqliteQuery: ExecContext not allowed on read-only adapter")
}
func (a sqliteQuery) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteQuery: PrepareContext not supported")
}
func (a sqliteQuery) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a sqliteQuery) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}
