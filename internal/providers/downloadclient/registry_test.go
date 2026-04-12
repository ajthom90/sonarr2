package downloadclient

import (
	"context"
	"strings"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// stubClient is a minimal implementation of DownloadClient used only in tests.
type stubClient struct{}

func (s *stubClient) Implementation() string             { return "Stub" }
func (s *stubClient) DefaultName() string                { return "Stub Client" }
func (s *stubClient) Settings() any                      { return nil }
func (s *stubClient) Test(_ context.Context) error       { return nil }
func (s *stubClient) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }
func (s *stubClient) Add(_ context.Context, _, _ string) (string, error) {
	return "dl-1", nil
}
func (s *stubClient) Items(_ context.Context) ([]Item, error)          { return nil, nil }
func (s *stubClient) Remove(_ context.Context, _ string, _ bool) error { return nil }
func (s *stubClient) Status(_ context.Context) (Status, error) {
	return Status{IsLocalhost: true}, nil
}

func TestDCRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register("Stub", func() DownloadClient { return &stubClient{} })

	f, err := r.Get("Stub")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	dc := f()
	if dc.Implementation() != "Stub" {
		t.Errorf("Implementation() = %q, want %q", dc.Implementation(), "Stub")
	}
}

func TestDCRegistryGetMissing(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("DoesNotExist")
	if err == nil {
		t.Fatal("expected error for missing factory, got nil")
	}
	if !strings.Contains(err.Error(), "DoesNotExist") {
		t.Errorf("error message %q does not mention the missing key", err.Error())
	}
}

func TestDCRegistryAll(t *testing.T) {
	r := NewRegistry()
	r.Register("A", func() DownloadClient { return &stubClient{} })
	r.Register("B", func() DownloadClient { return &stubClient{} })

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
