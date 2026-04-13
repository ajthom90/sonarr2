package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// --- stubs ---

// stubDCStore is an in-memory InstanceStore.
type stubDCStore struct {
	instances []downloadclient.Instance
}

func (s *stubDCStore) Create(_ context.Context, inst downloadclient.Instance) (downloadclient.Instance, error) {
	return inst, nil
}
func (s *stubDCStore) GetByID(_ context.Context, id int) (downloadclient.Instance, error) {
	return downloadclient.Instance{}, errors.New("not found")
}
func (s *stubDCStore) List(_ context.Context) ([]downloadclient.Instance, error) {
	return s.instances, nil
}
func (s *stubDCStore) Update(_ context.Context, inst downloadclient.Instance) error { return nil }
func (s *stubDCStore) Delete(_ context.Context, id int) error                       { return nil }

// stubDownloadClient is a minimal DownloadClient that returns a fixed item list.
type stubDownloadClient struct {
	items []downloadclient.Item
}

func (c *stubDownloadClient) Implementation() string { return "Stub" }
func (c *stubDownloadClient) DefaultName() string    { return "Stub" }
func (c *stubDownloadClient) Settings() any          { return &struct{}{} }
func (c *stubDownloadClient) Test(_ context.Context) error {
	return nil
}
func (c *stubDownloadClient) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }
func (c *stubDownloadClient) Add(_ context.Context, url, title string) (string, error) {
	return "", nil
}
func (c *stubDownloadClient) Items(_ context.Context) ([]downloadclient.Item, error) {
	return c.items, nil
}
func (c *stubDownloadClient) Remove(_ context.Context, downloadID string, deleteData bool) error {
	return nil
}
func (c *stubDownloadClient) Status(_ context.Context) (downloadclient.Status, error) {
	return downloadclient.Status{}, nil
}

// stubHistoryStore records calls to FindByDownloadID.
type stubHistoryStore struct {
	entries map[string][]history.Entry
}

func (s *stubHistoryStore) Create(_ context.Context, e history.Entry) (history.Entry, error) {
	return e, nil
}
func (s *stubHistoryStore) ListForSeries(_ context.Context, seriesID int64) ([]history.Entry, error) {
	return nil, nil
}
func (s *stubHistoryStore) ListForEpisode(_ context.Context, episodeID int64) ([]history.Entry, error) {
	return nil, nil
}
func (s *stubHistoryStore) FindByDownloadID(_ context.Context, downloadID string) ([]history.Entry, error) {
	return s.entries[downloadID], nil
}
func (s *stubHistoryStore) DeleteForSeries(_ context.Context, seriesID int64) error { return nil }
func (s *stubHistoryStore) ListAll(_ context.Context) ([]history.Entry, error)      { return nil, nil }

// stubQueue records Enqueue calls and allows FindDuplicate to be controlled.
type stubQueue struct {
	enqueued      []commands.Command
	duplicateKeys map[string]bool
}

func (q *stubQueue) Enqueue(_ context.Context, name string, body json.RawMessage, p commands.Priority, t commands.Trigger, dedupKey string) (commands.Command, error) {
	cmd := commands.Command{
		ID:       int64(len(q.enqueued) + 1),
		Name:     name,
		Body:     body,
		Priority: p,
		Trigger:  t,
		DedupKey: dedupKey,
		Status:   commands.StatusQueued,
		QueuedAt: time.Now(),
	}
	q.enqueued = append(q.enqueued, cmd)
	return cmd, nil
}
func (q *stubQueue) Claim(_ context.Context, workerID string, leaseDuration time.Duration) (*commands.Command, error) {
	return nil, nil
}
func (q *stubQueue) Complete(_ context.Context, id, durationMs int64, result json.RawMessage, message string) error {
	return nil
}
func (q *stubQueue) Fail(_ context.Context, id, durationMs int64, exception, message string) error {
	return nil
}
func (q *stubQueue) RefreshLease(_ context.Context, id int64, leaseDuration time.Duration) error {
	return nil
}
func (q *stubQueue) SweepExpiredLeases(_ context.Context) (int64, error) { return 0, nil }
func (q *stubQueue) FindDuplicate(_ context.Context, dedupKey string) (int64, bool, error) {
	if q.duplicateKeys != nil && q.duplicateKeys[dedupKey] {
		return 1, true, nil
	}
	return 0, false, nil
}
func (q *stubQueue) DeleteOldCompleted(_ context.Context, olderThan time.Time) (int64, error) {
	return 0, nil
}
func (q *stubQueue) Get(_ context.Context, id int64) (commands.Command, error) {
	return commands.Command{}, errors.New("not found")
}
func (q *stubQueue) ListRecent(_ context.Context, _ int) ([]commands.Command, error) {
	return nil, nil
}

// --- helpers ---

// makeRefreshHandler builds a handler wired to controlled stubs.
func makeRefreshHandler(
	instances []downloadclient.Instance,
	factory downloadclient.Factory,
	histEntries map[string][]history.Entry,
	duplicateKeys map[string]bool,
) (*RefreshMonitoredDownloadsHandler, *stubQueue) {
	dcStore := &stubDCStore{instances: instances}
	dcReg := downloadclient.NewRegistry()
	if factory != nil {
		dcReg.Register("Stub", factory)
	}
	q := &stubQueue{duplicateKeys: duplicateKeys}
	hist := &stubHistoryStore{entries: histEntries}
	log := slogDiscard()
	return NewRefreshMonitoredDownloadsHandler(dcStore, dcReg, q, hist, log), q
}

