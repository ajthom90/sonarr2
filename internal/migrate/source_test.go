package migrate

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/library"
	_ "modernc.org/sqlite"
)

func createFixtureDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open fixture db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create Sonarr-compatible tables
	_, err = db.Exec(`
		CREATE TABLE Series (
			Id INTEGER PRIMARY KEY, TvdbId INTEGER, Title TEXT, CleanTitle TEXT,
			Status TEXT, Path TEXT, Monitored INTEGER, SeriesType TEXT, Added TEXT
		);
		CREATE TABLE Seasons (
			Id INTEGER PRIMARY KEY, SeriesId INTEGER, SeasonNumber INTEGER, Monitored INTEGER
		);
		CREATE TABLE Episodes (
			Id INTEGER PRIMARY KEY, SeriesId INTEGER, SeasonNumber INTEGER,
			EpisodeNumber INTEGER, AbsoluteEpisodeNumber INTEGER,
			Title TEXT, Overview TEXT, AirDateUtc TEXT, Monitored INTEGER, EpisodeFileId INTEGER
		);
		CREATE TABLE EpisodeFiles (
			Id INTEGER PRIMARY KEY, SeriesId INTEGER, SeasonNumber INTEGER,
			RelativePath TEXT, Size INTEGER, DateAdded TEXT, ReleaseGroup TEXT, Quality TEXT
		);
		CREATE TABLE QualityProfiles (
			Id INTEGER PRIMARY KEY, Name TEXT, UpgradeAllowed INTEGER, Cutoff INTEGER, Items TEXT
		);
		CREATE TABLE Indexers (
			Id INTEGER PRIMARY KEY, Name TEXT, Implementation TEXT, Settings TEXT,
			EnableRss INTEGER, EnableAutomaticSearch INTEGER, EnableInteractiveSearch INTEGER, Priority INTEGER
		);
		CREATE TABLE DownloadClients (
			Id INTEGER PRIMARY KEY, Name TEXT, Implementation TEXT, Settings TEXT,
			Enable INTEGER, Priority INTEGER
		);
		CREATE TABLE Notifications (
			Id INTEGER PRIMARY KEY, Name TEXT, Implementation TEXT, Settings TEXT,
			OnGrab INTEGER, OnDownload INTEGER, OnHealthIssue INTEGER
		);
		CREATE TABLE Config (Key TEXT PRIMARY KEY, Value TEXT);
		CREATE TABLE History (
			Id INTEGER PRIMARY KEY, EpisodeId INTEGER, SeriesId INTEGER,
			SourceTitle TEXT, Quality TEXT, Date TEXT, EventType TEXT, DownloadId TEXT, Data TEXT
		);

		-- Insert fixture data
		INSERT INTO Config (Key, Value) VALUES ('ApiKey', 'test-api-key-12345');

		INSERT INTO Series VALUES (1, 71663, 'The Simpsons', 'thesimpsons', 'continuing', '/tv/The Simpsons', 1, 'standard', '2023-01-01 00:00:00');
		INSERT INTO Series VALUES (2, 81189, 'Breaking Bad', 'breakingbad', 'ended', '/tv/Breaking Bad', 1, 'standard', '2023-06-15 12:00:00');

		INSERT INTO Seasons VALUES (1, 1, 1, 1);
		INSERT INTO Seasons VALUES (2, 1, 2, 0);
		INSERT INTO Seasons VALUES (3, 2, 1, 1);

		INSERT INTO EpisodeFiles VALUES (1, 1, 1, 'Season 01/The Simpsons - S01E01 - Pilot.mkv', 1500000000, '2023-01-10 00:00:00', 'ABCD', '{"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080}}');

		INSERT INTO Episodes VALUES (1, 1, 1, 1, 0, 'Simpsons Roasting on an Open Fire', 'The first episode.', '1989-12-17 00:00:00', 1, 1);
		INSERT INTO Episodes VALUES (2, 1, 1, 2, 0, 'Bart the Genius', 'Bart cheats.', '1990-01-14 00:00:00', 1, 0);
		INSERT INTO Episodes VALUES (3, 2, 1, 1, 0, 'Pilot', 'Walter starts cooking.', '2008-01-20 00:00:00', 1, 0);

		INSERT INTO QualityProfiles VALUES (1, 'HD-1080p', 1, 7, '[{"quality":{"id":7,"name":"Bluray-1080p"},"allowed":true}]');

		INSERT INTO Indexers VALUES (1, 'NZBgeek', 'Newznab', '{"baseUrl":"https://nzbgeek.info","apiKey":"abc123"}', 1, 1, 0, 25);

		INSERT INTO DownloadClients VALUES (1, 'SABnzbd', 'Sabnzbd', '{"host":"localhost","port":8080,"apiKey":"sab123"}', 1, 1);

		INSERT INTO Notifications VALUES (1, 'Discord', 'Discord', '{"webhookUrl":"https://discord.com/api/webhooks/123"}', 1, 1, 0);

		INSERT INTO History VALUES (1, 1, 1, 'The.Simpsons.S01E01.1080p.BluRay', '{"quality":{"id":7,"name":"Bluray-1080p"}}', '2023-01-10 12:00:00', 'grabbed', 'dl-123', '{}');
	`)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	return db
}

