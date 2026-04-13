package health

import (
	"context"
	"testing"
)

func TestAuthCheckFormsMode(t *testing.T) {
	c := NewAuthCheck("forms")
	results := c.Check(context.Background())
	if len(results) != 0 {
		t.Errorf("forms mode: got %d results, want 0", len(results))
	}
}

func TestAuthCheckNoneMode(t *testing.T) {
	c := NewAuthCheck("none")
	results := c.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("none mode: got %d results, want 1", len(results))
	}
	if results[0].Type != LevelWarning {
		t.Errorf("Type = %q, want warning", results[0].Type)
	}
}
