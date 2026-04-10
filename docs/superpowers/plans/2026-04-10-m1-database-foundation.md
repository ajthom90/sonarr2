# Milestone 1 — Database Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a dual-dialect (Postgres + SQLite) database layer to sonarr2. The layer must support connection pooling, versioned migrations via `goose`, typed query generation via `sqlc`, and a single-writer discipline on SQLite. Ship a first migration that creates a `host_config` table, a `HostConfigStore` interface with both dialect implementations, and an end-to-end smoke test proving the whole pipeline works from `app.New` → DB open → migrations up → repository query.

**Architecture:** `internal/db/` owns connection management, migration running, and contains `sqlc`-generated code per dialect. `internal/hostconfig/` owns the first domain entity with a `Store` interface and two implementations, one per dialect, each using the respective `sqlc` output. `internal/app/` opens the DB on startup, runs migrations, closes on shutdown. The `Config` struct gains a `DB DBConfig` field consumed by `app.New`, which now returns an error because DB open can fail.

**Tech Stack:** Go 1.23 (existing), `github.com/jackc/pgx/v5` + `pgxpool` (Postgres driver + pool), `modernc.org/sqlite` (pure-Go SQLite driver, no CGO), `github.com/pressly/goose/v3` (migration runner), `sqlc` (codegen tool, not a runtime dep), `github.com/testcontainers/testcontainers-go/modules/postgres` (test-only, for real-Postgres integration tests).

---

## Project layout this milestone creates or modifies

```
sonarr2/
├── internal/
│   ├── config/
│   │   ├── config.go              # MODIFIED: adds DBConfig field and env/flag loading
│   │   └── config_test.go         # MODIFIED: adds DB config tests
│   ├── db/                        # NEW
│   │   ├── db.go                  #   Pool interface + dialect enum
│   │   ├── postgres.go            #   pgx Pool impl
│   │   ├── postgres_test.go       #   integration tests (testcontainers-go)
│   │   ├── sqlite.go              #   modernc SQLite pool impl with single-writer
│   │   ├── sqlite_test.go         #   integration tests (in-memory)
│   │   ├── migrate.go             #   goose-based migration runner
│   │   ├── migrate_test.go        #   migration integration tests
│   │   ├── migrations/            #   raw .sql files
│   │   │   ├── postgres/
│   │   │   │   └── 00001_host_config.sql
│   │   │   └── sqlite/
│   │   │       └── 00001_host_config.sql
│   │   ├── queries/               #   sqlc input SQL
│   │   │   ├── postgres/
│   │   │   │   └── host_config.sql
│   │   │   └── sqlite/
│   │   │       └── host_config.sql
│   │   └── gen/                   #   sqlc-generated code (committed)
│   │       ├── postgres/
│   │       │   ├── db.go          #     generated
│   │       │   ├── models.go      #     generated
│   │       │   └── host_config.sql.go  # generated
│   │       └── sqlite/
│   │           ├── db.go
│   │           ├── models.go
│   │           └── host_config.sql.go
│   ├── hostconfig/                # NEW
│   │   ├── hostconfig.go          #   HostConfig type + Store interface
│   │   ├── hostconfig_test.go
│   │   ├── postgres.go            #   Store impl wrapping db/gen/postgres
│   │   ├── postgres_test.go
│   │   ├── sqlite.go              #   Store impl wrapping db/gen/sqlite
│   │   └── sqlite_test.go
│   ├── app/
│   │   ├── app.go                 # MODIFIED: New returns error, opens db, runs migrations
│   │   └── app_test.go            # MODIFIED: tests still pass with new signature
│   └── api/
│       └── server.go              # MODIFIED: statusHandler reports db status
├── cmd/sonarr/main.go             # MODIFIED: handles New's new error return
├── sqlc.yaml                      # NEW: sqlc config
├── .github/workflows/test.yml     # MODIFIED: runs integration tests against Docker Postgres
└── go.mod / go.sum                # MODIFIED: pgx, modernc-sqlite, goose, testcontainers
```

Existing files not modified: `LICENSE`, `README.md`, `.gitignore`, `Makefile`, `docker/Dockerfile`, `.dockerignore`, `.golangci.yml`, `CONTRIBUTING.md`, `docs/`, `internal/buildinfo/`, `internal/logging/`.

---

## Prerequisite: one-time tooling install

`sqlc` is a code generator invoked by developers and CI. It's not a runtime dependency, but you need it on your PATH to regenerate generated code. Tasks that invoke `sqlc` will check for its presence and install via `go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0` if missing. This install happens in Task 9; if `sqlc` is already available from a previous install, the task is a no-op.

`goose` is used as a library for running migrations in tests — **not** as a CLI tool. It's added via `go get` in Task 5.

Docker is required for Postgres integration tests via testcontainers-go. If Docker isn't running locally, Postgres tests will be skipped via `testing.Short()` or a `testcontainers` package-level check. CI has Docker available.

---

## Task 1 — Extend Config with DBConfig

Add a `DB DBConfig` field to the root `Config` struct with dialect, DSN, pool sizing, and SQLite-specific options. Add env/flag overrides. Add validation.

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing tests**

Open `internal/config/config_test.go` and add the following tests **after** the existing tests (do not modify existing tests). These four new tests verify DB defaults, env override, flag override, and validation:

```go
func TestDefaultDBConfig(t *testing.T) {
	cfg := Default()
	if cfg.DB.Dialect != "sqlite" {
		t.Errorf("default dialect = %q, want sqlite", cfg.DB.Dialect)
	}
	if cfg.DB.DSN == "" {
		t.Errorf("default DSN must not be empty")
	}
	if cfg.DB.MaxOpenConns != 20 {
		t.Errorf("default max_open_conns = %d, want 20", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 2 {
		t.Errorf("default max_idle_conns = %d, want 2", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.BusyTimeout != 5*time.Second {
		t.Errorf("default busy_timeout = %v, want 5s", cfg.DB.BusyTimeout)
	}
}

func TestLoadDBEnvOverride(t *testing.T) {
	env := map[string]string{
		"SONARR2_DB_DIALECT":           "postgres",
		"SONARR2_DB_DSN":               "postgres://user:pass@localhost/sonarr2",
		"SONARR2_DB_MAX_OPEN_CONNS":    "50",
		"SONARR2_DB_MAX_IDLE_CONNS":    "5",
		"SONARR2_DB_BUSY_TIMEOUT":      "10s",
	}
	cfg, err := Load(nil, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DB.Dialect != "postgres" {
		t.Errorf("dialect = %q, want postgres", cfg.DB.Dialect)
	}
	if cfg.DB.DSN != "postgres://user:pass@localhost/sonarr2" {
		t.Errorf("DSN = %q", cfg.DB.DSN)
	}
	if cfg.DB.MaxOpenConns != 50 {
		t.Errorf("max_open_conns = %d, want 50", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 5 {
		t.Errorf("max_idle_conns = %d, want 5", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.BusyTimeout != 10*time.Second {
		t.Errorf("busy_timeout = %v, want 10s", cfg.DB.BusyTimeout)
	}
}

func TestLoadDBFlagsOverrideEnv(t *testing.T) {
	env := map[string]string{"SONARR2_DB_DIALECT": "sqlite"}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load(
		[]string{"-db-dialect", "postgres", "-db-dsn", "postgres://x"},
		getenv,
	)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DB.Dialect != "postgres" {
		t.Errorf("dialect = %q, want postgres", cfg.DB.Dialect)
	}
	if cfg.DB.DSN != "postgres://x" {
		t.Errorf("dsn = %q", cfg.DB.DSN)
	}
}

func TestValidateDBConfig(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"empty dialect", func(c *Config) { c.DB.Dialect = "" }, true},
		{"unknown dialect", func(c *Config) { c.DB.Dialect = "mysql" }, true},
		{"empty DSN", func(c *Config) { c.DB.DSN = "" }, true},
		{"negative max open", func(c *Config) { c.DB.MaxOpenConns = -1 }, true},
		{"negative max idle", func(c *Config) { c.DB.MaxIdleConns = -1 }, true},
		{"valid sqlite", func(c *Config) { c.DB.Dialect = "sqlite"; c.DB.DSN = "file:test.db" }, false},
		{"valid postgres", func(c *Config) { c.DB.Dialect = "postgres"; c.DB.DSN = "postgres://x" }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			tc.mutate(&cfg)
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("Validate() error = nil, want non-nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
		})
	}
}
```

