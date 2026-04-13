package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// discardLogger returns a slog.Logger that discards all output.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// tarEntries opens a tar.gz file and returns the list of entry names.
func tarEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

// readManifest extracts and decodes manifest.json from a tar.gz archive.
func readManifest(t *testing.T, path string) Manifest {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		if hdr.Name != "manifest.json" {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		var m Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		return m
	}
	t.Fatal("manifest.json not found in archive")
	return Manifest{}
}

// hasEntry reports whether names contains target.
func hasEntry(names []string, target string) bool {
	for _, n := range names {
		if n == target {
			return true
		}
	}
	return false
}

func TestCreateBackupSQLite(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sonarr2.db")
	if err := os.WriteFile(dbPath, []byte("fake-db-contents"), 0o644); err != nil {
		t.Fatalf("write fake db: %v", err)
	}

	backupDir := filepath.Join(dir, "backups")
	svc := New(Options{
		BackupDir:  backupDir,
		DBPath:     dbPath,
		DBDialect:  "sqlite",
		AppVersion: "0.1.0",
		Retention:  7,
		Log:        discardLogger(),
	})

	before := time.Now().Add(-time.Second)
	info, err := svc.Create(context.Background())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	after := time.Now().Add(time.Second)

	// Verify returned Info
	if info.Name == "" {
		t.Error("expected non-empty Name")
	}
	if info.Size <= 0 {
		t.Errorf("expected Size > 0, got %d", info.Size)
	}
	if info.Timestamp.Before(before) || info.Timestamp.After(after) {
		t.Errorf("Timestamp %v not in expected range [%v, %v]", info.Timestamp, before, after)
	}

	// Verify file exists on disk
	archivePath := filepath.Join(backupDir, info.Name)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive file not found: %v", err)
	}

	// Verify tar.gz contents
	names := tarEntries(t, archivePath)
	if !hasEntry(names, "manifest.json") {
		t.Errorf("archive missing manifest.json; entries: %v", names)
	}
	if !hasEntry(names, "sonarr2.db") {
		t.Errorf("archive missing sonarr2.db; entries: %v", names)
	}

	// Verify manifest fields
	m := readManifest(t, archivePath)
	if m.Version != "0.1.0" {
		t.Errorf("manifest.Version = %q, want %q", m.Version, "0.1.0")
	}
	if m.DBDialect != "sqlite" {
		t.Errorf("manifest.DBDialect = %q, want %q", m.DBDialect, "sqlite")
	}
	if m.Timestamp.IsZero() {
		t.Error("manifest.Timestamp is zero")
	}
}

func TestCreateBackupPostgres(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backups")
	svc := New(Options{
		BackupDir:  backupDir,
		DBPath:     "", // no SQLite file for postgres
		DBDialect:  "postgres",
		AppVersion: "0.1.0",
		Retention:  7,
		Log:        discardLogger(),
	})

	info, err := svc.Create(context.Background())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	archivePath := filepath.Join(backupDir, info.Name)
	names := tarEntries(t, archivePath)

	if !hasEntry(names, "manifest.json") {
		t.Errorf("archive missing manifest.json; entries: %v", names)
	}
	if hasEntry(names, "sonarr2.db") {
		t.Errorf("postgres backup should not contain sonarr2.db; entries: %v", names)
	}

	m := readManifest(t, archivePath)
	if m.DBDialect != "postgres" {
		t.Errorf("manifest.DBDialect = %q, want %q", m.DBDialect, "postgres")
	}
}

func TestListBackups(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sonarr2.db")
	if err := os.WriteFile(dbPath, []byte("fake-db-contents"), 0o644); err != nil {
		t.Fatalf("write fake db: %v", err)
	}

	backupDir := filepath.Join(dir, "backups")
	svc := New(Options{
		BackupDir:  backupDir,
		DBPath:     dbPath,
		DBDialect:  "sqlite",
		AppVersion: "0.1.0",
		Retention:  7,
		Log:        discardLogger(),
	})

	if _, err := svc.Create(context.Background()); err != nil {
		t.Fatalf("Create 1: %v", err)
	}
	// Sleep long enough that the filenames (second-resolution) differ.
	time.Sleep(1100 * time.Millisecond)
	if _, err := svc.Create(context.Background()); err != nil {
		t.Fatalf("Create 2: %v", err)
	}

	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(list))
	}
	// Newest should be first
	if !list[0].Timestamp.After(list[1].Timestamp) {
		t.Errorf("expected newest first: [0].Timestamp=%v [1].Timestamp=%v",
			list[0].Timestamp, list[1].Timestamp)
	}
}

func TestDeleteBackup(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sonarr2.db")
	if err := os.WriteFile(dbPath, []byte("fake-db-contents"), 0o644); err != nil {
		t.Fatalf("write fake db: %v", err)
	}

	backupDir := filepath.Join(dir, "backups")
	svc := New(Options{
		BackupDir:  backupDir,
		DBPath:     dbPath,
		DBDialect:  "sqlite",
		AppVersion: "0.1.0",
		Retention:  7,
		Log:        discardLogger(),
	})

	info, err := svc.Create(context.Background())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(context.Background(), info.Name); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 backups after delete, got %d", len(list))
	}
}

func TestRetention(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sonarr2.db")
	if err := os.WriteFile(dbPath, []byte("fake-db-contents"), 0o644); err != nil {
		t.Fatalf("write fake db: %v", err)
	}

	backupDir := filepath.Join(dir, "backups")
	svc := New(Options{
		BackupDir:  backupDir,
		DBPath:     dbPath,
		DBDialect:  "sqlite",
		AppVersion: "0.1.0",
		Retention:  2,
		Log:        discardLogger(),
	})

	for i := 0; i < 3; i++ {
		if _, err := svc.Create(context.Background()); err != nil {
			t.Fatalf("Create %d: %v", i+1, err)
		}
		// Ensure distinct timestamps in filenames (second-resolution).
		if i < 2 {
			time.Sleep(1100 * time.Millisecond)
		}
	}

	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 backups after retention enforcement, got %d", len(list))
	}
}
