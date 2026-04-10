# Milestone 0 — Project Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a bootable `sonarr2` binary that starts, loads config from file + env + flags, logs via `log/slog`, serves `/ping` and a stub `/api/v3/system/status`, and shuts down cleanly on SIGTERM/SIGINT. Ship a Makefile, Dockerfile, CONTRIBUTING.md, and GitHub Actions workflows for lint + test. No domain logic — this milestone is the skeleton every later milestone builds on.

**Architecture:** Single Go module at `github.com/ajthom90/sonarr2`. `cmd/sonarr/main.go` is a thin entry point that loads config and calls `internal/app.App.Run(ctx)`. The `app` package wires config → logger → HTTP server and owns the graceful shutdown lifecycle. Later milestones add database, scheduler, workers, and domain packages to the same composition root.

**Tech Stack:** Go 1.23, `net/http` + `github.com/go-chi/chi/v5` (HTTP router + middleware), `log/slog` (stdlib structured logging), `gopkg.in/yaml.v3` (config file parsing). No Cobra, no Viper, no Gin/Echo/Fiber.

---

## Project layout this milestone creates

```
sonarr2/
├── cmd/sonarr/main.go
├── internal/
│   ├── app/
│   │   ├── app.go
│   │   └── app_test.go
│   ├── api/
│   │   ├── server.go
│   │   └── server_test.go
│   ├── buildinfo/
│   │   ├── buildinfo.go
│   │   └── buildinfo_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── logging/
│       ├── logging.go
│       └── logging_test.go
├── docker/
│   └── Dockerfile
├── .github/workflows/
│   ├── lint.yml
│   └── test.yml
├── .golangci.yml
├── Makefile
├── CONTRIBUTING.md
├── go.mod
└── go.sum
```

Existing files not modified: `LICENSE`, `README.md`, `.gitignore`, `docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md`.

---

## Task 1 — Initialize Go module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize the module**

Run from the repo root:
```bash
go mod init github.com/ajthom90/sonarr2
```

Expected output:
```
go: creating new go.mod: module github.com/ajthom90/sonarr2
```

- [ ] **Step 2: Pin the Go version**

Edit `go.mod` so the `go` directive reads exactly:
```
module github.com/ajthom90/sonarr2

go 1.23
```

- [ ] **Step 3: Verify it builds (no code yet, so no-op is OK)**

Run:
```bash
go build ./...
```

Expected: no output, exit 0. `go build ./...` with no packages exits cleanly.

- [ ] **Step 4: Commit**

```bash
git add go.mod
git commit -m "chore: initialize go module github.com/ajthom90/sonarr2"
```

---

## Task 2 — Add buildinfo package

The `buildinfo` package exposes version, commit, and build date strings that are injected at link time via `-ldflags="-X ..."`. Defaults are used during `go test` and `go run` workflows.

**Files:**
- Create: `internal/buildinfo/buildinfo.go`
- Create: `internal/buildinfo/buildinfo_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/buildinfo/buildinfo_test.go`:
```go
package buildinfo

import "testing"

func TestGetReturnsNonEmptyDefaults(t *testing.T) {
	info := Get()
	if info.Version == "" {
		t.Error("Version must not be empty")
	}
	if info.Commit == "" {
		t.Error("Commit must not be empty")
	}
	if info.Date == "" {
		t.Error("Date must not be empty")
	}
}

func TestGetReflectsVariables(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	})

	Version = "1.2.3"
	Commit = "abcdef"
	Date = "2026-04-10T00:00:00Z"

	info := Get()
	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", info.Version)
	}
	if info.Commit != "abcdef" {
		t.Errorf("Commit = %q, want abcdef", info.Commit)
	}
	if info.Date != "2026-04-10T00:00:00Z" {
		t.Errorf("Date = %q, want 2026-04-10T00:00:00Z", info.Date)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/buildinfo/...
```

Expected: compilation failure because `Get`, `Version`, `Commit`, `Date` don't exist yet.

- [ ] **Step 3: Implement the package**

Create `internal/buildinfo/buildinfo.go`:
```go
// Package buildinfo exposes build-time metadata injected via -ldflags.
package buildinfo

// These values are overridden at link time with:
//
//	go build -ldflags="-X github.com/ajthom90/sonarr2/internal/buildinfo.Version=1.2.3 ..."
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Info is a snapshot of build metadata for serialization.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Get returns the current build metadata snapshot.
func Get() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./internal/buildinfo/...
```