func TestReadSeries(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	series, err := readSeries(ctx, db)
	if err != nil {
		t.Fatalf("readSeries: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(series))
	}

	s := series[0]
	if s.ID != 1 {
		t.Errorf("series[0].ID = %d, want 1", s.ID)
	}
	if s.TvdbID != 71663 {
		t.Errorf("series[0].TvdbID = %d, want 71663", s.TvdbID)
	}
	if s.Title != "The Simpsons" {
		t.Errorf("series[0].Title = %q, want %q", s.Title, "The Simpsons")
	}
	if s.CleanTitle != "thesimpsons" {
		t.Errorf("series[0].CleanTitle = %q, want %q", s.CleanTitle, "thesimpsons")
	}
	if s.Status != "continuing" {
		t.Errorf("series[0].Status = %q, want %q", s.Status, "continuing")
	}
	if s.Path != "/tv/The Simpsons" {
		t.Errorf("series[0].Path = %q, want %q", s.Path, "/tv/The Simpsons")
	}
	if !s.Monitored {
		t.Error("series[0].Monitored = false, want true")
	}
	if s.SeriesType != "standard" {
		t.Errorf("series[0].SeriesType = %q, want %q", s.SeriesType, "standard")
	}
	wantAdded := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	if !s.Added.Equal(wantAdded) {
		t.Errorf("series[0].Added = %v, want %v", s.Added, wantAdded)
	}

	s2 := series[1]
	if s2.ID != 2 {
		t.Errorf("series[1].ID = %d, want 2", s2.ID)
	}
	if s2.Status != "ended" {
		t.Errorf("series[1].Status = %q, want %q", s2.Status, "ended")
	}
}

func TestReadSeasons(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	seasons, err := readSeasons(ctx, db)
	if err != nil {
		t.Fatalf("readSeasons: %v", err)
	}
	if len(seasons) != 3 {
		t.Fatalf("expected 3 seasons, got %d", len(seasons))
	}

	s := seasons[0]
	if s.ID != 1 {
		t.Errorf("seasons[0].ID = %d, want 1", s.ID)
	}
	if s.SeriesID != 1 {
		t.Errorf("seasons[0].SeriesID = %d, want 1", s.SeriesID)
	}
	if s.SeasonNumber != 1 {
		t.Errorf("seasons[0].SeasonNumber = %d, want 1", s.SeasonNumber)
	}
	if !s.Monitored {
		t.Error("seasons[0].Monitored = false, want true")
	}

	s2 := seasons[1]
	if s2.Monitored {
		t.Error("seasons[1].Monitored = true, want false")
	}
}

func TestReadEpisodes(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	episodes, err := readEpisodes(ctx, db)
	if err != nil {
		t.Fatalf("readEpisodes: %v", err)
	}
	if len(episodes) != 3 {
		t.Fatalf("expected 3 episodes, got %d", len(episodes))
	}

	// Episode 1 has EpisodeFileId=1 (non-zero) and AbsoluteEpisodeNumber=0 (nil)
	e1 := episodes[0]
	if e1.ID != 1 {
		t.Errorf("episodes[0].ID = %d, want 1", e1.ID)
	}
	if e1.AbsoluteEpisodeNumber != nil {
		t.Errorf("episodes[0].AbsoluteEpisodeNumber = %v, want nil (was 0 in DB)", e1.AbsoluteEpisodeNumber)
	}
	if e1.EpisodeFileID == nil {
		t.Error("episodes[0].EpisodeFileID = nil, want non-nil (was 1 in DB)")
	} else if *e1.EpisodeFileID != 1 {
		t.Errorf("episodes[0].EpisodeFileID = %d, want 1", *e1.EpisodeFileID)
	}
	if e1.AirDateUtc == nil {
		t.Error("episodes[0].AirDateUtc = nil, want non-nil")
	} else {
		wantAir := time.Date(1989, 12, 17, 0, 0, 0, 0, time.UTC)
		if !e1.AirDateUtc.Equal(wantAir) {
			t.Errorf("episodes[0].AirDateUtc = %v, want %v", *e1.AirDateUtc, wantAir)
		}
	}
	if e1.Title != "Simpsons Roasting on an Open Fire" {
		t.Errorf("episodes[0].Title = %q, want %q", e1.Title, "Simpsons Roasting on an Open Fire")
	}
	if !e1.Monitored {
		t.Error("episodes[0].Monitored = false, want true")
	}

	// Episode 2 has EpisodeFileId=0 (should be nil)
	e2 := episodes[1]
	if e2.EpisodeFileID != nil {
		t.Errorf("episodes[1].EpisodeFileID = %v, want nil (was 0 in DB)", e2.EpisodeFileID)
	}

	// Episode 3 also has EpisodeFileId=0
	e3 := episodes[2]
	if e3.EpisodeFileID != nil {
		t.Errorf("episodes[2].EpisodeFileID = %v, want nil (was 0 in DB)", e3.EpisodeFileID)
	}
}

func TestReadEpisodeFiles(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	files, err := readEpisodeFiles(ctx, db)
	if err != nil {
		t.Fatalf("readEpisodeFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 episode file, got %d", len(files))
	}

	f := files[0]
	if f.ID != 1 {
		t.Errorf("files[0].ID = %d, want 1", f.ID)
	}
	if f.SeriesID != 1 {
		t.Errorf("files[0].SeriesID = %d, want 1", f.SeriesID)
	}
	if f.SeasonNumber != 1 {
		t.Errorf("files[0].SeasonNumber = %d, want 1", f.SeasonNumber)
	}
	if f.RelativePath != "Season 01/The Simpsons - S01E01 - Pilot.mkv" {
		t.Errorf("files[0].RelativePath = %q", f.RelativePath)
	}
	if f.Size != 1500000000 {
		t.Errorf("files[0].Size = %d, want 1500000000", f.Size)
	}
	if f.ReleaseGroup != "ABCD" {
		t.Errorf("files[0].ReleaseGroup = %q, want %q", f.ReleaseGroup, "ABCD")
	}
	if extractQualityName(f.Quality) != "Bluray-1080p" {
		t.Errorf("extractQualityName(files[0].Quality) = %q, want %q", extractQualityName(f.Quality), "Bluray-1080p")
	}
	wantDateAdded := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)
	if !f.DateAdded.Equal(wantDateAdded) {
		t.Errorf("files[0].DateAdded = %v, want %v", f.DateAdded, wantDateAdded)
	}
}