// --- tests ---

// TestRefreshDownloadsEnqueuesForCompleted verifies that a completed item with a
// matching grab history entry results in a ProcessDownload command being enqueued
// with the correct JSON body.
func TestRefreshDownloadsEnqueuesForCompleted(t *testing.T) {
	const downloadID = "dl-123"
	const seriesID = int64(42)
	const outputPath = "/tmp/downloads/show"

	instances := []downloadclient.Instance{{
		ID:             1,
		Name:           "MyClient",
		Implementation: "Stub",
		Enable:         true,
	}}

	factory := downloadclient.Factory(func() downloadclient.DownloadClient {
		return &stubDownloadClient{
			items: []downloadclient.Item{{
				DownloadID: downloadID,
				Title:      "Some.Show.S01E01",
				Status:     "completed",
				OutputPath: outputPath,
			}},
		}
	})

	histEntries := map[string][]history.Entry{
		downloadID: {{
			ID:         1,
			SeriesID:   seriesID,
			EpisodeID:  10,
			DownloadID: downloadID,
			EventType:  history.EventGrabbed,
		}},
	}

	handler, q := makeRefreshHandler(instances, factory, histEntries, nil)
	ctx := context.Background()

	if err := handler.Handle(ctx, commands.Command{Name: "RefreshMonitoredDownloads"}); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(q.enqueued) != 1 {
		t.Fatalf("enqueued count = %d, want 1", len(q.enqueued))
	}

	cmd := q.enqueued[0]
	if cmd.Name != "ProcessDownload" {
		t.Errorf("command name = %q, want ProcessDownload", cmd.Name)
	}
	if cmd.DedupKey != "ProcessDownload:"+downloadID {
		t.Errorf("dedup key = %q, want ProcessDownload:%s", cmd.DedupKey, downloadID)
	}

	// Verify the JSON body.
	var body struct {
		DownloadFolder string `json:"downloadFolder"`
		SeriesID       int64  `json:"seriesId"`
		DownloadID     string `json:"downloadId"`
	}
	if err := json.Unmarshal(cmd.Body, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body.DownloadFolder != outputPath {
		t.Errorf("downloadFolder = %q, want %q", body.DownloadFolder, outputPath)
	}
	if body.SeriesID != seriesID {
		t.Errorf("seriesId = %d, want %d", body.SeriesID, seriesID)
	}
	if body.DownloadID != downloadID {
		t.Errorf("downloadId = %q, want %q", body.DownloadID, downloadID)
	}
}

// TestRefreshDownloadsSkipsInProgress verifies that an item with status
// "downloading" does not result in any command being enqueued.
func TestRefreshDownloadsSkipsInProgress(t *testing.T) {
	instances := []downloadclient.Instance{{
		ID:             1,
		Name:           "MyClient",
		Implementation: "Stub",
		Enable:         true,
	}}

	factory := downloadclient.Factory(func() downloadclient.DownloadClient {
		return &stubDownloadClient{
			items: []downloadclient.Item{{
				DownloadID: "dl-456",
				Title:      "Some.Show.S01E02",
				Status:     "downloading",
				OutputPath: "/tmp/downloads/show2",
			}},
		}
	})

	handler, q := makeRefreshHandler(instances, factory, nil, nil)
	ctx := context.Background()

	if err := handler.Handle(ctx, commands.Command{Name: "RefreshMonitoredDownloads"}); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(q.enqueued) != 0 {
		t.Errorf("enqueued count = %d, want 0", len(q.enqueued))
	}
}

// TestRefreshDownloadsSkipsDuplicate verifies that when FindDuplicate returns
// true for the dedup key no additional command is enqueued.
func TestRefreshDownloadsSkipsDuplicate(t *testing.T) {
	const downloadID = "dl-789"
	const seriesID = int64(99)

	instances := []downloadclient.Instance{{
		ID:             1,
		Name:           "MyClient",
		Implementation: "Stub",
		Enable:         true,
	}}

	factory := downloadclient.Factory(func() downloadclient.DownloadClient {
		return &stubDownloadClient{
			items: []downloadclient.Item{{
				DownloadID: downloadID,
				Title:      "Some.Show.S01E03",
				Status:     "completed",
				OutputPath: "/tmp/downloads/show3",
			}},
		}
	})

	histEntries := map[string][]history.Entry{
		downloadID: {{
			ID:         2,
			SeriesID:   seriesID,
			EpisodeID:  11,
			DownloadID: downloadID,
			EventType:  history.EventGrabbed,
		}},
	}

	// Mark the dedup key as already present.
	duplicateKeys := map[string]bool{
		"ProcessDownload:" + downloadID: true,
	}

	handler, q := makeRefreshHandler(instances, factory, histEntries, duplicateKeys)
	ctx := context.Background()

	if err := handler.Handle(ctx, commands.Command{Name: "RefreshMonitoredDownloads"}); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(q.enqueued) != 0 {
		t.Errorf("enqueued count = %d, want 0 (duplicate should be skipped)", len(q.enqueued))
	}
}
