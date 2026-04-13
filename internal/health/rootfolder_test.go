package health

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeSeriesLister struct {
	paths []string
	err   error
}

func (f *fakeSeriesLister) ListRootPaths(_ context.Context) ([]string, error) {
	return f.paths, f.err
}

func TestRootFolderCheckAllExist(t *testing.T) {
	dir := t.TempDir()
	// Create the series subdirectory so filepath.Dir(path) exists on disk.
	seriesDir := filepath.Join(dir, "series")
	if err := os.MkdirAll(seriesDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// The series path; filepath.Dir(path) == seriesDir which now exists.
	path := filepath.Join(seriesDir, "Show.Name")
	lister := &fakeSeriesLister{paths: []string{path}}
	check := NewRootFolderCheck(lister)
	results := check.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("expected 0 results for existing root, got %d: %v", len(results), results)
	}
}

func TestRootFolderCheckMissing(t *testing.T) {
	path := "/nonexistent/path/xyz/Show.Name"
	lister := &fakeSeriesLister{paths: []string{path}}
	check := NewRootFolderCheck(lister)
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result for missing root, got %d", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("expected warning, got %q", results[0].Type)
	}
	if results[0].Source != "RootFolderCheck" {
		t.Errorf("expected source RootFolderCheck, got %q", results[0].Source)
	}
}

func TestRootFolderCheckListError(t *testing.T) {
	lister := &fakeSeriesLister{err: errors.New("db error")}
	check := NewRootFolderCheck(lister)
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 error result for list failure, got %d", len(results))
	}
	if results[0].Type != LevelError {
		t.Errorf("expected error level, got %q", results[0].Type)
	}
}
