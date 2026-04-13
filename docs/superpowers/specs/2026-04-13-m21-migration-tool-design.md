# M21 — Migration Tool

## Overview

A `sonarr-migrate` CLI binary that reads an existing Sonarr v3/v4 SQLite database and imports series, episodes, episode files, quality profiles, and provider settings into a fresh sonarr2 database. Supports path remapping and dry-run validation.

## Architecture

### CLI

```
sonarr-migrate --source /path/to/sonarr.db --dest /path/to/sonarr2.db [flags]

Flags:
  --source <path>     Path to source Sonarr SQLite database (required)
  --dest <path>       Path to destination sonarr2 SQLite database (required)
  --remap <old:new>   Path remapping (repeatable, e.g., --remap /tv:/media/tv)
  --dry-run           Validate without writing
  --skip-history      Don't import history entries
  --verbose           Verbose logging
```

### Migration Package

`internal/migrate/` with:

```go
// Migrator orchestrates the migration from source to destination.
type Migrator struct {
    source    *sql.DB      // read-only connection to Sonarr DB
    dest      db.Pool      // sonarr2 DB pool
    remaps    []PathRemap  // path remapping rules
    dryRun    bool
    skipHist  bool
    log       *slog.Logger
}

type PathRemap struct {
    Old string
    New string
}

func New(opts Options) (*Migrator, error)
func (m *Migrator) Run(ctx context.Context) (*Report, error)
func (m *Migrator) Close() error
```

### Migration Order (Topological)

1. **Host config** — read API key from source, upsert into dest
2. **Quality definitions** — sonarr2 seeds defaults; skip import (definitions are standardized)
3. **Quality profiles** — read profiles + items, write with ID mapping
4. **Series** — read series, apply path remaps, write
5. **Seasons** — read seasons, map series IDs, write
6. **Episodes** — read episodes, map series IDs, write
7. **Episode files** — read files, map series IDs, write
8. **Indexers** — read indexer configs, write as sonarr2 instances
9. **Download clients** — read DC configs, write as sonarr2 instances
10. **Notifications** — read notification configs, write as sonarr2 instances
11. **History** — read history entries, map series/episode IDs, write (optional)

### Source Schema Reading

Sonarr v3/v4 SQLite tables (read-only):

- `Series` — Id, TvdbId, Title, CleanTitle, Status, Path, Monitored, SeriesType, Added, ...
- `Seasons` — Id, SeriesId, SeasonNumber, Monitored
- `Episodes` — Id, SeriesId, SeasonNumber, EpisodeNumber, AbsoluteEpisodeNumber, Title, Overview, AirDateUtc, Monitored, EpisodeFileId, ...
- `EpisodeFiles` — Id, SeriesId, SeasonNumber, RelativePath, Size, DateAdded, ReleaseGroup, Quality (JSON), ...
- `QualityProfiles` — Id, Name, UpgradeAllowed, Cutoff, Items (JSON)
- `Indexers` — Id, Name, Implementation, Settings (JSON), EnableRss, EnableAutomaticSearch, EnableInteractiveSearch, Priority
- `DownloadClients` — Id, Name, Implementation, Settings (JSON), Enable, Priority
- `Notifications` — Id, Name, Implementation, Settings (JSON), OnGrab, OnDownload, OnHealthIssue, ...
- `Config` — Key, Value (key-value pairs including ApiKey)
- `History` — Id, EpisodeId, SeriesId, SourceTitle, Quality (JSON), Date, EventType, DownloadId, Data (JSON)

### ID Mapping

Source IDs don't match dest IDs. The migrator maintains maps:
- `sourceSeriesID → destSeriesID`
- `sourceEpisodeID → destEpisodeID`
- `sourceEpisodeFileID → destEpisodeFileID`
- `sourceQualityProfileID → destQualityProfileID`

### Path Remapping

For each `--remap old:new` flag, series paths are transformed:
```go
func (m *Migrator) remapPath(path string) string {
    for _, r := range m.remaps {
        if strings.HasPrefix(path, r.Old) {
            return r.New + path[len(r.Old):]
        }
    }
    return path
}
```

### Quality JSON Parsing

Sonarr stores quality as JSON: `{"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080},...}`. The migrator extracts the quality name string for sonarr2's simpler `QualityName` field.

### Report

```go
type Report struct {
    Series       int
    Seasons      int
    Episodes     int
    EpisodeFiles int
    Profiles     int
    Indexers     int
    DownloadClients int
    Notifications   int
    History      int
    Warnings     []string
    Errors       []string
}
```

## Testing

- **Unit tests** with a minimal Sonarr SQLite fixture (created in test setup)
- **Integration test**: create a source DB with known data, run migration, verify dest contains expected data
- The user's live Sonarr instance is available for manual testing

## Out of Scope

- Postgres source database
- Custom format import (format differs between v3 and sonarr2)
- Blocklist, import lists, tags import
- Resume capability
- Remote path mappings
- Version detection / multi-version support (assumed v3/v4 compatible schema)
