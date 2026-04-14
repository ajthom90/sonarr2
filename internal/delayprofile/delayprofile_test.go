package delayprofile_test

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/delayprofile"
)

func newStore(t *testing.T) delayprofile.Store {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{DSN: ":memory:", BusyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return delayprofile.NewSQLiteStore(pool)
}

func TestCRUDAndSeed(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	// Migration seeds one default profile.
	initial, _ := s.List(ctx)
	if len(initial) != 1 {
		t.Fatalf("expected 1 seeded profile, got %d", len(initial))
	}

	created, err := s.Create(ctx, delayprofile.Profile{
		EnableUsenet:      true,
		EnableTorrent:     true,
		PreferredProtocol: delayprofile.ProtocolTorrent,
		UsenetDelay:       60,
		TorrentDelay:      30,
		Order:             1,
		Tags:              []int{5, 6},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("want non-zero id")
	}

	got, _ := s.GetByID(ctx, created.ID)
	if got.PreferredProtocol != delayprofile.ProtocolTorrent {
		t.Errorf("preferredProtocol = %q", got.PreferredProtocol)
	}
	if got.ProtocolDelay(delayprofile.ProtocolUsenet) != 60 {
		t.Error("usenet delay wrong")
	}
	if got.ProtocolDelay(delayprofile.ProtocolTorrent) != 30 {
		t.Error("torrent delay wrong")
	}
}

func TestApplicableProfile(t *testing.T) {
	profiles := []delayprofile.Profile{
		{ID: 1, Order: 1, Tags: []int{10}, UsenetDelay: 120},
		{ID: 2, Order: 2, Tags: []int{20}, UsenetDelay: 60},
		{ID: 99, Order: 1 << 30, Tags: []int{}, UsenetDelay: 0}, // catch-all
	}

	tests := []struct {
		name   string
		tags   []int
		wantID int
	}{
		{"matches first", []int{10}, 1},
		{"matches second", []int{20, 30}, 2},
		{"falls through to catch-all", []int{99}, 99},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, ok := delayprofile.ApplicableProfile(profiles, tc.tags)
			if !ok {
				t.Fatal("expected a profile")
			}
			if p.ID != tc.wantID {
				t.Errorf("got ID %d, want %d", p.ID, tc.wantID)
			}
		})
	}
}
