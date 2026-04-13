package migrate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/providers/notification"

	_ "modernc.org/sqlite"
)

// Options controls Migrator behaviour.
type Options struct {
	SourcePath  string
	DestPool    db.Pool
	Remaps      []PathRemap
	DryRun      bool
	SkipHistory bool
	Log         *slog.Logger
}

// PathRemap rewrites a path prefix during migration.
type PathRemap struct {
	Old string
	New string
}

// Report summarises what was imported in a single Run.
type Report struct {
	Series          int      `json:"series"`
	Seasons         int      `json:"seasons"`
	Episodes        int      `json:"episodes"`
	EpisodeFiles    int      `json:"episodeFiles"`
	QualityProfiles int      `json:"qualityProfiles"`
	Indexers        int      `json:"indexers"`
	DownloadClients int      `json:"downloadClients"`
	Notifications   int      `json:"notifications"`
	History         int      `json:"history"`
	Warnings        []string `json:"warnings,omitempty"`
}

// Migrator reads from a source Sonarr SQLite database and writes into a
// sonarr2 destination database via the store interfaces.
type Migrator struct {
	source *sql.DB
	dest   db.Pool
	opts   Options
	log    *slog.Logger

	// ID mappings: source ID → dest ID
	seriesMap      map[int64]int64
	episodeMap     map[int64]int64
	episodeFileMap map[int64]int64
	profileMap     map[int64]int64
}

// New opens the source database and returns a ready Migrator.
// Call Close when done.
func New(opts Options) (*Migrator, error) {
	sourceDB, err := sql.Open("sqlite", opts.SourcePath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open source database: %w", err)
	}
	if err := sourceDB.Ping(); err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("ping source database: %w", err)
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	return &Migrator{
		source:         sourceDB,
		dest:           opts.DestPool,
		opts:           opts,
		log:            log,
		seriesMap:      make(map[int64]int64),
		episodeMap:     make(map[int64]int64),
		episodeFileMap: make(map[int64]int64),
		profileMap:     make(map[int64]int64),
	}, nil
}

// Close releases the source database connection.
func (m *Migrator) Close() error {
	return m.source.Close()
}

// Run orchestrates the full migration in topological order and returns a
// summary Report.
func (m *Migrator) Run(ctx context.Context) (*Report, error) {
	report := &Report{}

	// 1. API key
	if err := m.migrateAPIKey(ctx); err != nil {
		return nil, fmt.Errorf("migrate API key: %w", err)
	}

	// 2. Quality profiles
	n, err := m.migrateQualityProfiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate quality profiles: %w", err)
	}
	report.QualityProfiles = n

	// 3. Series
	n, err = m.migrateSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate series: %w", err)
	}
	report.Series = n

	// 4. Seasons
	n, err = m.migrateSeasons(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate seasons: %w", err)
	}
	report.Seasons = n

	// 5. Episode files (before episodes so we can map file IDs)
	n, err = m.migrateEpisodeFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate episode files: %w", err)
	}
	report.EpisodeFiles = n

	// 6. Episodes
	n, err = m.migrateEpisodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate episodes: %w", err)
	}
	report.Episodes = n

	// 7. Indexers
	n, err = m.migrateIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate indexers: %w", err)
	}
	report.Indexers = n

	// 8. Download clients
	n, err = m.migrateDownloadClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate download clients: %w", err)
	}
	report.DownloadClients = n

	// 9. Notifications
	n, err = m.migrateNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrate notifications: %w", err)
	}
	report.Notifications = n

	// 10. History (optional)
	if !m.opts.SkipHistory {
		n, err = m.migrateHistory(ctx, report)
		if err != nil {
			return nil, fmt.Errorf("migrate history: %w", err)
		}
		report.History = n
	}

	return report, nil
}

// remapPath rewrites path prefixes according to opts.Remaps. Returns the
// path unchanged if no remap matches.
func (m *Migrator) remapPath(path string) string {
	for _, r := range m.opts.Remaps {
		if strings.HasPrefix(path, r.Old) {
			return r.New + path[len(r.Old):]
		}
	}
	return path
}

// newHostConfigStore returns the Store for the destination dialect.
func (m *Migrator) newHostConfigStore() (hostconfig.Store, error) {
	switch p := m.dest.(type) {
	case *db.PostgresPool:
		return hostconfig.NewPostgresStore(p), nil
	case *db.SQLitePool:
		return hostconfig.NewSQLiteStore(p), nil
	default:
		return nil, fmt.Errorf("migrate: unsupported pool type %T", m.dest)
	}
}

