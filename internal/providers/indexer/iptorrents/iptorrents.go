// Package iptorrents implements a stub indexer.Indexer for IPTorrents.
// Full RSS + search integration is shipped in a follow-up PR.
package iptorrents

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// IPTorrents is a stub indexer for IPTorrents.
type IPTorrents struct {
	settings Settings
	client   *http.Client
}

// New constructs an IPTorrents indexer stub.
func New(settings Settings, client *http.Client) *IPTorrents {
	if client == nil {
		client = http.DefaultClient
	}
	return &IPTorrents{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (i *IPTorrents) Implementation() string { return "IPTorrents" }

// DefaultName satisfies providers.Provider.
func (i *IPTorrents) DefaultName() string { return "IPTorrents" }

// Settings satisfies providers.Provider.
func (i *IPTorrents) Settings() any { return &i.settings }

// Protocol satisfies indexer.Indexer.
func (i *IPTorrents) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// SupportsRss satisfies indexer.Indexer.
func (i *IPTorrents) SupportsRss() bool { return true }

// SupportsSearch satisfies indexer.Indexer — stub does not support search yet.
func (i *IPTorrents) SupportsSearch() bool { return false }

// FetchRss is not yet implemented.
func (i *IPTorrents) FetchRss(_ context.Context) ([]indexer.Release, error) {
	return nil, errors.New("iptorrents: not yet implemented")
}

// Search is not yet implemented.
func (i *IPTorrents) Search(_ context.Context, _ indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("iptorrents: not yet implemented")
}

// Test always returns nil for the stub.
func (i *IPTorrents) Test(_ context.Context) error { return nil }
