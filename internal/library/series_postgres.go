package library

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// postgresSeriesStore implements SeriesStore against a Postgres pool.
type postgresSeriesStore struct {
	q   *pggen.Queries
	bus events.Bus
}

// newPostgresSeriesStore returns a SeriesStore backed by pool. Exported via
// New() in library.go.
func newPostgresSeriesStore(pool *db.PostgresPool, bus events.Bus) *postgresSeriesStore {
	return &postgresSeriesStore{
		q:   pggen.New(pool.Raw()),
		bus: bus,
	}
}

func (s *postgresSeriesStore) Create(ctx context.Context, in Series) (Series, error) {
	monitorNewItems := in.MonitorNewItems
	if monitorNewItems == "" {
		monitorNewItems = "all"
	}
	row, err := s.q.CreateSeries(ctx, pggen.CreateSeriesParams{
		TvdbID:           in.TvdbID,
		Title:            in.Title,
		Slug:             in.Slug,
		Status:           in.Status,
		SeriesType:       in.SeriesType,
		Path:             in.Path,
		Monitored:        in.Monitored,
		QualityProfileID: qualityProfileIDToPg(in.QualityProfileID),
		SeasonFolder:     in.SeasonFolder,
		MonitorNewItems:  monitorNewItems,
	})
	if err != nil {
		return Series{}, fmt.Errorf("library: create series: %w", err)
	}
	out := seriesFromPostgres(row)

	if err := s.bus.Publish(ctx, SeriesAdded{
		ID:     out.ID,
		TvdbID: out.TvdbID,
		Title:  out.Title,
	}); err != nil {
		return out, fmt.Errorf("library: publish SeriesAdded: %w", err)
	}
	return out, nil
}

func (s *postgresSeriesStore) Get(ctx context.Context, id int64) (Series, error) {
	row, err := s.q.GetSeries(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	if err != nil {
		return Series{}, fmt.Errorf("library: get series: %w", err)
	}
	return seriesFromPostgres(row), nil
}

func (s *postgresSeriesStore) GetByTvdbID(ctx context.Context, tvdbID int64) (Series, error) {
	row, err := s.q.GetSeriesByTvdbID(ctx, tvdbID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	if err != nil {
		return Series{}, fmt.Errorf("library: get series by tvdb_id: %w", err)
	}
	return seriesFromPostgres(row), nil
}

func (s *postgresSeriesStore) GetBySlug(ctx context.Context, slug string) (Series, error) {
	row, err := s.q.GetSeriesBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	if err != nil {
		return Series{}, fmt.Errorf("library: get series by slug: %w", err)
	}
	return seriesFromPostgres(row), nil
}

func (s *postgresSeriesStore) List(ctx context.Context) ([]Series, error) {
	rows, err := s.q.ListSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("library: list series: %w", err)
	}
	out := make([]Series, 0, len(rows))
	for _, r := range rows {
		out = append(out, seriesFromPostgres(r))
	}
	return out, nil
}

func (s *postgresSeriesStore) Update(ctx context.Context, in Series) error {
	monitorNewItems := in.MonitorNewItems
	if monitorNewItems == "" {
		monitorNewItems = "all"
	}
	if err := s.q.UpdateSeries(ctx, pggen.UpdateSeriesParams{
		ID:               in.ID,
		TvdbID:           in.TvdbID,
		Title:            in.Title,
		Slug:             in.Slug,
		Status:           in.Status,
		SeriesType:       in.SeriesType,
		Path:             in.Path,
		Monitored:        in.Monitored,
		QualityProfileID: qualityProfileIDToPg(in.QualityProfileID),
		SeasonFolder:     in.SeasonFolder,
		MonitorNewItems:  monitorNewItems,
	}); err != nil {
		return fmt.Errorf("library: update series: %w", err)
	}
	if err := s.bus.Publish(ctx, SeriesUpdated{ID: in.ID, TvdbID: in.TvdbID}); err != nil {
		return fmt.Errorf("library: publish SeriesUpdated: %w", err)
	}
	return nil
}

func (s *postgresSeriesStore) Delete(ctx context.Context, id int64) error {
	if err := s.q.DeleteSeries(ctx, id); err != nil {
		return fmt.Errorf("library: delete series: %w", err)
	}
	if err := s.bus.Publish(ctx, SeriesDeleted{ID: id}); err != nil {
		return fmt.Errorf("library: publish SeriesDeleted: %w", err)
	}
	return nil
}

// seriesFromPostgres converts a sqlc-generated pggen.Series row to the
// canonical library.Series struct.
func seriesFromPostgres(r pggen.Series) Series {
	qpID := int64(0)
	if r.QualityProfileID.Valid {
		qpID = int64(r.QualityProfileID.Int32)
	}
	return Series{
		ID:               r.ID,
		TvdbID:           r.TvdbID,
		Title:            r.Title,
		Slug:             r.Slug,
		Status:           r.Status,
		SeriesType:       r.SeriesType,
		Path:             r.Path,
		Monitored:        r.Monitored,
		QualityProfileID: qpID,
		SeasonFolder:     r.SeasonFolder,
		MonitorNewItems:  r.MonitorNewItems,
		Added:            r.Added.Time,
		CreatedAt:        r.CreatedAt.Time,
		UpdatedAt:        r.UpdatedAt.Time,
	}
}

// qualityProfileIDToPg converts a domain quality-profile id (0 == unassigned) to
// a pgtype.Int4 nullable value. quality_profile_id column is Postgres INTEGER
// because quality_profiles.id is SERIAL (int32).
func qualityProfileIDToPg(id int64) pgtype.Int4 {
	if id == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(id), Valid: true}
}
