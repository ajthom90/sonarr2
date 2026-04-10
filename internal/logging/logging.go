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
