package releaseprofile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

type sqliteStore struct{ pool *db.SQLitePool }

// NewSQLiteStore returns a Store backed by SQLite.
func NewSQLiteStore(pool *db.SQLitePool) Store {
	return &sqliteStore{pool: pool}
}

func (s *sqliteStore) Create(ctx context.Context, p Profile) (Profile, error) {
	req, _ := json.Marshal(nonNilStr(p.Required))
	ign, _ := json.Marshal(nonNilStr(p.Ignored))
	tags, _ := json.Marshal(nonNilInt(p.Tags))
	var out Profile
	err := s.pool.Write(ctx, func(exec db.Executor) error {
		q := sqlitegen.New(&sqliteExec{exec: exec})
		row, err := q.CreateReleaseProfile(ctx, sqlitegen.CreateReleaseProfileParams{
			Name:      p.Name,
			Enabled:   boolToInt64(p.Enabled),
			Required:  string(req),
			Ignored:   string(ign),
			IndexerID: int64(p.IndexerID),
			Tags:      string(tags),
		})
		if err != nil {
			return err
		}
		out, err = fromSQLite(row)
		return err
	})
	if err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: create: %w", err)
	}
	return out, nil
}

func (s *sqliteStore) GetByID(ctx context.Context, id int) (Profile, error) {
	q := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	row, err := q.GetReleaseProfileByID(ctx, int64(id))
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: get: %w", err)
	}
	return fromSQLite(row)
}

func (s *sqliteStore) List(ctx context.Context) ([]Profile, error) {
	q := sqlitegen.New(sqliteQuery{q: s.pool.RawReader()})
	rows, err := q.ListReleaseProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("releaseprofile: list: %w", err)
	}
	out := make([]Profile, 0, len(rows))
	for _, r := range rows {
		p, err := fromSQLite(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *sqliteStore) Update(ctx context.Context, p Profile) error {
	req, _ := json.Marshal(nonNilStr(p.Required))
	ign, _ := json.Marshal(nonNilStr(p.Ignored))
	tags, _ := json.Marshal(nonNilInt(p.Tags))
	return s.pool.Write(ctx, func(exec db.Executor) error {
		q := sqlitegen.New(&sqliteExec{exec: exec})
		return q.UpdateReleaseProfile(ctx, sqlitegen.UpdateReleaseProfileParams{
			ID:        int64(p.ID),
			Name:      p.Name,
			Enabled:   boolToInt64(p.Enabled),
			Required:  string(req),
			Ignored:   string(ign),
			IndexerID: int64(p.IndexerID),
			Tags:      string(tags),
		})
	})
}

func (s *sqliteStore) Delete(ctx context.Context, id int) error {
	return s.pool.Write(ctx, func(exec db.Executor) error {
		q := sqlitegen.New(&sqliteExec{exec: exec})
		return q.DeleteReleaseProfile(ctx, int64(id))
	})
}

func fromSQLite(r sqlitegen.ReleaseProfile) (Profile, error) {
	var req, ign []string
	var tags []int
	if err := json.Unmarshal([]byte(r.Required), &req); err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: unmarshal required: %w", err)
	}
	if err := json.Unmarshal([]byte(r.Ignored), &ign); err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: unmarshal ignored: %w", err)
	}
	if err := json.Unmarshal([]byte(r.Tags), &tags); err != nil {
		return Profile{}, fmt.Errorf("releaseprofile: unmarshal tags: %w", err)
	}
	return Profile{
		ID:        int(r.ID),
		Name:      r.Name,
		Enabled:   r.Enabled != 0,
		Required:  nonNilStr(req),
		Ignored:   nonNilStr(ign),
		IndexerID: int(r.IndexerID),
		Tags:      nonNilInt(tags),
	}, nil
}

func nonNilStr(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilInt(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

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

type sqliteQuery struct{ q db.Querier }

func (a sqliteQuery) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("sqliteQuery: ExecContext not allowed")
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
