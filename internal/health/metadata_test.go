package health

import (
	"context"
	"testing"
)

func TestMetadataSourceCheckConfigured(t *testing.T) {
	check := NewMetadataSourceCheck("my-api-key")
	results := check.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("expected 0 results when API key configured, got %d", len(results))
	}
}

func TestMetadataSourceCheckUnconfigured(t *testing.T) {
	check := NewMetadataSourceCheck("")
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result when no API key, got %d", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("expected warning, got %q", results[0].Type)
	}
	if results[0].Source != "MetadataSourceCheck" {
		t.Errorf("expected source MetadataSourceCheck, got %q", results[0].Source)
	}
}
