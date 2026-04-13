// Package migrate implements one-shot import from an existing Sonarr v3/v4
// SQLite database into sonarr2.
package migrate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const sonarrTimeLayout = "2006-01-02 15:04:05"

// SourceSeries represents a row from Sonarr's Series table.
type SourceSeries struct {
	ID         int64
	TvdbID     int64
	Title      string
	CleanTitle string
	Status     string
	Path       string
	Monitored  bool
	SeriesType string
	Added      time.Time
}

// SourceSeason represents a row from Sonarr's Seasons table.
type SourceSeason struct {
	ID           int64
	SeriesID     int64
	SeasonNumber int
	Monitored    bool
}

// SourceEpisode represents a row from Sonarr's Episodes table.
type SourceEpisode struct {
	ID                    int64
	SeriesID              int64
	SeasonNumber          int
	EpisodeNumber         int
	AbsoluteEpisodeNumber *int
	Title                 string
	Overview              string
	AirDateUtc            *time.Time
	Monitored             bool
	EpisodeFileID         *int64
}

// SourceEpisodeFile represents a row from Sonarr's EpisodeFiles table.
type SourceEpisodeFile struct {
	ID           int64
	SeriesID     int64
	SeasonNumber int
	RelativePath string
	Size         int64
	DateAdded    time.Time
	ReleaseGroup string
	Quality      string // JSON blob
}

// SourceQualityProfile represents a row from Sonarr's QualityProfiles table.
type SourceQualityProfile struct {
	ID             int64
	Name           string
	UpgradeAllowed bool
	Cutoff         int
	Items          string // JSON
}

// SourceIndexer represents a row from Sonarr's Indexers table.
type SourceIndexer struct {
	ID                      int64
	Name                    string
	Implementation          string
	Settings                string // JSON
	EnableRss               bool
	EnableAutomaticSearch   bool
	EnableInteractiveSearch bool
	Priority                int
}

// SourceDownloadClient represents a row from Sonarr's DownloadClients table.
type SourceDownloadClient struct {
	ID             int64
	Name           string
	Implementation string
	Settings       string // JSON
	Enable         bool
	Priority       int
}

// SourceNotification represents a row from Sonarr's Notifications table.
type SourceNotification struct {
	ID             int64
	Name           string
	Implementation string
	Settings       string // JSON
	OnGrab         bool
	OnDownload     bool
	OnHealthIssue  bool
}

// SourceHistory represents a row from Sonarr's History table.
type SourceHistory struct {
	ID          int64
	EpisodeID   int64
	SeriesID    int64
	SourceTitle string
	Quality     string // JSON
	Date        time.Time
	EventType   string
	DownloadID  string
	Data        string // JSON
}

// parseSonarrTime parses a Sonarr SQLite datetime string. Returns zero time on
// empty input or parse errors.
func parseSonarrTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(sonarrTimeLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// extractQualityName extracts the quality name from a Sonarr quality JSON blob.
// Returns "Unknown" if the JSON is invalid or the name is absent.
func extractQualityName(qualityJSON string) string {
	var q struct {
		Quality struct {
			Name string `json:"name"`
		} `json:"quality"`
	}
	if json.Unmarshal([]byte(qualityJSON), &q) == nil && q.Quality.Name != "" {
		return q.Quality.Name
	}
	return "Unknown"
}

// readSeries reads all rows from the Sonarr Series table.
func readSeries(ctx context.Context, db *sql.DB) ([]SourceSeries, error) {
	const q = `SELECT Id, TvdbId, Title, CleanTitle, Status, Path, Monitored, SeriesType, Added FROM Series`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query Series: %w", err)
	}
	defer rows.Close()

	var out []SourceSeries
	for rows.Next() {
		var s SourceSeries
		var monitored int64
		var added string
		if err := rows.Scan(&s.ID, &s.TvdbID, &s.Title, &s.CleanTitle, &s.Status,
			&s.Path, &monitored, &s.SeriesType, &added); err != nil {
			return nil, fmt.Errorf("migrate: scan Series: %w", err)
		}
		s.Monitored = monitored != 0
		s.Added = parseSonarrTime(added)
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows Series: %w", err)
	}
	return out, nil
}