Expected:
```
ok  	github.com/ajthom90/sonarr2/internal/buildinfo	0.00Xs
```

- [ ] **Step 5: Commit**

```bash
git add internal/buildinfo/
git commit -m "feat(buildinfo): add build metadata package"
```

---

## Task 3 — Add logging package

Thin wrapper around `log/slog` that picks JSON vs text and maps string level names to `slog.Level`.

**Files:**
- Create: `internal/logging/logging.go`
- Create: `internal/logging/logging_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/logging/logging_test.go`:
```go
package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: FormatJSON, Level: LevelInfo}, &buf)
	logger.Info("hello", slog.String("k", "v"))
	out := buf.String()
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("expected JSON with msg=hello, got: %s", out)
	}
	if !strings.Contains(out, `"k":"v"`) {
		t.Errorf("expected JSON with k=v, got: %s", out)
	}
}

func TestNewTextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: FormatText, Level: LevelInfo}, &buf)
	logger.Info("hello", slog.String("k", "v"))
	out := buf.String()
	if !strings.Contains(out, "msg=hello") {
		t.Errorf("expected text with msg=hello, got: %s", out)
	}
	if !strings.Contains(out, "k=v") {
		t.Errorf("expected text with k=v, got: %s", out)
	}
}

func TestNewDefaultsToJSONOnUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: "bogus", Level: LevelInfo}, &buf)
	logger.Info("hello")
	if !strings.Contains(buf.String(), `"msg":"hello"`) {
		t.Errorf("expected JSON fallback, got: %s", buf.String())
	}
}

func TestNewLevelDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: FormatJSON, Level: LevelDebug}, &buf)
	logger.Debug("debug message")
	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("expected debug message, got: %s", buf.String())
	}
}

func TestNewLevelInfoSuppressesDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := New(Config{Format: FormatJSON, Level: LevelInfo}, &buf)
	logger.Debug("debug message")
	if strings.Contains(buf.String(), "debug message") {
		t.Errorf("expected debug to be suppressed at info level, got: %s", buf.String())
	}
}

func TestParseLevelUnknownDefaultsToInfo(t *testing.T) {
	if parseLevel("nonsense") != slog.LevelInfo {
		t.Error("expected unknown level to default to info")
	}
}

func TestParseLevelKnown(t *testing.T) {
	cases := map[Level]slog.Level{
		LevelDebug: slog.LevelDebug,
		LevelInfo:  slog.LevelInfo,
		LevelWarn:  slog.LevelWarn,
		LevelError: slog.LevelError,
	}
	for in, want := range cases {
		if got := parseLevel(in); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./internal/logging/...
```

Expected: compilation failure — `New`, `Config`, `Format*`, `Level*`, `parseLevel` do not exist.

- [ ] **Step 3: Implement the package**

Create `internal/logging/logging.go`:
```go
// Package logging builds structured loggers on top of log/slog.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// Format selects the slog handler to use.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

// Level is a string name for a slog level.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Config controls logger construction.
type Config struct {
	Format Format `yaml:"format"`
	Level  Level  `yaml:"level"`
}

// New returns a slog.Logger writing to w using the handler and level from cfg.
// Unknown formats fall back to JSON; unknown levels fall back to info.
func New(cfg Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}
	var handler slog.Handler
	switch cfg.Format {
	case FormatText:
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}
	return slog.New(handler)
}

func parseLevel(l Level) slog.Level {
	switch strings.ToLower(string(l)) {
	case string(LevelDebug):
		return slog.LevelDebug
	case string(LevelWarn):
		return slog.LevelWarn
	case string(LevelError):
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/logging/...
```

Expected:
```
ok  	github.com/ajthom90/sonarr2/internal/logging	0.00Xs
```

- [ ] **Step 5: Commit**

```bash
git add internal/logging/
git commit -m "feat(logging): add slog-based logging package"
```

---

## Task 4 — Add config package

Config loads defaults, then a YAML file, then environment variables, then CLI flags — each layer overriding the previous one.

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Modify: `go.mod` (adds `gopkg.in/yaml.v3` dependency)

- [ ] **Step 1: Add the YAML dependency**

```bash
go get gopkg.in/yaml.v3@v3.0.1
```

Expected: `go.mod` gains a `require gopkg.in/yaml.v3 v3.0.1` line; `go.sum` is created/updated.

- [ ] **Step 2: Write the failing tests**

