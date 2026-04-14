// Package rootfolder provides CRUD storage for Sonarr-compatible root folders.
// A root folder is a filesystem path under which Sonarr organizes series
// directories (e.g. "/tv", "/anime"). Root folders drive default path
// selection for adds, library scanning, and free-space reporting.
//
// Sonarr's root folder model is {id, path}; sonarr2 adds a created_at
// timestamp for debugging and auditability.
package rootfolder

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by Store lookups when the requested root folder does not exist.
var ErrNotFound = errors.New("rootfolder: not found")

// ErrAlreadyExists is returned when a root folder path collides with an existing one.
var ErrAlreadyExists = errors.New("rootfolder: already exists")

// RootFolder represents a single root folder row. Matches Sonarr's RootFolder
// model (src/NzbDrone.Core/RootFolders/RootFolder.cs), extended with a
// created_at timestamp.
type RootFolder struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
}

// Store provides CRUD access to root folders.
type Store interface {
	Create(ctx context.Context, path string) (RootFolder, error)
	Get(ctx context.Context, id int64) (RootFolder, error)
	GetByPath(ctx context.Context, path string) (RootFolder, error)
	List(ctx context.Context) ([]RootFolder, error)
	Delete(ctx context.Context, id int64) error
}
