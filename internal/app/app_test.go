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
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second

	a, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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
			_, _ = io.Copy(io.Discard, resp.Body)
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
