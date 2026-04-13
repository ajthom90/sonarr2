# M21 — Migration Tool Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a `sonarr-migrate` CLI that imports data from an existing Sonarr v3/v4 SQLite database into sonarr2.

**Architecture:** A `Migrator` in `internal/migrate/` reads from the source Sonarr DB, maintains ID mappings, applies path remaps, and writes through sonarr2's store interfaces. A CLI in `cmd/sonarr-migrate/` parses flags and drives the migration.

**Tech Stack:** Go stdlib (`database/sql`, `encoding/json`, `flag`), `modernc.org/sqlite` (already a dep)

---

## File Map

| Action | Path | Responsibility |
|---|---|---|
| Create | `internal/migrate/migrate.go` | Migrator struct, Run orchestration, ID mapping |
| Create | `internal/migrate/source.go` | Source DB reading functions |
| Create | `internal/migrate/migrate_test.go` | Integration test with fixture DB |
| Create | `cmd/sonarr-migrate/main.go` | CLI entry point |
| Modify | `Makefile` | Add build target for sonarr-migrate |
| Modify | `README.md` | Update for M21 |

---

### Task 1: Source Reader

**Files:**
- Create: `internal/migrate/source.go`
- Create: `internal/migrate/source_test.go`

Read from a Sonarr SQLite database. Each function opens a read-only query and returns typed slices.

```go
package migrate

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Source types — match Sonarr's schema.

type SourceSeries struct {
	ID         int64
	TvdbID     int64
	Title      string
	CleanTitle string
	Status     string // "continuing", "ended", "upcoming", "deleted"
	Path       string
	Monitored  bool
	SeriesType string // "standard", "daily", "anime"
	Added      time.Time
}

type SourceSeason struct {
	ID           int64
	SeriesID     int64
	SeasonNumber int
	Monitored    bool
}

type SourceEpisode struct {
	ID                    int64
	SeriesID              int64
	SeasonNumber          int
	EpisodeNumber         int
	AbsoluteEpisodeNumber *int
	Title                 string
	Overview              string
	AirDateUtc            *time.Time
	Monitored             bool
	EpisodeFileID         *int64
}

type SourceEpisodeFile struct {
	ID           int64
	SeriesID     int64
	SeasonNumber int
	RelativePath string
	Size         int64
	DateAdded    time.Time
	ReleaseGroup string
	Quality      string // JSON blob — extract quality name
}

type SourceQualityProfile struct {
	ID             int64
	Name           string
	UpgradeAllowed bool
	Cutoff         int
	Items          string // JSON
}

type SourceIndexer struct {
	ID                      int64
	Name                    string
	Implementation          string
	Settings                string // JSON
	EnableRss               bool
	EnableAutomaticSearch   bool
	EnableInteractiveSearch bool
	Priority                int
}

type SourceDownloadClient struct {
	ID             int64
	Name           string
	Implementation string
	Settings       string // JSON
	Enable         bool
	Priority       int
}

type SourceNotification struct {
	ID             int64
	Name           string
	Implementation string
	Settings       string // JSON
	OnGrab         bool
	OnDownload     bool
	OnHealthIssue  bool
}

type SourceHistory struct {
	ID          int64
	EpisodeID   int64
	SeriesID    int64
	SourceTitle string
	Quality     string // JSON
	Date        time.Time
	EventType   string
	DownloadID  string
	Data        string // JSON
}

// Reader functions
func readSeries(ctx context.Context, db *sql.DB) ([]SourceSeries, error)
func readSeasons(ctx context.Context, db *sql.DB) ([]SourceSeason, error)
func readEpisodes(ctx context.Context, db *sql.DB) ([]SourceEpisode, error)
func readEpisodeFiles(ctx context.Context, db *sql.DB) ([]SourceEpisodeFile, error)
func readQualityProfiles(ctx context.Context, db *sql.DB) ([]SourceQualityProfile, error)
func readIndexers(ctx context.Context, db *sql.DB) ([]SourceIndexer, error)
func readDownloadClients(ctx context.Context, db *sql.DB) ([]SourceDownloadClient, error)
func readNotifications(ctx context.Context, db *sql.DB) ([]SourceNotification, error)
func readHistory(ctx context.Context, db *sql.DB) ([]SourceHistory, error)
func readAPIKey(ctx context.Context, db *sql.DB) (string, error)
```

Each reader function runs a `SELECT` query and scans rows into structs. Sonarr stores booleans as integers (0/1).

**Quality JSON parsing helper:**
```go
// extractQualityName parses Sonarr's quality JSON and returns the quality name.
// Example input: {"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080},...}
func extractQualityName(qualityJSON string) string {
	var q struct {
		Quality struct {
			Name string `json:"name"`
		} `json:"quality"`
	}
	if json.Unmarshal([]byte(qualityJSON), &q) == nil && q.Quality.Name != "" {
		return q.Quality.Name
	}
	return "Unknown"
}
```

**Tests:** Create a minimal fixture Sonarr DB in test setup using `CREATE TABLE` + `INSERT` statements, then verify each reader returns expected data.

Commit: `feat(migrate): add Sonarr source database reader`

---

### Task 2: Migrator Core

**Files:**
- Create: `internal/migrate/migrate.go`
- Create: `internal/migrate/migrate_test.go`

