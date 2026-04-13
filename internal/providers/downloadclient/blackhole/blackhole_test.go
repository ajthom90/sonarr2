package blackhole

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlackholeAdd_NZBUrl(t *testing.T) {
	const nzbContent = `<?xml version="1.0" encoding="UTF-8"?><nzb xmlns="http://www.newzbin.com/DTD/2003/nzb"></nzb>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-nzb")
		_, _ = w.Write([]byte(nzbContent))
	}))
	defer srv.Close()

	dir := t.TempDir()
	b := New(Settings{WatchFolder: dir}, srv.Client())

	dest, err := b.Add(context.Background(), srv.URL+"/test.nzb", "Show.S01E01.720p")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	if !strings.HasSuffix(dest, ".nzb") {
		t.Errorf("expected .nzb extension, got %q", dest)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != nzbContent {
		t.Errorf("file content mismatch: got %q", string(data))
	}
}

func TestBlackholeAdd_TorrentUrl(t *testing.T) {
	const torrentContent = "d8:announce..."

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write([]byte(torrentContent))
	}))
	defer srv.Close()

	dir := t.TempDir()
	b := New(Settings{WatchFolder: dir}, srv.Client())

	dest, err := b.Add(context.Background(), srv.URL+"/test.torrent", "Show.S01E01.720p")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if !strings.HasSuffix(dest, ".torrent") {
		t.Errorf("expected .torrent extension, got %q", dest)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != torrentContent {
		t.Errorf("file content mismatch")
	}
}

func TestBlackholeAdd_MagnetLink(t *testing.T) {
	dir := t.TempDir()
	b := New(Settings{WatchFolder: dir}, nil)

	magnet := "magnet:?xt=urn:btih:abc123&dn=Show.S01E01"
	dest, err := b.Add(context.Background(), magnet, "Show.S01E01")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if !strings.HasSuffix(dest, ".torrent") {
		t.Errorf("expected .torrent extension, got %q", dest)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != magnet {
		t.Errorf("magnet content mismatch: got %q", string(data))
	}
}

func TestBlackholeAdd_TitleSanitization(t *testing.T) {
	const content = "data"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer srv.Close()

	dir := t.TempDir()
	b := New(Settings{WatchFolder: dir}, srv.Client())

	dest, err := b.Add(context.Background(), srv.URL+"/test.nzb", "Show: The Movie/Part 1")
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	// Path should not contain : or /
	base := filepath.Base(dest)
	if strings.Contains(base, ":") || strings.Contains(base, "/") {
		t.Errorf("sanitized filename still contains invalid chars: %q", base)
	}
}

func TestBlackholeItems(t *testing.T) {
	b := New(Settings{WatchFolder: "/tmp"}, nil)
	items, err := b.Items(context.Background())
	if err != nil {
		t.Fatalf("Items returned error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty items for blackhole, got %d", len(items))
	}
}

func TestBlackholeRemoveIsNoop(t *testing.T) {
	b := New(Settings{WatchFolder: "/tmp"}, nil)
	if err := b.Remove(context.Background(), "anything", true); err != nil {
		t.Errorf("Remove should be a no-op, got error: %v", err)
	}
}

func TestBlackholeTest_ValidFolder(t *testing.T) {
	dir := t.TempDir()
	b := New(Settings{WatchFolder: dir}, nil)
	if err := b.Test(context.Background()); err != nil {
		t.Errorf("Test() returned unexpected error: %v", err)
	}
}

func TestBlackholeTest_MissingFolder(t *testing.T) {
	b := New(Settings{WatchFolder: "/nonexistent/path/xyz"}, nil)
	if err := b.Test(context.Background()); err == nil {
		t.Fatal("Test() should return error for non-existent folder")
	}
}

func TestBlackholeTest_EmptyFolder(t *testing.T) {
	b := New(Settings{WatchFolder: ""}, nil)
	if err := b.Test(context.Background()); err == nil {
		t.Fatal("Test() should return error when WatchFolder is empty")
	}
}
