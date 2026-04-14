package health_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ajthom90/sonarr2/internal/health"
)

func TestApiKeyValidation(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		level health.Level
	}{
		{"empty", "", health.LevelError},
		{"short", "abc123", health.LevelWarning},
		{"ok", "1234567890abcdef1234567890abcdef", ""}, // OK = no results
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			chk := health.NewApiKeyValidationCheck(func() string { return c.key })
			results := chk.Check(context.Background())
			if c.level == "" {
				if len(results) != 0 {
					t.Errorf("expected no results, got %v", results)
				}
				return
			}
			if len(results) != 1 || results[0].Type != c.level {
				t.Errorf("expected one result of %q, got %v", c.level, results)
			}
		})
	}
}

func TestAppDataLocationCheck(t *testing.T) {
	dir := t.TempDir()
	chk := health.NewAppDataLocationCheck(func() string { return dir })
	if got := chk.Check(context.Background()); len(got) != 0 {
		t.Errorf("writable dir should be OK, got %v", got)
	}

	// Missing dir should error.
	bad := health.NewAppDataLocationCheck(func() string { return filepath.Join(dir, "nonexistent", "deep") })
	res := bad.Check(context.Background())
	if len(res) == 0 || res[0].Type != health.LevelError {
		t.Errorf("missing dir should error, got %v", res)
	}
}

func TestMountCheck(t *testing.T) {
	good := t.TempDir()
	bad := filepath.Join(good, "missing-mount")
	chk := health.NewMountCheck(func() []string { return []string{good, bad} })
	results := chk.Check(context.Background())
	if len(results) != 1 || results[0].Type != health.LevelError {
		t.Errorf("expected 1 error result, got %v", results)
	}
}

func TestRecyclingBinCheck(t *testing.T) {
	dir := t.TempDir()
	chk := health.NewRecyclingBinCheck(func() string { return dir })
	if got := chk.Check(context.Background()); len(got) != 0 {
		t.Errorf("writable recycle dir should be OK, got %v", got)
	}

	// Disabled (empty string) is OK.
	disabled := health.NewRecyclingBinCheck(func() string { return "" })
	if got := disabled.Check(context.Background()); len(got) != 0 {
		t.Errorf("disabled recycle bin should be OK, got %v", got)
	}

	// Not a directory → error.
	f, _ := os.CreateTemp(dir, "file")
	f.Close()
	notDir := health.NewRecyclingBinCheck(func() string { return f.Name() })
	if got := notDir.Check(context.Background()); len(got) == 0 || got[0].Type != health.LevelError {
		t.Errorf("non-dir recycle path should error")
	}
}

func TestProxyCheck(t *testing.T) {
	disabled := health.NewProxyCheck(func() bool { return false }, func() string { return "" }, func() int { return 0 })
	if got := disabled.Check(context.Background()); len(got) != 0 {
		t.Errorf("disabled proxy should be OK, got %v", got)
	}

	misconfig := health.NewProxyCheck(func() bool { return true }, func() string { return "" }, func() int { return 0 })
	if got := misconfig.Check(context.Background()); len(got) == 0 {
		t.Errorf("enabled but misconfigured proxy should error")
	}

	ok := health.NewProxyCheck(func() bool { return true }, func() string { return "proxy.local" }, func() int { return 3128 })
	if got := ok.Check(context.Background()); len(got) != 0 {
		t.Errorf("fully-configured proxy should be OK, got %v", got)
	}
}

func TestRemovedSeriesCheck(t *testing.T) {
	nil_fn := health.NewRemovedSeriesCheck(nil)
	if got := nil_fn.Check(context.Background()); len(got) != 0 {
		t.Error("nil callback should be OK")
	}
	chk := health.NewRemovedSeriesCheck(func() []string { return []string{"Show A", "Show B"} })
	results := chk.Check(context.Background())
	if len(results) != 1 || results[0].Type != health.LevelWarning {
		t.Errorf("expected 1 warning, got %v", results)
	}
}

func TestImportListStatusCheck(t *testing.T) {
	ok := health.NewImportListStatusCheck(nil)
	if got := ok.Check(context.Background()); len(got) != 0 {
		t.Error("nil callback should be OK")
	}
	failing := health.NewImportListStatusCheck(func() []string { return []string{"AniList"} })
	if got := failing.Check(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 warning, got %v", got)
	}
}

func TestNotificationStatusCheck(t *testing.T) {
	ok := health.NewNotificationStatusCheck(nil)
	if got := ok.Check(context.Background()); len(got) != 0 {
		t.Error("nil callback should be OK")
	}
	failing := health.NewNotificationStatusCheck(func() []string { return []string{"Discord"} })
	if got := failing.Check(context.Background()); len(got) != 1 {
		t.Errorf("expected 1 warning, got %v", got)
	}
}
