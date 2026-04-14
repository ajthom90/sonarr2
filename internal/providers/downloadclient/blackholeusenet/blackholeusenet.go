// Package blackholeusenet is the Usenet variant of the Blackhole download
// client. Wraps internal/providers/downloadclient/blackhole with a fixed
// Usenet protocol and .nzb extension. Distinct from Blackhole itself so the
// UI shows "Usenet Blackhole" and "Torrent Blackhole" as separate options,
// matching Sonarr's provider registry.
package blackholeusenet

import (
	"context"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/blackhole"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Settings is a rename of the shared Blackhole settings for schema clarity.
type Settings = blackhole.Settings

// UsenetBlackhole wraps a base Blackhole client and forces ProtocolUsenet.
type UsenetBlackhole struct{ inner *blackhole.Blackhole }

// New constructs a Usenet Blackhole client.
func New(s Settings, client *http.Client) *UsenetBlackhole {
	return &UsenetBlackhole{inner: blackhole.New(s, client)}
}

func (u *UsenetBlackhole) Implementation() string             { return "UsenetBlackhole" }
func (u *UsenetBlackhole) DefaultName() string                { return "Usenet Blackhole" }
func (u *UsenetBlackhole) Settings() any                      { return u.inner.Settings() }
func (u *UsenetBlackhole) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }
func (u *UsenetBlackhole) Add(ctx context.Context, url, title string) (string, error) {
	return u.inner.Add(ctx, url, title)
}
func (u *UsenetBlackhole) Items(ctx context.Context) ([]downloadclient.Item, error) {
	return u.inner.Items(ctx)
}
func (u *UsenetBlackhole) Remove(ctx context.Context, id string, removeData bool) error {
	return u.inner.Remove(ctx, id, removeData)
}
func (u *UsenetBlackhole) Status(ctx context.Context) (downloadclient.Status, error) {
	return u.inner.Status(ctx)
}
func (u *UsenetBlackhole) Test(ctx context.Context) error { return u.inner.Test(ctx) }
