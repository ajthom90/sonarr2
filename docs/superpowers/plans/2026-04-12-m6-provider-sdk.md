# Milestone 6 — Provider SDK + First Providers

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the provider SDK — the pluggable integration surface for indexers and download clients. Ship two reference providers: Newznab (indexer) and SABnzbd (download client). After M6, the system can search an indexer for releases and send a grab to a download client.

**Architecture:** `internal/providers/` defines the interfaces; each provider is a separate subpackage. Provider instances are stored in the DB (migration for `indexers` and `download_clients` tables). A reflection-based schema generator produces JSON form schemas so the frontend can render settings forms dynamically. Status tracking uses a decorator pattern for failure backoff.

---

## Layout

```
internal/
├── providers/
│   ├── provider.go          # Provider interface, Settings, SchemaFor
│   ├── schema.go            # Reflection-based form schema generation
│   ├── schema_test.go
│   ├── indexer/
│   │   ├── indexer.go       # Indexer interface, Release type, SearchRequest
│   │   ├── registry.go      # Register/Get/All
│   │   └── newznab/
│   │       ├── newznab.go   # Newznab implementation
│   │       ├── settings.go  # Newznab settings struct with form tags
│   │       ├── parser.go    # RSS/search XML response parser
│   │       └── newznab_test.go
│   └── downloadclient/
│       ├── downloadclient.go # DownloadClient interface, Item type
│       ├── registry.go
│       └── sabnzbd/
│           ├── sabnzbd.go
│           ├── settings.go
│           └── sabnzbd_test.go
└── db/
    ├── migrations/{postgres,sqlite}/00012_indexers.sql
    ├── migrations/{postgres,sqlite}/00013_download_clients.sql
    ├── queries/...
    └── gen/ (regenerated)
```

---

## Task 1 — Migrations + queries + sqlc

Add `indexers` and `download_clients` tables. Both follow the same shape: id, name, implementation, settings (JSONB/TEXT), enable flags, priority, tags, added timestamp.

### Postgres 00012_indexers.sql

```sql
-- +goose Up
CREATE TABLE indexers (
    id                        SERIAL PRIMARY KEY,
    name                      TEXT NOT NULL,
    implementation            TEXT NOT NULL,
    settings                  JSONB NOT NULL DEFAULT '{}',
    enable_rss                BOOLEAN NOT NULL DEFAULT TRUE,
    enable_automatic_search   BOOLEAN NOT NULL DEFAULT TRUE,
    enable_interactive_search BOOLEAN NOT NULL DEFAULT TRUE,
    priority                  INTEGER NOT NULL DEFAULT 25,
    added                     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE indexers;
```

### Postgres 00013_download_clients.sql

```sql
-- +goose Up
CREATE TABLE download_clients (
    id              SERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    implementation  TEXT NOT NULL,
    settings        JSONB NOT NULL DEFAULT '{}',
    enable          BOOLEAN NOT NULL DEFAULT TRUE,
    priority        INTEGER NOT NULL DEFAULT 1,
    remove_completed_downloads BOOLEAN NOT NULL DEFAULT TRUE,
    remove_failed_downloads    BOOLEAN NOT NULL DEFAULT TRUE,
    added           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE download_clients;
```

SQLite variants: `INTEGER PRIMARY KEY AUTOINCREMENT`, `INTEGER` for booleans, `TEXT` for JSONB.

Queries: standard CRUD (Create, GetByID, List, Update, Delete) for both tables.

### Steps

- [ ] Create 4 migration files + 4 query files
- [ ] `sqlc generate`, verify compile
- [ ] Run tests
- [ ] Commit: `feat(db): add indexers and download_clients tables`

---

## Task 2 — Provider SDK: interfaces + schema generator

Define the core provider interfaces and the reflection-based form schema generator.

**Files:** `internal/providers/provider.go`, `schema.go`, `schema_test.go`

### provider.go

```go
package providers

import "context"

// Provider is the base interface all provider kinds extend.
type Provider interface {
    Implementation() string
    DefaultName() string
    Settings() any
    Test(ctx context.Context) error
}
```

### schema.go

Walks struct tags on a settings struct and emits a JSON-serializable schema:

```go
type FieldSchema struct {
    Name        string   `json:"name"`
    Label       string   `json:"label"`
    Type        string   `json:"type"`        // text, password, number, checkbox, select, multiselect
    Required    bool     `json:"required,omitempty"`
    Default     string   `json:"default,omitempty"`
    Placeholder string   `json:"placeholder,omitempty"`
    HelpText    string   `json:"helpText,omitempty"`
    Advanced    bool     `json:"advanced,omitempty"`
}

type Schema struct {
    Fields []FieldSchema `json:"fields"`
}

// SchemaFor generates a form schema from a settings struct's field tags.
// Tags: form:"text", label:"URL", required:"true", default:"/api", etc.
func SchemaFor(settings any) Schema
```

