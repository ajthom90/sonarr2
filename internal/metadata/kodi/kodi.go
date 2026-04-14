// SPDX-License-Identifier: GPL-3.0-or-later
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Extras/Metadata/Consumers/Xbmc/).
// Copyright (c) Team Sonarr, licensed under GPL-3.0.

// Package kodi is a metadata.Consumer that writes Kodi/XBMC-compatible
// sidecar files (tvshow.nfo, <episode>.nfo) next to series and episode
// media files.
package kodi

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/metadata"
)

// Settings for the Kodi metadata consumer. Matches Sonarr's
// XbmcMetadataSettings.
type Settings struct {
	SeriesMetadata        bool `json:"seriesMetadata" form:"checkbox" label:"Series Metadata"`
	SeriesMetadataURL     bool `json:"seriesMetadataUrl" form:"checkbox" label:"Series Metadata URL"`
	SeriesMetadataEpisodeGuide bool `json:"seriesMetadataEpisodeGuide" form:"checkbox" label:"Series Metadata Episode Guide"`
	EpisodeMetadata       bool `json:"episodeMetadata" form:"checkbox" label:"Episode Metadata"`
	SeriesImages          bool `json:"seriesImages" form:"checkbox" label:"Series Images"`
	SeasonImages          bool `json:"seasonImages" form:"checkbox" label:"Season Images"`
	EpisodeImages         bool `json:"episodeImages" form:"checkbox" label:"Episode Images"`
}

// Kodi implements metadata.Consumer.
type Kodi struct {
	settings Settings
}

// New constructs a Kodi metadata consumer with the given settings.
func New(s Settings) *Kodi { return &Kodi{settings: s} }

func (k *Kodi) Implementation() string { return "XbmcMetadata" }
func (k *Kodi) DefaultName() string    { return "Kodi (XBMC)" }
func (k *Kodi) Settings() any          { return &k.settings }

// --- XML schemas Kodi consumes --------------------------------------------

// tvshowNfo is the XML schema Kodi reads from tvshow.nfo at the series root.
type tvshowNfo struct {
	XMLName      xml.Name `xml:"tvshow"`
	Title        string   `xml:"title"`
	Plot         string   `xml:"plot,omitempty"`
	Year         int      `xml:"year,omitempty"`
	Runtime      int      `xml:"runtime,omitempty"`
	Status       string   `xml:"status,omitempty"`
	Studio       string   `xml:"studio,omitempty"`
	MPAA         string   `xml:"mpaa,omitempty"`
	Genres       []string `xml:"genre,omitempty"`
	UniqueIDs    []uniqueID `xml:"uniqueid,omitempty"`
	TvdbID       string   `xml:"tvdbid,omitempty"`
	ImdbID       string   `xml:"imdb_id,omitempty"`
	Actors       []actor  `xml:"actor,omitempty"`
	EpisodeGuide *guide   `xml:"episodeguide,omitempty"`
}

type uniqueID struct {
	XMLName xml.Name `xml:"uniqueid"`
	Type    string   `xml:"type,attr"`
	Default bool     `xml:"default,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

type actor struct {
	XMLName xml.Name `xml:"actor"`
	Name    string   `xml:"name"`
	Role    string   `xml:"role"`
	Order   int      `xml:"order,omitempty"`
	Thumb   string   `xml:"thumb,omitempty"`
}

type guide struct {
	XMLName xml.Name `xml:"episodeguide"`
	URL     string   `xml:"url,omitempty"`
}

// episodeNfo is the XML schema Kodi reads from <episode>.nfo next to a video file.
type episodeNfo struct {
	XMLName   xml.Name `xml:"episodedetails"`
	Title     string   `xml:"title"`
	Season    int      `xml:"season"`
	Episode   int      `xml:"episode"`
	Aired     string   `xml:"aired,omitempty"`
	Plot      string   `xml:"plot,omitempty"`
	Runtime   int      `xml:"runtime,omitempty"`
	UniqueIDs []uniqueID `xml:"uniqueid,omitempty"`
	Thumb     string   `xml:"thumb,omitempty"`
}

// --- Consumer methods ------------------------------------------------------

// OnEpisodeFileImport writes <file>.nfo next to the imported media file.
func (k *Kodi) OnEpisodeFileImport(_ context.Context, c metadata.Context) error {
	if !k.settings.EpisodeMetadata {
		return nil
	}
	if c.EpisodeFile.Path == "" {
		return nil
	}
	nfoPath := replaceExt(c.EpisodeFile.Path, ".nfo")
	ep := episodeNfo{
		Title:   c.Episode.Title,
		Season:  c.Episode.SeasonNumber,
		Episode: c.Episode.EpisodeNumber,
		Aired:   c.Episode.AirDate,
		Plot:    c.Episode.Overview,
		Runtime: c.Episode.Runtime,
	}
	if c.Episode.ID > 0 {
		ep.UniqueIDs = append(ep.UniqueIDs, uniqueID{
			Type: "tvdb", Default: true, Value: fmt.Sprintf("%d", c.Episode.ID),
		})
	}
	if c.Episode.ScreenshotURL != "" && k.settings.EpisodeImages {
		ep.Thumb = c.Episode.ScreenshotURL
	}
	return writeXML(nfoPath, ep)
}

// OnSeriesRefresh writes tvshow.nfo at the series root.
func (k *Kodi) OnSeriesRefresh(_ context.Context, s metadata.SeriesInfo) error {
	if !k.settings.SeriesMetadata {
		return nil
	}
	if s.Path == "" {
		return nil
	}
	nfo := tvshowNfo{
		Title:   s.Title,
		Plot:    s.Overview,
		Year:    s.Year,
		Runtime: s.Runtime,
		Status:  s.Status,
		Studio:  s.Network,
		MPAA:    s.Certification,
		Genres:  s.Genres,
	}
	if s.TvdbID > 0 {
		nfo.TvdbID = fmt.Sprintf("%d", s.TvdbID)
		nfo.UniqueIDs = append(nfo.UniqueIDs, uniqueID{
			Type: "tvdb", Default: true, Value: fmt.Sprintf("%d", s.TvdbID),
		})
	}
	if s.ImdbID != "" {
		nfo.ImdbID = s.ImdbID
		nfo.UniqueIDs = append(nfo.UniqueIDs, uniqueID{Type: "imdb", Value: s.ImdbID})
	}
	for _, a := range s.Actors {
		nfo.Actors = append(nfo.Actors, actor{
			Name: a.Name, Role: a.Role, Order: a.Order, Thumb: a.ThumbURL,
		})
	}
	if k.settings.SeriesMetadataEpisodeGuide && s.TvdbID > 0 {
		nfo.EpisodeGuide = &guide{URL: fmt.Sprintf("https://api.thetvdb.com/series/%d/episodes", s.TvdbID)}
	}
	nfoPath := filepath.Join(s.Path, "tvshow.nfo")
	return writeXML(nfoPath, nfo)
}

// --- Helpers ---------------------------------------------------------------

// replaceExt replaces the extension on path with newExt (must include dot).
func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return path + newExt
	}
	return strings.TrimSuffix(path, ext) + newExt
}

// writeXML serializes v to XML (prefixed with the standard header) at path.
// The containing directory is created if needed.
func writeXML(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("metadata: mkdir: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("metadata: create %s: %w", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(xml.Header); err != nil {
		return fmt.Errorf("metadata: write header: %w", err)
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("metadata: encode: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return fmt.Errorf("metadata: flush: %w", err)
	}
	return f.Sync()
}
