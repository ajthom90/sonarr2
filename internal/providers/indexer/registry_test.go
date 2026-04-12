package indexer

import (
	"context"
	"strings"
	"testing"
)

// stubIndexer is a minimal implementation of Indexer used only in tests.
type stubIndexer struct{}

func (s *stubIndexer) Implementation() string                        { return "Stub" }
func (s *stubIndexer) DefaultName() string                           { return "Stub Indexer" }
func (s *stubIndexer) Settings() any                                 { return nil }
func (s *stubIndexer) Test(_ context.Context) error                  { return nil }
func (s *stubIndexer) Protocol() DownloadProtocol                    { return ProtocolUsenet }
func (s *stubIndexer) SupportsRss() bool                             { return true }
func (s *stubIndexer) SupportsSearch() bool                          { return true }
func (s *stubIndexer) FetchRss(_ context.Context) ([]Release, error) { return nil, nil }
func (s *stubIndexer) Search(_ context.Context, _ SearchRequest) ([]Release, error) {
	return nil, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register("Stub", func() Indexer { return &stubIndexer{} })

	f, err := r.Get("Stub")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	idx := f()
	if idx.Implementation() != "Stub" {
		t.Errorf("Implementation() = %q, want %q", idx.Implementation(), "Stub")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("DoesNotExist")
	if err == nil {
		t.Fatal("expected error for missing factory, got nil")
	}
	if !strings.Contains(err.Error(), "DoesNotExist") {
		t.Errorf("error message %q does not mention the missing key", err.Error())
	}
}

func TestRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register("A", func() Indexer { return &stubIndexer{} })
	r.Register("B", func() Indexer { return &stubIndexer{} })

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d entries, want 2", len(all))
	}
	for _, name := range []string{"A", "B"} {
		if _, ok := all[name]; !ok {
			t.Errorf("All() missing key %q", name)
		}
	}
}
