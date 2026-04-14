package tags

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

func (s *postgresStore) Create(ctx context.Context, label string) (Tag, error) {
	label = NormalizeLabel(label)
	if label == "" {
		return Tag{}, errors.New("tags: label is required")
	}
	row, err := s.q.CreateTag(ctx, label)
	if err != nil {
		if isPostgresUniqueErr(err) {
			return Tag{}, ErrDuplicateLabel
		}
		return Tag{}, fmt.Errorf("tags: create: %w", err)
	}
	return Tag{ID: int(row.ID), Label: row.Label}, nil
}

func (s *postgresStore) GetByID(ctx context.Context, id int) (Tag, error) {
	row, err := s.q.GetTagByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	if err != nil {
		return Tag{}, fmt.Errorf("tags: get by id: %w", err)
	}
	return Tag{ID: int(row.ID), Label: row.Label}, nil
}

func (s *postgresStore) GetByLabel(ctx context.Context, label string) (Tag, error) {
	row, err := s.q.GetTagByLabel(ctx, NormalizeLabel(label))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tag{}, ErrNotFound
	}
	if err != nil {
		return Tag{}, fmt.Errorf("tags: get by label: %w", err)
	}
	return Tag{ID: int(row.ID), Label: row.Label}, nil
}

func (s *postgresStore) List(ctx context.Context) ([]Tag, error) {
	rows, err := s.q.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("tags: list: %w", err)
	}
	out := make([]Tag, 0, len(rows))
	for _, r := range rows {
		out = append(out, Tag{ID: int(r.ID), Label: r.Label})
	}
	return out, nil
}

func (s *postgresStore) Update(ctx context.Context, t Tag) error {
	label := NormalizeLabel(t.Label)
	if label == "" {
		return errors.New("tags: label is required")
	}
	if err := s.q.UpdateTag(ctx, pggen.UpdateTagParams{
		Label: label,
		ID:    int32(t.ID),
	}); err != nil {
		if isPostgresUniqueErr(err) {
			return ErrDuplicateLabel
		}
		return fmt.Errorf("tags: update: %w", err)
	}
	return nil
}

func (s *postgresStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteTag(ctx, int32(id)); err != nil {
		return fmt.Errorf("tags: delete: %w", err)
	}
	return nil
}

// isPostgresUniqueErr reports whether err is a unique_violation (SQLSTATE 23505).
func isPostgresUniqueErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