// readSeasons reads all rows from the Sonarr Seasons table.
func readSeasons(ctx context.Context, db *sql.DB) ([]SourceSeason, error) {
	const q = `SELECT Id, SeriesId, SeasonNumber, Monitored FROM Seasons`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query Seasons: %w", err)
	}
	defer rows.Close()

	var out []SourceSeason
	for rows.Next() {
		var s SourceSeason
		var monitored int64
		if err := rows.Scan(&s.ID, &s.SeriesID, &s.SeasonNumber, &monitored); err != nil {
			return nil, fmt.Errorf("migrate: scan Seasons: %w", err)
		}
		s.Monitored = monitored != 0
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows Seasons: %w", err)
	}
	return out, nil
}

// readEpisodes reads all rows from the Sonarr Episodes table.
func readEpisodes(ctx context.Context, db *sql.DB) ([]SourceEpisode, error) {
	const q = `SELECT Id, SeriesId, SeasonNumber, EpisodeNumber, AbsoluteEpisodeNumber,
		Title, Overview, AirDateUtc, Monitored, EpisodeFileId FROM Episodes`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query Episodes: %w", err)
	}
	defer rows.Close()

	var out []SourceEpisode
	for rows.Next() {
		var e SourceEpisode
		var monitored int64
		var absEpNum int64
		var episodeFileID int64
		var airDateUtc sql.NullString
		if err := rows.Scan(&e.ID, &e.SeriesID, &e.SeasonNumber, &e.EpisodeNumber,
			&absEpNum, &e.Title, &e.Overview, &airDateUtc, &monitored, &episodeFileID); err != nil {
			return nil, fmt.Errorf("migrate: scan Episodes: %w", err)
		}
		e.Monitored = monitored != 0
		if absEpNum != 0 {
			v := int(absEpNum)
			e.AbsoluteEpisodeNumber = &v
		}
		if episodeFileID != 0 {
			e.EpisodeFileID = &episodeFileID
		}
		if airDateUtc.Valid && airDateUtc.String != "" {
			t := parseSonarrTime(airDateUtc.String)
			if !t.IsZero() {
				e.AirDateUtc = &t
			}
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows Episodes: %w", err)
	}
	return out, nil
}

// readEpisodeFiles reads all rows from the Sonarr EpisodeFiles table.
func readEpisodeFiles(ctx context.Context, db *sql.DB) ([]SourceEpisodeFile, error) {
	const q = `SELECT Id, SeriesId, SeasonNumber, RelativePath, Size, DateAdded, ReleaseGroup, Quality FROM EpisodeFiles`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query EpisodeFiles: %w", err)
	}
	defer rows.Close()

	var out []SourceEpisodeFile
	for rows.Next() {
		var f SourceEpisodeFile
		var dateAdded string
		if err := rows.Scan(&f.ID, &f.SeriesID, &f.SeasonNumber, &f.RelativePath,
			&f.Size, &dateAdded, &f.ReleaseGroup, &f.Quality); err != nil {
			return nil, fmt.Errorf("migrate: scan EpisodeFiles: %w", err)
		}
		f.DateAdded = parseSonarrTime(dateAdded)
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows EpisodeFiles: %w", err)
	}
	return out, nil
}

// readQualityProfiles reads all rows from the Sonarr QualityProfiles table.
func readQualityProfiles(ctx context.Context, db *sql.DB) ([]SourceQualityProfile, error) {
	const q = `SELECT Id, Name, UpgradeAllowed, Cutoff, Items FROM QualityProfiles`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query QualityProfiles: %w", err)
	}
	defer rows.Close()

	var out []SourceQualityProfile
	for rows.Next() {
		var p SourceQualityProfile
		var upgradeAllowed int64
		if err := rows.Scan(&p.ID, &p.Name, &upgradeAllowed, &p.Cutoff, &p.Items); err != nil {
			return nil, fmt.Errorf("migrate: scan QualityProfiles: %w", err)
		}
		p.UpgradeAllowed = upgradeAllowed != 0
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows QualityProfiles: %w", err)
	}
	return out, nil
}

