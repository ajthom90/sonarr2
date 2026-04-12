# Milestone 7 — Metadata Source (TVDB)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the metadata source layer — the service that fetches series and episode information from TheTVDB. After M7, the system can search for new series by title, fetch full episode lists for a series, and populate the library's Series/Season/Episode tables from an external source. This is required by M8 (RSS sync) to match releases to known series.

**Architecture:** `internal/providers/metadatasource/` defines an abstract `MetadataSource` interface. `internal/providers/metadatasource/tvdb/` implements it against TheTVDB v4 API. A `RefreshSeriesMetadata` command handler fetches metadata for a series and upserts episodes into the library. The design spec calls for a TVDB proxy companion service — M7 ships the direct-to-TVDB implementation only; the proxy is M17.

---

## Layout

```
internal/
├── providers/metadatasource/
│   ├── metadatasource.go    # MetadataSource interface, SeriesInfo, EpisodeInfo types
│   ├── tvdb/
│   │   ├── tvdb.go          # TVDB v4 API client
│   │   ├── settings.go      # Settings struct
│   │   ├── auth.go          # JWT token management
│   │   ├── types.go         # TVDB API response types
│   │   └── tvdb_test.go     # httptest-based tests
│   └── metadatasource_test.go
├── commands/handlers/
│   └── refresh_series.go    # RefreshSeriesMetadata handler
│   └── refresh_series_test.go
└── app/
    └── app.go               # Wire metadata source + register handler
```

No new migrations — M7 uses existing Series/Season/Episode tables from M2.

---

## Task 1 — MetadataSource interface + types

Define the abstract interface and shared types.

**Files:** `internal/providers/metadatasource/metadatasource.go`

```go
package metadatasource

import (
    "context"
    "time"
)

// SeriesSearchResult is a brief result from searching for a series by title.
type SeriesSearchResult struct {
    TvdbID   int64
    Title    string
    Year     int
    Overview string
    Status   string
    Network  string
    Slug     string
}

// SeriesInfo is the full metadata for a single series.
type SeriesInfo struct {
    TvdbID   int64
    Title    string
    Year     int
    Overview string
    Status   string // Continuing, Ended, Upcoming
    Network  string
    Runtime  int
    AirTime  string
    Slug     string
    Genres   []string
}

// EpisodeInfo is metadata for a single episode.
type EpisodeInfo struct {
    TvdbID                int64
    SeasonNumber          int
    EpisodeNumber         int
    AbsoluteEpisodeNumber *int
    Title                 string
    Overview              string
    AirDate               *time.Time
}

// MetadataSource fetches series and episode information from an external
// provider (TVDB, TMDb, TVMaze, etc.).
type MetadataSource interface {
    // SearchSeries finds series matching the query string.
    SearchSeries(ctx context.Context, query string) ([]SeriesSearchResult, error)

    // GetSeries returns full metadata for a series by TVDB ID.
    GetSeries(ctx context.Context, tvdbID int64) (SeriesInfo, error)

    // GetEpisodes returns all episodes for a series.
    GetEpisodes(ctx context.Context, tvdbID int64) ([]EpisodeInfo, error)
}
```

Test: verify types compile and interface is satisfiable with a stub.

Commit: `feat(metadatasource): add MetadataSource interface and info types`

---

## Task 2 — TVDB v4 API client

Implement the TVDB v4 API client with JWT authentication.

**Files:** `internal/providers/metadatasource/tvdb/tvdb.go`, `settings.go`, `auth.go`, `types.go`, `tvdb_test.go`

### TVDB v4 API overview

- Base URL: `https://api4.thetvdb.com/v4`
- Auth: POST `/login` with `{"apikey": "..."}` → JWT token. Token expires after 30 days. Refresh by re-logging.
- Search: GET `/search?query={title}&type=series`
- Series: GET `/series/{id}`
- Episodes: GET `/series/{id}/episodes/default?page=0` (paginated)

### settings.go

```go
type Settings struct {
    ApiKey string `json:"apiKey" form:"password" label:"TVDB API Key" required:"true"`
}
```

### auth.go

Token management — cache the JWT, refresh when expired:

```go
type tokenCache struct {
    mu    sync.Mutex
    token string
    expiry time.Time
}

func (c *tokenCache) get(ctx context.Context, client *http.Client, baseURL, apiKey string) (string, error)
```

### types.go

TVDB API JSON response shapes:

