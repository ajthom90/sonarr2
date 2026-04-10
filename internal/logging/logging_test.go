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