Uses `reflect` to walk exported fields. Each field with a `form` tag becomes a `FieldSchema` entry.

### Tests

- `TestSchemaForGeneratesFields` — pass a struct with 3 tagged fields, verify 3 FieldSchema entries with correct types/labels
- `TestSchemaForSkipsUntaggedFields` — fields without `form` tag are omitted
- `TestSchemaForHandlesAllTypes` — text, password, number, checkbox, select

### Steps

- [ ] Write tests
- [ ] Implement SchemaFor
- [ ] Commit: `feat(providers): add Provider interface and reflection-based schema generator`

---

## Task 3 — Indexer interface + registry

Define the indexer-specific interface and a type-safe registry.

**Files:** `internal/providers/indexer/indexer.go`, `registry.go`

### indexer.go

```go
package indexer

import (
    "context"
    "time"

    "github.com/ajthom90/sonarr2/internal/providers"
)

// DownloadProtocol identifies usenet vs torrent.
type DownloadProtocol string

const (
    ProtocolUsenet  DownloadProtocol = "usenet"
    ProtocolTorrent DownloadProtocol = "torrent"
)

// Release is a single result from an indexer search or RSS feed.
type Release struct {
    Title       string
    GUID        string
    DownloadURL string
    InfoURL     string
    Size        int64
    PublishDate time.Time
    Indexer     string
    Protocol    DownloadProtocol
    Seeders     int
    Leechers    int
    Categories  []int
}

// SearchRequest describes what to search for.
type SearchRequest struct {
    SeriesTitle string
    TvdbID      int64
    Season      int
    Episode     int
    Categories  []int
}

// Indexer extends Provider with indexer-specific methods.
type Indexer interface {
    providers.Provider
    Protocol() DownloadProtocol
    SupportsRss() bool
    SupportsSearch() bool
    FetchRss(ctx context.Context) ([]Release, error)
    Search(ctx context.Context, req SearchRequest) ([]Release, error)
}
```

### registry.go

```go
package indexer

type Factory func() Indexer

type Registry struct {
    factories map[string]Factory
}

func NewRegistry() *Registry
func (r *Registry) Register(name string, f Factory)
func (r *Registry) Get(name string) (Factory, error)
func (r *Registry) All() map[string]Factory
```

### Steps

- [ ] Create both files
- [ ] Write basic registry test
- [ ] Commit: `feat(providers/indexer): add Indexer interface and registry`

---

## Task 4 — Download client interface + registry

Same pattern as Task 3 but for download clients.

**Files:** `internal/providers/downloadclient/downloadclient.go`, `registry.go`

### downloadclient.go

```go
package downloadclient

import (
    "context"

    "github.com/ajthom90/sonarr2/internal/providers"
    "github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Item represents an active download in the client.
type Item struct {
    DownloadID string
    Title      string
    Status     string   // queued, downloading, completed, failed, paused
    TotalSize  int64
    Remaining  int64
    OutputPath string
}

// Status reports the download client's state.
type Status struct {
    IsLocalhost         bool
    OutputRootFolders   []string
}

// DownloadClient extends Provider with download-client-specific methods.
type DownloadClient interface {
    providers.Provider
    Protocol() indexer.DownloadProtocol
    Add(ctx context.Context, url string, title string) (downloadID string, err error)
    Items(ctx context.Context) ([]Item, error)
    Remove(ctx context.Context, downloadID string, deleteData bool) error
    Status(ctx context.Context) (Status, error)
}
```

### Steps

- [ ] Create both files + registry test
- [ ] Commit: `feat(providers/downloadclient): add DownloadClient interface and registry`

---

## Task 5 — Newznab indexer implementation

The first real provider. Talks to Newznab-compatible indexers (NZBGeek, DrunkenSlug, etc.) over HTTP.

**Files:** `internal/providers/indexer/newznab/newznab.go`, `settings.go`, `parser.go`, `newznab_test.go`

### settings.go

```go
type Settings struct {
    BaseURL    string `json:"baseUrl"    form:"text"     label:"URL"      required:"true"`
    ApiPath    string `json:"apiPath"    form:"text"     label:"API Path" default:"/api"`
    ApiKey     string `json:"apiKey"     form:"password" label:"API Key"  required:"true"`
    Categories []int  `json:"categories" form:"text"     label:"Categories" helpText:"Comma-separated category IDs"`
}
```

### newznab.go

