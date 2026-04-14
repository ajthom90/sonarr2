// SPDX-License-Identifier: GPL-3.0-or-later
// Helpers for stub download-client providers. Sonarr v5 ships ~17 DL clients;
// sonarr2 currently provides working implementations for the common ones
// (SAB, NZBGet, qBittorrent, Transmission, Deluge, Blackhole{Usenet,Torrent},
// Aria2) and registers stubs for the remaining ones so the provider schema
// exposed at /api/v3/downloadclient/schema is complete. Users migrating
// from Sonarr will see every identifier they recognize; stub providers
// surface a clear "not yet implemented" error when actioned, and the
// implementation can be filled in later without changing the schema.
package downloadclient

import (
	"context"
	"errors"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// ErrStub is returned by stub providers' action methods.
var ErrStub = errors.New("download client not yet implemented in sonarr2")

// Stub is the shared struct backing DL client stubs. Each provider package
// embeds it and overrides Implementation / DefaultName / Protocol / Settings.
type Stub struct{}

func (Stub) Test(context.Context) error                          { return ErrStub }
func (Stub) Add(context.Context, string, string) (string, error) { return "", ErrStub }
func (Stub) Items(context.Context) ([]Item, error)               { return nil, nil }
func (Stub) Remove(context.Context, string, bool) error          { return ErrStub }
func (Stub) Status(context.Context) (Status, error)              { return Status{}, nil }

// StubUsenet returns ProtocolUsenet — embed StubUsenet when the provider is
// Usenet-only (NZBVortex, Pneumatic, UsenetBlackhole, etc.).
type StubUsenet struct{ Stub }

func (StubUsenet) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }

// StubTorrent returns ProtocolTorrent.
type StubTorrent struct{ Stub }

func (StubTorrent) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }
