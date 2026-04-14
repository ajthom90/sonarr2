package importlist_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ajthom90/sonarr2/internal/importlist"
)

func TestRegistryRoundTrip(t *testing.T) {
	r := importlist.NewRegistry()
	r.Register("TraktUserImport", func() importlist.ListProvider { return importlist.NewTraktUser() })
	r.Register("TraktListImport", func() importlist.ListProvider { return importlist.NewTraktList() })
	r.Register("AniListImport", func() importlist.ListProvider { return importlist.NewAniList() })
	r.Register("Rss", func() importlist.ListProvider { return importlist.NewRSS() })

	names := r.Names()
	if len(names) != 4 {
		t.Errorf("Names count = %d, want 4", len(names))
	}

	p, err := r.Build("AniListImport")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if p.Implementation() != "AniListImport" {
		t.Errorf("Implementation = %q", p.Implementation())
	}

	if _, err := r.Build("Unknown"); !errors.Is(err, importlist.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestStubFetchReturnsErrStub(t *testing.T) {
	// All providers are stubs for now; Fetch should return ErrStub cleanly.
	providers := []importlist.ListProvider{
		importlist.NewAniList(), importlist.NewMyAnimeList(),
		importlist.NewPlexWatchlist(), importlist.NewPlexRSS(),
		importlist.NewRSS(), importlist.NewSimkl(),
		importlist.NewSonarrImport(),
		importlist.NewTraktUser(), importlist.NewTraktList(), importlist.NewTraktPopular(),
		importlist.NewCustom(),
	}
	for _, p := range providers {
		_, err := p.Fetch(context.Background())
		if !errors.Is(err, importlist.ErrStub) {
			t.Errorf("%s Fetch returned %v, want ErrStub", p.Implementation(), err)
		}
	}
}
