package grab_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/grab"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// ---------------------------------------------------------------------------
// Stub download client
// ---------------------------------------------------------------------------

type fakeSettings struct {
	Host string `json:"host"`
}

type fakeDownloadClient struct {
	protocol   indexer.DownloadProtocol
	addErr     error
	addURL     string
	addTitle   string
	addCalled  bool
	downloadID string
	settings   fakeSettings
}

func (f *fakeDownloadClient) Implementation() string { return "FakeClient" }
func (f *fakeDownloadClient) DefaultName() string    { return "FakeClient" }
func (f *fakeDownloadClient) Settings() any          { return &f.settings }
func (f *fakeDownloadClient) Protocol() indexer.DownloadProtocol {
	return f.protocol
}
func (f *fakeDownloadClient) Test(_ context.Context) error { return nil }
func (f *fakeDownloadClient) Add(_ context.Context, url, title string) (string, error) {
	f.addCalled = true
	f.addURL = url
	f.addTitle = title
	if f.addErr != nil {
		return "", f.addErr
	}
	return f.downloadID, nil
}
func (f *fakeDownloadClient) Items(_ context.Context) ([]downloadclient.Item, error) {
	return nil, nil
}
func (f *fakeDownloadClient) Remove(_ context.Context, _ string, _ bool) error { return nil }
func (f *fakeDownloadClient) Status(_ context.Context) (downloadclient.Status, error) {
	return downloadclient.Status{}, nil
}

// ---------------------------------------------------------------------------
// Stub dcStore
// ---------------------------------------------------------------------------

type fakeDCStore struct {
	instances []downloadclient.Instance
}

func (s *fakeDCStore) Create(_ context.Context, inst downloadclient.Instance) (downloadclient.Instance, error) {
	inst.ID = len(s.instances) + 1
	s.instances = append(s.instances, inst)
	return inst, nil
}
func (s *fakeDCStore) GetByID(_ context.Context, id int) (downloadclient.Instance, error) {
	for _, i := range s.instances {
		if i.ID == id {
			return i, nil
		}
	}
	return downloadclient.Instance{}, downloadclient.ErrNotFound
}
func (s *fakeDCStore) List(_ context.Context) ([]downloadclient.Instance, error) {
	return s.instances, nil
}
func (s *fakeDCStore) Update(_ context.Context, inst downloadclient.Instance) error { return nil }
func (s *fakeDCStore) Delete(_ context.Context, _ int) error                        { return nil }

// ---------------------------------------------------------------------------
// Stub historyStore
// ---------------------------------------------------------------------------

type fakeHistoryStore struct {
	entries []history.Entry
}

func (h *fakeHistoryStore) Create(_ context.Context, e history.Entry) (history.Entry, error) {
	e.ID = int64(len(h.entries) + 1)
	h.entries = append(h.entries, e)
	return e, nil
}
func (h *fakeHistoryStore) ListForSeries(_ context.Context, _ int64) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) ListForEpisode(_ context.Context, _ int64) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) FindByDownloadID(_ context.Context, _ string) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) DeleteForSeries(_ context.Context, _ int64) error { return nil }
func (h *fakeHistoryStore) ListAll(_ context.Context) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) DeleteBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func buildService(
	t *testing.T,
	instances []downloadclient.Instance,
	fdc *fakeDownloadClient,
) (*grab.Service, *fakeHistoryStore, events.Bus) {
	t.Helper()
	store := &fakeDCStore{instances: instances}

	reg := downloadclient.NewRegistry()
	reg.Register("FakeClient", func() downloadclient.DownloadClient { return fdc })

	hist := &fakeHistoryStore{}
	bus := events.NewBus(4)

	svc := grab.New(store, reg, hist, bus, slog.Default())
	return svc, hist, bus
}

