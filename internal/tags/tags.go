// Package tags provides CRUD storage for Sonarr-compatible tags. A tag is a
// simple label (e.g. "anime", "4k") that can be attached to series, indexers,
// download clients, notifications, import lists, release profiles, delay
// profiles, and auto-tagging rules. Tags drive filtering and binding logic
// across the system.
//
// Sonarr's tag model is just {id, label}; sonarr2 mirrors that exactly.
package tags

import (
	"context"
	"errors"
	"strings"
)

// ErrNotFound is returned by Store lookups when the requested tag does not exist.
var ErrNotFound = errors.New("tags: not found")

// ErrDuplicateLabel is returned when a tag label collides with an existing one.
var ErrDuplicateLabel = errors.New("tags: duplicate label")

// Tag represents a single tag row. Matches Sonarr's Tag model
// (src/NzbDrone.Core/Tags/Tag.cs).
type Tag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// Store provides CRUD access to tags.
type Store interface {
	Create(ctx context.Context, label string) (Tag, error)
	GetByID(ctx context.Context, id int) (Tag, error)
	GetByLabel(ctx context.Context, label string) (Tag, error)
	List(ctx context.Context) ([]Tag, error)
	Update(ctx context.Context, t Tag) error
	Delete(ctx context.Context, id int) error
}

// NormalizeLabel lowercases and trims a tag label. Sonarr stores tags
// case-insensitively and normalizes on write.
func NormalizeLabel(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}