// readIndexers reads all rows from the Sonarr Indexers table.
func readIndexers(ctx context.Context, db *sql.DB) ([]SourceIndexer, error) {
	const q = `SELECT Id, Name, Implementation, Settings, EnableRss, EnableAutomaticSearch, EnableInteractiveSearch, Priority FROM Indexers`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query Indexers: %w", err)
	}
	defer rows.Close()

	var out []SourceIndexer
	for rows.Next() {
		var idx SourceIndexer
		var enableRss, enableAutoSearch, enableInteractiveSearch int64
		if err := rows.Scan(&idx.ID, &idx.Name, &idx.Implementation, &idx.Settings,
			&enableRss, &enableAutoSearch, &enableInteractiveSearch, &idx.Priority); err != nil {
			return nil, fmt.Errorf("migrate: scan Indexers: %w", err)
		}
		idx.EnableRss = enableRss != 0
		idx.EnableAutomaticSearch = enableAutoSearch != 0
		idx.EnableInteractiveSearch = enableInteractiveSearch != 0
		out = append(out, idx)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows Indexers: %w", err)
	}
	return out, nil
}

// readDownloadClients reads all rows from the Sonarr DownloadClients table.
func readDownloadClients(ctx context.Context, db *sql.DB) ([]SourceDownloadClient, error) {
	const q = `SELECT Id, Name, Implementation, Settings, Enable, Priority FROM DownloadClients`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query DownloadClients: %w", err)
	}
	defer rows.Close()

	var out []SourceDownloadClient
	for rows.Next() {
		var dc SourceDownloadClient
		var enable int64
		if err := rows.Scan(&dc.ID, &dc.Name, &dc.Implementation, &dc.Settings,
			&enable, &dc.Priority); err != nil {
			return nil, fmt.Errorf("migrate: scan DownloadClients: %w", err)
		}
		dc.Enable = enable != 0
		out = append(out, dc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows DownloadClients: %w", err)
	}
	return out, nil
}

// readNotifications reads all rows from the Sonarr Notifications table.
func readNotifications(ctx context.Context, db *sql.DB) ([]SourceNotification, error) {
	const q = `SELECT Id, Name, Implementation, Settings, OnGrab, OnDownload, OnHealthIssue FROM Notifications`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query Notifications: %w", err)
	}
	defer rows.Close()

	var out []SourceNotification
	for rows.Next() {
		var n SourceNotification
		var onGrab, onDownload, onHealthIssue int64
		if err := rows.Scan(&n.ID, &n.Name, &n.Implementation, &n.Settings,
			&onGrab, &onDownload, &onHealthIssue); err != nil {
			return nil, fmt.Errorf("migrate: scan Notifications: %w", err)
		}
		n.OnGrab = onGrab != 0
		n.OnDownload = onDownload != 0
		n.OnHealthIssue = onHealthIssue != 0
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows Notifications: %w", err)
	}
	return out, nil
}

// readHistory reads all rows from the Sonarr History table.
func readHistory(ctx context.Context, db *sql.DB) ([]SourceHistory, error) {
	const q = `SELECT Id, EpisodeId, SeriesId, SourceTitle, Quality, Date, EventType, DownloadId, Data FROM History`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("migrate: query History: %w", err)
	}
	defer rows.Close()

	var out []SourceHistory
	for rows.Next() {
		var h SourceHistory
		var date string
		if err := rows.Scan(&h.ID, &h.EpisodeID, &h.SeriesID, &h.SourceTitle,
			&h.Quality, &date, &h.EventType, &h.DownloadID, &h.Data); err != nil {
			return nil, fmt.Errorf("migrate: scan History: %w", err)
		}
		h.Date = parseSonarrTime(date)
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrate: rows History: %w", err)
	}
	return out, nil
}

// readAPIKey reads the API key from Sonarr's Config table.
func readAPIKey(ctx context.Context, db *sql.DB) (string, error) {
	const q = `SELECT Value FROM Config WHERE Key = 'ApiKey'`
	var apiKey string
	if err := db.QueryRowContext(ctx, q).Scan(&apiKey); err != nil {
		return "", fmt.Errorf("migrate: read ApiKey: %w", err)
	}
	return apiKey, nil
}
