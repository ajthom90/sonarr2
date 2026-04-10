// Package config loads sonarr2 configuration from file, environment, and CLI flags.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
	fs.SetOutput(io.Discard)
	configFile := fs.String("config-file", "", "Path to YAML config file")
	bindAddress := fs.String("bind", "", "HTTP bind address")
	port := fs.Int("port", 0, "HTTP port")
	logFormat := fs.String("log-format", "", "Log format (json|text)")
	logLevel := fs.String("log-level", "", "Log level (debug|info|warn|error)")

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