// newQualityProfileStore returns the QualityProfileStore for the destination dialect.
func (m *Migrator) newQualityProfileStore() (profiles.QualityProfileStore, error) {
	switch p := m.dest.(type) {
	case *db.PostgresPool:
		return profiles.NewPostgresQualityProfileStore(p), nil
	case *db.SQLitePool:
		return profiles.NewSQLiteQualityProfileStore(p), nil
	default:
		return nil, fmt.Errorf("migrate: unsupported pool type %T", m.dest)
	}
}

// newIndexerStore returns the indexer InstanceStore for the destination dialect.
func (m *Migrator) newIndexerStore() (indexer.InstanceStore, error) {
	switch p := m.dest.(type) {
	case *db.PostgresPool:
		return indexer.NewPostgresInstanceStore(p), nil
	case *db.SQLitePool:
		return indexer.NewSQLiteInstanceStore(p), nil
	default:
		return nil, fmt.Errorf("migrate: unsupported pool type %T", m.dest)
	}
}

// newDownloadClientStore returns the download client InstanceStore for the destination dialect.
func (m *Migrator) newDownloadClientStore() (downloadclient.InstanceStore, error) {
	switch p := m.dest.(type) {
	case *db.PostgresPool:
		return downloadclient.NewPostgresInstanceStore(p), nil
	case *db.SQLitePool:
		return downloadclient.NewSQLiteInstanceStore(p), nil
	default:
		return nil, fmt.Errorf("migrate: unsupported pool type %T", m.dest)
	}
}

// newNotificationStore returns the notification InstanceStore for the destination dialect.
func (m *Migrator) newNotificationStore() (notification.InstanceStore, error) {
	switch p := m.dest.(type) {
	case *db.PostgresPool:
		return notification.NewPostgresInstanceStore(p), nil
	case *db.SQLitePool:
		return notification.NewSQLiteInstanceStore(p), nil
	default:
		return nil, fmt.Errorf("migrate: unsupported pool type %T", m.dest)
	}
}

// newHistoryStore returns the history Store for the destination dialect.
func (m *Migrator) newHistoryStore() (history.Store, error) {
	switch p := m.dest.(type) {
	case *db.PostgresPool:
		return history.NewPostgresStore(p), nil
	case *db.SQLitePool:
		return history.NewSQLiteStore(p), nil
	default:
		return nil, fmt.Errorf("migrate: unsupported pool type %T", m.dest)
	}
}

// migrateAPIKey copies the source API key into the destination host_config.
func (m *Migrator) migrateAPIKey(ctx context.Context) error {
	apiKey, err := readAPIKey(ctx, m.source)
	if err != nil {
		m.log.Warn("could not read source API key, skipping", slog.String("err", err.Error()))
		return nil
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate API key")
		return nil
	}
	hcStore, err := m.newHostConfigStore()
	if err != nil {
		return err
	}
	return hcStore.Upsert(ctx, hostconfig.HostConfig{
		APIKey:   apiKey,
		AuthMode: "none",
	})
}

// migrateQualityProfiles migrates quality profiles from source to destination.
func (m *Migrator) migrateQualityProfiles(ctx context.Context) (int, error) {
	sources, err := readQualityProfiles(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate quality profiles", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	qpStore, err := m.newQualityProfileStore()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, s := range sources {
		// Parse items JSON from source into the canonical items slice.
		// The source format is [{"quality":{"id":N,"name":"..."},"allowed":true}].
		// We map it to QualityProfileItem by pulling out id and allowed.
		var rawItems []struct {
			Quality struct {
				ID int `json:"id"`
			} `json:"quality"`
			Allowed bool `json:"allowed"`
		}
		if err := json.Unmarshal([]byte(s.Items), &rawItems); err != nil {
			m.log.Warn("skip quality profile: bad items JSON",
				slog.String("name", s.Name), slog.String("err", err.Error()))
			continue
		}
		items := make([]profiles.QualityProfileItem, 0, len(rawItems))
		for _, ri := range rawItems {
			items = append(items, profiles.QualityProfileItem{
				QualityID: ri.Quality.ID,
				Allowed:   ri.Allowed,
			})
		}

		created, err := qpStore.Create(ctx, profiles.QualityProfile{
			Name:           s.Name,
			UpgradeAllowed: s.UpgradeAllowed,
			Cutoff:         s.Cutoff,
			Items:          items,
			FormatItems:    []profiles.FormatScoreItem{},
		})
		if err != nil {
			m.log.Warn("skip quality profile",
				slog.String("name", s.Name), slog.String("err", err.Error()))
			continue
		}
		m.profileMap[s.ID] = int64(created.ID)
		count++
	}
	return count, nil
}

