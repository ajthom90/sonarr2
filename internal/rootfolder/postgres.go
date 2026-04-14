package rootfolder

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type postgresStore struct {
	q *pggen.Queries
}

// NewPostgresStore returns a Store backed by a Postgres pool.
func NewPostgresStore(pool *db.PostgresPool) Store {
	return &postgresStore{q: pggen.New(pool.Raw())}
}

func (s *postgresStore) Create(ctx context.Context, path string) (RootFolder, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return RootFolder{}, errors.New("rootfolder: path is required")
	}
	row, err := s.q.CreateRootFolder(ctx, path)
	if err != nil {
		if isPostgresUniqueErr(err) {
			return RootFolder{}, ErrAlreadyExists
		}
		return RootFolder{}, fmt.Errorf("rootfolder: create: %w", err)
	}
	return pgRowToDomain(row), nil
}

func (s *postgresStore) Get(ctx context.Context, id int64) (RootFolder, error) {
	row, err := s.q.GetRootFolder(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return RootFolder{}, ErrNotFound
	}
	if err != nil {
		return RootFolder{}, fmt.Errorf("rootfolder: get: %w", err)
	}
	return pgRowToDomain(row), nil
}

func (s *postgresStore) GetByPath(ctx context.Context, path string) (RootFolder, error) {
	row, err := s.q.GetRootFolderByPath(ctx, strings.TrimSpace(path))
	if errors.Is(err, pgx.ErrNoRows) {
		return RootFolder{}, ErrNotFound
	}
	if err != nil {
		return RootFolder{}, fmt.Errorf("rootfolder: get by path: %w", err)
	}
	return pgRowToDomain(row), nil
}

func (s *postgresStore) List(ctx context.Context) ([]RootFolder, error) {
	rows, err := s.q.ListRootFolders(ctx)
	if err != nil {
		return nil, fmt.Errorf("rootfolder: list: %w", err)
	}
	out := make([]RootFolder, 0, len(rows))
	for _, r := range rows {
		out = append(out, pgRowToDomain(r))
	}
	return out, nil
}

func (s *postgresStore) Delete(ctx context.Context, id int64) error {
	if err := s.q.DeleteRootFolder(ctx, int32(id)); err != nil {
		return fmt.Errorf("rootfolder: delete: %w", err)
	}
	return nil
}

// pgRowToDomain converts a sqlc-generated Postgres RootFolder row into the
// package's domain type. The ID widens int32 -> int64 and CreatedAt unwraps
// pgtype.Timestamptz.
func pgRowToDomain(row pggen.RootFolder) RootFolder {
	return RootFolder{
		ID:        int64(row.ID),
		Path:      row.Path,
		CreatedAt: row.CreatedAt.Time,
	}
}

// isPostgresUniqueErr reports whether err is a unique_violation (SQLSTATE 23505).
func isPostgresUniqueErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
