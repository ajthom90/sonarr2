// Package customformats provides custom format matching, scoring, and storage.
// A custom format is a named set of specifications (regex rules, source/resolution
// matchers, etc.) that can be attached to quality profiles with numeric score weights.
// The decision engine uses these scores to rank releases.
package customformats

import "context"

// Specification is a single matching rule within a custom format.
// The Implementation field identifies which matching strategy to use.
type Specification struct {
	Name           string `json:"name"`
	Implementation string `json:"implementation"` // e.g. "ReleaseTitleSpecification"
	Negate         bool   `json:"negate"`
	Required       bool   `json:"required"`
	Value          string `json:"value"` // regex pattern or enum value
}

// CustomFormat is a named collection of specifications. A release matches a
// custom format when all of its specifications match (AND logic).
type CustomFormat struct {
	ID                  int
	Name                string
	IncludeWhenRenaming bool
	Specifications      []Specification
}

// Store provides CRUD access to custom formats.
type Store interface {
	Create(ctx context.Context, cf CustomFormat) (CustomFormat, error)
	GetByID(ctx context.Context, id int) (CustomFormat, error)
	List(ctx context.Context) ([]CustomFormat, error)
	Update(ctx context.Context, cf CustomFormat) error
	Delete(ctx context.Context, id int) error
}
