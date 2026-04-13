// Package config loads sonarr2 configuration from file, environment, and CLI flags.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/ajthom90/sonarr2/internal/logging"
	"gopkg.in/yaml.v3"
)

// Config is the full runtime configuration for sonarr2.
type Config struct {
	HTTP             HTTPConfig     `yaml:"http"`
	Logging          logging.Config `yaml:"logging"`
	Paths            PathsConfig    `yaml:"paths"`
	DB               DBConfig       `yaml:"db"`
	TVDB             TVDBConfig     `yaml:"tvdb"`
	HistoryRetention time.Duration  `yaml:"history_retention"`
	BackupRetention  int            `yaml:"backup_retention"`
	BackupInterval   time.Duration  `yaml:"backup_interval"`
	APIRateLimit     float64        `yaml:"api_rate_limit"`
	APIRateBurst     int            `yaml:"api_rate_burst"`
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

// DBConfig controls the database connection.
type DBConfig struct {
	Dialect         string        `yaml:"dialect"`
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	BusyTimeout     time.Duration `yaml:"busy_timeout"`
}

// TVDBConfig controls the TVDB metadata source.
type TVDBConfig struct {
	ApiKey           string        `yaml:"api_key"`
	CacheSeriesTTL   time.Duration `yaml:"cache_series_ttl"`
	CacheEpisodesTTL time.Duration `yaml:"cache_episodes_ttl"`
	CacheSearchTTL   time.Duration `yaml:"cache_search_ttl"`
	RateLimit        float64       `yaml:"rate_limit"`
	RateBurst        int           `yaml:"rate_burst"`
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
		DB: DBConfig{
			Dialect:         "sqlite",
			DSN:             "file:./data/sonarr2.db?_journal=WAL&_busy_timeout=5000",
			MaxOpenConns:    20,
			MaxIdleConns:    2,
			ConnMaxLifetime: 30 * time.Minute,
			BusyTimeout:     5 * time.Second,
		},
		TVDB: TVDBConfig{
			CacheSeriesTTL:   24 * time.Hour,
			CacheEpisodesTTL: 6 * time.Hour,
			CacheSearchTTL:   time.Hour,
			RateLimit:        5,
			RateBurst:        10,
		},
		HistoryRetention: 90 * 24 * time.Hour,
		BackupRetention:  7,
		BackupInterval:   7 * 24 * time.Hour,
		APIRateLimit:     30,
		APIRateBurst:     100,
	}
}

// Load builds a Config from (lowest to highest precedence):
// defaults → YAML config file → environment variables → CLI flags.
// args is argv without the program name (typically os.Args[1:]).
func Load(args []string, getenv func(string) string) (Config, error) {
	cfg := Default()

	fs := flag.NewFlagSet("sonarr2", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configFile := fs.String("config-file", "", "Path to YAML config file")
	bindAddress := fs.String("bind", "", "HTTP bind address")
	port := fs.Int("port", 0, "HTTP port")
	logFormat := fs.String("log-format", "", "Log format (json|text)")
	logLevel := fs.String("log-level", "", "Log level (debug|info|warn|error)")
	dbDialect := fs.String("db-dialect", "", "Database dialect (sqlite|postgres)")
	dbDSN := fs.String("db-dsn", "", "Database DSN")

	if err := fs.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse flags: %w", err)
	}

	// 1. Config file path — flag takes priority over SONARR2_CONFIG_FILE env var.
	//    The file's contents are then overridden by env vars and flags in steps 2-3.
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
			return Config{}, fmt.Errorf("SONARR2_PORT must be an integer, got %q: %w", v, err)
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
	if v := getenv("SONARR2_TVDB_API_KEY"); v != "" {
		cfg.TVDB.ApiKey = v
	}
	if v := getenv("SONARR2_TVDB_CACHE_SERIES_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_TVDB_CACHE_SERIES_TTL must be a duration, got %q: %w", v, err)
		}
		cfg.TVDB.CacheSeriesTTL = d
	}
	if v := getenv("SONARR2_TVDB_CACHE_EPISODES_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_TVDB_CACHE_EPISODES_TTL must be a duration, got %q: %w", v, err)
		}
		cfg.TVDB.CacheEpisodesTTL = d
	}
	if v := getenv("SONARR2_TVDB_CACHE_SEARCH_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_TVDB_CACHE_SEARCH_TTL must be a duration, got %q: %w", v, err)
		}
		cfg.TVDB.CacheSearchTTL = d
	}
	if v := getenv("SONARR2_TVDB_RATE_LIMIT"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_TVDB_RATE_LIMIT must be a number, got %q: %w", v, err)
		}
		cfg.TVDB.RateLimit = f
	}
	if v := getenv("SONARR2_TVDB_RATE_BURST"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_TVDB_RATE_BURST must be an integer, got %q: %w", v, err)
		}
		cfg.TVDB.RateBurst = n
	}
	if v := getenv("SONARR2_HISTORY_RETENTION"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_HISTORY_RETENTION must be a duration, got %q: %w", v, err)
		}
		cfg.HistoryRetention = d
	}
	if v := getenv("SONARR2_BACKUP_RETENTION"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_BACKUP_RETENTION must be an integer, got %q: %w", v, err)
		}
		cfg.BackupRetention = n
	}
	if v := getenv("SONARR2_BACKUP_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_BACKUP_INTERVAL must be a duration, got %q: %w", v, err)
		}
		cfg.BackupInterval = d
	}
	if v := getenv("SONARR2_API_RATE_LIMIT"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_API_RATE_LIMIT must be a number, got %q: %w", v, err)
		}
		cfg.APIRateLimit = f
	}
	if v := getenv("SONARR2_API_RATE_BURST"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("SONARR2_API_RATE_BURST must be an integer, got %q: %w", v, err)
		}
		cfg.APIRateBurst = n
	}

	// 3. CLI flags override environment.
	if *bindAddress != "" {
		cfg.HTTP.BindAddress = *bindAddress
	}
	// port==0 means the flag was not provided (0 is also invalid, so Validate catches misuse).
	if *port != 0 {
		cfg.HTTP.Port = *port
	}
	if *logFormat != "" {
		cfg.Logging.Format = logging.Format(*logFormat)
	}
	if *logLevel != "" {
		cfg.Logging.Level = logging.Level(*logLevel)
	}
	if *dbDialect != "" {
		cfg.DB.Dialect = *dbDialect
	}
	if *dbDSN != "" {
		cfg.DB.DSN = *dbDSN
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
	return nil
}
