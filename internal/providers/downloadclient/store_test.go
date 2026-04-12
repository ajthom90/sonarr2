package downloadclient_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

func newTestDCStore(t *testing.T) downloadclient.InstanceStore {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return downloadclient.NewSQLiteInstanceStore(pool)
}

func TestDownloadClientStoreCRUD(t *testing.T) {
	store := newTestDCStore(t)
	ctx := context.Background()

	in := downloadclient.Instance{
		Name:                     "My SABnzbd",
		Implementation:           "SABnzbd",
		Settings:                 json.RawMessage(`{"host":"localhost","port":8080,"apiKey":"secret"}`),
		Enable:                   true,
		Priority:                 1,
		RemoveCompletedDownloads: true,
		RemoveFailedDownloads:    true,
	}

	// Create
	created, err := store.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("created.ID must be non-zero")
	}
	if created.Name != in.Name {
		t.Errorf("Name = %q, want %q", created.Name, in.Name)
	}
	if created.Implementation != in.Implementation {
		t.Errorf("Implementation = %q, want %q", created.Implementation, in.Implementation)
	}
	if created.Enable != in.Enable {
		t.Errorf("Enable = %v, want %v", created.Enable, in.Enable)
	}
	if created.Priority != in.Priority {
		t.Errorf("Priority = %d, want %d", created.Priority, in.Priority)
	}
	if created.RemoveCompletedDownloads != in.RemoveCompletedDownloads {
		t.Errorf("RemoveCompletedDownloads = %v, want %v", created.RemoveCompletedDownloads, in.RemoveCompletedDownloads)
	}
	if created.RemoveFailedDownloads != in.RemoveFailedDownloads {
		t.Errorf("RemoveFailedDownloads = %v, want %v", created.RemoveFailedDownloads, in.RemoveFailedDownloads)
	}

	// GetByID
	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != in.Name {
		t.Errorf("got.Name = %q, want %q", got.Name, in.Name)
	}
	if got.Priority != in.Priority {
		t.Errorf("got.Priority = %d, want %d", got.Priority, in.Priority)
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List len = %d, want 1", len(list))
	}

	// Update
	got.Name = "Updated SABnzbd"
	got.Enable = false
	got.Priority = 5
	got.RemoveCompletedDownloads = false
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := store.GetByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if after.Name != "Updated SABnzbd" {
		t.Errorf("Name after update = %q, want %q", after.Name, "Updated SABnzbd")
	}
	if after.Enable {
		t.Error("Enable after update = true, want false")
	}
	if after.Priority != 5 {
		t.Errorf("Priority after update = %d, want 5", after.Priority)
	}
	if after.RemoveCompletedDownloads {
		t.Error("RemoveCompletedDownloads after update = true, want false")
	}

	// Delete
	if err := store.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.GetByID(ctx, got.ID)
	if !errors.Is(err, downloadclient.ErrNotFound) {
		t.Errorf("GetByID after delete = %v, want ErrNotFound", err)
	}
}

func TestDownloadClientStoreGetByIDNotFound(t *testing.T) {
	store := newTestDCStore(t)
	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, downloadclient.ErrNotFound) {
		t.Errorf("GetByID(missing) = %v, want ErrNotFound", err)
	}
}

func TestDownloadClientStoreSettingsJSONRoundtrip(t *testing.T) {
	store := newTestDCStore(t)
	ctx := context.Background()

	settings := json.RawMessage(`{"host":"sabnzbd.local","port":8080,"apiKey":"myapikey","useSsl":true,"category":"tv"}`)

	created, err := store.Create(ctx, downloadclient.Instance{
		Name:           "Roundtrip SABnzbd",
		Implementation: "SABnzbd",
		Settings:       settings,
		Enable:         true,
		Priority:       1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	// Normalize both JSON values for comparison.
	var wantNorm, gotNorm any
	if err := json.Unmarshal(settings, &wantNorm); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if err := json.Unmarshal(got.Settings, &gotNorm); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	wantBytes, _ := json.Marshal(wantNorm)
	gotBytes, _ := json.Marshal(gotNorm)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Errorf("Settings roundtrip mismatch:\n  want %s\n   got %s", wantBytes, gotBytes)
	}
}
