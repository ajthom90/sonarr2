# Milestone 5 — Custom Formats + Decision Engine

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add quality profiles, custom formats, and the decision engine that filters and ranks releases. After M5, given a list of releases from an indexer, the system can decide which ones are acceptable, which are rejected (and why), and which one is the best. This is the "judgment" layer that M8 (RSS sync + grab) needs to pick the right download.

**Architecture:** Three new packages:

1. **`internal/profiles/`** — Quality profiles (ordered quality list with cutoff, upgrade allowed flag, min/max custom format scores). Migration + Store for `quality_profiles` and `quality_definitions` tables.
2. **`internal/customformats/`** — Custom format specifications (regex-based matching against parsed release info). Migration + Store for `custom_formats` table. Scoring via profile-attached weights.
3. **`internal/decisionengine/`** — Takes a release + profile + existing files and runs it through a chain of specifications that each return Accept or Reject(reason). Ranks surviving releases by quality → CF score → preference.

**Tech Stack:** Existing stack. 3 new migrations (00009-00011). New sqlc queries. Parser package (M4) feeds into the decision engine.

---

## Scope for M5 (focused MVP)

The full design spec lists ~30 decision specs. For M5 we implement the **core 8** that are testable without external services:

1. `QualityAllowed` — quality must be in profile's allowed list
2. `CustomFormatScore` — CF score must meet profile minimum
3. `UpgradeAllowed` — profile must permit upgrades if file exists
4. `Upgradable` — new release must be higher quality/CF than existing
5. `AcceptableSize` — release size between min/max for quality definition
6. `NotSample` — reject releases < 40MB (likely samples)
7. `Repack` — prefer repacks of same release group
8. `AlreadyImported` — skip if same release already in history

Remaining specs (Blocklist, Queue, FreeSpace, MinimumAge, Retention, Protocol, TorrentSeeding, etc.) require infrastructure from M6-M9 and will be added in those milestones. The engine is designed so specs can be added incrementally.

---

## Layout

```
internal/
├── profiles/
│   ├── quality.go            # QualityDefinition, QualityProfile types
│   ├── quality_store.go      # QualityStore interface
│   ├── quality_postgres.go
│   ├── quality_sqlite.go
│   └── quality_test.go
├── customformats/
│   ├── customformat.go       # CustomFormat, Specification types
│   ├── matcher.go            # Match(release, format) → bool
│   ├── scorer.go             # Score(release, formats, profile) → int
│   ├── store.go              # CustomFormatStore interface
│   ├── store_postgres.go
│   ├── store_sqlite.go
│   └── customformat_test.go
├── decisionengine/
│   ├── engine.go             # Engine type, Evaluate, Rank
│   ├── types.go              # Decision, Rejection, RemoteEpisode
│   ├── specs/                # One file per spec
│   │   ├── quality_allowed.go
│   │   ├── customformat_score.go
│   │   ├── upgrade_allowed.go
│   │   ├── upgradable.go
│   │   ├── acceptable_size.go
│   │   ├── not_sample.go
│   │   ├── repack.go
│   │   └── already_imported.go
│   └── engine_test.go
└── db/
    ├── migrations/{postgres,sqlite}/00009_quality_definitions.sql
    ├── migrations/{postgres,sqlite}/00010_quality_profiles.sql
    ├── migrations/{postgres,sqlite}/00011_custom_formats.sql
    ├── queries/{postgres,sqlite}/quality_definitions.sql
    ├── queries/{postgres,sqlite}/quality_profiles.sql
    ├── queries/{postgres,sqlite}/custom_formats.sql
    └── gen/ (regenerated)
```

---

## Task 1 — Migrations + queries + sqlc regen

Add `quality_definitions`, `quality_profiles`, and `custom_formats` tables.

### Postgres 00009_quality_definitions.sql

```sql
-- +goose Up
CREATE TABLE quality_definitions (
    id         SERIAL PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    source     TEXT NOT NULL,
    resolution TEXT NOT NULL,
    min_size   REAL NOT NULL DEFAULT 0,
    max_size   REAL NOT NULL DEFAULT 0,
    preferred_size REAL NOT NULL DEFAULT 0
);

-- Seed the standard quality definitions.
INSERT INTO quality_definitions (name, source, resolution, min_size, max_size, preferred_size) VALUES
    ('SDTV', 'television', '480p', 0.5, 100, 50),
    ('WEBDL-480p', 'webdl', '480p', 0.5, 100, 50),
    ('WEBRip-480p', 'webrip', '480p', 0.5, 100, 50),
    ('DVD', 'dvd', '480p', 0.5, 100, 50),
    ('HDTV-720p', 'television', '720p', 3, 200, 95),
    ('WEBDL-720p', 'webdl', '720p', 3, 200, 95),
    ('WEBRip-720p', 'webrip', '720p', 3, 200, 95),
    ('Bluray-720p', 'bluray', '720p', 3, 200, 95),
    ('HDTV-1080p', 'television', '1080p', 3, 400, 190),
    ('WEBDL-1080p', 'webdl', '1080p', 3, 400, 190),
    ('WEBRip-1080p', 'webrip', '1080p', 3, 400, 190),
    ('Bluray-1080p', 'bluray', '1080p', 3, 400, 190),
    ('Bluray-1080p Remux', 'remux', '1080p', 10, 600, 400),
    ('HDTV-2160p', 'television', '2160p', 10, 800, 400),
    ('WEBDL-2160p', 'webdl', '2160p', 10, 800, 400),
    ('WEBRip-2160p', 'webrip', '2160p', 10, 800, 400),
    ('Bluray-2160p', 'bluray', '2160p', 10, 800, 400),
    ('Bluray-2160p Remux', 'remux', '2160p', 20, 1200, 800);

-- +goose Down
DROP TABLE quality_definitions;
```

