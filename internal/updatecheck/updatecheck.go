// Package updatecheck queries GitHub Releases to determine if a newer
// version of sonarr2 is available.
package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Result describes the version comparison outcome.
type Result struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	LatestVersion   string `json:"latestVersion"`
	CurrentVersion  string `json:"currentVersion"`
}

// Checker compares the running version against the latest GitHub Release.
type Checker struct {
	currentVersion string
	releasesURL    string // https://api.github.com/repos/{owner}/{repo}/releases/latest
	httpClient     *http.Client
	mu             sync.Mutex
	cached         *Result
	cachedAt       time.Time
	cacheTTL       time.Duration
}

// New creates a Checker. Pass nil for httpClient to use a default with 10s timeout.
func New(currentVersion, repoOwner, repoName string, httpClient *http.Client) *Checker {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Checker{
		currentVersion: currentVersion,
		releasesURL:    fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName),
		httpClient:     httpClient,
		cacheTTL:       24 * time.Hour,
	}
}

// WithBaseURL overrides the GitHub API URL (for tests).
func (c *Checker) WithBaseURL(url string) *Checker {
	c.releasesURL = url
	return c
}

// Check returns whether an update is available. Results are cached for 24 hours.
func (c *Checker) Check(ctx context.Context) (*Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil && time.Since(c.cachedAt) < c.cacheTTL {
		return c.cached, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.releasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Don't fail hard — just report no update available.
		result := &Result{
			CurrentVersion: c.currentVersion,
			LatestVersion:  c.currentVersion,
		}
		c.cached = result
		c.cachedAt = time.Now()
		return result, nil
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("updatecheck: decode response: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	result := &Result{
		CurrentVersion:  c.currentVersion,
		LatestVersion:   latest,
		UpdateAvailable: latest != c.currentVersion && c.currentVersion != "dev",
	}
	c.cached = result
	c.cachedAt = time.Now()
	return result, nil
}
