// Package notification — HTTP helpers shared across webhook-style providers.
// Keeps the per-provider code focused on payload shape.
package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultHTTPClient is used when a provider's constructor receives nil. Kept
// small and in-package so every provider shares the same default transport.
var DefaultHTTPClient = http.DefaultClient

// PostJSON marshals body, POSTs it to url with Content-Type application/json,
// and returns an error when the response status is not 2xx.
func PostJSON(ctx context.Context, client *http.Client, url string, body any) error {
	return PostJSONWithHeaders(ctx, client, url, body, nil)
}

// PostJSONWithHeaders is PostJSON with extra headers (e.g. API keys).
func PostJSONWithHeaders(ctx context.Context, client *http.Client, url string, body any, headers map[string]string) error {
	if url == "" {
		return fmt.Errorf("post: URL is empty")
	}
	if client == nil {
		client = DefaultHTTPClient
	}
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("post: marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("post: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return do(client, req)
}

// PostForm POSTs form-encoded body to url, returning an error on non-2xx.
func PostForm(ctx context.Context, client *http.Client, url, body string, headers map[string]string) error {
	if url == "" {
		return fmt.Errorf("post-form: URL is empty")
	}
	if client == nil {
		client = DefaultHTTPClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("post-form: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return do(client, req)
}

// Get performs an HTTP GET with custom headers and returns an error on non-2xx.
func Get(ctx context.Context, client *http.Client, url string, headers map[string]string) error {
	if url == "" {
		return fmt.Errorf("get: URL is empty")
	}
	if client == nil {
		client = DefaultHTTPClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("get: build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return do(client, req)
}

// Put performs an HTTP PUT with a plain text body (used by ntfy).
func Put(ctx context.Context, client *http.Client, url, contentType, body string, headers map[string]string) error {
	if url == "" {
		return fmt.Errorf("put: URL is empty")
	}
	if client == nil {
		client = DefaultHTTPClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("put: build request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return do(client, req)
}

func do(client *http.Client, req *http.Request) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, req.URL.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	// Read up to 512B of response body for error context.
	buf, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("%s %s: status %d: %s",
		req.Method, req.URL.Host, resp.StatusCode, strings.TrimSpace(string(buf)))
}
