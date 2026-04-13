# Milestone 9 — Import Pipeline

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the download loop: when a download completes, scan the output folder, parse filenames, match to episodes, move/hardlink files into the series library folder, rename using configurable tokens, update episode_files records, and record history. After M9, a grabbed release goes from "downloading" to "in your library, properly organized."

**Architecture:** `internal/import/` owns the import pipeline. `internal/organizer/` handles filename building from naming tokens. A `ProcessDownload` command handler is triggered when the download monitoring detects a completed item.

---

## Layout

```
internal/
├── import/
│   ├── import.go            # ImportService: scan → parse → match → move
│   ├── import_test.go
│   └── specs/               # Import specifications (filters)
│       ├── not_sample.go
│       ├── not_unpacking.go
│       └── has_audio.go     # stub — full ffprobe in later milestone
├── organizer/
│   ├── filenambuilder.go    # Build destination filename from tokens
│   ├── tokens.go            # Token definitions
│   └── organizer_test.go
├── commands/handlers/
│   └── process_download.go  # ProcessDownload command handler
└── app/
    └── app.go               # Wire import service + register handler
```

No new migrations — uses existing episode_files and history tables from M2/M8.

---

## Task 1 — Organizer: filename builder with naming tokens

Build destination filenames from configurable naming tokens.

### tokens.go

```go
package organizer

// Token patterns that get replaced in the naming format string.
// Example format: "{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality Full}]"
const (
    TokenSeriesTitle   = "{Series Title}"
    TokenSeason        = "{season:00}"
    TokenEpisode       = "{episode:00}"
    TokenEpisodeTitle  = "{Episode Title}"
    TokenQualityFull   = "{Quality Full}"
    TokenReleaseGroup  = "{Release Group}"
    TokenAirDate       = "{Air-Date}"
)
```

### filenamebuilder.go

```go
package organizer

import "fmt"

type EpisodeInfo struct {
    SeriesTitle   string
    SeasonNumber  int
    EpisodeNumber int
    EpisodeTitle  string
    QualityName   string
    ReleaseGroup  string
    AirDate       string // YYYY-MM-DD
}

// DefaultEpisodeFormat is the standard naming format.
const DefaultEpisodeFormat = "{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality Full}]"

// BuildFilename applies the naming format to episode info, producing
// a filename (without extension).
func BuildFilename(format string, info EpisodeInfo) string {
    // Replace each token with the actual value.
    result := format
    result = strings.ReplaceAll(result, TokenSeriesTitle, info.SeriesTitle)
    result = strings.ReplaceAll(result, TokenSeason, fmt.Sprintf("%02d", info.SeasonNumber))
    result = strings.ReplaceAll(result, TokenEpisode, fmt.Sprintf("%02d", info.EpisodeNumber))
    result = strings.ReplaceAll(result, TokenEpisodeTitle, info.EpisodeTitle)
    result = strings.ReplaceAll(result, TokenQualityFull, info.QualityName)
    result = strings.ReplaceAll(result, TokenReleaseGroup, info.ReleaseGroup)
    result = strings.ReplaceAll(result, TokenAirDate, info.AirDate)
    // Clean illegal filename characters.
    return cleanFilename(result)
}

// BuildSeasonFolder returns the season subfolder name.
func BuildSeasonFolder(seasonNumber int) string {
    return fmt.Sprintf("Season %02d", seasonNumber)
}

func cleanFilename(name string) string {
    // Remove characters illegal in most filesystems.
    for _, c := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
        name = strings.ReplaceAll(name, c, "")
    }
    return strings.TrimSpace(name)
}
```

### Tests

- `TestBuildFilenameDefault` — default format with "The Simpsons" S35E10 "Pilot" → `"The Simpsons - S35E10 - Pilot [WEBDL-1080p]"`
- `TestBuildFilenameCleansIllegalChars` — title with `:` and `?` → stripped
- `TestBuildSeasonFolder` — season 1 → "Season 01", season 12 → "Season 12"

Commit: `feat(organizer): add filename builder with naming tokens`

---

## Task 2 — Import service

The core import logic: given a download folder path, scan files, parse, match, decide, move.

### import.go