Create `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ajthom90/sonarr2/internal/logging"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.HTTP.Port != 8989 {
		t.Errorf("default port = %d, want 8989", cfg.HTTP.Port)
	}
	if cfg.HTTP.BindAddress != "0.0.0.0" {
		t.Errorf("default bind = %q, want 0.0.0.0", cfg.HTTP.BindAddress)
	}
	if cfg.Logging.Format != logging.FormatJSON {
		t.Errorf("default log format = %q, want json", cfg.Logging.Format)
	}
	if cfg.Logging.Level != logging.LevelInfo {
		t.Errorf("default log level = %q, want info", cfg.Logging.Level)
	}
}

func TestLoadNoArgsNoEnvUsesDefaults(t *testing.T) {
	cfg, err := Load(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 8989 {
		t.Errorf("port = %d, want 8989", cfg.HTTP.Port)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	env := map[string]string{
		"SONARR2_PORT":         "9999",
		"SONARR2_BIND_ADDRESS": "127.0.0.1",
		"SONARR2_LOG_LEVEL":    "debug",
		"SONARR2_LOG_FORMAT":   "text",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load(nil, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 9999 {
		t.Errorf("port = %d, want 9999", cfg.HTTP.Port)
	}
	if cfg.HTTP.BindAddress != "127.0.0.1" {
		t.Errorf("bind = %q, want 127.0.0.1", cfg.HTTP.BindAddress)
	}
	if cfg.Logging.Level != logging.LevelDebug {
		t.Errorf("level = %q, want debug", cfg.Logging.Level)
	}
	if cfg.Logging.Format != logging.FormatText {
		t.Errorf("format = %q, want text", cfg.Logging.Format)
	}
}

func TestLoadFlagsOverrideEnv(t *testing.T) {
	env := map[string]string{"SONARR2_PORT": "9999"}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load([]string{"-port", "7777", "-bind", "10.0.0.1"}, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 7777 {
		t.Errorf("port = %d, want 7777", cfg.HTTP.Port)
	}
	if cfg.HTTP.BindAddress != "10.0.0.1" {
		t.Errorf("bind = %q, want 10.0.0.1", cfg.HTTP.BindAddress)
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
http:
  port: 12345
  bind_address: 127.0.0.1
  url_base: /sonarr
logging:
  format: text
  level: warn
paths:
  config: /etc/sonarr2
  data: /var/lib/sonarr2
  logs: /var/log/sonarr2
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load([]string{"-config-file", path}, func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 12345 {
		t.Errorf("port = %d, want 12345", cfg.HTTP.Port)
	}
	if cfg.HTTP.BindAddress != "127.0.0.1" {
		t.Errorf("bind = %q, want 127.0.0.1", cfg.HTTP.BindAddress)
	}
	if cfg.HTTP.URLBase != "/sonarr" {
		t.Errorf("url_base = %q, want /sonarr", cfg.HTTP.URLBase)
	}
	if cfg.Logging.Format != logging.FormatText {
		t.Errorf("format = %q, want text", cfg.Logging.Format)
	}
	if cfg.Paths.Data != "/var/lib/sonarr2" {
		t.Errorf("paths.data = %q, want /var/lib/sonarr2", cfg.Paths.Data)
	}
}

func TestLoadConfigFilePrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  port: 12345\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{"SONARR2_PORT": "22222"}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load([]string{"-config-file", path, "-port", "33333"}, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 33333 {
		t.Errorf("precedence: flag should win, got port = %d", cfg.HTTP.Port)
	}
}

func TestLoadConfigFileFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  port: 12345\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{"SONARR2_CONFIG_FILE": path}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load(nil, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 12345 {
		t.Errorf("port = %d, want 12345", cfg.HTTP.Port)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := Load([]string{"-config-file", "/nonexistent/nope.yaml"}, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoadInvalidEnvPort(t *testing.T) {
	env := map[string]string{"SONARR2_PORT": "not-a-number"}
	_, err := Load(nil, func(k string) string { return env[k] })
	if err == nil {
		t.Fatal("expected error for invalid SONARR2_PORT")
	}
}

func TestValidateInvalidPort(t *testing.T) {
	cfg := Default()
	cfg.HTTP.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port 0")
	}
	cfg.HTTP.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for port 70000")
	}
}

func TestValidateEmptyBindAddress(t *testing.T) {
	cfg := Default()
	cfg.HTTP.BindAddress = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty bind address")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/config/...
```

Expected: compilation failure — `Default`, `Load`, `Config`, `HTTPConfig`, etc. do not exist.

