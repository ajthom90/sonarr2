package iptorrents

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// compile-time interface check.
var _ indexer.Indexer = (*IPTorrents)(nil)

func TestIPTorrentsInterface(t *testing.T) {
	ipt := New(Settings{FeedURL: "https://iptorrents.com/t.rss?u=1;tp=abc", Cookie: "uid=1; pass=abc"}, nil)

	if ipt.Implementation() != "IPTorrents" {
		t.Errorf("Implementation() = %q, want IPTorrents", ipt.Implementation())
	}
	if ipt.Protocol() != indexer.ProtocolTorrent {
		t.Errorf("Protocol() = %q, want torrent", ipt.Protocol())
	}
	if !ipt.SupportsRss() {
		t.Error("SupportsRss() should be true")
	}
	if ipt.SupportsSearch() {
		t.Error("SupportsSearch() should be false for stub")
	}
}

func TestIPTorrentsTestReturnsNil(t *testing.T) {
	ipt := New(Settings{FeedURL: "https://iptorrents.com/t.rss", Cookie: "uid=1; pass=abc"}, nil)
	if err := ipt.Test(context.Background()); err != nil {
		t.Errorf("Test() should return nil for stub, got: %v", err)
	}
}