### SQLite 00009_quality_definitions.sql

Same content but `SERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT` and `REAL` stays (SQLite supports REAL natively).

### Postgres 00010_quality_profiles.sql

```sql
-- +goose Up
CREATE TABLE quality_profiles (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    upgrade_allowed  BOOLEAN NOT NULL DEFAULT TRUE,
    cutoff           INTEGER NOT NULL DEFAULT 0,
    items            JSONB NOT NULL DEFAULT '[]',
    min_format_score INTEGER NOT NULL DEFAULT 0,
    cutoff_format_score INTEGER NOT NULL DEFAULT 0,
    format_items     JSONB NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE quality_profiles;
```

`items` is a JSONB array of `{qualityId: int, allowed: bool}`. `format_items` is `{formatId: int, score: int}`.

### SQLite 00010_quality_profiles.sql

Same but `SERIAL` → `INTEGER PRIMARY KEY AUTOINCREMENT`, `BOOLEAN` → `INTEGER`, `JSONB` → `TEXT`.

### Postgres 00011_custom_formats.sql

```sql
-- +goose Up
CREATE TABLE custom_formats (
    id             SERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    include_when_renaming BOOLEAN NOT NULL DEFAULT FALSE,
    specifications JSONB NOT NULL DEFAULT '[]'
);

-- +goose Down
DROP TABLE custom_formats;
```

`specifications` is a JSON array of `{name, implementation, fields, negate, required}`.

### SQLite 00011_custom_formats.sql

Same pattern.

### Queries

**quality_definitions:** GetAll, GetByID, UpdateSizes (user can adjust min/max/preferred)
**quality_profiles:** Create, GetByID, List, Update, Delete
**custom_formats:** Create, GetByID, List, Update, Delete

Write full query files for both dialects. Standard CRUD patterns — the implementer writes them based on the schema.

### Steps

- [ ] Create 6 migration files
- [ ] Create 6 query files
- [ ] Run `sqlc generate`
- [ ] Verify migrations + compile
- [ ] Commit: `feat(db): add quality definitions, profiles, and custom formats tables`

---

## Task 2 — profiles package: QualityDefinition + QualityProfile

Define the domain types and Store interfaces for quality definitions and quality profiles.

**Files:** `internal/profiles/quality.go`, `quality_store.go`, `quality_postgres.go`, `quality_sqlite.go`, `quality_test.go`

### Key types

```go
type QualityDefinition struct {
    ID            int
    Name          string
    Source        string    // maps to parser.QualitySource
    Resolution    string    // maps to parser.Resolution
    MinSize       float64   // MB per minute of runtime
    MaxSize       float64
    PreferredSize float64
}

type QualityProfileItem struct {
    QualityID int
    Allowed   bool
}

type FormatScoreItem struct {
    FormatID int
    Score    int
}

type QualityProfile struct {
    ID                 int
    Name               string
    UpgradeAllowed     bool
    Cutoff             int       // quality definition ID that's "good enough"
    Items              []QualityProfileItem
    MinFormatScore     int
    CutoffFormatScore  int
    FormatItems        []FormatScoreItem
}
```

### Stores

```go
type QualityDefinitionStore interface {
    GetAll(ctx) ([]QualityDefinition, error)
    GetByID(ctx, id) (QualityDefinition, error)
}

type QualityProfileStore interface {
    Create(ctx, QualityProfile) (QualityProfile, error)
    GetByID(ctx, id) (QualityProfile, error)
    List(ctx) ([]QualityProfile, error)
    Update(ctx, QualityProfile) error
    Delete(ctx, id) error
}
```

Items/FormatItems are stored as JSONB in Postgres, TEXT in SQLite. The stores marshal/unmarshal them.

### Tests

- Seed quality definitions are loadable after migration
- QualityProfile CRUD roundtrip
- Items JSON marshaling works

