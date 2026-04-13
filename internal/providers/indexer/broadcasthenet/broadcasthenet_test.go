package broadcasthenet

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// compile-time interface check.
var _ indexer.Indexer = (*BroadcastheNet)(nil)

func TestBroadcastheNetInterface(t *testing.T) {
	b := New(Settings{ApiKey: "myapikey"}, nil)

	if b.Implementation() != "BroadcastheNet" {
		t.Errorf("Implementation() = %q, want BroadcastheNet", b.Implementation())
	}
	if b.Protocol() != indexer.ProtocolTorrent {
		t.Errorf("Protocol() = %q, want torrent", b.Protocol())
	}
	if b.SupportsRss() {
		t.Error("SupportsRss() should be false")
	}
	if !b.SupportsSearch() {
		t.Error("SupportsSearch() should be true")
	}
}

func TestBroadcastheNetTestReturnsNil(t *testing.T) {
	b := New(Settings{ApiKey: "myapikey"}, nil)
	if err := b.Test(context.Background()); err != nil {
		t.Errorf("Test() should return nil for stub, got: %v", err)
	}
}