- [ ] **Step 4: Implement the package**

Create `internal/config/config.go`:
```go
// Package config loads sonarr2 configuration from file, environment, and CLI flags.
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/ajthom90/sonarr2/internal/logging"
	"gopkg.in/yaml.v3"
)

// Config is the full runtime configuration for sonarr2.
type Config struct {
	HTTP    HTTPConfig     `yaml:"http"`
	Logging logging.Config `yaml:"logging"`
	Paths   PathsConfig    `yaml:"paths"`
}

// HTTPConfig controls the HTTP listener.
type HTTPConfig struct {
	BindAddress string `yaml:"bind_address"`
	Port        int    `yaml:"port"`
	URLBase     string `yaml:"url_base"`
}

// PathsConfig holds runtime filesystem locations.
type PathsConfig struct {
	Config string `yaml:"config"`
	Data   string `yaml:"data"`
	Logs   string `yaml:"logs"`
}

// Default returns the built-in defaults.
func Default() Config {
	return Config{
		HTTP: HTTPConfig{
			BindAddress: "0.0.0.0",
			Port:        8989,
			URLBase:     "",
		},
		Logging: logging.Config{
			Format: logging.FormatJSON,
			Level:  logging.LevelInfo,
		},
		Paths: PathsConfig{
			Config: "./config",
			Data:   "./data",
			Logs:   "./logs",
		},
	}
}

// Load builds a Config from (lowest to highest precedence):
// defaults → YAML config file → environment variables → CLI flags.
// args is argv without the program name (typically os.Args[1:]).
func Load(args []string, getenv func(string) string) (Config, error) {
	cfg := Default()

	fs := flag.NewFlagSet("sonarr2", flag.ContinueOnError)
	configFile := fs.String("config-file", "", "Path to YAML config file")
	bindAddress := fs.String("bind", "", "HTTP bind address")
	port := fs.Int("port", 0, "HTTP port")
	logFormat := fs.String("log-format", "", "Log format (json|text)")
	logLevel := fs.String("log-level", "", "Log level (debug|info|warn|error)")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	// 1. Config file (from flag or env).
	path := *configFile
	if path == "" {
		path = getenv("SONARR2_CONFIG_FILE")
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config file %q: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
		}
	}

	// 2. Environment variables override file.
	if v := getenv("SONARR2_BIND_ADDRESS"); v != "" {
		cfg.HTTP.BindAddress = v
	}
	if v := getenv("SONARR2_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_PORT must be an integer, got %q", v)
		}
		cfg.HTTP.Port = n
	}
	if v := getenv("SONARR2_URL_BASE"); v != "" {
		cfg.HTTP.URLBase = v
	}
	if v := getenv("SONARR2_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = logging.Format(v)
	}
	if v := getenv("SONARR2_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = logging.Level(v)
	}

	// 3. CLI flags override environment.
	if *bindAddress != "" {
		cfg.HTTP.BindAddress = *bindAddress
	}
	if *port != 0 {
		cfg.HTTP.Port = *port
	}
	if *logFormat != "" {
		cfg.Logging.Format = logging.Format(*logFormat)
	}
	if *logLevel != "" {
		cfg.Logging.Level = logging.Level(*logLevel)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate returns an error if the configuration is not usable.
func (c Config) Validate() error {
	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		return fmt.Errorf("http.port must be 1-65535, got %d", c.HTTP.Port)
	}
	if c.HTTP.BindAddress == "" {
		return errors.New("http.bind_address must not be empty")
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/config/...
```

Expected:
```
ok  	github.com/ajthom90/sonarr2/internal/config	0.00Xs
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/config/
git commit -m "feat(config): load from defaults, file, env, and flags"
```

---

## Task 5 — Add HTTP server package

The `api` package holds the chi router, common middleware, and two handlers: `/ping` (liveness) and `/api/v3/system/status` (stub returning build info).

**Files:**
- Create: `internal/api/server.go`
- Create: `internal/api/server_test.go`
- Modify: `go.mod` (adds `github.com/go-chi/chi/v5` dependency)

- [ ] **Step 1: Add the chi dependency**

```bash
go get github.com/go-chi/chi/v5@v5.1.0
```

Expected: `go.mod` gains a `require github.com/go-chi/chi/v5 v5.1.0` line; `go.sum` is updated.

- [ ] **Step 2: Write the failing tests**

