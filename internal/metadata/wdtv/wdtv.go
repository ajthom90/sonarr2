// Package wdtv is a metadata.Consumer for WDTV Live Hub / WDTV Live (legacy).
// Emits <episode>.xml sidecar files matching WDTV's schema.
package wdtv

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/metadata"
)

// Settings for the WDTV metadata consumer.
type Settings struct {
	EpisodeMetadata bool `json:"episodeMetadata" form:"checkbox" label:"Episode Metadata"`
	EpisodeImages   bool `json:"episodeImages" form:"checkbox" label:"Episode Images"`
}

type WDTV struct {
	settings Settings
}

func New(s Settings) *WDTV { return &WDTV{settings: s} }

func (w *WDTV) Implementation() string { return "WdtvMetadata" }
func (w *WDTV) DefaultName() string    { return "WDTV" }
func (w *WDTV) Settings() any          { return &w.settings }

type wdtvEpisodeXML struct {
	XMLName    xml.Name `xml:"details"`
	Title      string   `xml:"title"`
	Season     int      `xml:"season_number"`
	Episode    int      `xml:"episode_number"`
	FirstAired string   `xml:"firstaired,omitempty"`
	Overview   string   `xml:"overview,omitempty"`
	Runtime    int      `xml:"runtime,omitempty"`
}

func (w *WDTV) OnEpisodeFileImport(_ context.Context, c metadata.Context) error {
	if !w.settings.EpisodeMetadata || c.EpisodeFile.Path == "" {
		return nil
	}
	dest := replaceExt(c.EpisodeFile.Path, ".xml")
	return writeXML(dest, wdtvEpisodeXML{
		Title: c.Episode.Title, Season: c.Episode.SeasonNumber,
		Episode: c.Episode.EpisodeNumber, FirstAired: c.Episode.AirDate,
		Overview: c.Episode.Overview, Runtime: c.Episode.Runtime,
	})
}

func (w *WDTV) OnSeriesRefresh(_ context.Context, _ metadata.SeriesInfo) error { return nil }

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return strings.TrimSuffix(path, ext) + newExt
}

func writeXML(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("wdtv: mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("wdtv: create: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(xml.Header); err != nil {
		return fmt.Errorf("wdtv: header: %w", err)
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("wdtv: encode: %w", err)
	}
	return enc.Flush()
}
