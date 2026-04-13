// Package hostconfig owns the host_config entity and its Store interface.
// host_config is a singleton row holding the API key, authentication mode,
// and migration state. The Store interface has one implementation per
// database dialect.
package hostconfig

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

// ErrNotFound is returned by Store.Get when the host_config row does not exist.
var ErrNotFound = errors.New("hostconfig: not found")

// HostConfig is the singleton host configuration row.
type HostConfig struct {
	APIKey         string
	AuthMode       string
	MigrationState string
	TvdbApiKey     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Store reads and writes the host_config row.
type Store interface {
	// Get returns the current host config. If the row does not exist,
	// returns ErrNotFound.
	Get(ctx context.Context) (HostConfig, error)

	// Upsert inserts or updates the singleton host_config row. The
	// created_at / updated_at timestamps are managed by the database.
	Upsert(ctx context.Context, hc HostConfig) error
}

// NewAPIKey returns a cryptographically random 64-character hex API key
// suitable for first-run initialization of host_config.
func NewAPIKey() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is catastrophic; panic is the right response.
		panic("hostconfig: crypto/rand read failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
