// Package nyaa implements a stub indexer.Indexer for Nyaa.si anime torrents.
// Full RSS + search integration is shipped in a follow-up PR.
package nyaa

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Nyaa is a stub indexer for Nyaa.si.
type Nyaa struct {
	settings Settings
	client   *http.Client
}

// New constructs a Nyaa indexer stub.
func New(settings Settings, client *http.Client) *Nyaa {
	if client == nil {
		client = http.DefaultClient
	}
	return &Nyaa{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (n *Nyaa) Implementation() string { return "Nyaa" }

// DefaultName satisfies providers.Provider.
func (n *Nyaa) DefaultName() string { return "Nyaa" }

// Settings satisfies providers.Provider.
func (n *Nyaa) Settings() any { return &n.settings }

// Protocol satisfies indexer.Indexer.
func (n *Nyaa) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// SupportsRss satisfies indexer.Indexer.
func (n *Nyaa) SupportsRss() bool { return true }

// SupportsSearch satisfies indexer.Indexer — stub does not support search yet.
func (n *Nyaa) SupportsSearch() bool { return false }

// FetchRss is not yet implemented.
func (n *Nyaa) FetchRss(_ context.Context) ([]indexer.Release, error) {
	return nil, errors.New("nyaa: not yet implemented")
}

// Search is not yet implemented.
func (n *Nyaa) Search(_ context.Context, _ indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("nyaa: not yet implemented")
}

// Test always returns nil for the stub.
func (n *Nyaa) Test(_ context.Context) error { return nil }
