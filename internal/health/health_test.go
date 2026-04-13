package health

import (
	"context"
	"testing"
)

// fakeCheck implements Check for testing.
type fakeCheck struct {
	name    string
	results []Result
}

func (f *fakeCheck) Name() string { return f.name }
func (f *fakeCheck) Check(_ context.Context) []Result {
	return f.results
}

func TestCheckerRunAll(t *testing.T) {
	ctx := context.Background()

	checks := []Check{
		&fakeCheck{name: "ok", results: nil}, // returns nil → no results
		&fakeCheck{name: "warn", results: []Result{
			{Source: "warn", Type: LevelWarning, Message: "something"},
		}},
		&fakeCheck{name: "errs", results: []Result{
			{Source: "errs", Type: LevelError, Message: "bad thing 1"},
			{Source: "errs", Type: LevelError, Message: "bad thing 2"},
		}},
	}

	c := NewChecker(checks...)
	got := c.RunAll(ctx)

	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}

	// Results() should return the cached copy.
	cached := c.Results()
	if len(cached) != 3 {
		t.Fatalf("cached results: expected 3, got %d", len(cached))
	}
}

func TestCheckerEmpty(t *testing.T) {
	c := NewChecker()

	got := c.RunAll(context.Background())
	if len(got) != 0 {
		t.Fatalf("expected 0 results from empty checker, got %d", len(got))
	}

	// Results() before RunAll should also return empty slice (not nil).
	fresh := NewChecker()
	results := fresh.Results()
	if results == nil {
		t.Fatal("Results() should return empty slice, not nil")
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results before RunAll, got %d", len(results))
	}
}