Implements `indexer.Indexer`:
- `FetchRss` → GET `{baseURL}{apiPath}?t=tvsearch&cat={categories}&apikey={key}&dl=1`
- `Search` → GET `{baseURL}{apiPath}?t=tvsearch&tvdbid={id}&season={s}&ep={e}&cat={categories}&apikey={key}`
- `Test` → GET `{baseURL}{apiPath}?t=caps&apikey={key}` and verify 200

Parses Newznab XML responses (RSS format with `newznab:attr` extensions for size, category, etc.).

### parser.go

Parses the XML RSS response into `[]Release`. Uses `encoding/xml`.

### Tests

Use `httptest.NewServer` to serve canned XML responses:
- `TestNewznabFetchRss` — canned RSS response with 2 items, verify 2 releases parsed
- `TestNewznabSearch` — canned search response
- `TestNewznabTestConnectionSuccess` — caps endpoint returns 200
- `TestNewznabTestConnectionFailure` — caps returns 401

### Steps

- [ ] Write canned XML test fixtures
- [ ] Write failing tests
- [ ] Implement settings, parser, newznab
- [ ] Commit: `feat(providers/indexer/newznab): add Newznab indexer implementation`

---

## Task 6 — SABnzbd download client implementation

Talks to SABnzbd's API for adding NZBs and checking queue status.

**Files:** `internal/providers/downloadclient/sabnzbd/sabnzbd.go`, `settings.go`, `sabnzbd_test.go`

### settings.go

```go
type Settings struct {
    Host    string `json:"host"    form:"text"     label:"Host"    required:"true" default:"localhost"`
    Port    int    `json:"port"    form:"number"   label:"Port"    required:"true" default:"8080"`
    ApiKey  string `json:"apiKey"  form:"password" label:"API Key" required:"true"`
    UseSsl  bool   `json:"useSsl"  form:"checkbox" label:"Use SSL"`
    Category string `json:"category" form:"text"   label:"Category" default:"tv"`
}
```

### sabnzbd.go

Implements `downloadclient.DownloadClient`:
- `Add` → GET `{host}/api?mode=addurl&name={url}&cat={category}&apikey={key}&output=json`
- `Items` → GET `{host}/api?mode=queue&apikey={key}&output=json`, parse JSON queue
- `Remove` → GET `{host}/api?mode=queue&name=delete&value={id}&del_files={1|0}&apikey={key}&output=json`
- `Status` → GET `{host}/api?mode=fullstatus&apikey={key}&output=json`
- `Test` → GET `{host}/api?mode=version&apikey={key}&output=json`

### Tests

Use `httptest.NewServer` with canned JSON responses:
- `TestSabnzbdAdd` — verify download ID returned
- `TestSabnzbdItems` — canned queue JSON, verify parsed items
- `TestSabnzbdTestSuccess` / `TestSabnzbdTestFailure`

### Steps

- [ ] Write canned JSON fixtures
- [ ] Write failing tests
- [ ] Implement
- [ ] Commit: `feat(providers/downloadclient/sabnzbd): add SABnzbd download client`

---

## Task 7 — Provider instance stores

Add stores for indexer and download client instances (persisted in DB as rows with JSON settings).

**Files:** `internal/providers/indexer/store.go`, `store_postgres.go`, `store_sqlite.go`, `store_test.go` (and same pattern for downloadclient)

### IndexerInstance type

```go
type Instance struct {
    ID                      int
    Name                    string
    Implementation          string
    Settings                json.RawMessage
    EnableRss               bool
    EnableAutomaticSearch   bool
    EnableInteractiveSearch bool
    Priority                int
    Added                   time.Time
}

type InstanceStore interface {
    Create(ctx, Instance) (Instance, error)
    GetByID(ctx, id) (Instance, error)
    List(ctx) ([]Instance, error)
    Update(ctx, Instance) error
    Delete(ctx, id) error
}
```

Same pattern for download clients.

### Steps

- [ ] Write stores for both indexer + download client instances
- [ ] Tests (SQLite CRUD roundtrip)
- [ ] Commit: `feat(providers): add indexer and download client instance stores`

---

## Task 8 — Wire into app + final verification + push

- Wire indexer/download client registries and instance stores into app.New
- Register Newznab factory: `indexerRegistry.Register("Newznab", func() indexer.Indexer { return newznab.New() })`
- Register SABnzbd factory: `dcRegistry.Register("SABnzbd", func() downloadclient.DownloadClient { return sabnzbd.New() })`
- Final: tidy, lint, test, build, push, CI watch

Commit: `feat(app): wire provider registries and instance stores`

Then: push + CI watch.

---

## Done

After Task 8, the system has a pluggable provider architecture with two reference implementations. M7 (metadata source) adds TVDB. M8 (RSS sync + grab) connects the indexer to the decision engine to the download client — the first end-to-end "grab a release" flow.
