package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	// Flag should beat env and file (flag > env > file).
	cfg, err := Load([]string{"-config-file", path, "-port", "33333"}, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Port != 33333 {
		t.Errorf("precedence: flag should win, got port = %d", cfg.HTTP.Port)
	}

	// Env should beat file when no flag is provided (env > file).
	cfg2, err := Load([]string{"-config-file", path}, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg2.HTTP.Port != 22222 {
		t.Errorf("precedence: env should beat file, got port = %d", cfg2.HTTP.Port)
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
		"SONARR2_DB_DIALECT":        "postgres",
		"SONARR2_DB_DSN":            "postgres://user:pass@localhost/sonarr2",
		"SONARR2_DB_MAX_OPEN_CONNS": "50",
		"SONARR2_DB_MAX_IDLE_CONNS": "5",
		"SONARR2_DB_BUSY_TIMEOUT":   "10s",
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

func TestLoadTVDBEnvOverrides(t *testing.T) {
	env := map[string]string{
		"SONARR2_TVDB_API_KEY":            "my-key",
		"SONARR2_TVDB_CACHE_SERIES_TTL":   "48h",
		"SONARR2_TVDB_CACHE_EPISODES_TTL": "12h",
		"SONARR2_TVDB_CACHE_SEARCH_TTL":   "2h",
		"SONARR2_TVDB_RATE_LIMIT":         "10",
		"SONARR2_TVDB_RATE_BURST":         "20",
	}
	getenv := func(k string) string { return env[k] }

	cfg, err := Load(nil, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TVDB.ApiKey != "my-key" {
		t.Errorf("TVDB.ApiKey = %q, want my-key", cfg.TVDB.ApiKey)
	}
	if cfg.TVDB.CacheSeriesTTL != 48*time.Hour {
		t.Errorf("CacheSeriesTTL = %v, want 48h", cfg.TVDB.CacheSeriesTTL)
	}
	if cfg.TVDB.CacheEpisodesTTL != 12*time.Hour {
		t.Errorf("CacheEpisodesTTL = %v, want 12h", cfg.TVDB.CacheEpisodesTTL)
	}
	if cfg.TVDB.CacheSearchTTL != 2*time.Hour {
		t.Errorf("CacheSearchTTL = %v, want 2h", cfg.TVDB.CacheSearchTTL)
	}
	if cfg.TVDB.RateLimit != 10 {
		t.Errorf("RateLimit = %v, want 10", cfg.TVDB.RateLimit)
	}
	if cfg.TVDB.RateBurst != 20 {
		t.Errorf("RateBurst = %d, want 20", cfg.TVDB.RateBurst)
	}
}

func TestTVDBDefaults(t *testing.T) {
	cfg := Default()
	if cfg.TVDB.CacheSeriesTTL != 24*time.Hour {
		t.Errorf("default CacheSeriesTTL = %v, want 24h", cfg.TVDB.CacheSeriesTTL)
	}
	if cfg.TVDB.CacheEpisodesTTL != 6*time.Hour {
		t.Errorf("default CacheEpisodesTTL = %v, want 6h", cfg.TVDB.CacheEpisodesTTL)
	}
	if cfg.TVDB.CacheSearchTTL != time.Hour {
		t.Errorf("default CacheSearchTTL = %v, want 1h", cfg.TVDB.CacheSearchTTL)
	}
	if cfg.TVDB.RateLimit != 5 {
		t.Errorf("default RateLimit = %v, want 5", cfg.TVDB.RateLimit)
	}
	if cfg.TVDB.RateBurst != 10 {
		t.Errorf("default RateBurst = %d, want 10", cfg.TVDB.RateBurst)
	}
}

func TestHistoryRetentionDefault(t *testing.T) {
	cfg := Default()
	if cfg.HistoryRetention != 90*24*time.Hour {
		t.Errorf("default HistoryRetention = %v, want 2160h", cfg.HistoryRetention)
	}
}

func TestHistoryRetentionEnvOverride(t *testing.T) {
	env := map[string]string{"SONARR2_HISTORY_RETENTION": "720h"}
	cfg, err := Load(nil, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HistoryRetention != 720*time.Hour {
		t.Errorf("HistoryRetention = %v, want 720h", cfg.HistoryRetention)
	}
}
