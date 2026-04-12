package commands_test

import (
	"log/slog"
	"testing"
)

// newTestLogger returns a slog.Logger that writes to t.Log at debug level,
// automatically scoped to the test.
func newTestLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(testWriter{t: t}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// testWriter adapts testing.T to an io.Writer so slog can write to the test log.
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
