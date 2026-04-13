package health

import (
	"context"
	"testing"
)

// fakeInstanceCounter is defined in indexer_test.go and shared here.

func TestDownloadClientCheckHasClient(t *testing.T) {
	check := NewDownloadClientCheck(&fakeInstanceCounter{count: 2})
	results := check.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("expected 0 results when clients configured, got %d", len(results))
	}
}

func TestDownloadClientCheckNoClient(t *testing.T) {
	check := NewDownloadClientCheck(&fakeInstanceCounter{count: 0})
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result when no clients, got %d", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("expected warning, got %q", results[0].Type)
	}
	if results[0].Source != "DownloadClientCheck" {
		t.Errorf("expected source DownloadClientCheck, got %q", results[0].Source)
	}
}
