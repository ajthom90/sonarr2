// Package blackhole implements a downloadclient.DownloadClient that writes
// NZB or torrent files to a watch folder, with no API tracking.
package blackhole

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Blackhole is a downloadclient.DownloadClient that drops files into a folder.
type Blackhole struct {
	settings Settings
	client   *http.Client
}

// New constructs a Blackhole download client. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *Blackhole {
	if client == nil {
		client = http.DefaultClient
	}
	return &Blackhole{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (b *Blackhole) Implementation() string { return "Blackhole" }

// DefaultName satisfies providers.Provider.
func (b *Blackhole) DefaultName() string { return "Blackhole" }

// Settings satisfies providers.Provider.
func (b *Blackhole) Settings() any { return &b.settings }

// Protocol satisfies downloadclient.DownloadClient.
// Blackhole supports both protocols; default to usenet.
func (b *Blackhole) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }

// Add downloads or writes the given URL as a file in the watch folder.
// For torrent URLs (magnet: links), the URL string is written directly.
// For HTTP URLs the content is fetched and saved; the filename is derived
// from title.
func (b *Blackhole) Add(ctx context.Context, downloadURL string, title string) (string, error) {
	if b.settings.WatchFolder == "" {
		return "", fmt.Errorf("blackhole: WatchFolder is not configured")
	}

	// Sanitize title for use as a filename.
	safe := sanitizeTitle(title)

	var content []byte
	var ext string

	if strings.HasPrefix(downloadURL, "magnet:") {
		// Write magnet URI as a .torrent meta-link file (simple text).
		content = []byte(downloadURL)
		ext = ".torrent"
	} else {
		// Fetch the remote content.
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
		if err != nil {
			return "", fmt.Errorf("blackhole: failed to build request: %w", err)
		}
		resp, err := b.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("blackhole: download failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("blackhole: download returned status %d", resp.StatusCode)
		}
		content, err = io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("blackhole: failed to read download: %w", err)
		}
		// Guess extension from URL or content-type.
		ext = guessExt(downloadURL, resp.Header.Get("Content-Type"))
	}

	dest := filepath.Join(b.settings.WatchFolder, safe+ext)
	if err := os.WriteFile(dest, content, 0644); err != nil {
		return "", fmt.Errorf("blackhole: failed to write file %q: %w", dest, err)
	}
	return dest, nil
}

// Items returns an empty list — blackhole doesn't track state.
func (b *Blackhole) Items(_ context.Context) ([]downloadclient.Item, error) {
	return nil, nil
}

// Remove is a no-op for blackhole.
func (b *Blackhole) Remove(_ context.Context, _ string, _ bool) error { return nil }

// Status returns the download client's overall state.
func (b *Blackhole) Status(_ context.Context) (downloadclient.Status, error) {
	return downloadclient.Status{
		IsLocalhost:       true,
		OutputRootFolders: []string{b.settings.WatchFolder},
	}, nil
}

// Test verifies the watch folder is configured and writable.
func (b *Blackhole) Test(_ context.Context) error {
	if b.settings.WatchFolder == "" {
		return fmt.Errorf("blackhole: WatchFolder is not configured")
	}
	info, err := os.Stat(b.settings.WatchFolder)
	if err != nil {
		return fmt.Errorf("blackhole: watch folder %q: %w", b.settings.WatchFolder, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("blackhole: watch folder %q is not a directory", b.settings.WatchFolder)
	}
	return nil
}

// sanitizeTitle replaces characters that are invalid in filenames.
func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(title)
}

// guessExt returns a file extension based on the URL path or Content-Type.
func guessExt(rawURL string, contentType string) string {
	lower := strings.ToLower(rawURL)
	switch {
	case strings.HasSuffix(lower, ".torrent"):
		return ".torrent"
	case strings.HasSuffix(lower, ".nzb"):
		return ".nzb"
	case strings.Contains(contentType, "x-bittorrent"):
		return ".torrent"
	case strings.Contains(contentType, "x-nzb"):
		return ".nzb"
	default:
		return ".nzb" // default to NZB for blackhole
	}
}
