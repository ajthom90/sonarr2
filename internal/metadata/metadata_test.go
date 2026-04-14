package metadata_test

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ajthom90/sonarr2/internal/metadata"
	"github.com/ajthom90/sonarr2/internal/metadata/kodi"
	"github.com/ajthom90/sonarr2/internal/metadata/plex"
	"github.com/ajthom90/sonarr2/internal/metadata/roksbox"
	"github.com/ajthom90/sonarr2/internal/metadata/wdtv"
)

func TestKodiEpisodeNfo(t *testing.T) {
	tmp := t.TempDir()
	videoPath := filepath.Join(tmp, "Show", "Season 01", "S01E01.mkv")
	mustMkdir(t, filepath.Dir(videoPath))

	c := kodi.New(kodi.Settings{EpisodeMetadata: true, EpisodeImages: true})
	err := c.OnEpisodeFileImport(context.Background(), metadata.Context{
		Series:      metadata.SeriesInfo{Title: "Show"},
		Episode:     metadata.EpisodeInfo{Title: "Pilot", SeasonNumber: 1, EpisodeNumber: 1, AirDate: "2020-01-01", Overview: "A pilot."},
		EpisodeFile: metadata.EpisodeFileInfo{Path: videoPath},
	})
	if err != nil {
		t.Fatalf("OnEpisodeFileImport: %v", err)
	}
	nfo, err := os.ReadFile(filepath.Join(tmp, "Show", "Season 01", "S01E01.nfo"))
	if err != nil {
		t.Fatalf("reading .nfo: %v", err)
	}
	if !strings.Contains(string(nfo), "<title>Pilot</title>") {
		t.Errorf("expected <title>Pilot</title> in nfo, got:\n%s", string(nfo))
	}
	if !strings.Contains(string(nfo), "<episodedetails>") {
		t.Errorf("expected episodedetails root element, got:\n%s", string(nfo))
	}
}

func TestKodiSeriesNfo(t *testing.T) {
	tmp := t.TempDir()
	c := kodi.New(kodi.Settings{SeriesMetadata: true})
	err := c.OnSeriesRefresh(context.Background(), metadata.SeriesInfo{
		Title:    "Breaking Bad",
		Path:     tmp,
		TvdbID:   81189,
		ImdbID:   "tt0903747",
		Year:     2008,
		Genres:   []string{"Crime", "Drama"},
		Overview: "A chemistry teacher turned meth cook.",
	})
	if err != nil {
		t.Fatalf("OnSeriesRefresh: %v", err)
	}
	// Validate the file parses as XML and contains expected fields.
	raw, err := os.ReadFile(filepath.Join(tmp, "tvshow.nfo"))
	if err != nil {
		t.Fatalf("reading tvshow.nfo: %v", err)
	}
	var got struct {
		XMLName xml.Name `xml:"tvshow"`
		Title   string   `xml:"title"`
	}
	if err := xml.Unmarshal(raw, &got); err != nil {
		t.Fatalf("tvshow.nfo did not parse: %v", err)
	}
	if got.Title != "Breaking Bad" {
		t.Errorf("title = %q, want Breaking Bad", got.Title)
	}
}

func TestWDTVAndRoksboxAndPlex(t *testing.T) {
	// Smoke: all three emit something without error.
	tmp := t.TempDir()
	video := filepath.Join(tmp, "S01E01.mkv")
	mustMkdir(t, tmp)

	ctx := metadata.Context{
		Episode:     metadata.EpisodeInfo{Title: "Pilot", SeasonNumber: 1, EpisodeNumber: 1},
		EpisodeFile: metadata.EpisodeFileInfo{Path: video},
	}

	for _, tc := range []struct {
		name string
		c    metadata.Consumer
	}{
		{"wdtv", wdtv.New(wdtv.Settings{EpisodeMetadata: true})},
		{"roksbox", roksbox.New(roksbox.Settings{EpisodeMetadata: true})},
		{"plex", plex.New(plex.Settings{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.c.OnEpisodeFileImport(context.Background(), ctx); err != nil {
				t.Fatalf("%s OnEpisodeFileImport: %v", tc.name, err)
			}
		})
	}
}

func TestRegistry(t *testing.T) {
	r := metadata.NewRegistry()
	r.Register("XbmcMetadata", func() metadata.Consumer {
		return kodi.New(kodi.Settings{})
	})
	if _, err := r.Build("unknown"); err == nil {
		t.Error("expected ErrNotFound for unknown identifier")
	}
	c, err := r.Build("XbmcMetadata")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if c.Implementation() != "XbmcMetadata" {
		t.Errorf("Implementation = %q", c.Implementation())
	}
	names := r.Names()
	if len(names) != 1 || names[0] != "XbmcMetadata" {
		t.Errorf("Names = %v", names)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