Create `internal/api/server_test.go`:
```go
package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestPingHandler(t *testing.T) {
	srv := httptest.NewServer(Handler(discardLogger()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q, want application/json; charset=utf-8", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestStatusHandlerReturnsBuildInfo(t *testing.T) {
	origVersion := buildinfo.Version
	buildinfo.Version = "testversion"
	t.Cleanup(func() { buildinfo.Version = origVersion })

	srv := httptest.NewServer(Handler(discardLogger()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v3/system/status")
	if err != nil {
		t.Fatalf("GET status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["appName"] != "sonarr2" {
		t.Errorf("appName = %v, want sonarr2", body["appName"])
	}
	if body["version"] != "testversion" {
		t.Errorf("version = %v, want testversion", body["version"])
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	srv := httptest.NewServer(Handler(discardLogger()))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/does-not-exist")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
go test ./internal/api/...
```

Expected: compilation failure — `Handler` does not exist.

- [ ] **Step 4: Implement the package**

Create `internal/api/server.go`:
```go
// Package api hosts the HTTP server, router, and top-level request handlers.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server wraps a net/http server configured with the sonarr2 router.
type Server struct {
	log     *slog.Logger
	httpsrv *http.Server
}

// New builds a Server bound to cfg.BindAddress:cfg.Port.
func New(cfg config.HTTPConfig, log *slog.Logger) *Server {
	return &Server{
		log: log,
		httpsrv: &http.Server{
			Addr:              net.JoinHostPort(cfg.BindAddress, strconv.Itoa(cfg.Port)),
			Handler:           Handler(log),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Start blocks until the server stops or errors. ErrServerClosed from a clean
// Shutdown is not returned as an error.
func (s *Server) Start() error {
	s.log.Info("http server listening", slog.String("addr", s.httpsrv.Addr))
	if err := s.httpsrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server within the context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.httpsrv.Shutdown(ctx)
}

// Handler builds the chi router without wrapping it in a full server. Useful
// for tests that need to drive the handler directly.
func Handler(log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)

	r.Get("/ping", pingHandler)
	r.Get("/api/v3/system/status", statusHandler)

	return r
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	info := buildinfo.Get()
	resp := map[string]any{
		"appName":   "sonarr2",
		"version":   info.Version,
		"buildTime": info.Date,
		"commit":    info.Commit,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				log.Info("http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Duration("dur", time.Since(start)),
					slog.String("request_id", middleware.GetReqID(r.Context())),
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/api/...
```

Expected:
```
ok  	github.com/ajthom90/sonarr2/internal/api	0.0XXs
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/api/
git commit -m "feat(api): add chi-based HTTP server with /ping and /api/v3/system/status"
```

---

## Task 6 — Add app composition root

The `app` package wires `config`, `logging`, and `api` together and owns the shutdown lifecycle. Its tests start a real HTTP server on an ephemeral port and verify the `/ping` endpoint before cancelling the context.

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/app/app_test.go`:
```go
package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/ajthom90/sonarr2/internal/logging"
)

func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestAppRunAndShutdown(t *testing.T) {
	port := findFreePort(t)
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError // quiet tests

	a := New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("server did not start within 3s")
		}
		resp, err := http.Get(base + "/ping")
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}

func TestSignalContextCancelsOnParentCancel(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	ctx, cancel := SignalContext(parent)
	defer cancel()

	parentCancel()

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("SignalContext did not cancel when parent was cancelled")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/app/...
```

Expected: compilation failure — `New`, `App.Run`, `SignalContext` do not exist.

- [ ] **Step 3: Implement the package**

Create `internal/app/app.go`:
```go
// Package app is the composition root for sonarr2 — it wires the logger,
// HTTP server, and graceful shutdown together.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ajthom90/sonarr2/internal/api"
	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/ajthom90/sonarr2/internal/logging"
)

// App is the running sonarr2 process.
type App struct {
	cfg    config.Config
	log    *slog.Logger
	server *api.Server
}

// New constructs an App from the given config. It creates the logger and
// server but does not start any goroutines.
func New(cfg config.Config) *App {
	log := logging.New(cfg.Logging, os.Stderr)
	return &App{
		cfg:    cfg,
		log:    log,
		server: api.New(cfg.HTTP, log),
	}
}

