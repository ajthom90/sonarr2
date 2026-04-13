package notification_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

func newTestNotifStore(t *testing.T) notification.InstanceStore {
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
	return notification.NewSQLiteInstanceStore(pool)
}

func TestNotificationStoreCRUD(t *testing.T) {
	store := newTestNotifStore(t)
	ctx := context.Background()

	in := notification.Instance{
		Name:           "My Discord",
		Implementation: "Discord",
		Settings:       json.RawMessage(`{"webhookUrl":"https://discord.com/api/webhooks/123/abc"}`),
		OnGrab:         true,
		OnDownload:     true,
		OnHealthIssue:  false,
		Tags:           json.RawMessage(`[]`),
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
	if created.OnGrab != in.OnGrab {
		t.Errorf("OnGrab = %v, want %v", created.OnGrab, in.OnGrab)
	}
	if created.OnDownload != in.OnDownload {
		t.Errorf("OnDownload = %v, want %v", created.OnDownload, in.OnDownload)
	}
	if created.OnHealthIssue != in.OnHealthIssue {
		t.Errorf("OnHealthIssue = %v, want %v", created.OnHealthIssue, in.OnHealthIssue)
	}

	// GetByID
	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != in.Name {
		t.Errorf("got.Name = %q, want %q", got.Name, in.Name)
	}
	if got.OnGrab != in.OnGrab {
		t.Errorf("got.OnGrab = %v, want %v", got.OnGrab, in.OnGrab)
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
	got.Name = "Updated Discord"
	got.OnGrab = false
	got.OnHealthIssue = true
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := store.GetByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if after.Name != "Updated Discord" {
		t.Errorf("Name after update = %q, want %q", after.Name, "Updated Discord")
	}
	if after.OnGrab {
		t.Error("OnGrab after update = true, want false")
	}
	if !after.OnHealthIssue {
		t.Error("OnHealthIssue after update = false, want true")
	}

	// Delete
	if err := store.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.GetByID(ctx, got.ID)
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("GetByID after delete = %v, want ErrNotFound", err)
	}
}

func TestNotificationStoreGetByIDNotFound(t *testing.T) {
	store := newTestNotifStore(t)
	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, notification.ErrNotFound) {
		t.Errorf("GetByID(missing) = %v, want ErrNotFound", err)
	}
}

func TestNotificationStoreSettingsJSONRoundtrip(t *testing.T) {
	store := newTestNotifStore(t)
	ctx := context.Background()

	settings := json.RawMessage(`{"webhookUrl":"https://discord.com/api/webhooks/456/xyz","username":"Sonarr","avatar":"https://example.com/icon.png"}`)

	created, err := store.Create(ctx, notification.Instance{
		Name:           "Roundtrip Discord",
		Implementation: "Discord",
		Settings:       settings,
		OnGrab:         true,
		OnDownload:     true,
		OnHealthIssue:  true,
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

func TestNotificationStoreTagsJSONRoundtrip(t *testing.T) {
	store := newTestNotifStore(t)
	ctx := context.Background()

	tags := json.RawMessage(`[1,2,3]`)

	created, err := store.Create(ctx, notification.Instance{
		Name:           "Tagged Notification",
		Implementation: "Slack",
		Settings:       json.RawMessage(`{"webhookUrl":"https://hooks.slack.com/services/T00/B00/xxx"}`),
		OnGrab:         true,
		OnDownload:     false,
		OnHealthIssue:  true,
		Tags:           tags,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	var wantNorm, gotNorm any
	if err := json.Unmarshal(tags, &wantNorm); err != nil {
		t.Fatalf("unmarshal want tags: %v", err)
	}
	if err := json.Unmarshal(got.Tags, &gotNorm); err != nil {
		t.Fatalf("unmarshal got tags: %v", err)
	}
	wantBytes, _ := json.Marshal(wantNorm)
	gotBytes, _ := json.Marshal(gotNorm)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Errorf("Tags roundtrip mismatch:\n  want %s\n   got %s", wantBytes, gotBytes)
	}
}
