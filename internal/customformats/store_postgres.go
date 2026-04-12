package customformats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned when a requested custom format does not exist.
var ErrNotFound = errors.New("customformats: not found")

type postgresStore struct {
	q *pggen.Queries
}

// NewPostgresStore returns a Store backed by a Postgres pool.
func NewPostgresStore(pool *db.PostgresPool) Store {
	return &postgresStore{q: pggen.New(pool.Raw())}
}

func (s *postgresStore) Create(ctx context.Context, cf CustomFormat) (CustomFormat, error) {
	specsJSON, err := json.Marshal(cf.Specifications)
	if err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: marshal specifications: %w", err)
	}
	row, err := s.q.CreateCustomFormat(ctx, pggen.CreateCustomFormatParams{
		Name:                cf.Name,
		IncludeWhenRenaming: cf.IncludeWhenRenaming,
		Specifications:      specsJSON,
	})
	if err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: create: %w", err)
	}
	return customFormatFromPostgres(row)
}

func (s *postgresStore) GetByID(ctx context.Context, id int) (CustomFormat, error) {
	row, err := s.q.GetCustomFormatByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomFormat{}, ErrNotFound
	}
	if err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: get by id: %w", err)
	}
	return customFormatFromPostgres(row)
}

func (s *postgresStore) List(ctx context.Context) ([]CustomFormat, error) {
	rows, err := s.q.ListCustomFormats(ctx)
	if err != nil {
		return nil, fmt.Errorf("customformats: list: %w", err)
	}
	out := make([]CustomFormat, 0, len(rows))
	for _, r := range rows {
		cf, err := customFormatFromPostgres(r)
		if err != nil {
			return nil, err
		}
		out = append(out, cf)
	}
	return out, nil
}

func (s *postgresStore) Update(ctx context.Context, cf CustomFormat) error {
	specsJSON, err := json.Marshal(cf.Specifications)
	if err != nil {
		return fmt.Errorf("customformats: marshal specifications: %w", err)
	}
	if err := s.q.UpdateCustomFormat(ctx, pggen.UpdateCustomFormatParams{
		ID:                  int32(cf.ID),
		Name:                cf.Name,
		IncludeWhenRenaming: cf.IncludeWhenRenaming,
		Specifications:      specsJSON,
	}); err != nil {
		return fmt.Errorf("customformats: update: %w", err)
	}
	return nil
}

func (s *postgresStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteCustomFormat(ctx, int32(id)); err != nil {
		return fmt.Errorf("customformats: delete: %w", err)
	}
	return nil
}

func customFormatFromPostgres(r pggen.CustomFormat) (CustomFormat, error) {
	var specs []Specification
	if err := json.Unmarshal(r.Specifications, &specs); err != nil {
		return CustomFormat{}, fmt.Errorf("customformats: unmarshal specifications: %w", err)
	}
	if specs == nil {
		specs = []Specification{}
	}
	return CustomFormat{
		ID:                  int(r.ID),
		Name:                r.Name,
		IncludeWhenRenaming: r.IncludeWhenRenaming,
		Specifications:      specs,
	}, nil
}