// Run starts the HTTP server and blocks until ctx is cancelled or the server
// errors. It then performs a graceful shutdown with a 30s deadline.
func (a *App) Run(ctx context.Context) error {
	info := buildinfo.Get()
	a.log.Info("sonarr2 starting",
		slog.String("version", info.Version),
		slog.String("commit", info.Commit),
		slog.String("date", info.Date),
	)

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.server.Start(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	wg.Wait()
	a.log.Info("sonarr2 stopped")
	return nil
}

// SignalContext returns a context that cancels on SIGINT or SIGTERM, or when
// parent is cancelled.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/app/...
```

Expected:
```
ok  	github.com/ajthom90/sonarr2/internal/app	0.XXs
```

- [ ] **Step 5: Commit**

```bash
git add internal/app/
git commit -m "feat(app): add composition root with graceful shutdown"
```

---

## Task 7 — Add cmd/sonarr entry point

Thin `main` that loads config and calls `app.Run`.

**Files:**
- Create: `cmd/sonarr/main.go`

- [ ] **Step 1: Create the entry point**

Create `cmd/sonarr/main.go`:
```go
// Command sonarr is the sonarr2 server binary.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ajthom90/sonarr2/internal/app"
	"github.com/ajthom90/sonarr2/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "sonarr2: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	a := app.New(cfg)
	ctx, cancel := app.SignalContext(context.Background())
	defer cancel()
	return a.Run(ctx)
}
```

- [ ] **Step 2: Build the binary**

```bash
go build -o /tmp/sonarr2 ./cmd/sonarr
```

Expected: no output, exit 0. Binary created at `/tmp/sonarr2`.

- [ ] **Step 3: Run the binary in the background and smoke-test it**

```bash
/tmp/sonarr2 -port 18989 -bind 127.0.0.1 -log-format text &
SONARR_PID=$!
sleep 1
curl -sf http://127.0.0.1:18989/ping
echo
curl -sf http://127.0.0.1:18989/api/v3/system/status
echo
kill $SONARR_PID
wait $SONARR_PID 2>/dev/null || true
```

Expected:
```
{"status":"ok"}
{"appName":"sonarr2","buildTime":"unknown","commit":"unknown","version":"dev"}
```

And the process exits cleanly (no "killed" message because we caught SIGTERM).

- [ ] **Step 4: Commit**

```bash
git add cmd/sonarr/
git commit -m "feat(cmd): add sonarr2 main entry point"
```

---

## Task 8 — Add Makefile

The Makefile captures the canonical build/test/lint commands and injects build metadata via `-ldflags`.

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Create the Makefile**

Create `Makefile`:
```make
.PHONY: build test lint tidy run clean

BIN := sonarr2
OUT := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/ajthom90/sonarr2/internal/buildinfo.Version=$(VERSION) \
	-X github.com/ajthom90/sonarr2/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/ajthom90/sonarr2/internal/buildinfo.Date=$(DATE)

build:
	@mkdir -p $(OUT)
	CGO_ENABLED=0 go build -ldflags='$(LDFLAGS)' -o $(OUT)/$(BIN) ./cmd/sonarr

test:
	go test -race -count=1 ./...

lint:
	go vet ./...
	@fmt=$$(gofmt -l -s .); if [ -n "$$fmt" ]; then echo "gofmt issues:"; echo "$$fmt"; exit 1; fi

tidy:
	go mod tidy

run: build
	./$(OUT)/$(BIN)

clean:
	rm -rf $(OUT)
```

- [ ] **Step 2: Verify `make build` works**

```bash
make build
./dist/sonarr2 -help 2>&1 | head -20
```

Expected: `./dist/sonarr2` exists. `-help` prints the flag usage including `-port`, `-bind`, `-config-file`, `-log-format`, `-log-level` (exit status may be nonzero on `-help`, that's OK).

- [ ] **Step 3: Verify `make test` works**

```bash
make test
```

Expected: all packages `ok`, race detector enabled.

- [ ] **Step 4: Verify `make lint` works**

```bash
make lint
```

Expected: no output, exit 0.

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile with build/test/lint targets"
```

---

## Task 9 — Add Dockerfile

Multi-stage build producing a static binary in a distroless base image.

**Files:**
- Create: `docker/Dockerfile`
- Create: `docker/.dockerignore` (at repo root — dockerignore applies to build context)

Actually, `.dockerignore` lives at the root of the build context, not under `docker/`. We put it at the repo root.

- [ ] **Step 1: Create the Dockerfile**

