// Package customscript implements a notification.Notification that executes
// an external script, passing event data as environment variables.
package customscript

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// CustomScript executes a user-configured script for each notification event.
type CustomScript struct {
	settings Settings
}

// New constructs a CustomScript notification provider.
func New(settings Settings) *CustomScript {
	return &CustomScript{settings: settings}
}

// Implementation satisfies providers.Provider.
func (c *CustomScript) Implementation() string { return "CustomScript" }

// DefaultName satisfies providers.Provider.
func (c *CustomScript) DefaultName() string { return "Custom Script" }

// Settings satisfies providers.Provider.
func (c *CustomScript) Settings() any { return &c.settings }

// Test verifies the script path is configured and the file is executable.
func (c *CustomScript) Test(_ context.Context) error {
	if c.settings.Path == "" {
		return fmt.Errorf("customscript: Path is not configured")
	}
	info, err := os.Stat(c.settings.Path)
	if err != nil {
		return fmt.Errorf("customscript: stat %q: %w", c.settings.Path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("customscript: path %q is a directory, not a file", c.settings.Path)
	}
	return nil
}

// OnGrab runs the configured script with grab event environment variables.
func (c *CustomScript) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	env := []string{
		"sonarr_eventtype=Grab",
		"sonarr_series_title=" + msg.SeriesTitle,
		"sonarr_episode_title=" + msg.EpisodeTitle,
		"sonarr_quality=" + msg.Quality,
		"sonarr_indexer=" + msg.Indexer,
	}
	return c.run(ctx, env)
}

// OnDownload runs the configured script with download event environment variables.
func (c *CustomScript) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	env := []string{
		"sonarr_eventtype=Download",
		"sonarr_series_title=" + msg.SeriesTitle,
		"sonarr_episode_title=" + msg.EpisodeTitle,
		"sonarr_quality=" + msg.Quality,
	}
	return c.run(ctx, env)
}

// OnHealthIssue runs the configured script with health issue event environment variables.
func (c *CustomScript) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	env := []string{
		"sonarr_eventtype=HealthIssue",
		"sonarr_health_issue_type=" + msg.Type,
		"sonarr_health_issue_message=" + msg.Message,
	}
	return c.run(ctx, env)
}

// run executes the configured script with the provided environment variables appended.
func (c *CustomScript) run(ctx context.Context, env []string) error {
	if c.settings.Path == "" {
		return fmt.Errorf("customscript: Path is not configured")
	}

	var args []string
	if c.settings.Arguments != "" {
		args = strings.Fields(c.settings.Arguments)
	}

	cmd := exec.CommandContext(ctx, c.settings.Path, args...)
	cmd.Env = append(cmd.Environ(), env...)

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("customscript: script failed: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