```go
package importpkg

type Service struct {
    library      *library.Library
    history      history.Store
    bus          events.Bus
    log          *slog.Logger
}

func New(library *library.Library, history history.Store,
    bus events.Bus, log *slog.Logger) *Service

// ProcessFolder scans a folder for importable media files, matches them
// to episodes, and moves them into the library.
func (s *Service) ProcessFolder(ctx context.Context, downloadFolder string,
    seriesID int64, downloadID string) error {
    // 1. List files in downloadFolder (filter by media extensions: .mkv, .mp4, .avi)
    // 2. For each file:
    //    a. Parse filename → ParsedEpisodeInfo
    //    b. Match to episodes by (seriesID, season, episode)
    //    c. If no match, skip (log warning)
    //    d. Run import specs (NotSample, NotUnpacking)
    //    e. If rejected, skip
    //    f. Determine destination: series.Path / SeasonFolder / BuildFilename + ext
    //    g. Move or hardlink the file (hardlink if same filesystem, copy otherwise)
    //    h. Create episode_files record
    //    i. Update episode.episode_file_id
    //    j. Record history entry (EventDownloadImported)
    //    k. Publish EpisodeFileAdded event (triggers stats recompute)
    // 3. Return nil (individual file errors are logged, not fatal)
}
```

### File operations

```go
// moveFile moves src to dst. Tries hardlink first (instant, saves disk),
// falls back to copy+delete if hardlink fails (cross-device).
func moveFile(src, dst string) error {
    // Ensure destination directory exists.
    if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
        return err
    }
    // Try hardlink first.
    if err := os.Link(src, dst); err == nil {
        return nil
    }
    // Fallback: copy + delete.
    return copyFile(src, dst)
}

func copyFile(src, dst string) error {
    in, err := os.Open(src)
    if err != nil { return err }
    defer in.Close()
    out, err := os.Create(dst + ".partial")
    if err != nil { return err }
    if _, err := io.Copy(out, in); err != nil {
        out.Close()
        os.Remove(dst + ".partial")
        return err
    }
    out.Close()
    return os.Rename(dst + ".partial", dst)
}
```

### Import specs

Simple go/no-go filters:

- **NotSample** — reject files < 40MB (same logic as decision engine's NotSample but applied to actual file size)
- **NotUnpacking** — reject if `.r00`, `.rar`, `.part` files exist in the folder (still being extracted)
- **HasAudio** — stub for M9 that always accepts. Full ffprobe integration in a later milestone.

### Tests

Use `t.TempDir()` for real filesystem operations:

- `TestImportProcessFolderMovesFile` — create a fake .mkv file in a temp dir, set up a series + episode in SQLite, run ProcessFolder, verify the file was moved to the series path, episode_file_id is set, history entry exists
- `TestImportSkipsSampleFiles` — create a tiny file (100 bytes), verify it's skipped
- `TestImportHardlinksWhenSameFS` — verify hardlink used (file exists at both source and dest with same inode)
- `TestImportSkipsNonMediaFiles` — .txt and .nfo files ignored

Commit: `feat(import): add import service with file move/hardlink and episode matching`

---

## Task 3 — ProcessDownload command handler

A command handler that wraps the import service and is triggered when a download completes.

### process_download.go

```go
type ProcessDownloadHandler struct {
    importSvc *importpkg.Service
}

func NewProcessDownloadHandler(importSvc *importpkg.Service) *ProcessDownloadHandler

func (h *ProcessDownloadHandler) Handle(ctx context.Context, cmd commands.Command) error {
    var body struct {
        DownloadFolder string `json:"downloadFolder"`
        SeriesID       int64  `json:"seriesId"`
        DownloadID     string `json:"downloadId"`
    }
    json.Unmarshal(cmd.Body, &body)
    return h.importSvc.ProcessFolder(ctx, body.DownloadFolder, body.SeriesID, body.DownloadID)
}
```

Test: same as import tests but through the command handler interface.

Commit: `feat(commands/handlers): add ProcessDownload handler`

---

## Task 4 — Wire into app + README + push

1. Wire import service and ProcessDownload handler in app.New
2. Register: `reg.Register("ProcessDownload", processDownloadHandler)`
3. Update README: add import pipeline to implemented list, bump to M9
4. Final verification: tidy, lint, test, build, smoke test, push, CI watch

Commit: `feat(app): wire import service and ProcessDownload handler` then `docs: update README to reflect M9 progress`

---

## Done

After Task 4, the full download lifecycle works: RSS sync grabs a release → download client downloads it → ProcessDownload imports the completed file → file is renamed and moved into the series library → episode_files record created → stats recomputed. M10 (filesystem watcher) will automate the "detect completed download" trigger; for M9 the ProcessDownload command must be enqueued manually or by polling download client status.
