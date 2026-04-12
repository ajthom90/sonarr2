package tvdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// tokenTTL is how long a TVDB JWT token is considered valid.
// TVDB tokens expire after 30 days; we refresh 1 hour early as a safety margin.
const tokenTTL = 30*24*time.Hour - time.Hour

// tokenCache caches the TVDB JWT token and refreshes it when expired.
type tokenCache struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
}

// get returns a valid JWT token, logging in if the cached token is absent or
// expired.
func (c *tokenCache) get(ctx context.Context, client *http.Client, baseURL, apiKey string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expiry) {
		return c.token, nil
	}

	token, err := c.login(ctx, client, baseURL, apiKey)
	if err != nil {
		return "", err
	}

	c.token = token
	c.expiry = time.Now().Add(tokenTTL)
	return c.token, nil
}

// invalidate forces the next get() to re-login regardless of expiry.
func (c *tokenCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.expiry = time.Time{}
}

// login performs POST /v4/login and returns the JWT token.
func (c *tokenCache) login(ctx context.Context, client *http.Client, baseURL, apiKey string) (string, error) {
	body, err := json.Marshal(tvdbLoginRequest{APIKey: apiKey})
	if err != nil {
		return "", fmt.Errorf("tvdb: marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v4/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("tvdb: build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tvdb: login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tvdb: login returned status %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("tvdb: read login response: %w", err)
	}

	var result tvdbLoginResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("tvdb: decode login response: %w", err)
	}

	if result.Data.Token == "" {
		return "", fmt.Errorf("tvdb: login response contained empty token")
	}

	return result.Data.Token, nil
}
