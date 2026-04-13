package health

import (
	"context"
	"errors"
	"testing"
)

type fakeInstanceCounter struct {
	count int
	err   error
}

func (f *fakeInstanceCounter) CountEnabled(_ context.Context) (int, error) {
	return f.count, f.err
}

func TestIndexerCheckHasIndexer(t *testing.T) {
	check := NewIndexerCheck(&fakeInstanceCounter{count: 1})
	results := check.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("expected 0 results when indexers configured, got %d", len(results))
	}
}

func TestIndexerCheckNoIndexer(t *testing.T) {
	check := NewIndexerCheck(&fakeInstanceCounter{count: 0})
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result when no indexers, got %d", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("expected warning, got %q", results[0].Type)
	}
	if results[0].Source != "IndexerCheck" {
		t.Errorf("expected source IndexerCheck, got %q", results[0].Source)
	}
}

func TestIndexerCheckCountError(t *testing.T) {
	check := NewIndexerCheck(&fakeInstanceCounter{err: errors.New("db error")})
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 error result for count failure, got %d", len(results))
	}
	if results[0].Type != LevelError {
		t.Errorf("expected error level, got %q", results[0].Type)
	}
}
