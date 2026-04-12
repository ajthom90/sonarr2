package tvdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

const defaultBaseURL = "https://api4.thetvdb.com"

// Client implements metadatasource.MetadataSource against the TVDB v4 API.
type Client struct {
	settings Settings
	http     *http.Client
	baseURL  string
	auth     tokenCache
}

// New constructs a TVDB Client. Pass nil for httpClient to use http.DefaultClient.
// If settings.BaseURL is empty the production TVDB endpoint is used.
func New(settings Settings, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	base := settings.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		settings: settings,
		http:     httpClient,
		baseURL:  base,
	}
}

// WithBaseURL overrides the base URL and returns the receiver for chaining.
// Intended for tests that point at an httptest server.
func (c *Client) WithBaseURL(u string) *Client {
	c.baseURL = u
	return c
}

// SearchSeries implements metadatasource.MetadataSource.
// GET /v4/search?query={query}&type=series
func (c *Client) SearchSeries(ctx context.Context, query string) ([]metadatasource.SeriesSearchResult, error) {
	u, err := url.Parse(c.baseURL + "/v4/search")
	if err != nil {
		return nil, fmt.Errorf("tvdb: parse search URL: %w", err)
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("type", "series")
	u.RawQuery = q.Encode()

	body, err := c.get(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var resp tvdbSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("tvdb: decode search response: %w", err)
	}

	results := make([]metadatasource.SeriesSearchResult, 0, len(resp.Data))
	for _, r := range resp.Data {
		id, err := strconv.ParseInt(r.TvdbID, 10, 64)
		if err != nil {
			// Skip entries with malformed IDs rather than failing the whole call.
			continue
		}
		year, _ := strconv.Atoi(r.Year)
		results = append(results, metadatasource.SeriesSearchResult{
			TvdbID:   id,
			Title:    r.Name,
			Year:     year,
			Overview: r.Overview,
			Status:   r.Status.Name,
			Network:  r.Network,
			Slug:     r.Slug,
		})
	}
	return results, nil
}

// GetSeries implements metadatasource.MetadataSource.
// GET /v4/series/{id}
func (c *Client) GetSeries(ctx context.Context, tvdbID int64) (metadatasource.SeriesInfo, error) {
	rawURL := fmt.Sprintf("%s/v4/series/%d", c.baseURL, tvdbID)

	body, err := c.get(ctx, rawURL)
	if err != nil {
		return metadatasource.SeriesInfo{}, err
	}

	var resp tvdbSeriesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return metadatasource.SeriesInfo{}, fmt.Errorf("tvdb: decode series response: %w", err)
	}

	s := resp.Data
	year, _ := strconv.Atoi(s.Year)

	genres := make([]string, 0, len(s.Genres))
	for _, g := range s.Genres {
		genres = append(genres, g.Name)
	}

	return metadatasource.SeriesInfo{
		TvdbID:   s.ID,
		Title:    s.Name,
		Year:     year,
		Overview: s.Overview,
		Status:   s.Status.Name,
		Network:  s.OriginalNetwork.Name,
		Runtime:  s.AverageRuntime,
		AirTime:  s.AirsTime,
		Slug:     s.Slug,
		Genres:   genres,
	}, nil
}

// GetEpisodes implements metadatasource.MetadataSource.
// GET /v4/series/{id}/episodes/default?page=0 — follows links.next for pagination.
func (c *Client) GetEpisodes(ctx context.Context, tvdbID int64) ([]metadatasource.EpisodeInfo, error) {
	var all []metadatasource.EpisodeInfo
	nextURL := fmt.Sprintf("%s/v4/series/%d/episodes/default?page=0", c.baseURL, tvdbID)

	for nextURL != "" {
		body, err := c.get(ctx, nextURL)
		if err != nil {
			return nil, err
		}

		var resp tvdbEpisodesResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("tvdb: decode episodes response: %w", err)
		}

		for _, ep := range resp.Data.Episodes {
			info := metadatasource.EpisodeInfo{
				TvdbID:        ep.ID,
				SeasonNumber:  ep.SeasonNumber,
				EpisodeNumber: ep.Number,
				Title:         ep.Name,
				Overview:      ep.Overview,
			}

			if ep.AbsoluteNumber != 0 {
				abs := ep.AbsoluteNumber
				info.AbsoluteEpisodeNumber = &abs
			}

			if ep.Aired != "" {
				t, err := time.Parse("2006-01-02", ep.Aired)
				if err == nil {
					info.AirDate = &t
				}
			}

			all = append(all, info)
		}

		// Follow pagination.
		nextURL = ""
		if resp.Links.Next != nil && *resp.Links.Next != "" {
			// links.next may be a relative reference like "page=1" or a full URL.
			// Build the full URL by resolving against the base series episodes URL.
			base := fmt.Sprintf("%s/v4/series/%d/episodes/default", c.baseURL, tvdbID)
			parsed, err := url.Parse(base)
			if err == nil {
				next, err := url.Parse("?" + *resp.Links.Next)
				if err == nil {
					nextURL = parsed.ResolveReference(next).String()
				}
			}
		}
	}

	return all, nil
}

// get performs an authenticated GET request and returns the response body.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	token, err := c.auth.get(ctx, c.http, c.baseURL, c.settings.ApiKey)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tvdb: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tvdb: request %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tvdb: request to %s returned status %d", rawURL, resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tvdb: read response body: %w", err)
	}
	return raw, nil
}
