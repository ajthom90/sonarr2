package recyclebin_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/recyclebin"
)

func TestDeleteFilePermanent(t *testing.T) {
	src := filepath.Join(t.TempDir(), "file.mkv")
	must(t, os.WriteFile(src, []byte("x"), 0o644))

	dest, err := recyclebin.DeleteFile(src, "", "")
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if dest != "" {
		t.Errorf("expected empty dest for permanent delete, got %q", dest)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("file should be gone: %v", err)
	}
}

func TestDeleteFileToRecycle(t *testing.T) {
	tmp := t.TempDir()
	recycle := filepath.Join(tmp, "recycle")
	must(t, os.MkdirAll(recycle, 0o755))
	src := filepath.Join(tmp, "file.mkv")
	must(t, os.WriteFile(src, []byte("x"), 0o644))

	dest, err := recyclebin.DeleteFile(src, recycle, "Series1")
	if err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	want := filepath.Join(recycle, "Series1", "file.mkv")
	if dest != want {
		t.Errorf("dest = %q, want %q", dest, want)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("file not at recycle dest: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should be gone: %v", err)
	}
}

func TestCleanupRemovesOld(t *testing.T) {
	recycle := t.TempDir()
	old := filepath.Join(recycle, "old.mkv")
	new := filepath.Join(recycle, "new.mkv")
	must(t, os.WriteFile(old, []byte("old"), 0o644))
	must(t, os.WriteFile(new, []byte("new"), 0o644))
	past := time.Now().Add(-30 * 24 * time.Hour)
	must(t, os.Chtimes(old, past, past))

	n, err := recyclebin.Cleanup(recycle, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}
	if _, err := os.Stat(new); err != nil {
		t.Error("new file should remain")
	}
}

func TestCleanupDisabled(t *testing.T) {
	// maxAge=0 means never purge.
	recycle := t.TempDir()
	must(t, os.WriteFile(filepath.Join(recycle, "x"), []byte("x"), 0o644))
	n, err := recyclebin.Cleanup(recycle, 0)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if n != 0 {
		t.Errorf("removed = %d, want 0 when maxAge=0", n)
	}
}

func TestEmpty(t *testing.T) {
	recycle := t.TempDir()
	must(t, os.WriteFile(filepath.Join(recycle, "a"), []byte("a"), 0o644))
	must(t, os.WriteFile(filepath.Join(recycle, "b"), []byte("b"), 0o644))
	if err := recyclebin.Empty(recycle); err != nil {
		t.Fatalf("Empty: %v", err)
	}
	entries, _ := os.ReadDir(recycle)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