```go
type tvdbSearchResponse struct {
    Data []tvdbSearchResult `json:"data"`
}

type tvdbSearchResult struct {
    TvdbID    string `json:"tvdb_id"`
    Name      string `json:"name"`
    Year      string `json:"year"`
    Overview  string `json:"overview"`
    Status    string `json:"status"`
    Network   string `json:"network"`
    Slug      string `json:"slug"`
}

type tvdbSeriesResponse struct {
    Data tvdbSeries `json:"data"`
}

type tvdbSeries struct {
    ID          int64    `json:"id"`
    Name        string   `json:"name"`
    Year        string   `json:"year"`
    Overview    string   `json:"overview"`
    Status      tvdbStatus `json:"status"`
    // etc.
}

type tvdbEpisodesResponse struct {
    Data struct {
        Episodes []tvdbEpisode `json:"episodes"`
    } `json:"data"`
    Links tvdbLinks `json:"links"`
}
```

**Note:** TVDB v4 API fields vary — the implementer should look at https://thetvdb.github.io/v4-api/ for the actual shapes. The test will use canned responses, so the exact field names only matter for the parser. The key contract is: our types.go defines the wire format, tvdb.go converts to `SeriesInfo`/`EpisodeInfo`.

### tvdb.go

```go
type Client struct {
    settings Settings
    http     *http.Client
    baseURL  string
    auth     tokenCache
}

func New(settings Settings, httpClient *http.Client) *Client
func (c *Client) SearchSeries(ctx, query) ([]SeriesSearchResult, error)
func (c *Client) GetSeries(ctx, tvdbID) (SeriesInfo, error)
func (c *Client) GetEpisodes(ctx, tvdbID) ([]EpisodeInfo, error)
```

For tests, use `httptest.NewServer` that serves canned JSON for `/v4/login`, `/v4/search`, `/v4/series/{id}`, `/v4/series/{id}/episodes/default`.

### Tests

- `TestTVDBSearchSeries` — canned search response, verify results
- `TestTVDBGetSeries` — canned series response
- `TestTVDBGetEpisodes` — canned episodes response with pagination (2 pages)
- `TestTVDBAuthCachesToken` — two calls reuse the same token (login called once)
- `TestTVDBAuthRefreshesExpiredToken` — expired token triggers re-login

Commit: `feat(metadatasource/tvdb): add TVDB v4 API client with JWT auth`

---

## Task 3 — RefreshSeriesMetadata command handler

A command handler that fetches metadata for a series from the metadata source and upserts episodes into the library.

**Files:** `internal/commands/handlers/refresh_series.go`, `refresh_series_test.go`

### Handler

```go
type RefreshSeriesHandler struct {
    source   metadatasource.MetadataSource
    library  *library.Library
}

func NewRefreshSeriesHandler(source metadatasource.MetadataSource, lib *library.Library) *RefreshSeriesHandler

func (h *RefreshSeriesHandler) Handle(ctx context.Context, cmd commands.Command) error {
    // 1. Parse cmd.Body for {"seriesId": 123}
    // 2. Get the series from library to find tvdbID
    // 3. Call source.GetSeries(ctx, tvdbID) → update series metadata
    // 4. Call source.GetEpisodes(ctx, tvdbID)
    // 5. For each episode: upsert into library (create if new, update if exists)
    // 6. Upsert seasons based on distinct season numbers from episodes
}
```

The body JSON: `{"seriesId": 123}`. The handler loads the series from the library by ID, gets its TVDB ID, fetches metadata, and syncs episodes.

### Test

Use a stub MetadataSource that returns canned SeriesInfo + EpisodeInfo. Use an in-memory SQLite library. Create a series first, then run the handler, verify episodes were created.

Commit: `feat(commands/handlers): add RefreshSeriesMetadata handler`

---

## Task 4 — Wire into app + register handler

Wire the TVDB metadata source and RefreshSeriesMetadata handler into app.New.

**Files:** Modify `internal/app/app.go`

```go
// In New:
tvdbSource := tvdb.New(tvdb.Settings{ApiKey: ""}, nil) // API key will come from config/UI later
refreshHandler := handlers.NewRefreshSeriesHandler(tvdbSource, lib)
reg.Register("RefreshSeriesMetadata", refreshHandler)
```

For M7, the TVDB API key is empty by default — users will configure it via the UI (M15) or env var (add `SONARR2_TVDB_API_KEY` to config). The handler gracefully errors if the key is missing.

Commit: `feat(app): wire TVDB metadata source and RefreshSeriesMetadata handler`

---

## Task 5 — Update README + final verification + push

1. Update README.md: add "Metadata source (TVDB)" to the implemented list, bump milestone count to 7
2. `go mod tidy`
3. `make lint`
4. `go test -race -count=1 -timeout 120s -short ./...`
5. `make clean && make build`
6. Smoke test with SQLite
7. `git push origin main`
8. Watch both CI workflows

---

## Done

After Task 5, the system can fetch series and episode metadata from TVDB. M8 (RSS sync + grab) connects the indexer → parser → decision engine → download client flow. M7's RefreshSeriesMetadata handler is the first command that talks to an external API and writes to the library — it proves the full metadata pipeline works.
