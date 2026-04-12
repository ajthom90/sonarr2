package profiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

// --- QualityDefinition ---

type sqliteQualityDefinitionStore struct {
	pool *db.SQLitePool
}

// NewSQLiteQualityDefinitionStore returns a QualityDefinitionStore backed by
// a SQLite pool.
func NewSQLiteQualityDefinitionStore(pool *db.SQLitePool) QualityDefinitionStore {
	return &sqliteQualityDefinitionStore{pool: pool}
}

func (s *sqliteQualityDefinitionStore) GetAll(ctx context.Context) ([]QualityDefinition, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.GetAllQualityDefinitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("profiles: get all quality definitions: %w", err)
	}
	out := make([]QualityDefinition, 0, len(rows))
	for _, r := range rows {
		out = append(out, qualityDefFromSQLite(r))
	}
	return out, nil
}

func (s *sqliteQualityDefinitionStore) GetByID(ctx context.Context, id int) (QualityDefinition, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetQualityDefinitionByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return QualityDefinition{}, ErrNotFound
	}
	if err != nil {
		return QualityDefinition{}, fmt.Errorf("profiles: get quality definition by id: %w", err)
	}
	return qualityDefFromSQLite(row), nil
}

func qualityDefFromSQLite(r sqlitegen.QualityDefinition) QualityDefinition {
	return QualityDefinition{
		ID:            int(r.ID),
		Name:          r.Name,
		Source:        r.Source,
		Resolution:    r.Resolution,
		MinSize:       r.MinSize,
		MaxSize:       r.MaxSize,
		PreferredSize: r.PreferredSize,
	}
}

// --- QualityProfile ---

type sqliteQualityProfileStore struct {
	pool *db.SQLitePool
}

// NewSQLiteQualityProfileStore returns a QualityProfileStore backed by a
// SQLite pool.
func NewSQLiteQualityProfileStore(pool *db.SQLitePool) QualityProfileStore {
	return &sqliteQualityProfileStore{pool: pool}
}

func (s *sqliteQualityProfileStore) Create(ctx context.Context, p QualityProfile) (QualityProfile, error) {
	itemsJSON, err := json.Marshal(p.Items)
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: marshal items: %w", err)
	}
	formatItemsJSON, err := json.Marshal(p.FormatItems)
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: marshal format_items: %w", err)
	}

	var out QualityProfile
	err = s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := queries.CreateQualityProfile(ctx, sqlitegen.CreateQualityProfileParams{
			Name:              p.Name,
			UpgradeAllowed:    boolToInt64(p.UpgradeAllowed),
			Cutoff:            int64(p.Cutoff),
			Items:             string(itemsJSON),
			MinFormatScore:    int64(p.MinFormatScore),
			CutoffFormatScore: int64(p.CutoffFormatScore),
			FormatItems:       string(formatItemsJSON),
		})
		if err != nil {
			return fmt.Errorf("profiles: create quality profile: %w", err)
		}
		var convErr error
		out, convErr = qualityProfileFromSQLite(row)
		return convErr
	})
	if err != nil {
		return QualityProfile{}, err
	}
	return out, nil
}

func (s *sqliteQualityProfileStore) GetByID(ctx context.Context, id int) (QualityProfile, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := queries.GetQualityProfileByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return QualityProfile{}, ErrNotFound
	}
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: get quality profile by id: %w", err)
	}
	return qualityProfileFromSQLite(row)
}

func (s *sqliteQualityProfileStore) List(ctx context.Context) ([]QualityProfile, error) {
	queries := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := queries.ListQualityProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("profiles: list quality profiles: %w", err)
	}
	out := make([]QualityProfile, 0, len(rows))
	for _, r := range rows {
		p, err := qualityProfileFromSQLite(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *sqliteQualityProfileStore) Update(ctx context.Context, p QualityProfile) error {
	itemsJSON, err := json.Marshal(p.Items)
	if err != nil {
		return fmt.Errorf("profiles: marshal items: %w", err)
	}
	formatItemsJSON, err := json.Marshal(p.FormatItems)
	if err != nil {
		return fmt.Errorf("profiles: marshal format_items: %w", err)
	}

	err = s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.UpdateQualityProfile(ctx, sqlitegen.UpdateQualityProfileParams{
			ID:                int64(p.ID),
			Name:              p.Name,
			UpgradeAllowed:    boolToInt64(p.UpgradeAllowed),
			Cutoff:            int64(p.Cutoff),
			Items:             string(itemsJSON),
			MinFormatScore:    int64(p.MinFormatScore),
			CutoffFormatScore: int64(p.CutoffFormatScore),
			FormatItems:       string(formatItemsJSON),
		})
	})
	if err != nil {
		return fmt.Errorf("profiles: update quality profile: %w", err)
	}
	return nil
}

func (s *sqliteQualityProfileStore) Delete(ctx context.Context, id int) error {
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteExec{exec: exec})
		return queries.DeleteQualityProfile(ctx, int64(id))
	})
	if err != nil {
		return fmt.Errorf("profiles: delete quality profile: %w", err)
	}
	return nil
}

// qualityProfileFromSQLite converts a sqlc row to the canonical domain type,
// unmarshaling the JSON text columns.
func qualityProfileFromSQLite(r sqlitegen.QualityProfile) (QualityProfile, error) {
	var items []QualityProfileItem
	if err := json.Unmarshal([]byte(r.Items), &items); err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: unmarshal items: %w", err)
	}
	if items == nil {
		items = []QualityProfileItem{}
	}

	var formatItems []FormatScoreItem
	if err := json.Unmarshal([]byte(r.FormatItems), &formatItems); err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: unmarshal format_items: %w", err)
	}
	if formatItems == nil {
		formatItems = []FormatScoreItem{}
	}

	return QualityProfile{
		ID:                int(r.ID),
		Name:              r.Name,
		UpgradeAllowed:    r.UpgradeAllowed != 0,
		Cutoff:            int(r.Cutoff),
		Items:             items,
		MinFormatScore:    int(r.MinFormatScore),
		CutoffFormatScore: int(r.CutoffFormatScore),
		FormatItems:       formatItems,
	}, nil
}

// boolToInt64 converts a Go bool to the SQLite integer representation.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// sqliteExec adapts a db.Executor to sqlc's DBTX interface for writes.
// Mirrors the adapter in internal/library/series_sqlite.go.
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

// sqliteQuery adapts a read-only db.Querier to sqlc's DBTX interface.
// Mirrors the adapter in internal/library/series_sqlite.go.
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
