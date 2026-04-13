package health

import (
	"context"
	"errors"
	"testing"
)

type fakePinger struct {
	err error
}

func (f *fakePinger) Ping(_ context.Context) error { return f.err }

func TestDatabaseCheckHealthy(t *testing.T) {
	check := NewDatabaseCheck(&fakePinger{err: nil})
	results := check.Check(context.Background())
	if len(results) != 0 {
		t.Fatalf("expected 0 results for healthy db, got %d", len(results))
	}
}

func TestDatabaseCheckUnhealthy(t *testing.T) {
	check := NewDatabaseCheck(&fakePinger{err: errors.New("connection refused")})
	results := check.Check(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 result for unhealthy db, got %d", len(results))
	}
	if results[0].Source != "DatabaseCheck" {
		t.Errorf("expected source DatabaseCheck, got %q", results[0].Source)
	}
	if results[0].Type != LevelError {
		t.Errorf("expected level error, got %q", results[0].Type)
	}
}
