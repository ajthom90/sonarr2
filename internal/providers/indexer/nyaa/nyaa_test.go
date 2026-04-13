package nyaa

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// compile-time interface check.
var _ indexer.Indexer = (*Nyaa)(nil)

func TestNyaaInterface(t *testing.T) {
	n := New(Settings{BaseURL: "https://nyaa.si", Categories: "1_2", Filter: "2"}, nil)

	if n.Implementation() != "Nyaa" {
		t.Errorf("Implementation() = %q, want Nyaa", n.Implementation())
	}
	if n.Protocol() != indexer.ProtocolTorrent {
		t.Errorf("Protocol() = %q, want torrent", n.Protocol())
	}
	if !n.SupportsRss() {
		t.Error("SupportsRss() should be true")
	}
	if n.SupportsSearch() {
		t.Error("SupportsSearch() should be false for stub")
	}
}

func TestNyaaTestReturnsNil(t *testing.T) {
	n := New(Settings{BaseURL: "https://nyaa.si"}, nil)
	if err := n.Test(context.Background()); err != nil {
		t.Errorf("Test() should return nil for stub, got: %v", err)
	}
}