### Steps

- [ ] Write failing tests
- [ ] Implement stores (both dialects)
- [ ] Verify all pass
- [ ] Commit: `feat(profiles): add QualityDefinition and QualityProfile stores`

---

## Task 3 — customformats package

Define `CustomFormat` with regex-based specifications and a `Match` function.

**Files:** `internal/customformats/customformat.go`, `matcher.go`, `scorer.go`, `store.go`, `store_postgres.go`, `store_sqlite.go`, `customformat_test.go`

### Key types

```go
type Specification struct {
    Name           string `json:"name"`
    Implementation string `json:"implementation"` // "ReleaseTitleSpecification", "SourceSpecification", etc.
    Negate         bool   `json:"negate"`
    Required       bool   `json:"required"`
    Value          string `json:"value"` // regex pattern or enum value
}

type CustomFormat struct {
    ID                    int
    Name                  string
    IncludeWhenRenaming   bool
    Specifications        []Specification
}
```

### Matcher

```go
// Match checks if a parsed release matches a custom format.
// All specifications must match (AND logic). Negated specs invert.
func Match(info parser.ParsedEpisodeInfo, cf CustomFormat) bool
```

Implementation types supported in M5:
- `ReleaseTitleSpecification` — regex match on `info.ReleaseTitle`
- `SourceSpecification` — match on `info.Quality.Source`
- `ResolutionSpecification` — match on `info.Quality.Resolution`
- `ReleaseGroupSpecification` — regex match on `info.ReleaseGroup`

### Scorer

```go
// Score computes the total custom format score for a release against a
// quality profile's format items.
func Score(info parser.ParsedEpisodeInfo, formats []CustomFormat, profile profiles.QualityProfile) int
```

Sums scores of all matching formats using the profile's `FormatItems` weights.

### Tests

- `TestMatchReleaseTitleSpec` — regex matches title
- `TestMatchNegatedSpec` — negated spec inverts
- `TestMatchAllSpecsRequired` — all specs must match (AND)
- `TestScoreMultipleFormats` — formats with weights produce correct total
- Store CRUD roundtrip

### Steps

- [ ] Write tests
- [ ] Implement matcher + scorer + store
- [ ] Commit: `feat(customformats): add regex-based custom format matching and scoring`

---

## Task 4 — Decision engine types + core

Define the engine and its public types.

**Files:** `internal/decisionengine/types.go`, `engine.go`

### Types

```go
type Decision int
const (
    Accept Decision = iota
    Reject
)

type RejectionType int
const (
    Permanent RejectionType = iota
    Temporary
)

type Rejection struct {
    Type   RejectionType
    Reason string
    Spec   string // which spec rejected it
}

// RemoteEpisode is a release from an indexer paired with its parsed info
// and the target series/episodes.
type RemoteEpisode struct {
    Release       Release
    ParsedInfo    parser.ParsedEpisodeInfo
    SeriesID      int64
    EpisodeIDs    []int64
    Quality       parser.ParsedQuality
    CustomFormats []int // matched CF IDs
    CFScore       int
}

type Release struct {
    Title    string
    Size     int64     // bytes
    Indexer  string
    Age      int       // days (usenet) or minutes (torrent)
    Protocol string    // usenet or torrent
    Seeders  int
}

// Spec is a single evaluation rule.
type Spec interface {
    Name() string
    Evaluate(ctx context.Context, remote RemoteEpisode, profile profiles.QualityProfile) (Decision, []Rejection)
}
```

### Engine

```go
type Engine struct {
    specs []Spec
}

func New(specs ...Spec) *Engine

// Evaluate runs all specs against the release. Returns all rejections
// (not just the first) so the UI can show "why wasn't this grabbed?"
func (e *Engine) Evaluate(ctx context.Context, remote RemoteEpisode, profile profiles.QualityProfile) (Decision, []Rejection)

// Rank sorts a slice of accepted RemoteEpisodes by preference.
// Higher is better: CF score → quality order → size proximity.
func (e *Engine) Rank(remotes []RemoteEpisode, profile profiles.QualityProfile) []RemoteEpisode
```

### Steps

- [ ] Write types + engine
- [ ] Write basic test: engine with no specs accepts everything
- [ ] Commit: `feat(decisionengine): add engine types and evaluate/rank framework`

---

## Task 5 — Decision specs (the core 8)

Implement the 8 M5-scope specs, each in its own file under `internal/decisionengine/specs/`.

**Files:** 8 spec files + `internal/decisionengine/engine_test.go`

### Spec implementations

Each spec is a struct satisfying the `Spec` interface. Quick summary:

