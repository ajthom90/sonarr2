package indexer

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Instance represents a configured indexer stored in the database.
type Instance struct {
	ID                      int
	Name                    string
	Implementation          string
	Settings                json.RawMessage
	EnableRss               bool
	EnableAutomaticSearch   bool
	EnableInteractiveSearch bool
	Priority                int
	Added                   time.Time
}

// InstanceStore provides CRUD access to indexer instances.
type InstanceStore interface {
	Create(ctx context.Context, inst Instance) (Instance, error)
	GetByID(ctx context.Context, id int) (Instance, error)
	List(ctx context.Context) ([]Instance, error)
	Update(ctx context.Context, inst Instance) error
	Delete(ctx context.Context, id int) error
}

// ErrNotFound is returned when a requested indexer instance does not exist.
var ErrNotFound = errors.New("indexer: not found")
