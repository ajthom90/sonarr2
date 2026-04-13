package notification

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Instance represents a configured notification provider stored in the database.
type Instance struct {
	ID             int
	Name           string
	Implementation string
	Settings       json.RawMessage
	OnGrab         bool
	OnDownload     bool
	OnHealthIssue  bool
	Tags           json.RawMessage
	Added          time.Time
}

// InstanceStore provides CRUD access to notification instances.
type InstanceStore interface {
	Create(ctx context.Context, inst Instance) (Instance, error)
	GetByID(ctx context.Context, id int) (Instance, error)
	List(ctx context.Context) ([]Instance, error)
	Update(ctx context.Context, inst Instance) error
	Delete(ctx context.Context, id int) error
}

// ErrNotFound is returned when a requested notification instance does not exist.
var ErrNotFound = errors.New("notification: not found")
