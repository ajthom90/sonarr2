package downloadclient

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Instance represents a configured download client stored in the database.
type Instance struct {
	ID                       int
	Name                     string
	Implementation           string
	Settings                 json.RawMessage
	Enable                   bool
	Priority                 int
	RemoveCompletedDownloads bool
	RemoveFailedDownloads    bool
	Added                    time.Time
}

// InstanceStore provides CRUD access to download client instances.
type InstanceStore interface {
	Create(ctx context.Context, inst Instance) (Instance, error)
	GetByID(ctx context.Context, id int) (Instance, error)
	List(ctx context.Context) ([]Instance, error)
	Update(ctx context.Context, inst Instance) error
	Delete(ctx context.Context, id int) error
}

// ErrNotFound is returned when a requested download client instance does not exist.
var ErrNotFound = errors.New("downloadclient: not found")