func usenetRelease(url, title string) indexer.Release {
	return indexer.Release{
		Title:       title,
		DownloadURL: url,
		Protocol:    indexer.ProtocolUsenet,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGrabSuccess(t *testing.T) {
	ctx := context.Background()
	fdc := &fakeDownloadClient{
		protocol:   indexer.ProtocolUsenet,
		downloadID: "nzo-abc123",
	}
	inst := downloadclient.Instance{
		ID:             1,
		Name:           "My SABnzbd",
		Implementation: "FakeClient",
		Settings:       json.RawMessage(`{"host":"localhost"}`),
		Enable:         true,
		Priority:       1,
	}

	svc, hist, bus := buildService(t, []downloadclient.Instance{inst}, fdc)

	// Subscribe to ReleasesGrabbed to verify event published.
	var capturedEvent *grab.ReleasesGrabbed
	events.SubscribeSync[grab.ReleasesGrabbed](bus, func(_ context.Context, e grab.ReleasesGrabbed) error {
		capturedEvent = &e
		return nil
	})

	rel := usenetRelease("https://example.com/nzb/show.nzb", "Show.S01E01.720p-GROUP")
	err := svc.Grab(ctx, rel, 10, []int64{100, 101}, "720p")
	if err != nil {
		t.Fatalf("Grab returned unexpected error: %v", err)
	}

	// Verify Add was called with correct args.
	if !fdc.addCalled {
		t.Error("Add was not called on the download client")
	}
	if fdc.addURL != rel.DownloadURL {
		t.Errorf("Add URL = %q, want %q", fdc.addURL, rel.DownloadURL)
	}
	if fdc.addTitle != rel.Title {
		t.Errorf("Add title = %q, want %q", fdc.addTitle, rel.Title)
	}

	// Verify history recorded (one entry per episode ID).
	if len(hist.entries) != 2 {
		t.Fatalf("history entries = %d, want 2", len(hist.entries))
	}
	for _, entry := range hist.entries {
		if entry.SeriesID != 10 {
			t.Errorf("history entry SeriesID = %d, want 10", entry.SeriesID)
		}
		if entry.DownloadID != "nzo-abc123" {
			t.Errorf("history entry DownloadID = %q, want nzo-abc123", entry.DownloadID)
		}
		if entry.EventType != history.EventGrabbed {
			t.Errorf("history entry EventType = %q, want %q", entry.EventType, history.EventGrabbed)
		}
		if entry.QualityName != "720p" {
			t.Errorf("history entry QualityName = %q, want 720p", entry.QualityName)
		}
	}

	// Verify event published.
	if capturedEvent == nil {
		t.Fatal("ReleasesGrabbed event was not published")
	}
	if capturedEvent.SeriesID != 10 {
		t.Errorf("event SeriesID = %d, want 10", capturedEvent.SeriesID)
	}
	if capturedEvent.DownloadID != "nzo-abc123" {
		t.Errorf("event DownloadID = %q, want nzo-abc123", capturedEvent.DownloadID)
	}
	if len(capturedEvent.EpisodeIDs) != 2 {
		t.Errorf("event EpisodeIDs len = %d, want 2", len(capturedEvent.EpisodeIDs))
	}
}

func TestGrabNoEnabledClient(t *testing.T) {
	ctx := context.Background()
	fdc := &fakeDownloadClient{protocol: indexer.ProtocolUsenet}

	// Disabled instance — should not be used.
	inst := downloadclient.Instance{
		ID:             1,
		Name:           "Disabled",
		Implementation: "FakeClient",
		Settings:       json.RawMessage(`{}`),
		Enable:         false, // disabled
		Priority:       1,
	}

	svc, hist, _ := buildService(t, []downloadclient.Instance{inst}, fdc)

	rel := usenetRelease("https://example.com/show.nzb", "Show.S01E01")
	err := svc.Grab(ctx, rel, 1, []int64{1}, "720p")
	if err == nil {
		t.Fatal("expected error when no enabled clients, got nil")
	}

	// No history should have been recorded.
	if len(hist.entries) != 0 {
		t.Errorf("history entries = %d, want 0", len(hist.entries))
	}
}

func TestGrabClientAddFails(t *testing.T) {
	ctx := context.Background()
	addErr := errors.New("SABnzbd API error: queue full")
	fdc := &fakeDownloadClient{
		protocol: indexer.ProtocolUsenet,
		addErr:   addErr,
	}
	inst := downloadclient.Instance{
		ID:             1,
		Name:           "My SABnzbd",
		Implementation: "FakeClient",
		Settings:       json.RawMessage(`{}`),
		Enable:         true,
		Priority:       1,
	}

	svc, hist, _ := buildService(t, []downloadclient.Instance{inst}, fdc)

	rel := usenetRelease("https://example.com/show.nzb", "Show.S01E01")
	err := svc.Grab(ctx, rel, 1, []int64{1}, "720p")
	if err == nil {
		t.Fatal("expected error when Add fails, got nil")
	}

	// No history should have been recorded on failure.
	if len(hist.entries) != 0 {
		t.Errorf("history entries = %d, want 0 on Add failure", len(hist.entries))
	}
}