You'll also need to add `"time"` to the test file's import block if it isn't already imported.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/ajthom90/projects/sonarr2
go test ./internal/config/...
```

Expected: compilation failure because `DBConfig`, `cfg.DB`, etc. don't exist yet.

- [ ] **Step 3: Implement the changes**

Open `internal/config/config.go` and apply the following changes:

**Add the `DBConfig` type** — insert immediately after the `PathsConfig` type definition:

```go
// DBConfig controls the database connection.
type DBConfig struct {
	Dialect         string        `yaml:"dialect"`
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	BusyTimeout     time.Duration `yaml:"busy_timeout"`
}
```

**Add `DB DBConfig` to the `Config` struct:**

```go
// Config is the full runtime configuration for sonarr2.
type Config struct {
	HTTP    HTTPConfig     `yaml:"http"`
	Logging logging.Config `yaml:"logging"`
	Paths   PathsConfig    `yaml:"paths"`
	DB      DBConfig       `yaml:"db"`
}
```

**Update `Default()`** to include DB defaults — the returned literal should now include:

```go
		DB: DBConfig{
			Dialect:         "sqlite",
			DSN:             "file:./data/sonarr2.db?_journal=WAL&_busy_timeout=5000",
			MaxOpenConns:    20,
			MaxIdleConns:    2,
			ConnMaxLifetime: 30 * time.Minute,
			BusyTimeout:     5 * time.Second,
		},
```

Add `"time"` to the imports if it isn't already there.

**Add DB flags to `Load`** — inside `Load`, after the existing `logLevel` flag declaration and before `fs.Parse`, add:

```go
	dbDialect := fs.String("db-dialect", "", "Database dialect (sqlite|postgres)")
	dbDSN := fs.String("db-dsn", "", "Database DSN")
```

**Add DB env var handling** — inside `Load`, after the existing `SONARR2_LOG_LEVEL` block, add:

```go
	if v := getenv("SONARR2_DB_DIALECT"); v != "" {
		cfg.DB.Dialect = v
	}
	if v := getenv("SONARR2_DB_DSN"); v != "" {
		cfg.DB.DSN = v
	}
	if v := getenv("SONARR2_DB_MAX_OPEN_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_DB_MAX_OPEN_CONNS must be an integer, got %q: %w", v, err)
		}
		cfg.DB.MaxOpenConns = n
	}
	if v := getenv("SONARR2_DB_MAX_IDLE_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_DB_MAX_IDLE_CONNS must be an integer, got %q: %w", v, err)
		}
		cfg.DB.MaxIdleConns = n
	}
	if v := getenv("SONARR2_DB_BUSY_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_DB_BUSY_TIMEOUT must be a duration, got %q: %w", v, err)
		}
		cfg.DB.BusyTimeout = d
	}
```

**Add DB flag handling** — inside `Load`, after the existing `if *logLevel != ""` block, add:

```go
	if *dbDialect != "" {
		cfg.DB.Dialect = *dbDialect
	}
	if *dbDSN != "" {
		cfg.DB.DSN = *dbDSN
	}
```

**Extend `Validate`** — add DB validation at the bottom of `Validate`, just before `return nil`:

```go
	switch c.DB.Dialect {
	case "sqlite", "postgres":
		// ok
	default:
		return fmt.Errorf("db.dialect must be sqlite or postgres, got %q", c.DB.Dialect)
	}
	if c.DB.DSN == "" {
		return errors.New("db.dsn must not be empty")
	}
	if c.DB.MaxOpenConns < 0 {
		return fmt.Errorf("db.max_open_conns must be >= 0, got %d", c.DB.MaxOpenConns)
	}
	if c.DB.MaxIdleConns < 0 {
		return fmt.Errorf("db.max_idle_conns must be >= 0, got %d", c.DB.MaxIdleConns)
	}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -v ./internal/config/...
```

Expected: all tests pass (the new four + the original 11 = 15 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add DBConfig with dialect, DSN, pool sizing, validation"
```

---

## Task 2 — internal/db package skeleton with Pool interface

Create the `internal/db` package with a `Pool` interface abstraction that both Postgres and SQLite implementations will satisfy.

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/db_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/db/db_test.go`:
```go
package db

import (
	"errors"
	"testing"
)