// migrateSeries migrates TV series from source to destination.
func (m *Migrator) migrateSeries(ctx context.Context) (int, error) {
	sources, err := readSeries(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate series", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	lib, err := library.New(m.dest, events.NewNoopBus())
	if err != nil {
		return 0, fmt.Errorf("create library: %w", err)
	}

	count := 0
	for _, s := range sources {
		created, err := lib.Series.Create(ctx, library.Series{
			TvdbID:     s.TvdbID,
			Title:      s.Title,
			Slug:       s.CleanTitle,
			Status:     s.Status,
			SeriesType: s.SeriesType,
			Path:       m.remapPath(s.Path),
			Monitored:  s.Monitored,
			Added:      s.Added,
		})
		if err != nil {
			m.log.Warn("skip series",
				slog.String("title", s.Title), slog.String("err", err.Error()))
			continue
		}
		m.seriesMap[s.ID] = created.ID
		count++
	}
	return count, nil
}

// migrateSeasons migrates seasons from source to destination.
func (m *Migrator) migrateSeasons(ctx context.Context) (int, error) {
	sources, err := readSeasons(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate seasons", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	lib, err := library.New(m.dest, events.NewNoopBus())
	if err != nil {
		return 0, fmt.Errorf("create library: %w", err)
	}

	count := 0
	for _, s := range sources {
		destSeriesID, ok := m.seriesMap[s.SeriesID]
		if !ok {
			m.log.Warn("skip season: series not mapped",
				slog.Int64("sourceSeriesID", s.SeriesID),
				slog.Int("seasonNumber", s.SeasonNumber))
			continue
		}
		if err := lib.Seasons.Upsert(ctx, library.Season{
			SeriesID:     destSeriesID,
			SeasonNumber: int32(s.SeasonNumber),
			Monitored:    s.Monitored,
		}); err != nil {
			m.log.Warn("skip season",
				slog.Int64("seriesID", destSeriesID),
				slog.Int("seasonNumber", s.SeasonNumber),
				slog.String("err", err.Error()))
			continue
		}
		count++
	}
	return count, nil
}

// migrateEpisodeFiles migrates episode files from source to destination.
func (m *Migrator) migrateEpisodeFiles(ctx context.Context) (int, error) {
	sources, err := readEpisodeFiles(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate episode files", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	lib, err := library.New(m.dest, events.NewNoopBus())
	if err != nil {
		return 0, fmt.Errorf("create library: %w", err)
	}

	count := 0
	for _, f := range sources {
		destSeriesID, ok := m.seriesMap[f.SeriesID]
		if !ok {
			m.log.Warn("skip episode file: series not mapped",
				slog.Int64("sourceSeriesID", f.SeriesID))
			continue
		}
		created, err := lib.EpisodeFiles.Create(ctx, library.EpisodeFile{
			SeriesID:     destSeriesID,
			SeasonNumber: int32(f.SeasonNumber),
			RelativePath: f.RelativePath,
			Size:         f.Size,
			DateAdded:    f.DateAdded,
			ReleaseGroup: f.ReleaseGroup,
			QualityName:  extractQualityName(f.Quality),
		})
		if err != nil {
			m.log.Warn("skip episode file",
				slog.Int64("seriesID", destSeriesID),
				slog.String("err", err.Error()))
			continue
		}
		m.episodeFileMap[f.ID] = created.ID
		count++
	}
	return count, nil
}

// migrateEpisodes migrates episodes from source to destination.
func (m *Migrator) migrateEpisodes(ctx context.Context) (int, error) {
	sources, err := readEpisodes(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate episodes", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	lib, err := library.New(m.dest, events.NewNoopBus())
	if err != nil {
		return 0, fmt.Errorf("create library: %w", err)
	}

	count := 0
	for _, e := range sources {
		destSeriesID, ok := m.seriesMap[e.SeriesID]
		if !ok {
			m.log.Warn("skip episode: series not mapped",
				slog.Int64("sourceSeriesID", e.SeriesID))
			continue
		}

		ep := library.Episode{
			SeriesID:      destSeriesID,
			SeasonNumber:  int32(e.SeasonNumber),
			EpisodeNumber: int32(e.EpisodeNumber),
			Title:         e.Title,
			Overview:      e.Overview,
			AirDateUtc:    e.AirDateUtc,
			Monitored:     e.Monitored,
		}

		if e.AbsoluteEpisodeNumber != nil {
			v := int32(*e.AbsoluteEpisodeNumber)
			ep.AbsoluteEpisodeNumber = &v
		}

		if e.EpisodeFileID != nil {
			if destFileID, ok := m.episodeFileMap[*e.EpisodeFileID]; ok {
				ep.EpisodeFileID = &destFileID
			}
		}

		created, err := lib.Episodes.Create(ctx, ep)
		if err != nil {
			m.log.Warn("skip episode",
				slog.Int64("seriesID", destSeriesID),
				slog.Int("season", e.SeasonNumber),
				slog.Int("episode", e.EpisodeNumber),
				slog.String("err", err.Error()))
			continue
		}
		m.episodeMap[e.ID] = created.ID
		count++
	}
	return count, nil
}

// migrateIndexers migrates indexer configurations from source to destination.
func (m *Migrator) migrateIndexers(ctx context.Context) (int, error) {
	sources, err := readIndexers(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate indexers", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	idxStore, err := m.newIndexerStore()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, idx := range sources {
		_, err := idxStore.Create(ctx, indexer.Instance{
			Name:                    idx.Name,
			Implementation:          idx.Implementation,
			Settings:                json.RawMessage(idx.Settings),
			EnableRss:               idx.EnableRss,
			EnableAutomaticSearch:   idx.EnableAutomaticSearch,
			EnableInteractiveSearch: idx.EnableInteractiveSearch,
			Priority:                idx.Priority,
		})
		if err != nil {
			m.log.Warn("skip indexer",
				slog.String("name", idx.Name), slog.String("err", err.Error()))
			continue
		}
		count++
	}
	return count, nil
}

// migrateDownloadClients migrates download client configurations from source to destination.
func (m *Migrator) migrateDownloadClients(ctx context.Context) (int, error) {
	sources, err := readDownloadClients(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate download clients", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	dcStore, err := m.newDownloadClientStore()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, dc := range sources {
		_, err := dcStore.Create(ctx, downloadclient.Instance{
			Name:           dc.Name,
			Implementation: dc.Implementation,
			Settings:       json.RawMessage(dc.Settings),
			Enable:         dc.Enable,
			Priority:       dc.Priority,
		})
		if err != nil {
			m.log.Warn("skip download client",
				slog.String("name", dc.Name), slog.String("err", err.Error()))
			continue
		}
		count++
	}
	return count, nil
}

// migrateNotifications migrates notification configurations from source to destination.
func (m *Migrator) migrateNotifications(ctx context.Context) (int, error) {
	sources, err := readNotifications(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate notifications", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	notifStore, err := m.newNotificationStore()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, n := range sources {
		_, err := notifStore.Create(ctx, notification.Instance{
			Name:           n.Name,
			Implementation: n.Implementation,
			Settings:       json.RawMessage(n.Settings),
			OnGrab:         n.OnGrab,
			OnDownload:     n.OnDownload,
			OnHealthIssue:  n.OnHealthIssue,
		})
		if err != nil {
			m.log.Warn("skip notification",
				slog.String("name", n.Name), slog.String("err", err.Error()))
			continue
		}
		count++
	}
	return count, nil
}

// migrateHistory migrates grab/import history from source to destination.
// Entries whose series or episode IDs could not be mapped are skipped with a
// warning appended to the report.
func (m *Migrator) migrateHistory(ctx context.Context, report *Report) (int, error) {
	sources, err := readHistory(ctx, m.source)
	if err != nil {
		return 0, err
	}
	if m.opts.DryRun {
		m.log.Info("dry-run: would migrate history", slog.Int("count", len(sources)))
		return len(sources), nil
	}

	hStore, err := m.newHistoryStore()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, h := range sources {
		destSeriesID, ok := m.seriesMap[h.SeriesID]
		if !ok {
			msg := fmt.Sprintf("skip history entry %d: series %d not mapped", h.ID, h.SeriesID)
			m.log.Warn(msg)
			report.Warnings = append(report.Warnings, msg)
			continue
		}
		destEpisodeID, ok := m.episodeMap[h.EpisodeID]
		if !ok {
			msg := fmt.Sprintf("skip history entry %d: episode %d not mapped", h.ID, h.EpisodeID)
			m.log.Warn(msg)
			report.Warnings = append(report.Warnings, msg)
			continue
		}

		var data json.RawMessage
		if h.Data != "" {
			data = json.RawMessage(h.Data)
		} else {
			data = json.RawMessage("{}")
		}

		_, err := hStore.Create(ctx, history.Entry{
			EpisodeID:   destEpisodeID,
			SeriesID:    destSeriesID,
			SourceTitle: h.SourceTitle,
			QualityName: extractQualityName(h.Quality),
			EventType:   history.EventType(h.EventType),
			Date:        h.Date,
			DownloadID:  h.DownloadID,
			Data:        data,
		})
		if err != nil {
			m.log.Warn("skip history entry",
				slog.Int64("id", h.ID), slog.String("err", err.Error()))
			continue
		}
		count++
	}
	return count, nil
}
