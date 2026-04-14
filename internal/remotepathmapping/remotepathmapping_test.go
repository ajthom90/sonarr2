package remotepathmapping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/remotepathmapping"
)

func newStore(t *testing.T) remotepathmapping.Store {
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
	return remotepathmapping.NewSQLiteStore(pool)
}

func TestCRUD(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	m, err := s.Create(ctx, remotepathmapping.Mapping{
		Host: "sab.local", RemotePath: "/downloads/", LocalPath: "/mnt/nas/downloads/",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.ID == 0 {
		t.Error("want non-zero id")
	}

	got, err := s.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Host != "sab.local" {
		t.Errorf("host = %q", got.Host)
	}

	if err := s.Update(ctx, remotepathmapping.Mapping{
		ID: m.ID, Host: "sab.local", RemotePath: "/dl/", LocalPath: "/nas/dl/",
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ = s.GetByID(ctx, m.ID)
	if got.RemotePath != "/dl/" {
		t.Errorf("remotePath = %q", got.RemotePath)
	}

	list, _ := s.List(ctx)
	if len(list) != 1 {
		t.Errorf("want 1, got %d", len(list))
	}

	if err := s.Delete(ctx, m.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.GetByID(ctx, m.ID); !errors.Is(err, remotepathmapping.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestTranslate(t *testing.T) {
	mappings := []remotepathmapping.Mapping{
		{Host: "sab.local", RemotePath: "/downloads/", LocalPath: "/mnt/nas/downloads/"},
		{Host: "qbit", RemotePath: "C:\\Torrents\\", LocalPath: "Z:\\mount\\torrents\\"},
	}
	cases := []struct {
		host, in, want string
	}{
		{"sab.local", "/downloads/Show.S01E01/", "/mnt/nas/downloads/Show.S01E01/"},
		{"SAB.LOCAL", "/downloads/foo", "/mnt/nas/downloads/foo"},
		{"qbit", "C:\\Torrents\\foo", "Z:\\mount\\torrents\\foo"},
		{"other", "/downloads/x", "/downloads/x"}, // no matching host
		{"sab.local", "/other/path", "/other/path"}, // host match but no prefix
		{"", "/downloads/x", "/downloads/x"},
	}
	for _, c := range cases {
		got := remotepathmapping.Translate(mappings, c.host, c.in)
		if got != c.want {
			t.Errorf("Translate(%q,%q) = %q, want %q", c.host, c.in, got, c.want)
		}
	}
}