```go
package migrate

type Options struct {
	SourcePath  string
	DestPool    db.Pool
	Remaps      []PathRemap
	DryRun      bool
	SkipHistory bool
	Log         *slog.Logger
}

type PathRemap struct {
	Old string
	New string
}

type Report struct {
	Series          int `json:"series"`
	Seasons         int `json:"seasons"`
	Episodes        int `json:"episodes"`
	EpisodeFiles    int `json:"episodeFiles"`
	QualityProfiles int `json:"qualityProfiles"`
	Indexers        int `json:"indexers"`
	DownloadClients int `json:"downloadClients"`
	Notifications   int `json:"notifications"`
	History         int `json:"history"`
	Warnings        []string `json:"warnings,omitempty"`
}

type Migrator struct {
	source *sql.DB
	dest   db.Pool
	opts   Options

	// ID mappings (source → dest)
	seriesMap      map[int64]int64
	episodeMap     map[int64]int64
	episodeFileMap map[int64]int64
	profileMap     map[int64]int64
}
```

**Run method orchestration:**
1. Open source DB read-only
2. Read API key → upsert host config
3. Read quality profiles → write, build profileMap
4. Read series → apply path remaps → write, build seriesMap
5. Read seasons → map series IDs → write
6. Read episode files → map series IDs → write, build episodeFileMap
7. Read episodes → map series/file IDs → write, build episodeMap
8. Read indexers → write
9. Read download clients → write
10. Read notifications → write
11. If !skipHistory: read history → map IDs → write
12. Recompute all series statistics
13. Return Report

For dry-run: run all reads and transformations, log what would be written, but skip actual writes.

**Path remapping:**
```go
func (m *Migrator) remapPath(path string) string {
	for _, r := range m.opts.Remaps {
		if strings.HasPrefix(path, r.Old) {
			return r.New + path[len(r.Old):]
		}
	}
	return path
}
```

**Writing uses the existing store interfaces** — the Migrator gets stores from the dest pool using the same dialect-dispatch pattern as app.go. It calls `Create` for each entity.

**Test:** Create a source fixture DB with 2 series, 3 episodes, 1 file, 1 profile. Create a dest in-memory sonarr2 DB (migrate + seed). Run Migrator. Verify dest contains all data with correct ID mappings.

Commit: `feat(migrate): add Migrator with full import pipeline`

---

### Task 3: CLI Entry Point

**Files:**
- Create: `cmd/sonarr-migrate/main.go`
- Modify: `Makefile`

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/migrate"
)

func main() {
	source := flag.String("source", "", "Path to source Sonarr SQLite database (required)")
	dest := flag.String("dest", "", "Path to destination sonarr2 SQLite database (required)")
	var remaps remapFlags
	flag.Var(&remaps, "remap", "Path remapping old:new (repeatable)")
	dryRun := flag.Bool("dry-run", false, "Validate without writing")
	skipHistory := flag.Bool("skip-history", false, "Skip history import")
	verbose := flag.Bool("verbose", false, "Verbose logging")
	flag.Parse()

	if *source == "" || *dest == "" {
		fmt.Fprintln(os.Stderr, "Usage: sonarr-migrate --source <sonarr.db> --dest <sonarr2.db> [flags]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx := context.Background()

	// Open dest as sonarr2 SQLite pool, run migrations.
	destPool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN: "file:" + *dest + "?_journal=WAL&_busy_timeout=5000",
	})
	if err != nil {
		log.Error("open destination database", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer destPool.Close()

	if err := db.Migrate(ctx, destPool); err != nil {
		log.Error("migrate destination database", slog.String("err", err.Error()))
		os.Exit(1)
	}

	m, err := migrate.New(migrate.Options{
		SourcePath:  *source,
		DestPool:    destPool,
		Remaps:      remaps.parsed,
		DryRun:      *dryRun,
		SkipHistory: *skipHistory,
		Log:         log,
	})
	if err != nil {
		log.Error("initialize migrator", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer m.Close()

	report, err := m.Run(ctx)
	if err != nil {
		log.Error("migration failed", slog.String("err", err.Error()))
		os.Exit(1)
	}

	fmt.Printf("Migration complete:\n")
	fmt.Printf("  Series:          %d\n", report.Series)
	fmt.Printf("  Seasons:         %d\n", report.Seasons)
	fmt.Printf("  Episodes:        %d\n", report.Episodes)
	fmt.Printf("  Episode files:   %d\n", report.EpisodeFiles)
	fmt.Printf("  Quality profiles:%d\n", report.QualityProfiles)
	fmt.Printf("  Indexers:        %d\n", report.Indexers)
	fmt.Printf("  Download clients:%d\n", report.DownloadClients)
	fmt.Printf("  Notifications:   %d\n", report.Notifications)
	fmt.Printf("  History:         %d\n", report.History)
	if len(report.Warnings) > 0 {
		fmt.Printf("\nWarnings:\n")
		for _, w := range report.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}

// remapFlags implements flag.Value for repeatable --remap flags.
type remapFlags struct {
	parsed []migrate.PathRemap
}

func (f *remapFlags) String() string { return "" }

func (f *remapFlags) Set(val string) error {
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("remap must be old:new, got %q", val)
	}
	f.parsed = append(f.parsed, migrate.PathRemap{Old: parts[0], New: parts[1]})
	return nil
}
```

**Makefile** — add target:
```makefile
build-migrate:
	go build -o dist/sonarr-migrate ./cmd/sonarr-migrate
```

Commit: `feat(migrate): add sonarr-migrate CLI binary`

---

### Task 4: README Update

- Bump milestone to 21
- Add migration tool bullet
- Mention the `sonarr-migrate` binary

Commit: `docs: update README to reflect M21 progress`
