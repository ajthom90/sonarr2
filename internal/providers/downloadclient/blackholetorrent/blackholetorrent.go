// Package blackholetorrent is the Torrent variant of the Blackhole download
// client. Wraps internal/providers/downloadclient/blackhole with a fixed
// Torrent protocol and .torrent extension.
package blackholetorrent

import (
	"context"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/blackhole"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

type Settings = blackhole.Settings

type TorrentBlackhole struct{ inner *blackhole.Blackhole }

func New(s Settings, client *http.Client) *TorrentBlackhole {
	return &TorrentBlackhole{inner: blackhole.New(s, client)}
}

func (t *TorrentBlackhole) Implementation() string                { return "TorrentBlackhole" }
func (t *TorrentBlackhole) DefaultName() string                   { return "Torrent Blackhole" }
func (t *TorrentBlackhole) Settings() any                         { return t.inner.Settings() }
func (t *TorrentBlackhole) Protocol() indexer.DownloadProtocol    { return indexer.ProtocolTorrent }
func (t *TorrentBlackhole) Add(ctx context.Context, url, title string) (string, error) {
	return t.inner.Add(ctx, url, title)
}
func (t *TorrentBlackhole) Items(ctx context.Context) ([]downloadclient.Item, error) {
	return t.inner.Items(ctx)
}
func (t *TorrentBlackhole) Remove(ctx context.Context, id string, removeData bool) error {
	return t.inner.Remove(ctx, id, removeData)
}
func (t *TorrentBlackhole) Status(ctx context.Context) (downloadclient.Status, error) {
	return t.inner.Status(ctx)
}
func (t *TorrentBlackhole) Test(ctx context.Context) error { return t.inner.Test(ctx) }
