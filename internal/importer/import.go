// Package importer scans completed download folders, matches media files to
// episodes, moves/hardlinks them into the library, and records history.
package importer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/organizer"
	"github.com/ajthom90/sonarr2/internal/parser"
)

const minSizeBytes = 40 * 1024 * 1024 // 40 MB — anything smaller is considered a sample.

// mediaExtensions lists file extensions treated as media files.
var mediaExtensions = map[string]bool{
	".mkv": true,
	".mp4": true,
	".avi": true,
	".ts":  true,
	".wmv": true,
	".flv": true,
}

// unpackingExtensions marks files that indicate an in-progress extraction.
var unpackingExtensions = map[string]bool{
	".rar":  true,
	".r00":  true,
	".part": true,
}

// Service handles importing completed downloads into the library.
type Service struct {
	library *library.Library
	history history.Store
	bus     events.Bus
	log     *slog.Logger
}

// New constructs an import Service.
func New(lib *library.Library, hist history.Store, bus events.Bus, log *slog.Logger) *Service {
	return &Service{
		library: lib,
		history: hist,
		bus:     bus,
		log:     log,
	}
}

// ProcessFolder scans downloadFolder for importable media files, matches each
// to an episode in the library, and moves the file to the library path.
// Individual file errors are logged and skipped; the overall call returns nil.
func (s *Service) ProcessFolder(ctx context.Context, downloadFolder string, seriesID int64, downloadID string) error {
	entries, err := os.ReadDir(downloadFolder)
	if err != nil {
		return fmt.Errorf("importer: read dir %q: %w", downloadFolder, err)
	}

	// Check for unpacking sentinel files (any .rar/.r00/.part in the folder).
	for _, e := range entries {
		if !e.IsDir() && unpackingExtensions[strings.ToLower(filepath.Ext(e.Name()))] {
			s.log.WarnContext(ctx, "importer: folder still unpacking, skipping",
				"folder", downloadFolder, "file", e.Name())
			return nil
		}
	}

	// Load the series so we know its library path and title.
	series, err := s.library.Series.Get(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("importer: get series %d: %w", seriesID, err)
	}

	// Load all episodes for the series for matching.
	episodes, err := s.library.Episodes.ListForSeries(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("importer: list episodes for series %d: %w", seriesID, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isMediaFile(name) {
			continue
		}

		srcPath := filepath.Join(downloadFolder, name)
		if err := s.importFile(ctx, srcPath, name, series, episodes, downloadID); err != nil {
			s.log.WarnContext(ctx, "importer: failed to import file",
				"file", srcPath, "error", err)
		}
	}
	return nil
}

// importFile handles a single media file: size-check, parse, match, move,
// and record keeping.
func (s *Service) importFile(
	ctx context.Context,
	srcPath, filename string,
	series library.Series,
	episodes []library.Episode,
	downloadID string,
) error {
	// Size check — reject samples.
	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("stat %q: %w", srcPath, err)
	}
	if info.Size() < minSizeBytes {
		s.log.InfoContext(ctx, "importer: skipping sample file",
			"file", srcPath, "size", info.Size())
		return nil
	}

	// Parse the filename to extract season/episode numbers.
	parsed := parser.ParseTitle(strings.TrimSuffix(filename, filepath.Ext(filename)))

	if len(parsed.EpisodeNumbers) == 0 {
		s.log.WarnContext(ctx, "importer: could not parse episode number from filename",
			"file", filename)
		return nil
	}

	// Match to an episode in the library.
	ep, ok := findEpisode(episodes, int32(parsed.SeasonNumber), int32(parsed.EpisodeNumbers[0]))
	if !ok {
		s.log.WarnContext(ctx, "importer: no matching episode found",
			"series_id", series.ID,
			"season", parsed.SeasonNumber,
			"episode", parsed.EpisodeNumbers[0],
		)
		return nil
	}

	// Build destination path.
	ext := filepath.Ext(filename)
	qualityName := qualityFullName(parsed.Quality)
	builtName := organizer.BuildFilename(organizer.DefaultEpisodeFormat, organizer.EpisodeInfo{
		SeriesTitle:   series.Title,
		SeasonNumber:  parsed.SeasonNumber,
		EpisodeNumber: parsed.EpisodeNumbers[0],
		EpisodeTitle:  ep.Title,
		QualityName:   qualityName,
		ReleaseGroup:  parsed.ReleaseGroup,
	})
	seasonFolder := organizer.BuildSeasonFolder(parsed.SeasonNumber)
	relativePath := filepath.Join(seasonFolder, builtName+ext)
	dstPath := filepath.Join(series.Path, relativePath)

	// Move (hardlink or copy) the file.
	if err := moveFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("move %q → %q: %w", srcPath, dstPath, err)
	}

	// Create episode_files record.
	ef, err := s.library.EpisodeFiles.Create(ctx, library.EpisodeFile{
		SeriesID:     series.ID,
		SeasonNumber: int32(parsed.SeasonNumber),
		RelativePath: relativePath,
		Size:         info.Size(),
		ReleaseGroup: parsed.ReleaseGroup,
		QualityName:  qualityName,
	})
	if err != nil {
		return fmt.Errorf("create episode_file record: %w", err)
	}

	// Update episode to point at the new file.
	ep.EpisodeFileID = &ef.ID
	if err := s.library.Episodes.Update(ctx, ep); err != nil {
		return fmt.Errorf("update episode %d: %w", ep.ID, err)
	}

	// Record history entry.
	if _, err := s.history.Create(ctx, history.Entry{
		EpisodeID:   ep.ID,
		SeriesID:    series.ID,
		SourceTitle: filename,
		QualityName: qualityName,
		EventType:   history.EventDownloadImported,
		DownloadID:  downloadID,
	}); err != nil {
		// History failure is non-fatal — log and continue.
		s.log.WarnContext(ctx, "importer: failed to record history",
			"episode_id", ep.ID, "error", err)
	}

	s.log.InfoContext(ctx, "importer: imported file",
		"src", srcPath, "dst", dstPath, "episode_id", ep.ID)
	return nil
}

// findEpisode returns the first episode whose season and episode numbers match.
func findEpisode(episodes []library.Episode, season, episode int32) (library.Episode, bool) {
	for _, ep := range episodes {
		if ep.SeasonNumber == season && ep.EpisodeNumber == episode {
			return ep, true
		}
	}
	return library.Episode{}, false
}

// isMediaFile reports whether filename has a known media extension.
func isMediaFile(name string) bool {
	return mediaExtensions[strings.ToLower(filepath.Ext(name))]
}

// qualityFullName converts a ParsedQuality into a human-readable string
// like "WEBDL-1080p". Returns "Unknown" when no quality was detected.
func qualityFullName(q parser.ParsedQuality) string {
	if q.Source == "" && q.Resolution == "" {
		return "Unknown"
	}
	parts := []string{}
	if q.Source != "" {
		parts = append(parts, string(q.Source))
	}
	if q.Resolution != "" {
		parts = append(parts, string(q.Resolution))
	}
	return strings.Join(parts, "-")
}

// moveFile moves src to dst. It tries a hardlink first (instant, same
// filesystem), falling back to copy+atomic-rename on cross-device moves.
func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// Try hardlink (instant, saves disk space when on the same filesystem).
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	// Fallback: copy to .partial then rename atomically.
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	partial := dst + ".partial"
	out, err := os.Create(partial)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(partial)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(partial)
		return err
	}
	return os.Rename(partial, dst)
}