func TestParseDialect(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    Dialect
		wantErr bool
	}{
		"postgres":     {"postgres", DialectPostgres, false},
		"sqlite":       {"sqlite", DialectSQLite, false},
		"unknown":      {"mysql", "", true},
		"empty":        {"", "", true},
		"uppercase pg": {"POSTGRES", DialectPostgres, false},
		"uppercase lt": {"SQLite", DialectSQLite, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseDialect(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseDialect(%q) error = nil, want non-nil", tc.in)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDialect(%q) error = %v, want nil", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseDialect(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestErrNoRowsIsNotNil(t *testing.T) {
	if ErrNoRows == nil {
		t.Error("ErrNoRows must be defined")
	}
	if !errors.Is(ErrNoRows, ErrNoRows) {
		t.Error("ErrNoRows must be identifiable via errors.Is")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/ajthom90/projects/sonarr2
go test ./internal/db/...
```

Expected: compilation failure because `Dialect`, `DialectPostgres`, `DialectSQLite`, `ParseDialect`, `ErrNoRows` don't exist yet.

- [ ] **Step 3: Implement the package**

Create `internal/db/db.go`:
```go
// Package db provides database connection pooling, migration running, and
// typed query code for sonarr2's Postgres and SQLite backends.
package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Dialect identifies which database backend is in use.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// ErrNoRows is returned by repository methods when a lookup finds nothing.
// Store implementations translate driver-specific no-row errors into this
// sentinel so callers can use errors.Is without importing driver packages.
var ErrNoRows = errors.New("db: no rows")

// ParseDialect normalizes a dialect string. Accepts any casing of
// "postgres" or "sqlite".
func ParseDialect(s string) (Dialect, error) {
	switch strings.ToLower(s) {
	case "postgres":
		return DialectPostgres, nil
	case "sqlite":
		return DialectSQLite, nil
	default:
		return "", fmt.Errorf("db: unknown dialect %q", s)
	}
}

// Pool is a high-level abstraction over the backend connection pool. It
// exposes the minimum surface the rest of the application needs: query
// execution via the dialect-specific generated code (Queries accessor),
// liveness checks, and graceful close. The concrete types are returned by
// OpenPostgres and OpenSQLite.
type Pool interface {
	// Dialect reports which backend this pool uses.
	Dialect() Dialect

	// Ping verifies the database is reachable. Returns nil if the
	// connection is healthy.
	Ping(ctx context.Context) error

	// Close releases all resources held by the pool. Must be called
	// during shutdown.
	Close() error
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -v ./internal/db/...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/db/db.go internal/db/db_test.go
git commit -m "feat(db): add package skeleton with Pool interface and Dialect enum"
```

---

## Task 3 — Postgres Pool implementation

Add `pgx/v5` and `pgxpool`. Implement `OpenPostgres`, a struct satisfying `Pool`, and integration tests using `testcontainers-go`.

**Files:**
- Create: `internal/db/postgres.go`
- Create: `internal/db/postgres_test.go`
- Modify: `go.mod` (adds pgx/v5 and testcontainers-go modules)

- [ ] **Step 1: Add pgx dependency**

```bash
cd /Users/ajthom90/projects/sonarr2
go get github.com/jackc/pgx/v5@v5.7.1
go get github.com/jackc/pgx/v5/pgxpool@v5.7.1
```

Expected: `go.mod` gains `require github.com/jackc/pgx/v5 v5.7.1`; `go.sum` is updated.

- [ ] **Step 2: Add testcontainers dependency**

```bash
go get github.com/testcontainers/testcontainers-go@v0.33.0
go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.33.0
```

Expected: module adds direct dependencies on testcontainers-go.

- [ ] **Step 3: Write the failing tests**

Create `internal/db/postgres_test.go`:
```go
package db

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// postgresContainer starts a fresh Postgres instance via testcontainers-go
// and returns a connection DSN plus a cleanup function. Tests that need
// Postgres call this helper once at the top of the test.
func postgresContainer(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Postgres container in -short mode")
	}
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("sonarr2_test"),
		tcpostgres.WithUsername("sonarr2"),
		tcpostgres.WithPassword("sonarr2"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("postgres container failed to start (Docker unavailable?): %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Terminate(context.Background()); err != nil {
			t.Logf("container terminate: %v", err)
		}
	})

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	return dsn
}

func TestOpenPostgresConnectsAndPings(t *testing.T) {
	dsn := postgresContainer(t)

	pool, err := OpenPostgres(context.Background(), PostgresOptions{
		DSN:             dsn,
		MaxOpenConns:    4,
		MinOpenConns:    1,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if pool.Dialect() != DialectPostgres {
		t.Errorf("Dialect() = %q, want postgres", pool.Dialect())
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestOpenPostgresRejectsInvalidDSN(t *testing.T) {
	_, err := OpenPostgres(context.Background(), PostgresOptions{
		DSN:          "::::invalid::::",
		MaxOpenConns: 1,
	})
	if err == nil {
		t.Fatal("expected error for invalid DSN")
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
go test ./internal/db/...
```

Expected: compilation failure because `OpenPostgres`, `PostgresOptions`, and `postgresPool` don't exist yet.

- [ ] **Step 5: Implement postgres.go**

Create `internal/db/postgres.go`:
```go
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresOptions controls the Postgres connection pool.
type PostgresOptions struct {
	DSN             string
	MaxOpenConns    int
	MinOpenConns    int
	ConnMaxLifetime time.Duration
}

// PostgresPool is the Postgres implementation of Pool. It embeds a
// *pgxpool.Pool and exposes it via Raw for use by sqlc-generated code.
type PostgresPool struct {
	pool *pgxpool.Pool
}

// Raw returns the underlying pgxpool.Pool so sqlc-generated code can use it
// as a DBTX. Callers should NOT retain this pointer past the owning Pool's
// lifetime.
func (p *PostgresPool) Raw() *pgxpool.Pool {
	return p.pool
}

// Dialect implements Pool.
func (p *PostgresPool) Dialect() Dialect { return DialectPostgres }

// Ping implements Pool.
func (p *PostgresPool) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close implements Pool.
func (p *PostgresPool) Close() error {
	p.pool.Close()
	return nil
}

// OpenPostgres parses opts.DSN, applies pool sizing, and returns a
// connected *PostgresPool. It returns an error if the DSN is invalid or the
// initial Ping fails.
func OpenPostgres(ctx context.Context, opts PostgresOptions) (*PostgresPool, error) {
	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: parse postgres DSN: %w", err)
	}
	if opts.MaxOpenConns > 0 {
		cfg.MaxConns = int32(opts.MaxOpenConns)
	}
	if opts.MinOpenConns > 0 {
		cfg.MinConns = int32(opts.MinOpenConns)
	}
	if opts.ConnMaxLifetime > 0 {
		cfg.MaxConnLifetime = opts.ConnMaxLifetime
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open postgres pool: %w", err)
	}

	// Verify the connection is actually usable.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: postgres ping: %w", err)
	}

	return &PostgresPool{pool: pool}, nil
}
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test -race ./internal/db/...
```

Expected: `TestOpenPostgresRejectsInvalidDSN` passes. `TestOpenPostgresConnectsAndPings` requires Docker — if Docker is running it should pass; if Docker is down it should skip with a message (not fail).

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/db/postgres.go internal/db/postgres_test.go
git commit -m "feat(db): add Postgres pool via pgx/v5 with testcontainers integration test"
```

---

## Task 4 — SQLite Pool implementation with single-writer discipline

Add `modernc.org/sqlite`. Implement `OpenSQLite` returning a pool that enforces single-writer semantics via a serialized writer goroutine. Reads use a separate connection with `PRAGMA query_only=1`.

**Files:**
- Create: `internal/db/sqlite.go`
- Create: `internal/db/sqlite_test.go`
- Modify: `go.mod`

- [ ] **Step 1: Add modernc.org/sqlite dependency**

```bash
cd /Users/ajthom90/projects/sonarr2
go get modernc.org/sqlite@v1.33.1
```

- [ ] **Step 2: Write the failing tests**

Create `internal/db/sqlite_test.go`:
```go
package db

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestOpenSQLiteInMemory(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if pool.Dialect() != DialectSQLite {
		t.Errorf("Dialect() = %q, want sqlite", pool.Dialect())
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestOpenSQLiteFileDSN(t *testing.T) {
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?_journal=WAL&_busy_timeout=5000"

	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         dsn,
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := pool.Ping(context.Background()); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestSQLiteWriterSerializesWrites(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	// Create a test table via the writer.
	err = pool.Write(context.Background(), func(exec Executor) error {
		_, err := exec.ExecContext(context.Background(),
			`CREATE TABLE counter (n INTEGER)`)
		if err != nil {
			return err
		}
		_, err = exec.ExecContext(context.Background(),
			`INSERT INTO counter (n) VALUES (0)`)
		return err
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	// Launch 20 concurrent writers. Single-writer discipline means they
	// all complete without database-is-locked errors.
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := pool.Write(context.Background(), func(exec Executor) error {
				_, err := exec.ExecContext(context.Background(),
					`UPDATE counter SET n = n + 1`)
				return err
			})
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify the final count is 20.
	var n int
	err = pool.Read(context.Background(), func(q Querier) error {
		row := q.QueryRowContext(context.Background(), `SELECT n FROM counter`)
		return row.Scan(&n)
	})
	if err != nil {
		t.Fatalf("read counter: %v", err)
	}
	if n != 20 {
		t.Errorf("counter = %d, want 20", n)
	}
}

func TestOpenSQLiteRejectsEmptyDSN(t *testing.T) {
	_, err := OpenSQLite(context.Background(), SQLiteOptions{DSN: ""})
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/db/...
```

Expected: compilation failure because `OpenSQLite`, `SQLiteOptions`, `SQLitePool`, `Executor`, `Querier`, `pool.Write`, `pool.Read` don't exist.

- [ ] **Step 4: Implement sqlite.go**

Create `internal/db/sqlite.go`:
```go
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteOptions controls the SQLite connection pool.
type SQLiteOptions struct {
	DSN         string
	BusyTimeout time.Duration
}

// Executor is the interface satisfied by both *sql.DB and *sql.Tx for exec
// statements. Used by Write callbacks.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Querier is the interface satisfied by a read-only *sql.DB for queries.
// Used by Read callbacks.
type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SQLitePool owns two *sql.DB handles: a single-connection writer DB and a
// multi-connection read-only DB. All writes are serialized through a
// writer goroutine, which eliminates "database is locked" errors at the
// application layer.
type SQLitePool struct {
	writer   *sql.DB   // max open conns = 1; PRAGMA journal_mode=WAL
	reader   *sql.DB   // query_only; multi-conn
	writeReq chan writeReq
	wg       sync.WaitGroup
	closeOnce sync.Once
	closed    chan struct{}
}

type writeReq struct {
	ctx  context.Context
	fn   func(Executor) error
	done chan error
}

// Dialect implements Pool.
func (p *SQLitePool) Dialect() Dialect { return DialectSQLite }

// Ping implements Pool.
func (p *SQLitePool) Ping(ctx context.Context) error {
	if err := p.reader.PingContext(ctx); err != nil {
		return err
	}
	return p.writer.PingContext(ctx)
}

// Close implements Pool. Shuts down the writer loop and closes both sql.DB
// handles.
func (p *SQLitePool) Close() error {
	var err error
	p.closeOnce.Do(func() {
		close(p.closed)
		p.wg.Wait()
		if rerr := p.reader.Close(); rerr != nil {
			err = rerr
		}
		if werr := p.writer.Close(); werr != nil && err == nil {
			err = werr
		}
	})
	return err
}

// Write submits fn to the writer goroutine and blocks until it completes.
// fn receives an Executor that wraps the single writer connection; writes
// are serialized with respect to all other Write calls.
func (p *SQLitePool) Write(ctx context.Context, fn func(Executor) error) error {
	req := writeReq{ctx: ctx, fn: fn, done: make(chan error, 1)}
	select {
	case p.writeReq <- req:
	case <-p.closed:
		return errors.New("db: sqlite pool closed")
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-req.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Read executes fn against the read-only connection pool. Multiple Read
// calls may execute concurrently.
func (p *SQLitePool) Read(ctx context.Context, fn func(Querier) error) error {
	return fn(p.reader)
}

// RawWriter returns the underlying writer *sql.DB for use by sqlc-generated
// code that needs a DBTX. Only used by Store implementations.
func (p *SQLitePool) RawWriter() *sql.DB { return p.writer }

// RawReader returns the underlying reader *sql.DB for read-only query
// execution by sqlc-generated code.
func (p *SQLitePool) RawReader() *sql.DB { return p.reader }

// OpenSQLite opens a SQLite database using the modernc.org/sqlite pure-Go
// driver. It creates two sql.DB handles: a single-connection writer and a
// multi-connection reader with PRAGMA query_only=1.
func OpenSQLite(ctx context.Context, opts SQLiteOptions) (*SQLitePool, error) {
	if opts.DSN == "" {
		return nil, errors.New("db: sqlite DSN is required")
	}

	writer, err := sql.Open("sqlite", opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: open sqlite writer: %w", err)
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)

	if err := writer.PingContext(ctx); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: sqlite writer ping: %w", err)
	}

	// Apply WAL mode and busy_timeout via PRAGMA for safety.
	if _, err := writer.ExecContext(ctx, `PRAGMA journal_mode=WAL`); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: set WAL mode: %w", err)
	}
	if opts.BusyTimeout > 0 {
		ms := int(opts.BusyTimeout / time.Millisecond)
		if _, err := writer.ExecContext(ctx, fmt.Sprintf("PRAGMA busy_timeout=%d", ms)); err != nil {
			_ = writer.Close()
			return nil, fmt.Errorf("db: set busy_timeout: %w", err)
		}
	}

	reader, err := sql.Open("sqlite", opts.DSN)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("db: open sqlite reader: %w", err)
	}
	if _, err := reader.ExecContext(ctx, `PRAGMA query_only=1`); err != nil {
		_ = reader.Close()
		_ = writer.Close()
		return nil, fmt.Errorf("db: set query_only: %w", err)
	}

	p := &SQLitePool{
		writer:   writer,
		reader:   reader,
		writeReq: make(chan writeReq),
		closed:   make(chan struct{}),
	}

	p.wg.Add(1)
	go p.writerLoop()

	return p, nil
}

// writerLoop is the single goroutine that owns the writer connection.
// Every Write call flows through here, guaranteeing serial access.
func (p *SQLitePool) writerLoop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.closed:
			return
		case req := <-p.writeReq:
			err := req.fn(p.writer)
			req.done <- err
		}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test -race -v ./internal/db/...
```

Expected: all SQLite tests pass. `TestSQLiteWriterSerializesWrites` is the key test — it proves the single-writer discipline works under concurrency.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/db/sqlite.go internal/db/sqlite_test.go
git commit -m "feat(db): add SQLite pool with single-writer discipline via modernc/sqlite"
```

---

## Task 5 — Migration runner via goose

Wrap `goose` to run migrations for both dialects. Migrations live in `internal/db/migrations/{postgres,sqlite}/NNNNN_name.sql`. Expose `Migrate(ctx, pool) error`.

**Files:**
- Create: `internal/db/migrate.go`
- Create: `internal/db/migrate_test.go`
- Modify: `go.mod`

- [ ] **Step 1: Add goose dependency**

```bash
cd /Users/ajthom90/projects/sonarr2
go get github.com/pressly/goose/v3@v3.22.1
```

- [ ] **Step 2: Write the failing tests**

Create `internal/db/migrate_test.go`:
```go
package db

import (
	"context"
	"testing"
	"time"
)

func TestMigrateSQLiteCreatesHostConfig(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// The host_config table should exist after migrating.
	err = pool.Read(context.Background(), func(q Querier) error {
		row := q.QueryRowContext(context.Background(),
			`SELECT name FROM sqlite_master WHERE type='table' AND name='host_config'`)
		var name string
		return row.Scan(&name)
	})
	if err != nil {
		t.Errorf("host_config table missing: %v", err)
	}
}

func TestMigrateSQLiteIsIdempotent(t *testing.T) {
	pool, err := OpenSQLite(context.Background(), SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}

func TestMigratePostgresCreatesHostConfig(t *testing.T) {
	dsn := postgresContainer(t)

	pool, err := OpenPostgres(context.Background(), PostgresOptions{
		DSN:          dsn,
		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// The host_config table should exist.
	var exists bool
	err = pool.Raw().QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'host_config')`,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("check host_config existence: %v", err)
	}
	if !exists {
		t.Error("host_config table missing")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/db/...
```

Expected: compilation failure because `Migrate` doesn't exist and the migration files are missing.

- [ ] **Step 4: Create the migration directories and first migration**

Create `internal/db/migrations/postgres/00001_host_config.sql`:
```sql
-- +goose Up
CREATE TABLE host_config (
    id              SMALLINT PRIMARY KEY CHECK (id = 1),
    api_key         TEXT NOT NULL,
    auth_mode       TEXT NOT NULL DEFAULT 'forms',
    migration_state TEXT NOT NULL DEFAULT 'clean',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE host_config;
```

Create `internal/db/migrations/sqlite/00001_host_config.sql`:
```sql
-- +goose Up
CREATE TABLE host_config (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    api_key         TEXT NOT NULL,
    auth_mode       TEXT NOT NULL DEFAULT 'forms',
    migration_state TEXT NOT NULL DEFAULT 'clean',
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down
DROP TABLE host_config;
```

- [ ] **Step 5: Implement migrate.go**

Create `internal/db/migrate.go`:
```go
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/postgres/*.sql migrations/sqlite/*.sql
var migrationsFS embed.FS

// Migrate runs all pending migrations for the given pool's dialect. It is
// safe to call repeatedly; already-applied migrations are skipped.
func Migrate(ctx context.Context, pool Pool) error {
	switch p := pool.(type) {
	case *PostgresPool:
		return migratePostgres(ctx, p)
	case *SQLitePool:
		return migrateSQLite(ctx, p)
	default:
		return fmt.Errorf("db: unsupported pool type %T", pool)
	}
}

func migratePostgres(ctx context.Context, p *PostgresPool) error {
	// goose operates on *sql.DB, not pgxpool. Wrap via pgx's stdlib.
	conn := stdlib.OpenDBFromPool(p.pool)
	defer conn.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("db: goose set dialect: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations/postgres")
	if err != nil {
		return fmt.Errorf("db: subfs postgres migrations: %w", err)
	}
	goose.SetBaseFS(sub)
	defer goose.SetBaseFS(nil)

	if err := goose.UpContext(ctx, conn, "."); err != nil {
		return fmt.Errorf("db: postgres migrate up: %w", err)
	}
	return nil
}

func migrateSQLite(ctx context.Context, p *SQLitePool) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("db: goose set dialect: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations/sqlite")
	if err != nil {
		return fmt.Errorf("db: subfs sqlite migrations: %w", err)
	}
	goose.SetBaseFS(sub)
	defer goose.SetBaseFS(nil)

	if err := goose.UpContext(ctx, p.writer, "."); err != nil {
		return fmt.Errorf("db: sqlite migrate up: %w", err)
	}
	return nil
}

// Ensure the sql package is imported (used via type assertion paths above).
var _ = (*sql.DB)(nil)
```

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test -race -v ./internal/db/...
```

Expected: all migration tests pass. SQLite tests run unconditionally; Postgres tests require Docker and will skip if unavailable.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/db/migrate.go internal/db/migrate_test.go internal/db/migrations/
git commit -m "feat(db): add goose migration runner with host_config first migration"
```

---

## Task 6 — sqlc configuration and generated query code

Configure `sqlc` to read queries from `internal/db/queries/{postgres,sqlite}/` and emit generated code into `internal/db/gen/{postgres,sqlite}/`.

**Files:**
- Create: `sqlc.yaml`
- Create: `internal/db/queries/postgres/host_config.sql`
- Create: `internal/db/queries/sqlite/host_config.sql`
- Create: `internal/db/gen/postgres/*` (generated)
- Create: `internal/db/gen/sqlite/*` (generated)

- [ ] **Step 1: Install sqlc if missing**

```bash
which sqlc || go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
sqlc version
```

Expected: prints `v1.27.0` (or newer). If `go install` ends up in a location not on PATH, you may need `$(go env GOPATH)/bin/sqlc`.

- [ ] **Step 2: Create `sqlc.yaml` at repo root**

Create `sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "internal/db/migrations/postgres"
    queries: "internal/db/queries/postgres"
    gen:
      go:
        package: "postgres"
        sql_package: "pgx/v5"
        out: "internal/db/gen/postgres"
        emit_interface: true
        emit_json_tags: false
        emit_prepared_queries: false
        emit_exact_table_names: false
        emit_empty_slices: true
  - engine: "sqlite"
    schema: "internal/db/migrations/sqlite"
    queries: "internal/db/queries/sqlite"
    gen:
      go:
        package: "sqlite"
        sql_package: "database/sql"
        out: "internal/db/gen/sqlite"
        emit_interface: true
        emit_json_tags: false
        emit_prepared_queries: false
        emit_exact_table_names: false
        emit_empty_slices: true
```

- [ ] **Step 3: Create the Postgres query file**

Create `internal/db/queries/postgres/host_config.sql`:
```sql
-- name: GetHostConfig :one
SELECT id, api_key, auth_mode, migration_state, created_at, updated_at
FROM host_config
WHERE id = 1;

-- name: UpsertHostConfig :exec
INSERT INTO host_config (id, api_key, auth_mode, migration_state)
VALUES (1, $1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET api_key = EXCLUDED.api_key,
    auth_mode = EXCLUDED.auth_mode,
    migration_state = EXCLUDED.migration_state,
    updated_at = now();
```

- [ ] **Step 4: Create the SQLite query file**

Create `internal/db/queries/sqlite/host_config.sql`:
```sql
-- name: GetHostConfig :one
SELECT id, api_key, auth_mode, migration_state, created_at, updated_at
FROM host_config
WHERE id = 1;

-- name: UpsertHostConfig :exec
INSERT INTO host_config (id, api_key, auth_mode, migration_state)
VALUES (1, ?, ?, ?)
ON CONFLICT (id) DO UPDATE
SET api_key = excluded.api_key,
    auth_mode = excluded.auth_mode,
    migration_state = excluded.migration_state,
    updated_at = datetime('now');
```

- [ ] **Step 5: Generate code**

```bash
cd /Users/ajthom90/projects/sonarr2
sqlc generate
```

Expected: `internal/db/gen/postgres/{db.go,models.go,host_config.sql.go}` and `internal/db/gen/sqlite/{db.go,models.go,host_config.sql.go}` are created. If `sqlc` reports errors, read them — typical issues are missing schema files or incorrect query syntax. Fix by editing the query files (do NOT edit generated files).

- [ ] **Step 6: Verify generated code compiles**

```bash
go build ./internal/db/gen/...
```

Expected: no output, exit 0.

- [ ] **Step 7: Run all tests to ensure nothing regressed**

```bash
go test -race ./...
```

Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
git add sqlc.yaml internal/db/queries/ internal/db/gen/
git commit -m "feat(db): add sqlc config and generate host_config queries"
```

---

## Task 7 — hostconfig package with Store interface

Create `internal/hostconfig/` with a dialect-agnostic `HostConfig` struct and a `Store` interface. Provide two implementations — `NewPostgresStore` and `NewSQLiteStore` — each wrapping its respective `sqlc`-generated code.

**Files:**
- Create: `internal/hostconfig/hostconfig.go`
- Create: `internal/hostconfig/hostconfig_test.go`
- Create: `internal/hostconfig/postgres.go`
- Create: `internal/hostconfig/postgres_test.go`
- Create: `internal/hostconfig/sqlite.go`
- Create: `internal/hostconfig/sqlite_test.go`

- [ ] **Step 1: Write the failing unit tests**

Create `internal/hostconfig/hostconfig_test.go`:
```go
package hostconfig

import (
	"testing"
	"time"
)

func TestHostConfigFields(t *testing.T) {
	hc := HostConfig{
		APIKey:         "abc123",
		AuthMode:       "forms",
		MigrationState: "clean",
		CreatedAt:      time.Unix(0, 0),
		UpdatedAt:      time.Unix(0, 0),
	}
	if hc.APIKey != "abc123" {
		t.Errorf("APIKey = %q", hc.APIKey)
	}
}

func TestNewAPIKeyIsNonEmpty(t *testing.T) {
	k := NewAPIKey()
	if len(k) < 32 {
		t.Errorf("API key length = %d, want >= 32", len(k))
	}
}

func TestNewAPIKeyIsUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 10; i++ {
		k := NewAPIKey()
		if _, ok := seen[k]; ok {
			t.Errorf("duplicate API key generated: %q", k)
		}
		seen[k] = struct{}{}
	}
}
```

- [ ] **Step 2: Write the core package**

Create `internal/hostconfig/hostconfig.go`:
```go
// Package hostconfig owns the host_config entity and its Store interface.
// host_config is a singleton row holding the API key, authentication mode,
// and migration state. The Store interface has one implementation per
// database dialect.
package hostconfig

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// HostConfig is the singleton host configuration row.
type HostConfig struct {
	APIKey         string
	AuthMode       string
	MigrationState string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Store reads and writes the host_config row.
type Store interface {
	// Get returns the current host config. If the row does not exist,
	// returns ErrNotFound.
	Get(ctx context.Context) (HostConfig, error)

	// Upsert inserts or updates the singleton host_config row. The
	// created_at / updated_at timestamps are managed by the database.
	Upsert(ctx context.Context, hc HostConfig) error
}

// NewAPIKey returns a cryptographically random 64-character hex API key
// suitable for first-run initialization of host_config.
func NewAPIKey() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is catastrophic; panic is the right response.
		panic("hostconfig: crypto/rand read failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
```

- [ ] **Step 3: Run the unit tests to verify they pass**

```bash
cd /Users/ajthom90/projects/sonarr2
go test ./internal/hostconfig/...
```

Expected: the three unit tests pass.

- [ ] **Step 4: Add ErrNotFound to the package**

Create or append to `internal/hostconfig/hostconfig.go` (before the `HostConfig` type definition):

```go
import "errors"
```

(add to existing imports if already present)

And add:

```go
// ErrNotFound is returned by Store.Get when the host_config row does not exist.
var ErrNotFound = errors.New("hostconfig: not found")
```

Re-run tests to make sure nothing broke.

- [ ] **Step 5: Write the failing Postgres Store tests**

Create `internal/hostconfig/postgres_test.go`. The file embeds its own testcontainers-based setup helper to avoid cross-package test-helper plumbing — the db package's `postgresContainer` helper is test-file-scoped and cannot be imported from another package:

```go
package hostconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgresForTest(t *testing.T) *db.PostgresPool {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Postgres in -short mode")
	}
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("sonarr2_test"),
		tcpostgres.WithUsername("sonarr2"),
		tcpostgres.WithPassword("sonarr2"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Skipf("postgres container unavailable: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}

	pool, err := db.OpenPostgres(ctx, db.PostgresOptions{
		DSN:          dsn,
		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("OpenPostgres: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func TestPostgresStoreGetReturnsNotFoundWhenEmpty(t *testing.T) {
	pool := setupPostgresForTest(t)
	store := NewPostgresStore(pool)
	_, err := store.Get(context.Background())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestPostgresStoreUpsertAndGet(t *testing.T) {
	pool := setupPostgresForTest(t)
	store := NewPostgresStore(pool)

	want := HostConfig{
		APIKey:         "test-api-key",
		AuthMode:       "forms",
		MigrationState: "clean",
	}
	if err := store.Upsert(context.Background(), want); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if got.AuthMode != want.AuthMode {
		t.Errorf("AuthMode = %q, want %q", got.AuthMode, want.AuthMode)
	}
	if got.MigrationState != want.MigrationState {
		t.Errorf("MigrationState = %q, want %q", got.MigrationState, want.MigrationState)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestPostgresNewAPIKeyRoundtrip(t *testing.T) {
	pool := setupPostgresForTest(t)
	store := NewPostgresStore(pool)

	key := NewAPIKey()
	err := store.Upsert(context.Background(), HostConfig{
		APIKey:         key,
		AuthMode:       "forms",
		MigrationState: "clean",
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.APIKey != key {
		t.Errorf("APIKey roundtrip: got %q, want %q", got.APIKey, key)
	}
	if time.Since(got.CreatedAt) > time.Minute {
		t.Errorf("CreatedAt too old: %v", got.CreatedAt)
	}
}
```

- [ ] **Step 6: Implement postgres.go**

Create `internal/hostconfig/postgres.go`:
```go
package hostconfig

import (
	"context"
	"errors"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/db"
	pggen "github.com/ajthom90/sonarr2/internal/db/gen/postgres"
	"github.com/jackc/pgx/v5"
)

// PostgresStore implements Store against a Postgres database using the
// sqlc-generated queries in internal/db/gen/postgres.
type PostgresStore struct {
	q *pggen.Queries
}

// NewPostgresStore returns a Store backed by the given Postgres pool.
func NewPostgresStore(pool *db.PostgresPool) *PostgresStore {
	return &PostgresStore{q: pggen.New(pool.Raw())}
}

// Get implements Store.
func (s *PostgresStore) Get(ctx context.Context) (HostConfig, error) {
	row, err := s.q.GetHostConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return HostConfig{}, ErrNotFound
	}
	if err != nil {
		return HostConfig{}, fmt.Errorf("hostconfig: postgres get: %w", err)
	}
	return HostConfig{
		APIKey:         row.ApiKey,
		AuthMode:       row.AuthMode,
		MigrationState: row.MigrationState,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}, nil
}

// Upsert implements Store.
func (s *PostgresStore) Upsert(ctx context.Context, hc HostConfig) error {
	if err := s.q.UpsertHostConfig(ctx, pggen.UpsertHostConfigParams{
		ApiKey:         hc.APIKey,
		AuthMode:       hc.AuthMode,
		MigrationState: hc.MigrationState,
	}); err != nil {
		return fmt.Errorf("hostconfig: postgres upsert: %w", err)
	}
	return nil
}
```

**Note on field names:** The exact field names in the sqlc-generated `pggen.GetHostConfigRow` and `pggen.UpsertHostConfigParams` structs depend on sqlc's naming rules. Typically `api_key` becomes `ApiKey`. If sqlc produces different names (e.g., `APIKey` with upper-case acronym handling), adjust the code accordingly. Run `go doc github.com/ajthom90/sonarr2/internal/db/gen/postgres` after regenerating to confirm. The generated timestamp fields are typically `pgtype.Timestamptz` with a `.Time` accessor — the code above uses `.Time`, which is correct for pgx v5.

- [ ] **Step 7: Write the SQLite Store tests**

Create `internal/hostconfig/sqlite_test.go`:
```go
package hostconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
)

func setupSQLiteForTest(t *testing.T) *db.SQLitePool {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return pool
}

func TestSQLiteStoreGetReturnsNotFoundWhenEmpty(t *testing.T) {
	pool := setupSQLiteForTest(t)
	store := NewSQLiteStore(pool)
	_, err := store.Get(context.Background())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStoreUpsertAndGet(t *testing.T) {
	pool := setupSQLiteForTest(t)
	store := NewSQLiteStore(pool)

	want := HostConfig{
		APIKey:         "sqlite-test-key",
		AuthMode:       "forms",
		MigrationState: "clean",
	}
	if err := store.Upsert(context.Background(), want); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if got.AuthMode != want.AuthMode {
		t.Errorf("AuthMode = %q, want %q", got.AuthMode, want.AuthMode)
	}
}

func TestSQLiteNewAPIKeyRoundtrip(t *testing.T) {
	pool := setupSQLiteForTest(t)
	store := NewSQLiteStore(pool)

	key := NewAPIKey()
	err := store.Upsert(context.Background(), HostConfig{
		APIKey:         key,
		AuthMode:       "forms",
		MigrationState: "clean",
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.APIKey != key {
		t.Errorf("APIKey roundtrip: got %q, want %q", got.APIKey, key)
	}
}
```

- [ ] **Step 8: Implement sqlite.go**

Create `internal/hostconfig/sqlite.go`:
```go
package hostconfig

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	sqlitegen "github.com/ajthom90/sonarr2/internal/db/gen/sqlite"
)

// SQLiteStore implements Store against a SQLite database using the
// sqlc-generated queries in internal/db/gen/sqlite. All writes funnel
// through the pool's writer goroutine; reads go through the read-only
// pool.
type SQLiteStore struct {
	pool *db.SQLitePool
}

// NewSQLiteStore returns a Store backed by the given SQLite pool.
func NewSQLiteStore(pool *db.SQLitePool) *SQLiteStore {
	return &SQLiteStore{pool: pool}
}

// Get implements Store.
func (s *SQLiteStore) Get(ctx context.Context) (HostConfig, error) {
	var hc HostConfig
	err := s.pool.Read(ctx, func(q db.Querier) error {
		// The sqlc-generated code expects a DBTX that satisfies its interface
		// over database/sql. We wrap the Querier in a tiny adapter.
		reader := &sqliteReaderAdapter{q: q}
		queries := sqlitegen.New(reader)
		row, err := queries.GetHostConfig(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("hostconfig: sqlite get: %w", err)
		}
		createdAt, _ := time.Parse("2006-01-02 15:04:05", row.CreatedAt)
		updatedAt, _ := time.Parse("2006-01-02 15:04:05", row.UpdatedAt)
		hc = HostConfig{
			APIKey:         row.ApiKey,
			AuthMode:       row.AuthMode,
			MigrationState: row.MigrationState,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}
		return nil
	})
	return hc, err
}

// Upsert implements Store.
func (s *SQLiteStore) Upsert(ctx context.Context, hc HostConfig) error {
	return s.pool.Write(ctx, func(exec db.Executor) error {
		queries := sqlitegen.New(&sqliteWriterAdapter{exec: exec})
		return queries.UpsertHostConfig(ctx, sqlitegen.UpsertHostConfigParams{
			ApiKey:         hc.APIKey,
			AuthMode:       hc.AuthMode,
			MigrationState: hc.MigrationState,
		})
	})
}

// sqliteReaderAdapter adapts a db.Querier to sqlc's DBTX interface for
// read-only operations. Write methods panic because they must not be
// called on a read adapter.
type sqliteReaderAdapter struct{ q db.Querier }

func (a *sqliteReaderAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	panic("sqliteReaderAdapter: ExecContext called on read-only adapter")
}
func (a *sqliteReaderAdapter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	panic("sqliteReaderAdapter: PrepareContext called on read-only adapter")
}
func (a *sqliteReaderAdapter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.q.QueryContext(ctx, query, args...)
}
func (a *sqliteReaderAdapter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.q.QueryRowContext(ctx, query, args...)
}

// sqliteWriterAdapter adapts a db.Executor to sqlc's DBTX interface.
type sqliteWriterAdapter struct{ exec db.Executor }

func (a *sqliteWriterAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return a.exec.ExecContext(ctx, query, args...)
}
func (a *sqliteWriterAdapter) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, errors.New("sqliteWriterAdapter: PrepareContext not supported")
}
func (a *sqliteWriterAdapter) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return a.exec.QueryContext(ctx, query, args...)
}
func (a *sqliteWriterAdapter) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return a.exec.QueryRowContext(ctx, query, args...)
}
```

**Note on db.Executor extension:** The `db.Executor` interface in Task 4 already has `ExecContext`, `QueryRowContext`, `QueryContext`. If sqlc's generated `DBTX` interface has additional methods (e.g., `PrepareContext`), this adapter's panic/error stubs handle them. Verify by inspecting `internal/db/gen/sqlite/db.go` after generation — if it references more methods, add them to the adapter.

- [ ] **Step 9: Run all tests**

```bash
cd /Users/ajthom90/projects/sonarr2
go test -race -v ./internal/hostconfig/...
```

Expected: unit tests pass. SQLite tests pass. Postgres tests skip if Docker unavailable, pass if Docker available.

- [ ] **Step 10: Commit**

```bash
git add internal/hostconfig/
git commit -m "feat(hostconfig): add Store interface with Postgres and SQLite implementations"
```

---

## Task 8 — Open Pool from Config helper

Add a helper in `internal/db` that takes a `config.DBConfig` and returns the appropriate `Pool`.

**Files:**
- Create: `internal/db/open.go`
- Create: `internal/db/open_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/db/open_test.go`:
```go
package db

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/config"
)

func TestOpenFromConfigSQLite(t *testing.T) {
	pool, err := OpenFromConfig(context.Background(), config.DBConfig{
		Dialect:     "sqlite",
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenFromConfig: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if pool.Dialect() != DialectSQLite {
		t.Errorf("Dialect() = %q, want sqlite", pool.Dialect())
	}
}

func TestOpenFromConfigUnknownDialect(t *testing.T) {
	_, err := OpenFromConfig(context.Background(), config.DBConfig{
		Dialect: "mysql",
		DSN:     "mysql://",
	})
	if err == nil {
		t.Fatal("expected error for unknown dialect")
	}
}
```

- [ ] **Step 2: Run test, expect failure**

```bash
go test ./internal/db/...
```

Expected: compilation failure because `OpenFromConfig` doesn't exist.

- [ ] **Step 3: Implement open.go**

Create `internal/db/open.go`:
```go
package db

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/config"
)

// OpenFromConfig returns a Pool for the configured dialect. It is the
// single entry point used by app.New — callers do not need to know which
// concrete type they get back.
func OpenFromConfig(ctx context.Context, cfg config.DBConfig) (Pool, error) {
	dialect, err := ParseDialect(cfg.Dialect)
	if err != nil {
		return nil, err
	}

	switch dialect {
	case DialectPostgres:
		return OpenPostgres(ctx, PostgresOptions{
			DSN:             cfg.DSN,
			MaxOpenConns:    cfg.MaxOpenConns,
			MinOpenConns:    cfg.MaxIdleConns, // reuse as min
			ConnMaxLifetime: cfg.ConnMaxLifetime,
		})
	case DialectSQLite:
		return OpenSQLite(ctx, SQLiteOptions{
			DSN:         cfg.DSN,
			BusyTimeout: cfg.BusyTimeout,
		})
	default:
		return nil, fmt.Errorf("db: unsupported dialect %q", dialect)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -race ./internal/db/...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/db/open.go internal/db/open_test.go
git commit -m "feat(db): add OpenFromConfig helper for dialect-agnostic pool creation"
```

---

## Task 9 — Wire DB into app.New and main.go

Modify `app.New` to return `(*App, error)` because DB open can fail. Open the pool, run migrations, and seed a default host_config on first run. Close the pool on shutdown. Update `cmd/sonarr/main.go` to handle the new error.

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `cmd/sonarr/main.go`

- [ ] **Step 1: Update the app test to expect the new signature**

Edit `internal/app/app_test.go` — replace the `a := New(cfg)` line in `TestAppRunAndShutdown` with:
```go
	a, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
```

Also add DB config overrides to the test so it uses in-memory SQLite:
```go
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second
```

Make sure `time` is imported.

- [ ] **Step 2: Run tests, expect failure**

```bash
go test ./internal/app/...
```

Expected: compilation failure because `New` returns only `*App`, not `(*App, error)`.

- [ ] **Step 3: Update internal/app/app.go**

Open `internal/app/app.go` and apply these changes:

**Update the `App` struct** to hold the pool:
```go
// App is the running sonarr2 process.
type App struct {
	log    *slog.Logger
	server *api.Server
	pool   db.Pool
}
```

**Add `"github.com/ajthom90/sonarr2/internal/db"` and `"github.com/ajthom90/sonarr2/internal/hostconfig"` to imports.**

**Rewrite `New`** to open the pool, migrate, and seed default host_config:
```go
// New constructs an App from the given config. It opens the database,
// runs migrations, and wires the HTTP server. If any step fails, returns
// the error. The caller is responsible for calling Run to start serving.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	log := logging.New(cfg.Logging, os.Stderr)

	pool, err := db.OpenFromConfig(ctx, cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("app: open db: %w", err)
	}

	if err := db.Migrate(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: migrate db: %w", err)
	}

	if err := seedHostConfig(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: seed host config: %w", err)
	}

	addr := net.JoinHostPort(cfg.HTTP.BindAddress, strconv.Itoa(cfg.HTTP.Port))
	return &App{
		log:    log,
		server: api.New(addr, log),
		pool:   pool,
	}, nil
}

// seedHostConfig inserts a default host_config row with a freshly generated
// API key if none exists. Called once on startup from New.
func seedHostConfig(ctx context.Context, pool db.Pool) error {
	var store hostconfig.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		store = hostconfig.NewPostgresStore(p)
	case *db.SQLitePool:
		store = hostconfig.NewSQLiteStore(p)
	default:
		return fmt.Errorf("app: unsupported pool type %T", pool)
	}

	_, err := store.Get(ctx)
	if err == nil {
		return nil // already seeded
	}
	if !errors.Is(err, hostconfig.ErrNotFound) {
		return fmt.Errorf("app: get host config: %w", err)
	}

	return store.Upsert(ctx, hostconfig.HostConfig{
		APIKey:         hostconfig.NewAPIKey(),
		AuthMode:       "forms",
		MigrationState: "clean",
	})
}
```

**Update `Run`** to close the pool on shutdown. After the existing `wg.Wait()` and before the final log line, add:
```go
	if err := a.pool.Close(); err != nil {
		a.log.Error("db close error", slog.String("err", err.Error()))
	}
```

**Also update imports** to add `errors` if not already present.

- [ ] **Step 4: Update cmd/sonarr/main.go**

Edit `cmd/sonarr/main.go`, replacing the `run()` function body:
```go
func run() error {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := app.SignalContext(context.Background())
	defer cancel()

	a, err := app.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("new app: %w", err)
	}
	return a.Run(ctx)
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd /Users/ajthom90/projects/sonarr2
go test -race -v ./internal/app/... ./cmd/sonarr/...
```

Expected: all tests pass. The integration test in app_test.go now opens an in-memory SQLite, runs migrations, seeds host_config, and still verifies the /ping endpoint works end-to-end.

- [ ] **Step 6: Verify the binary still builds and runs**

```bash
make clean
make build
ls -l dist/sonarr2

mkdir -p /tmp/sonarr2-m1-data
./dist/sonarr2 \
  -port 18991 -bind 127.0.0.1 -log-format text \
  -db-dialect sqlite \
  -db-dsn "file:/tmp/sonarr2-m1-data/test.db?_journal=WAL&_busy_timeout=5000" &
PID=$!
sleep 2
curl -sf http://127.0.0.1:18991/ping
curl -sf http://127.0.0.1:18991/api/v3/system/status
kill $PID
wait $PID 2>/dev/null || true
ls /tmp/sonarr2-m1-data/
rm -rf /tmp/sonarr2-m1-data
```

Expected:
- `/ping` returns `{"status":"ok"}`
- `/api/v3/system/status` returns the standard JSON
- `/tmp/sonarr2-m1-data/test.db` exists (and `-wal`, `-shm` companion files)
- Binary exits cleanly

- [ ] **Step 7: Commit**

```bash
git add internal/app/ cmd/sonarr/
git commit -m "feat(app): open db pool, migrate, and seed host config on startup"
```

---

## Task 10 — Extend /api/v3/system/status with database status

Add a `database` field to the status response containing the dialect and connectivity state.

**Files:**
- Modify: `internal/api/server.go`
- Modify: `internal/api/server_test.go`
- Modify: `internal/app/app.go` (pass pool to api.New)

- [ ] **Step 1: Update api server_test.go**

Edit `internal/api/server_test.go`, adding a new test after the existing `TestStatusHandlerReturnsBuildInfo`:

```go
func TestStatusHandlerIncludesDatabase(t *testing.T) {
	ping := &stubPool{dialect: "sqlite", pingErr: nil}
	srv := httptest.NewServer(HandlerWithPool(discardLogger(), ping))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v3/system/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	dbv, ok := body["database"].(map[string]any)
	if !ok {
		t.Fatalf("database field missing or wrong type: %T", body["database"])
	}
	if dbv["dialect"] != "sqlite" {
		t.Errorf("dialect = %v", dbv["dialect"])
	}
	if dbv["connected"] != true {
		t.Errorf("connected = %v", dbv["connected"])
	}
}

func TestStatusHandlerReportsDatabaseDown(t *testing.T) {
	down := &stubPool{dialect: "postgres", pingErr: errPingDown}
	srv := httptest.NewServer(HandlerWithPool(discardLogger(), down))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v3/system/status")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	dbv, _ := body["database"].(map[string]any)
	if dbv["connected"] != false {
		t.Errorf("connected = %v, want false", dbv["connected"])
	}
}

// stubPool is a minimal test double for the pool shape used by the status
// handler. It is defined in the test file because it is only needed here.
type stubPool struct {
	dialect string
	pingErr error
}

func (s *stubPool) Dialect() string                      { return s.dialect }
func (s *stubPool) Ping(ctx context.Context) error       { return s.pingErr }

var errPingDown = stubPingError{}

type stubPingError struct{}

func (stubPingError) Error() string { return "db: ping down" }
```

Add `"context"` to the test file's imports if not already present.

- [ ] **Step 2: Run tests, expect failure**

```bash
go test ./internal/api/...
```

Expected: compilation failure because `HandlerWithPool` doesn't exist.

- [ ] **Step 3: Update the test to match the final shape**

The `stubPool` in `server_test.go` should satisfy the api package's `PoolPinger` interface, which we define below with `Dialect() string` (NOT `db.Dialect`). This keeps the api package free of a `db` import — we cut the `api → config` edge at the end of M0 and we're not going to reintroduce a similar edge here.

Update `stubPool` and its construction in the test file to use `string`:
```go
type stubPool struct {
	dialect string
	pingErr error
}

func (s *stubPool) Dialect() string                  { return s.dialect }
func (s *stubPool) Ping(ctx context.Context) error   { return s.pingErr }
```

And in the test bodies:
```go
ping := &stubPool{dialect: "sqlite", pingErr: nil}
// ...
down := &stubPool{dialect: "postgres", pingErr: errPingDown}
```

No `internal/db` import is needed in the test file for this to work.

- [ ] **Step 4: Implement the changes in server.go**

Open `internal/api/server.go`. The `statusHandler` currently reads `buildinfo.Get()` only. It must now also report database state.

**Add the `PoolPinger` interface** — lean, stdlib-only, no `db` import:
```go
// PoolPinger is the minimum interface the status handler needs to report
// database connectivity. The api package intentionally does not import
// internal/db to keep this layer free of database-package coupling — the
// app composition root adapts a db.Pool to this interface.
type PoolPinger interface {
	Dialect() string
	Ping(ctx context.Context) error
}
```

**Add `HandlerWithPool`** as a new constructor:
```go
// HandlerWithPool builds the chi router with a database pool reference so
// the /api/v3/system/status handler can report db connectivity. Most code
// should use this instead of Handler.
func HandlerWithPool(log *slog.Logger, pool PoolPinger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)

	r.Get("/ping", pingHandler)
	r.Get("/api/v3/system/status", statusHandlerWithPool(pool))

	return r
}

// Handler builds the chi router without wrapping it in a full server.
// Convenience for tests that don't need a pool; the status handler tolerates
// a nil pool and reports database connected=false.
func Handler(log *slog.Logger) http.Handler {
	return HandlerWithPool(log, nil)
}
```

**Replace the old `statusHandler` function with `statusHandlerWithPool`:**
```go
func statusHandlerWithPool(pool PoolPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := buildinfo.Get()
		resp := map[string]any{
			"appName":   "sonarr2",
			"version":   info.Version,
			"buildTime": info.Date,
			"commit":    info.Commit,
		}

		if pool != nil {
			connected := pool.Ping(r.Context()) == nil
			resp["database"] = map[string]any{
				"dialect":   pool.Dialect(),
				"connected": connected,
			}
		} else {
			resp["database"] = map[string]any{
				"dialect":   "",
				"connected": false,
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
```

**Delete the old `statusHandler`** (it's replaced by `statusHandlerWithPool`).

**Update `New`** to accept and pass a pool:
```go
// New builds a Server bound to addr.
func New(addr string, log *slog.Logger, pool PoolPinger) *Server {
	return &Server{
		log: log,
		httpsrv: &http.Server{
			Addr:              addr,
			Handler:           HandlerWithPool(log, pool),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}
```

**Import check:** `server.go` should NOT import `internal/db` — only `context`, `encoding/json`, `fmt`, `log/slog`, `net/http`, `time`, `internal/buildinfo`, `go-chi/chi/v5`, `go-chi/chi/v5/middleware`.

- [ ] **Step 5: Update internal/app/app.go to adapt db.Pool to api.PoolPinger**

Open `internal/app/app.go`. The `db.Pool` interface has `Dialect() db.Dialect`, which does not satisfy `api.PoolPinger`'s `Dialect() string` (Go's method set matching is exact on return types). The composition root adapts between them.

Add this adapter type at the bottom of `app.go`:
```go
// poolPingerAdapter wraps a db.Pool to satisfy api.PoolPinger by returning
// the dialect as a plain string. This keeps the api package free of a
// db-package import.
type poolPingerAdapter struct {
	pool db.Pool
}

func (p poolPingerAdapter) Dialect() string                { return string(p.pool.Dialect()) }
func (p poolPingerAdapter) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }
```

Update the `New` function's return to pass the adapter to `api.New`:
```go
	return &App{
		log:    log,
		server: api.New(addr, log, poolPingerAdapter{pool: pool}),
		pool:   pool,
	}, nil
```

Verify that `app.go` does NOT import `internal/api` in a way that creates a circular dependency — it already imports `api` so we're fine. No new imports needed in app.go.

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test -race ./...
```

Expected: all tests pass. The existing `TestPingHandler` still works because `Handler(log)` passes a nil pool, and the status test now uses `HandlerWithPool(log, stub)`.

- [ ] **Step 7: Verify the api package has no db import**

```bash
grep -l "internal/db" internal/api/*.go || echo "OK: api has no db import"
```

Expected: output is `OK: api has no db import`. If any file matches, the coupling was reintroduced — go back and fix it.

- [ ] **Step 8: Commit**

```bash
git add internal/api/server.go internal/api/server_test.go internal/app/app.go
git commit -m "feat(api): extend status endpoint with database dialect and connectivity"
```

---

## Task 11 — CI workflow update for Postgres integration tests

Update `.github/workflows/test.yml` so the test job runs with Docker available (already default on `ubuntu-latest`) and explicitly runs `go test -race -tags=...` or a non-short mode to exercise the Postgres container tests.

**Files:**
- Modify: `.github/workflows/test.yml`

- [ ] **Step 1: Read the current workflow**

```bash
cat .github/workflows/test.yml
```

- [ ] **Step 2: Update the workflow**

Replace the `test` job body so it explicitly runs with `-count=1` and without `-short`, ensuring Postgres tests run in CI. The build job is unchanged.

Replace this block:
```yaml
      - name: go test
        run: go test -race -count=1 -v ./...
```

With this block:
```yaml
      - name: go test
        run: |
          # ubuntu-latest has Docker preinstalled and running, so
          # testcontainers-based Postgres tests will run against a real PG.
          go test -race -count=1 -v ./...
```

(The comment is the only functional change — but it's worth documenting the expectation for future maintainers. If you want to skip Docker-gated tests on CI, you'd add `-short`, but for M1+ we want the real coverage.)

Also add a `services: {}` or a simple step that verifies Docker works before running tests, to give a clear error if the runner's Docker is down. Add this step **before** `go test`:

```yaml
      - name: verify docker available
        run: |
          docker version
          docker info | head -20
```

- [ ] **Step 3: Verify YAML is still valid**

If `python3` is available:
```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/test.yml')); print('ok')"
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/test.yml
git commit -m "chore(ci): verify docker available before running testcontainers tests"
```

---

## Task 12 — Final verification

Run every gate from a clean state, then push.

**Files:** none (verification only)

- [ ] **Step 1: go mod tidy**

```bash
cd /Users/ajthom90/projects/sonarr2
go mod tidy
git diff go.mod go.sum
```

If there are changes, commit them:
```bash
git add go.mod go.sum
git commit -m "chore: go mod tidy after M1"
```

- [ ] **Step 2: Lint**

```bash
make lint
```

Expected: no output, exit 0. If golangci-lint or staticcheck are available locally, run them too:
```bash
golangci-lint run ./... 2>&1 || true
staticcheck ./... 2>&1 || true
```

If these find issues, report BLOCKED with the exact issues — don't fix them in the verification task.

- [ ] **Step 3: Full test suite under race detector**

```bash
make test
```

Or to include Postgres tests explicitly:
```bash
go test -race -count=1 -v ./...
```

Expected: all tests pass. SQLite tests pass unconditionally. Postgres tests pass if Docker is running; otherwise they skip with a message.

- [ ] **Step 4: Clean build**

```bash
make clean && make build
ls -l dist/sonarr2
```

Expected: binary exists.

- [ ] **Step 5: End-to-end smoke test with a persistent SQLite file**

```bash
mkdir -p /tmp/sonarr2-m1-verify
./dist/sonarr2 \
  -port 18992 -bind 127.0.0.1 -log-format text \
  -db-dialect sqlite \
  -db-dsn "file:/tmp/sonarr2-m1-verify/sonarr2.db?_journal=WAL&_busy_timeout=5000" &
PID=$!
sleep 2
echo "=== /ping ==="
curl -sf http://127.0.0.1:18992/ping
echo
echo "=== /api/v3/system/status ==="
curl -sf http://127.0.0.1:18992/api/v3/system/status | python3 -m json.tool || curl -sf http://127.0.0.1:18992/api/v3/system/status
echo
kill -TERM $PID
wait $PID 2>/dev/null || true
ls -l /tmp/sonarr2-m1-verify/
rm -rf /tmp/sonarr2-m1-verify
```

Expected:
- `/ping` returns `{"status":"ok"}`
- `/api/v3/system/status` returns JSON with `appName`, `version`, `buildTime`, `commit`, and a `database: {dialect: "sqlite", connected: true}` block
- `/tmp/sonarr2-m1-verify/sonarr2.db` exists (with `-wal` and `-shm` companion files), proving migrations ran and data persisted
- Process exits cleanly

- [ ] **Step 6: Git log inspection**

```bash
git log --oneline b6998f9..HEAD
```

Expected commits (counts approximate; adjust for fix commits):
- `feat(config): add DBConfig ...`
- `feat(db): add package skeleton ...`
- `feat(db): add Postgres pool ...`
- `feat(db): add SQLite pool ...`
- `feat(db): add goose migration runner ...`
- `feat(db): add sqlc config and generate ...`
- `feat(hostconfig): add Store interface ...`
- `feat(db): add OpenFromConfig helper ...`
- `feat(app): open db pool, migrate, and seed ...`
- `feat(api): extend status endpoint with database ...`
- `chore(ci): verify docker available ...`
- Any tidy or review-fix commits

- [ ] **Step 7: Push to origin**

```bash
git push origin main
```

Expected: push succeeds.

---

## Done

After Task 12 completes, Milestone 1 is complete. The repository now has:

- Dual-dialect database support (Postgres via pgx, SQLite via modernc)
- Versioned migrations via goose with per-dialect subdirectories
- sqlc-generated typed query code committed and building
- A `HostConfig` domain entity with a `Store` interface and two implementations
- Single-writer discipline on SQLite (no "database is locked" at the app layer)
- `app.New` opens the DB, runs migrations, seeds a default host_config, and closes on shutdown
- `/api/v3/system/status` reports database dialect and connectivity
- CI matrix that exercises the DB layer on every PR

**Next milestone:** M2 — Domain core + event bus. Adds the `internal/events` package (typed generic pub/sub), the first domain entities (Series, Season, Episode, EpisodeFile with CRUD stores), and the `series_statistics` cached counter table. This builds directly on M1's sqlc + Store pattern.
