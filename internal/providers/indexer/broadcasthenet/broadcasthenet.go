// Package broadcasthenet implements a stub indexer.Indexer for BroadcastheNet.
// Full API integration is shipped in a follow-up PR.
package broadcasthenet

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// BroadcastheNet is a stub indexer for BroadcastheNet.
type BroadcastheNet struct {
	settings Settings
	client   *http.Client
}

// New constructs a BroadcastheNet indexer stub.
func New(settings Settings, client *http.Client) *BroadcastheNet {
	if client == nil {
		client = http.DefaultClient
	}
	return &BroadcastheNet{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (b *BroadcastheNet) Implementation() string { return "BroadcastheNet" }

// DefaultName satisfies providers.Provider.
func (b *BroadcastheNet) DefaultName() string { return "BroadcastheNet" }

// Settings satisfies providers.Provider.
func (b *BroadcastheNet) Settings() any { return &b.settings }

// Protocol satisfies indexer.Indexer.
func (b *BroadcastheNet) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// SupportsRss satisfies indexer.Indexer.
func (b *BroadcastheNet) SupportsRss() bool { return false }

// SupportsSearch satisfies indexer.Indexer.
func (b *BroadcastheNet) SupportsSearch() bool { return true }

// FetchRss is not yet implemented.
func (b *BroadcastheNet) FetchRss(_ context.Context) ([]indexer.Release, error) {
	return nil, errors.New("broadcasthenet: not yet implemented")
}

// Search is not yet implemented.
func (b *BroadcastheNet) Search(_ context.Context, _ indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("broadcasthenet: not yet implemented")
}

// Test always returns nil for the stub.
func (b *BroadcastheNet) Test(_ context.Context) error { return nil }
