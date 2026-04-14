// Package roksbox is a metadata.Consumer for Roksbox (obsolete set-top-box
// media player). Emits <video>.xml sidecar files matching Roksbox's schema.
package roksbox

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/metadata"
)

// Settings for the Roksbox metadata consumer.
type Settings struct {
	EpisodeMetadata bool `json:"episodeMetadata" form:"checkbox" label:"Episode Metadata"`
	EpisodeImages   bool `json:"episodeImages" form:"checkbox" label:"Episode Images"`
	SeasonImages    bool `json:"seasonImages" form:"checkbox" label:"Season Images"`
}

type Roksbox struct {
	settings Settings
}

func New(s Settings) *Roksbox { return &Roksbox{settings: s} }

func (r *Roksbox) Implementation() string { return "RoksboxMetadata" }
func (r *Roksbox) DefaultName() string    { return "Roksbox" }
func (r *Roksbox) Settings() any          { return &r.settings }

type videoXML struct {
	XMLName     xml.Name `xml:"video"`
	Title       string   `xml:"title"`
	Season      int      `xml:"season"`
	Episode     int      `xml:"episode"`
	Description string   `xml:"description,omitempty"`
}

func (r *Roksbox) OnEpisodeFileImport(_ context.Context, c metadata.Context) error {
	if !r.settings.EpisodeMetadata || c.EpisodeFile.Path == "" {
		return nil
	}
	dest := replaceExt(c.EpisodeFile.Path, ".xml")
	return writeXML(dest, videoXML{
		Title:       c.Episode.Title,
		Season:      c.Episode.SeasonNumber,
		Episode:     c.Episode.EpisodeNumber,
		Description: c.Episode.Overview,
	})
}

func (r *Roksbox) OnSeriesRefresh(_ context.Context, _ metadata.SeriesInfo) error { return nil }

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return strings.TrimSuffix(path, ext) + newExt
}

func writeXML(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("roksbox: mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("roksbox: create: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(xml.Header); err != nil {
		return fmt.Errorf("roksbox: header: %w", err)
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("roksbox: encode: %w", err)
	}
	return enc.Flush()
}