func TestReadQualityProfiles(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	profiles, err := readQualityProfiles(ctx, db)
	if err != nil {
		t.Fatalf("readQualityProfiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 quality profile, got %d", len(profiles))
	}

	p := profiles[0]
	if p.ID != 1 {
		t.Errorf("profiles[0].ID = %d, want 1", p.ID)
	}
	if p.Name != "HD-1080p" {
		t.Errorf("profiles[0].Name = %q, want %q", p.Name, "HD-1080p")
	}
	if !p.UpgradeAllowed {
		t.Error("profiles[0].UpgradeAllowed = false, want true")
	}
	if p.Cutoff != 7 {
		t.Errorf("profiles[0].Cutoff = %d, want 7", p.Cutoff)
	}
	if p.Items == "" {
		t.Error("profiles[0].Items is empty")
	}
}

func TestReadIndexers(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	indexers, err := readIndexers(ctx, db)
	if err != nil {
		t.Fatalf("readIndexers: %v", err)
	}
	if len(indexers) != 1 {
		t.Fatalf("expected 1 indexer, got %d", len(indexers))
	}

	idx := indexers[0]
	if idx.ID != 1 {
		t.Errorf("indexers[0].ID = %d, want 1", idx.ID)
	}
	if idx.Name != "NZBgeek" {
		t.Errorf("indexers[0].Name = %q, want %q", idx.Name, "NZBgeek")
	}
	if idx.Implementation != "Newznab" {
		t.Errorf("indexers[0].Implementation = %q, want %q", idx.Implementation, "Newznab")
	}
	if !idx.EnableRss {
		t.Error("indexers[0].EnableRss = false, want true")
	}
	if !idx.EnableAutomaticSearch {
		t.Error("indexers[0].EnableAutomaticSearch = false, want true")
	}
	if idx.EnableInteractiveSearch {
		t.Error("indexers[0].EnableInteractiveSearch = true, want false")
	}
	if idx.Priority != 25 {
		t.Errorf("indexers[0].Priority = %d, want 25", idx.Priority)
	}
}