Create `docker/Dockerfile`:
```dockerfile
# syntax=docker/dockerfile:1.7

# ---- Builder ----
FROM golang:1.23-alpine AS builder
WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source.
COPY . .

ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -trimpath \
      -ldflags="-s -w \
        -X github.com/ajthom90/sonarr2/internal/buildinfo.Version=$VERSION \
        -X github.com/ajthom90/sonarr2/internal/buildinfo.Commit=$COMMIT \
        -X github.com/ajthom90/sonarr2/internal/buildinfo.Date=$DATE" \
      -o /out/sonarr2 ./cmd/sonarr

# ---- Runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/sonarr2 /sonarr2
EXPOSE 8989
VOLUME ["/config", "/data"]
ENTRYPOINT ["/sonarr2"]
```

- [ ] **Step 2: Create .dockerignore**

Create `.dockerignore` at the repo root:
```
.git
.github
dist
docs
*.md
!README.md
.vscode
.idea
node_modules
coverage.out
*.test
```

- [ ] **Step 3: Build the image locally (if Docker is available)**

```bash
docker build -f docker/Dockerfile -t sonarr2:dev .
```

Expected: successful build. If Docker isn't available in the dev environment, skip this step and note it in the commit.

- [ ] **Step 4: Smoke-test the image (if built)**

```bash
docker run --rm -p 18989:8989 sonarr2:dev -bind 0.0.0.0 &
CID=$(docker ps -q --filter ancestor=sonarr2:dev | head -1)
sleep 2
curl -sf http://127.0.0.1:18989/ping && echo
docker stop "$CID" 2>/dev/null || true
```

Expected: `{"status":"ok"}`.

- [ ] **Step 5: Commit**

```bash
git add docker/Dockerfile .dockerignore
git commit -m "chore(docker): add multi-stage Dockerfile and .dockerignore"
```

---

## Task 10 — Add golangci-lint config and lint workflow

**Files:**
- Create: `.golangci.yml`
- Create: `.github/workflows/lint.yml`

- [ ] **Step 1: Create .golangci.yml**

Create `.golangci.yml`:
```yaml
run:
  timeout: 5m

linters:
  disable-all: true
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - gocritic
    - revive

linters-settings:
  revive:
    rules:
      - name: exported
        disabled: false
  gofmt:
    simplify: true

issues:
  exclude-dirs:
    - dist
    - vendor
```

- [ ] **Step 2: Create the lint workflow**

Create `.github/workflows/lint.yml`:
```yaml
name: lint

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: go vet
        run: go vet ./...

      - name: gofmt
        run: |
          output=$(gofmt -l -s .)
          if [ -n "$output" ]; then
            echo "::error::gofmt issues found in:"
            echo "$output"
            gofmt -d -s .
            exit 1
          fi

      - name: staticcheck
        uses: dominikh/staticcheck-action@v1.3.1
        with:
          version: "2024.1.1"
          install-go: false

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.60.3
```

- [ ] **Step 3: Run lint locally to verify clean state**

```bash
make lint
```

Expected: no output, exit 0. (CI will also run staticcheck and golangci-lint, which take longer — locally `make lint` covers vet + gofmt.)

- [ ] **Step 4: Commit**

```bash
git add .golangci.yml .github/workflows/lint.yml
git commit -m "chore(ci): add lint workflow and golangci-lint config"
```

---

## Task 11 — Add test workflow

**Files:**
- Create: `.github/workflows/test.yml`

- [ ] **Step 1: Create the workflow**

Create `.github/workflows/test.yml`:
```yaml
name: test

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: download modules
        run: go mod download

      - name: go test
        run: go test -race -count=1 -v ./...

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: build binary
        run: |
          CGO_ENABLED=0 go build -o /tmp/sonarr2 ./cmd/sonarr
          /tmp/sonarr2 -port 18989 -bind 127.0.0.1 -log-format text &
          PID=$!
          sleep 1
          curl -sf http://127.0.0.1:18989/ping
          curl -sf http://127.0.0.1:18989/api/v3/system/status
          kill $PID
          wait $PID 2>/dev/null || true
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/test.yml
git commit -m "chore(ci): add test workflow with build smoke test"
```

---

## Task 12 — Add CONTRIBUTING.md

**Files:**
- Create: `CONTRIBUTING.md`

- [ ] **Step 1: Create the file**

