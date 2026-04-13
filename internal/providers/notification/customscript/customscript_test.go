package customscript

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// TestCustomScriptOnGrab creates a temp script that writes env vars to a file,
// runs OnGrab, and verifies the file was written with the correct content.
func TestCustomScriptOnGrab(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "output.txt")
	scriptFile := filepath.Join(tmpDir, "notify.sh")

	// Write a script that echoes the event type env var to a file.
	scriptContent := "#!/bin/sh\necho \"$sonarr_eventtype:$sonarr_series_title\" > " + outFile + "\n"
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cs := New(Settings{Path: scriptFile})
	err := cs.OnGrab(context.Background(), notification.GrabMessage{
		SeriesTitle:  "Breaking Bad",
		EpisodeTitle: "Pilot",
		Quality:      "HDTV-720p",
		Indexer:      "TestIndexer",
	})
	if err != nil {
		t.Fatalf("OnGrab returned error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	got := string(data)
	if got == "" {
		t.Error("output file is empty — script did not run")
	}
	// The script writes "Grab:Breaking Bad"
	if len(got) < 4 {
		t.Errorf("output too short: %q", got)
	}
}

// TestCustomScriptOnDownload runs a no-op echo script via OnDownload.
func TestCustomScriptOnDownload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptFile := filepath.Join(tmpDir, "noop.sh")
	if err := os.WriteFile(scriptFile, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cs := New(Settings{Path: scriptFile})
	if err := cs.OnDownload(context.Background(), notification.DownloadMessage{
		SeriesTitle:  "Breaking Bad",
		EpisodeTitle: "Pilot",
		Quality:      "Bluray-1080p",
	}); err != nil {
		t.Fatalf("OnDownload returned error: %v", err)
	}
}

// TestCustomScriptOnHealthIssue runs a no-op echo script via OnHealthIssue.
func TestCustomScriptOnHealthIssue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptFile := filepath.Join(tmpDir, "noop.sh")
	if err := os.WriteFile(scriptFile, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cs := New(Settings{Path: scriptFile})
	if err := cs.OnHealthIssue(context.Background(), notification.HealthMessage{
		Type:    "IndexerFailure",
		Message: "Indexer is offline",
	}); err != nil {
		t.Fatalf("OnHealthIssue returned error: %v", err)
	}
}

// TestCustomScriptNoPath verifies that an empty path returns an error.
func TestCustomScriptNoPath(t *testing.T) {
	cs := New(Settings{})
	err := cs.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

// TestCustomScriptScriptFails verifies that a non-zero exit code returns an error.
func TestCustomScriptScriptFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping shell script test on Windows")
	}

	tmpDir := t.TempDir()
	scriptFile := filepath.Join(tmpDir, "fail.sh")
	if err := os.WriteFile(scriptFile, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	cs := New(Settings{Path: scriptFile})
	err := cs.OnGrab(context.Background(), notification.GrabMessage{SeriesTitle: "Test"})
	if err == nil {
		t.Fatal("expected error for exit code 1")
	}
}