1. **`QualityAllowed`** — checks `profile.Items` contains the release's quality ID with `Allowed=true`. Reject reason: "Quality X is not allowed in profile."
2. **`CustomFormatScore`** — checks `remote.CFScore >= profile.MinFormatScore`. Reject reason: "Custom format score N is below minimum M."
3. **`UpgradeAllowed`** — if an existing file exists and `profile.UpgradeAllowed == false`, reject. Needs: existing file quality passed via context or RemoteEpisode field.
4. **`Upgradable`** — if existing file's quality >= release quality, reject. Uses quality ordering from profile.
5. **`AcceptableSize`** — checks `release.Size` is between `qualityDef.MinSize` and `qualityDef.MaxSize` (scaled by episode runtime). For M5, use a simple byte range without runtime scaling.
6. **`NotSample`** — rejects releases < 40MB (`40 * 1024 * 1024` bytes).
7. **`Repack`** — if release is a repack (`ParsedQuality.Modifier == ModifierRepack`), verify the release group matches the existing file's group. If not, reject.
8. **`AlreadyImported`** — stub that always accepts (full implementation needs history store from M8).

### Tests

Table-driven per spec:
- `TestQualityAllowedAcceptsAllowed` / `RejectsDisallowed`
- `TestCustomFormatScoreAcceptsAboveMin` / `RejectsBelowMin`
- `TestUpgradeAllowedRejectsWhenDisabled`
- `TestUpgradableRejectsLowerQuality`
- `TestAcceptableSizeRejectsTooSmall` / `RejectsTooLarge`
- `TestNotSampleRejectsTiny`
- `TestRepackAcceptsMatchingGroup` / `RejectsNonMatchingGroup`

Plus an integration-style test: `TestEngineEvaluateFullPipeline` — create an engine with all 8 specs, feed it a well-formed release, verify it accepts.

### Steps

- [ ] Write tests for each spec
- [ ] Implement all 8 specs
- [ ] Write the pipeline test
- [ ] Commit: `feat(decisionengine): add 8 core decision specs`

---

## Task 6 — Release ranking

Implement the `Rank` function that sorts accepted releases by preference.

**Files:** Modify `internal/decisionengine/engine.go`, add ranking tests to `engine_test.go`

### Ranking order (highest priority first)

1. Custom format score (higher better)
2. Quality (per profile order — `Items` array index, lower index = higher preference)
3. Size proximity to preferred (closer to `qualityDef.PreferredSize` wins)

```go
func (e *Engine) Rank(remotes []RemoteEpisode, profile profiles.QualityProfile) []RemoteEpisode {
    sort.SliceStable(remotes, func(i, j int) bool {
        // CF score descending.
        if remotes[i].CFScore != remotes[j].CFScore {
            return remotes[i].CFScore > remotes[j].CFScore
        }
        // Quality order: lower index in profile.Items = better.
        qi := qualityIndex(remotes[i].Quality, profile)
        qj := qualityIndex(remotes[j].Quality, profile)
        if qi != qj {
            return qi < qj
        }
        // Size proximity to preferred (smaller diff = better).
        // Omitted for M5 — just use size descending as tiebreaker.
        return remotes[i].Release.Size > remotes[j].Release.Size
    })
    return remotes
}
```

### Tests

- `TestRankByCustomFormatScore` — 3 releases with different CF scores, ranked correctly
- `TestRankByQualityWhenCFEqual` — same CF score, different quality, ranked by profile order
- `TestRankTiebreaker` — same quality + CF, ranked by size

### Steps

- [ ] Implement Rank
- [ ] Write ranking tests
- [ ] Commit: `feat(decisionengine): add release ranking by CF score, quality, and size`

---

## Task 7 — Wire profiles + custom formats into app.New

Create profile and CF stores in `app.New`. Seed a default quality profile on first run.

**Files:** Modify `internal/app/app.go`

### Wiring

```go
// Create profile stores.
var qualityDefStore profiles.QualityDefinitionStore
var qualityProfileStore profiles.QualityProfileStore
var cfStore customformats.Store
// ... dialect dispatch ...
```

Seed a default "Any" quality profile that allows all qualities with upgrade enabled, cutoff at WEBDL-1080p. Only seed if no profiles exist.

### Steps

- [ ] Add stores to App struct
- [ ] Seed default profile
- [ ] Integration test: verify default profile exists after app.New
- [ ] Commit: `feat(app): wire quality profiles and custom format stores`

---

## Task 8 — Final verification + push

- [ ] `go mod tidy`
- [ ] `make lint`
- [ ] `go test -race -count=1 -timeout 120s -short ./...`
- [ ] `make clean && make build`
- [ ] `git log --oneline e724165..HEAD`
- [ ] `git push origin main`
- [ ] Watch both CI workflows

---

## Done

After Task 8, the system can evaluate releases against quality profiles and custom formats, rank them, and explain why rejects happened. M6 (provider SDK + first providers) will feed real releases from indexers into this engine, and M8 (RSS sync) will use the ranked output to pick the best download.