Create `CONTRIBUTING.md`:
```markdown
# Contributing to sonarr2

Thanks for your interest. This document explains how to set up a dev environment and submit changes.

## Dev requirements

- Go 1.23 or newer
- git
- Docker (for building release images and running integration tests against Postgres in later milestones)

## Getting started

```bash
git clone https://github.com/ajthom90/sonarr2.git
cd sonarr2
make test
make build
./dist/sonarr2 -port 18989 -bind 127.0.0.1 -log-format text
curl http://127.0.0.1:18989/ping
```

## Project layout

See the [design doc](./docs/superpowers/specs/2026-04-10-sonarr-rewrite-design.md) §3.2 for the full layout. At a high level:

- `cmd/sonarr/` — main entry point
- `internal/app/` — composition root
- `internal/api/` — HTTP server, routes, middleware
- `internal/config/` — configuration loading
- `internal/logging/` — structured logging setup
- `internal/buildinfo/` — build metadata

## Making changes

1. Open an issue first for any non-trivial change so we can discuss direction.
2. Create a branch off `main`. Use a descriptive name (e.g., `feat/rss-sync`, `fix/sqlite-busy-timeout`).
3. Follow TDD: write a failing test, make it pass, commit. Prefer many small commits over one large one.
4. Run `make lint test` before pushing.
5. Open a PR. CI must pass before review.

## Commit messages

- Lowercase, imperative, present tense.
- Use a scope: `api: return 404 on unknown routes`.
- Prefixes are free-form but common ones are `feat`, `fix`, `chore`, `docs`, `test`, `refactor`.
- Include a "why" in the body when the "what" isn't obvious from the diff.

## Coding standards

- `gofmt -s` formatted
- `go vet`, `staticcheck`, and `golangci-lint` must pass
- All exported identifiers have godoc comments
- Package-level comments on every `internal/*` package
- Tests in the same package (white-box) unless a test needs to avoid circular imports
- Table-driven tests preferred over multiple top-level test functions for closely related cases
- Run `go test -race ./...` — we gate CI on race detector output

## License

By contributing you agree your contribution will be licensed under the MIT license (see [LICENSE](./LICENSE)).
```

- [ ] **Step 2: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add CONTRIBUTING.md"
```

---

## Task 13 — Final verification

Run every gate one more time from a clean state to confirm the milestone is done.

**Files:** none (verification only)

- [ ] **Step 1: Run `go mod tidy` to clean any stray imports**

```bash
go mod tidy
git diff go.mod go.sum
```

Expected: no diff. If there is a diff, commit it.

- [ ] **Step 2: Run the full lint**

```bash
make lint
```

Expected: no output, exit 0.

- [ ] **Step 3: Run the full test suite with race detector**

```bash
make test
```

Expected: all packages `ok`.

- [ ] **Step 4: Clean build**

```bash
make clean
make build
ls -l dist/sonarr2
file dist/sonarr2
```

Expected: binary exists. On Linux, `file` reports a statically linked ELF binary. On macOS, Mach-O.

- [ ] **Step 5: Run the binary and exercise both endpoints**

```bash
./dist/sonarr2 -port 18990 -bind 127.0.0.1 -log-format text &
SONARR_PID=$!
sleep 1
echo "=== /ping ==="
curl -sf http://127.0.0.1:18990/ping
echo
echo "=== /api/v3/system/status ==="
curl -sf http://127.0.0.1:18990/api/v3/system/status
echo
kill -TERM $SONARR_PID
wait $SONARR_PID 2>/dev/null || true
```

Expected:
```
=== /ping ===
{"status":"ok"}
=== /api/v3/system/status ===
{"appName":"sonarr2","buildTime":"...","commit":"...","version":"..."}
```

Process exits cleanly because `app.SignalContext` catches SIGTERM.

- [ ] **Step 6: Commit any stray changes from verification (if needed)**

```bash
git status
# If there are any changes, commit them with an appropriate message.
# If clean, skip this step.
```

- [ ] **Step 7: Push to origin**

```bash
git push origin main
```

---

## Done

After Task 13 completes, Milestone 0 is complete. The repository now has:

- A buildable, runnable binary with config loading, structured logging, and graceful shutdown
- Unit tests for every internal package
- An integration test that boots the app, hits `/ping`, and shuts down
- CI for lint and test
- A multi-stage Dockerfile targeting `gcr.io/distroless/static-debian12:nonroot`
- Contributor documentation

**Next milestone:** M1 — Database foundation. Sets up `pgxpool`, `modernc.org/sqlite`, goose migrations (per-dialect), sqlc generation, repository interface pattern, and dual-dialect integration tests. The first real migration creates the `host_config` and `_migration_state` tables so the app has a place to persist its API key and migration state.
