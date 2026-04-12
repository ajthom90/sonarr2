package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

// --- QualityDefinition ---

type postgresQualityDefinitionStore struct {
	q *pggen.Queries
}

// NewPostgresQualityDefinitionStore returns a QualityDefinitionStore backed by
// a Postgres pool.
func NewPostgresQualityDefinitionStore(pool *db.PostgresPool) QualityDefinitionStore {
	return &postgresQualityDefinitionStore{q: pggen.New(pool.Raw())}
}

func (s *postgresQualityDefinitionStore) GetAll(ctx context.Context) ([]QualityDefinition, error) {
	rows, err := s.q.GetAllQualityDefinitions(ctx)
	if err != nil {
		return nil, fmt.Errorf("profiles: get all quality definitions: %w", err)
	}
	out := make([]QualityDefinition, 0, len(rows))
	for _, r := range rows {
		out = append(out, qualityDefFromPostgres(r))
	}
	return out, nil
}

func (s *postgresQualityDefinitionStore) GetByID(ctx context.Context, id int) (QualityDefinition, error) {
	row, err := s.q.GetQualityDefinitionByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return QualityDefinition{}, ErrNotFound
	}
	if err != nil {
		return QualityDefinition{}, fmt.Errorf("profiles: get quality definition by id: %w", err)
	}
	return qualityDefFromPostgres(row), nil
}

func qualityDefFromPostgres(r pggen.QualityDefinition) QualityDefinition {
	return QualityDefinition{
		ID:            int(r.ID),
		Name:          r.Name,
		Source:        r.Source,
		Resolution:    r.Resolution,
		MinSize:       float64(r.MinSize),
		MaxSize:       float64(r.MaxSize),
		PreferredSize: float64(r.PreferredSize),
	}
}

// --- QualityProfile ---

type postgresQualityProfileStore struct {
	q *pggen.Queries
}

// NewPostgresQualityProfileStore returns a QualityProfileStore backed by a
// Postgres pool.
func NewPostgresQualityProfileStore(pool *db.PostgresPool) QualityProfileStore {
	return &postgresQualityProfileStore{q: pggen.New(pool.Raw())}
}

func (s *postgresQualityProfileStore) Create(ctx context.Context, p QualityProfile) (QualityProfile, error) {
	itemsJSON, err := json.Marshal(p.Items)
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: marshal items: %w", err)
	}
	formatItemsJSON, err := json.Marshal(p.FormatItems)
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: marshal format_items: %w", err)
	}

	row, err := s.q.CreateQualityProfile(ctx, pggen.CreateQualityProfileParams{
		Name:              p.Name,
		UpgradeAllowed:    p.UpgradeAllowed,
		Cutoff:            int32(p.Cutoff),
		Items:             itemsJSON,
		MinFormatScore:    int32(p.MinFormatScore),
		CutoffFormatScore: int32(p.CutoffFormatScore),
		FormatItems:       formatItemsJSON,
	})
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: create quality profile: %w", err)
	}
	return qualityProfileFromPostgres(row)
}

func (s *postgresQualityProfileStore) GetByID(ctx context.Context, id int) (QualityProfile, error) {
	row, err := s.q.GetQualityProfileByID(ctx, int32(id))
	if errors.Is(err, pgx.ErrNoRows) {
		return QualityProfile{}, ErrNotFound
	}
	if err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: get quality profile by id: %w", err)
	}
	return qualityProfileFromPostgres(row)
}

func (s *postgresQualityProfileStore) List(ctx context.Context) ([]QualityProfile, error) {
	rows, err := s.q.ListQualityProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("profiles: list quality profiles: %w", err)
	}
	out := make([]QualityProfile, 0, len(rows))
	for _, r := range rows {
		p, err := qualityProfileFromPostgres(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *postgresQualityProfileStore) Update(ctx context.Context, p QualityProfile) error {
	itemsJSON, err := json.Marshal(p.Items)
	if err != nil {
		return fmt.Errorf("profiles: marshal items: %w", err)
	}
	formatItemsJSON, err := json.Marshal(p.FormatItems)
	if err != nil {
		return fmt.Errorf("profiles: marshal format_items: %w", err)
	}

	if err := s.q.UpdateQualityProfile(ctx, pggen.UpdateQualityProfileParams{
		ID:                int32(p.ID),
		Name:              p.Name,
		UpgradeAllowed:    p.UpgradeAllowed,
		Cutoff:            int32(p.Cutoff),
		Items:             itemsJSON,
		MinFormatScore:    int32(p.MinFormatScore),
		CutoffFormatScore: int32(p.CutoffFormatScore),
		FormatItems:       formatItemsJSON,
	}); err != nil {
		return fmt.Errorf("profiles: update quality profile: %w", err)
	}
	return nil
}

func (s *postgresQualityProfileStore) Delete(ctx context.Context, id int) error {
	if err := s.q.DeleteQualityProfile(ctx, int32(id)); err != nil {
		return fmt.Errorf("profiles: delete quality profile: %w", err)
	}
	return nil
}

// qualityProfileFromPostgres converts a sqlc row to the canonical domain type,
// unmarshaling the JSONB items fields.
func qualityProfileFromPostgres(r pggen.QualityProfile) (QualityProfile, error) {
	var items []QualityProfileItem
	if err := json.Unmarshal(r.Items, &items); err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: unmarshal items: %w", err)
	}
	if items == nil {
		items = []QualityProfileItem{}
	}

	var formatItems []FormatScoreItem
	if err := json.Unmarshal(r.FormatItems, &formatItems); err != nil {
		return QualityProfile{}, fmt.Errorf("profiles: unmarshal format_items: %w", err)
	}
	if formatItems == nil {
		formatItems = []FormatScoreItem{}
	}

	return QualityProfile{
		ID:                int(r.ID),
		Name:              r.Name,
		UpgradeAllowed:    r.UpgradeAllowed,
		Cutoff:            int(r.Cutoff),
		Items:             items,
		MinFormatScore:    int(r.MinFormatScore),
		CutoffFormatScore: int(r.CutoffFormatScore),
		FormatItems:       formatItems,
	}, nil
}
