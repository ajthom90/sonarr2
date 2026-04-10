package buildinfo

import "testing"

func TestGetReturnsNonEmptyDefaults(t *testing.T) {
	info := Get()
	if info.Version == "" {
		t.Error("Version must not be empty")
	}
	if info.Commit == "" {
		t.Error("Commit must not be empty")
	}
	if info.Date == "" {
		t.Error("Date must not be empty")
	}
}

func TestGetReflectsVariables(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	})

	Version = "1.2.3"
	Commit = "abcdef"
	Date = "2026-04-10T00:00:00Z"

	info := Get()
	if info.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", info.Version)
	}
	if info.Commit != "abcdef" {
		t.Errorf("Commit = %q, want abcdef", info.Commit)
	}
	if info.Date != "2026-04-10T00:00:00Z" {
		t.Errorf("Date = %q, want 2026-04-10T00:00:00Z", info.Date)
	}
}
