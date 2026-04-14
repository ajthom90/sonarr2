package releaseprofile_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/releaseprofile"
)

func newStore(t *testing.T) releaseprofile.Store {
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
	return releaseprofile.NewSQLiteStore(pool)
}

func TestCRUD(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	created, err := s.Create(ctx, releaseprofile.Profile{
		Name: "Prefer Remux", Enabled: true,
		Required: []string{"REMUX"}, Ignored: []string{"CAM", "TS"},
		IndexerID: 0, Tags: []int{1, 2},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("want non-zero id")
	}
	got, _ := s.GetByID(ctx, created.ID)
	if len(got.Required) != 1 || got.Required[0] != "REMUX" {
		t.Errorf("Required = %v", got.Required)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %v", got.Tags)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.GetByID(ctx, created.ID); !errors.Is(err, releaseprofile.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestMatch(t *testing.T) {
	p := releaseprofile.Profile{
		Name:     "p",
		Required: []string{"1080p", "/WEB-DL/"},
		Ignored:  []string{"CAM"},
	}
	cases := []struct {
		title string
		want  bool
	}{
		{"Show.S01E01.1080p.WEB-DL.x264", true},
		{"Show.S01E01.720p.WEB-DL.x264", false},          // missing 1080p
		{"Show.S01E01.1080p.HDTV.x264", false},           // missing WEB-DL regex
		{"Show.S01E01.1080p.WEB-DL.CAM.x264", false},     // ignored
		{"Show.S01E01.WEB-dl.1080P.x264", true},          // case-insensitive
	}
	for _, c := range cases {
		if got := releaseprofile.Match(p, c.title); got != c.want {
			t.Errorf("Match(%q) = %v, want %v", c.title, got, c.want)
		}
	}
}
