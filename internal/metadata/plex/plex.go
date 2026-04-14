// Package plex is a metadata.Consumer for Plex-style sidecar files.
// Plex mostly reads from TheTVDB/TMDB directly, but supports a subset of
// .nfo content and episode thumbnails; this consumer emits what Plex reads.
package plex

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/metadata"
)

// Settings for the Plex metadata consumer.
type Settings struct {
	EpisodeImages bool `json:"episodeImages" form:"checkbox" label:"Episode Images"`
}

type Plex struct {
	settings Settings
}

func New(s Settings) *Plex { return &Plex{settings: s} }

func (p *Plex) Implementation() string { return "MediaBrowserMetadata" }
func (p *Plex) DefaultName() string    { return "Plex" }
func (p *Plex) Settings() any          { return &p.settings }

func (p *Plex) OnEpisodeFileImport(_ context.Context, c metadata.Context) error {
	// Plex doesn't require sidecar .nfo files by default but can use them.
	// This emits a minimal <episode>.nfo for compatibility when Plex's agent
	// is set to "Local Media Assets" or similar.
	if c.EpisodeFile.Path == "" {
		return nil
	}
	return writeXML(replaceExt(c.EpisodeFile.Path, ".nfo"), struct {
		XMLName xml.Name `xml:"episodedetails"`
		Title   string   `xml:"title"`
		Season  int      `xml:"season"`
		Episode int      `xml:"episode"`
		Aired   string   `xml:"aired,omitempty"`
		Plot    string   `xml:"plot,omitempty"`
	}{
		Title:   c.Episode.Title,
		Season:  c.Episode.SeasonNumber,
		Episode: c.Episode.EpisodeNumber,
		Aired:   c.Episode.AirDate,
		Plot:    c.Episode.Overview,
	})
}

func (p *Plex) OnSeriesRefresh(_ context.Context, _ metadata.SeriesInfo) error {
	// Plex uses its own agent for series-level metadata by default.
	return nil
}

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return strings.TrimSuffix(path, ext) + newExt
}

func writeXML(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("plex metadata: mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("plex metadata: create: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(xml.Header); err != nil {
		return fmt.Errorf("plex metadata: header: %w", err)
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("plex metadata: encode: %w", err)
	}
	return enc.Flush()
}