func TestReadDownloadClients(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	clients, err := readDownloadClients(ctx, db)
	if err != nil {
		t.Fatalf("readDownloadClients: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 download client, got %d", len(clients))
	}

	dc := clients[0]
	if dc.ID != 1 {
		t.Errorf("clients[0].ID = %d, want 1", dc.ID)
	}
	if dc.Name != "SABnzbd" {
		t.Errorf("clients[0].Name = %q, want %q", dc.Name, "SABnzbd")
	}
	if dc.Implementation != "Sabnzbd" {
		t.Errorf("clients[0].Implementation = %q, want %q", dc.Implementation, "Sabnzbd")
	}
	if !dc.Enable {
		t.Error("clients[0].Enable = false, want true")
	}
	if dc.Priority != 1 {
		t.Errorf("clients[0].Priority = %d, want 1", dc.Priority)
	}
}

func TestReadNotifications(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	notifications, err := readNotifications(ctx, db)
	if err != nil {
		t.Fatalf("readNotifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}

	n := notifications[0]
	if n.ID != 1 {
		t.Errorf("notifications[0].ID = %d, want 1", n.ID)
	}
	if n.Name != "Discord" {
		t.Errorf("notifications[0].Name = %q, want %q", n.Name, "Discord")
	}
	if n.Implementation != "Discord" {
		t.Errorf("notifications[0].Implementation = %q, want %q", n.Implementation, "Discord")
	}
	if !n.OnGrab {
		t.Error("notifications[0].OnGrab = false, want true")
	}
	if !n.OnDownload {
		t.Error("notifications[0].OnDownload = false, want true")
	}
	if n.OnHealthIssue {
		t.Error("notifications[0].OnHealthIssue = true, want false")
	}
}

func TestReadHistory(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	history, err := readHistory(ctx, db)
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}

	h := history[0]
	if h.ID != 1 {
		t.Errorf("history[0].ID = %d, want 1", h.ID)
	}
	if h.EpisodeID != 1 {
		t.Errorf("history[0].EpisodeID = %d, want 1", h.EpisodeID)
	}
	if h.SeriesID != 1 {
		t.Errorf("history[0].SeriesID = %d, want 1", h.SeriesID)
	}
	if h.SourceTitle != "The.Simpsons.S01E01.1080p.BluRay" {
		t.Errorf("history[0].SourceTitle = %q", h.SourceTitle)
	}
	if h.EventType != "grabbed" {
		t.Errorf("history[0].EventType = %q, want %q", h.EventType, "grabbed")
	}
	if h.DownloadID != "dl-123" {
		t.Errorf("history[0].DownloadID = %q, want %q", h.DownloadID, "dl-123")
	}
	wantDate := time.Date(2023, 1, 10, 12, 0, 0, 0, time.UTC)
	if !h.Date.Equal(wantDate) {
		t.Errorf("history[0].Date = %v, want %v", h.Date, wantDate)
	}
}

func TestReadAPIKey(t *testing.T) {
	ctx := context.Background()
	db := createFixtureDB(t)

	key, err := readAPIKey(ctx, db)
	if err != nil {
		t.Fatalf("readAPIKey: %v", err)
	}
	if key != "test-api-key-12345" {
		t.Errorf("readAPIKey = %q, want %q", key, "test-api-key-12345")
	}
}

func TestExtractQualityName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "valid quality JSON",
			input: `{"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080}}`,
			want:  "Bluray-1080p",
		},
		{
			name:  "bad JSON",
			input: `not-valid-json`,
			want:  "Unknown",
		},
		{
			name:  "empty string",
			input: "",
			want:  "Unknown",
		},
		{
			name:  "missing quality name",
			input: `{"quality":{"id":7}}`,
			want:  "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractQualityName(tt.input)
			if got != tt.want {
				t.Errorf("extractQualityName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// createFixtureDBFile creates the same fixture as createFixtureDB but writes
// it to the given file path instead of in-memory. The returned *sql.DB is
// already closed; use the path to re-open it.
func createFixtureDBFile(t *testing.T, path string) {
	t.Helper()
	fileDB, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open fixture file db: %v", err)
	}
	defer fileDB.Close()

	_, err = fileDB.Exec(`
		CREATE TABLE Series (
			Id INTEGER PRIMARY KEY, TvdbId INTEGER, Title TEXT, CleanTitle TEXT,
			Status TEXT, Path TEXT, Monitored INTEGER, SeriesType TEXT, Added TEXT
		);
		CREATE TABLE Seasons (
			Id INTEGER PRIMARY KEY, SeriesId INTEGER, SeasonNumber INTEGER, Monitored INTEGER
		);
		CREATE TABLE Episodes (
			Id INTEGER PRIMARY KEY, SeriesId INTEGER, SeasonNumber INTEGER,
			EpisodeNumber INTEGER, AbsoluteEpisodeNumber INTEGER,
			Title TEXT, Overview TEXT, AirDateUtc TEXT, Monitored INTEGER, EpisodeFileId INTEGER
		);
		CREATE TABLE EpisodeFiles (
			Id INTEGER PRIMARY KEY, SeriesId INTEGER, SeasonNumber INTEGER,
			RelativePath TEXT, Size INTEGER, DateAdded TEXT, ReleaseGroup TEXT, Quality TEXT
		);
		CREATE TABLE QualityProfiles (
			Id INTEGER PRIMARY KEY, Name TEXT, UpgradeAllowed INTEGER, Cutoff INTEGER, Items TEXT
		);
		CREATE TABLE Indexers (
			Id INTEGER PRIMARY KEY, Name TEXT, Implementation TEXT, Settings TEXT,
			EnableRss INTEGER, EnableAutomaticSearch INTEGER, EnableInteractiveSearch INTEGER, Priority INTEGER
		);
		CREATE TABLE DownloadClients (
			Id INTEGER PRIMARY KEY, Name TEXT, Implementation TEXT, Settings TEXT,
			Enable INTEGER, Priority INTEGER
		);
		CREATE TABLE Notifications (
			Id INTEGER PRIMARY KEY, Name TEXT, Implementation TEXT, Settings TEXT,
			OnGrab INTEGER, OnDownload INTEGER, OnHealthIssue INTEGER
		);
		CREATE TABLE Config (Key TEXT PRIMARY KEY, Value TEXT);
		CREATE TABLE History (
			Id INTEGER PRIMARY KEY, EpisodeId INTEGER, SeriesId INTEGER,
			SourceTitle TEXT, Quality TEXT, Date TEXT, EventType TEXT, DownloadId TEXT, Data TEXT
		);

		-- Insert fixture data
		INSERT INTO Config (Key, Value) VALUES ('ApiKey', 'test-api-key-12345');

		INSERT INTO Series VALUES (1, 71663, 'The Simpsons', 'thesimpsons', 'continuing', '/tv/The Simpsons', 1, 'standard', '2023-01-01 00:00:00');
		INSERT INTO Series VALUES (2, 81189, 'Breaking Bad', 'breakingbad', 'ended', '/tv/Breaking Bad', 1, 'standard', '2023-06-15 12:00:00');

		INSERT INTO Seasons VALUES (1, 1, 1, 1);
		INSERT INTO Seasons VALUES (2, 1, 2, 0);
		INSERT INTO Seasons VALUES (3, 2, 1, 1);

		INSERT INTO EpisodeFiles VALUES (1, 1, 1, 'Season 01/The Simpsons - S01E01 - Pilot.mkv', 1500000000, '2023-01-10 00:00:00', 'ABCD', '{"quality":{"id":7,"name":"Bluray-1080p","source":"bluray","resolution":1080}}');

		INSERT INTO Episodes VALUES (1, 1, 1, 1, 0, 'Simpsons Roasting on an Open Fire', 'The first episode.', '1989-12-17 00:00:00', 1, 1);
		INSERT INTO Episodes VALUES (2, 1, 1, 2, 0, 'Bart the Genius', 'Bart cheats.', '1990-01-14 00:00:00', 1, 0);
		INSERT INTO Episodes VALUES (3, 2, 1, 1, 0, 'Pilot', 'Walter starts cooking.', '2008-01-20 00:00:00', 1, 0);

		INSERT INTO QualityProfiles VALUES (1, 'HD-1080p', 1, 7, '[{"quality":{"id":7,"name":"Bluray-1080p"},"allowed":true}]');

		INSERT INTO Indexers VALUES (1, 'NZBgeek', 'Newznab', '{"baseUrl":"https://nzbgeek.info","apiKey":"abc123"}', 1, 1, 0, 25);

		INSERT INTO DownloadClients VALUES (1, 'SABnzbd', 'Sabnzbd', '{"host":"localhost","port":8080,"apiKey":"sab123"}', 1, 1);

		INSERT INTO Notifications VALUES (1, 'Discord', 'Discord', '{"webhookUrl":"https://discord.com/api/webhooks/123"}', 1, 1, 0);

		INSERT INTO History VALUES (1, 1, 1, 'The.Simpsons.S01E01.1080p.BluRay', '{"quality":{"id":7,"name":"Bluray-1080p"}}', '2023-01-10 12:00:00', 'grabbed', 'dl-123', '{}');
	`)
	if err != nil {
		t.Fatalf("create fixture file: %v", err)
	}
}

func TestMigratorIntegration(t *testing.T) {
	ctx := context.Background()

	// Create source fixture DB at a temp file path.
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "sonarr.db")
	createFixtureDBFile(t, sourceFile)

	// Create destination sonarr2 DB.
	destPath := filepath.Join(tmpDir, "sonarr2.db")
	destPool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN: "file:" + destPath + "?_journal=WAL&_busy_timeout=5000",
	})
	if err != nil {
		t.Fatalf("open dest: %v", err)
	}
	defer destPool.Close()

	if err := db.Migrate(ctx, destPool); err != nil {
		t.Fatalf("migrate dest: %v", err)
	}

	// Run migration.
	m, err := New(Options{
		SourcePath: sourceFile,
		DestPool:   destPool,
		Remaps:     []PathRemap{{Old: "/tv", New: "/media/tv"}},
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	defer m.Close()

	report, err := m.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify report counts.
	if report.Series != 2 {
		t.Errorf("Series = %d, want 2", report.Series)
	}
	if report.Seasons != 3 {
		t.Errorf("Seasons = %d, want 3", report.Seasons)
	}
	if report.Episodes != 3 {
		t.Errorf("Episodes = %d, want 3", report.Episodes)
	}
	if report.EpisodeFiles != 1 {
		t.Errorf("EpisodeFiles = %d, want 1", report.EpisodeFiles)
	}
	if report.QualityProfiles != 1 {
		t.Errorf("QualityProfiles = %d, want 1", report.QualityProfiles)
	}
	if report.Indexers != 1 {
		t.Errorf("Indexers = %d, want 1", report.Indexers)
	}
	if report.DownloadClients != 1 {
		t.Errorf("DownloadClients = %d, want 1", report.DownloadClients)
	}
	if report.Notifications != 1 {
		t.Errorf("Notifications = %d, want 1", report.Notifications)
	}
	if report.History != 1 {
		t.Errorf("History = %d, want 1", report.History)
	}

	// Verify path remapping: series paths should have /tv replaced by /media/tv.
	lib, err := library.New(destPool, events.NewNoopBus())
	if err != nil {
		t.Fatalf("create library for verification: %v", err)
	}
	allSeries, err := lib.Series.List(ctx)
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	if len(allSeries) != 2 {
		t.Fatalf("listed %d series, want 2", len(allSeries))
	}
	for _, s := range allSeries {
		if strings.HasPrefix(s.Path, "/tv") {
			t.Errorf("series %q path not remapped: %s", s.Title, s.Path)
		}
		if !strings.HasPrefix(s.Path, "/media/tv") {
			t.Errorf("series %q path incorrect: %s", s.Title, s.Path)
		}
	}

	// Verify warnings slice is empty (all entities should have mapped cleanly).
	if len(report.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", report.Warnings)
	}
}
