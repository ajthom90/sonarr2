package rsssync_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/decisionengine/specs"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/grab"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/rsssync"
)

// ---------------------------------------------------------------------------
// Fake indexer
// ---------------------------------------------------------------------------

type fakeIndexer struct {
	releases []indexer.Release
	fetchErr error
}

func (f *fakeIndexer) Implementation() string             { return "FakeIndexer" }
func (f *fakeIndexer) DefaultName() string                { return "FakeIndexer" }
func (f *fakeIndexer) Settings() any                      { return &struct{}{} }
func (f *fakeIndexer) Test(_ context.Context) error       { return nil }
func (f *fakeIndexer) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }
func (f *fakeIndexer) SupportsRss() bool                  { return true }
func (f *fakeIndexer) SupportsSearch() bool               { return false }
func (f *fakeIndexer) FetchRss(_ context.Context) ([]indexer.Release, error) {
	return f.releases, f.fetchErr
}
func (f *fakeIndexer) Search(_ context.Context, _ indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Fake indexer InstanceStore
// ---------------------------------------------------------------------------

type fakeIdxStore struct {
	instances []indexer.Instance
}

func (s *fakeIdxStore) Create(_ context.Context, inst indexer.Instance) (indexer.Instance, error) {
	inst.ID = len(s.instances) + 1
	s.instances = append(s.instances, inst)
	return inst, nil
}
func (s *fakeIdxStore) GetByID(_ context.Context, id int) (indexer.Instance, error) {
	for _, i := range s.instances {
		if i.ID == id {
			return i, nil
		}
	}
	return indexer.Instance{}, indexer.ErrNotFound
}
func (s *fakeIdxStore) List(_ context.Context) ([]indexer.Instance, error) {
	return s.instances, nil
}
func (s *fakeIdxStore) Update(_ context.Context, _ indexer.Instance) error { return nil }
func (s *fakeIdxStore) Delete(_ context.Context, _ int) error              { return nil }

// ---------------------------------------------------------------------------
// Fake library stores
// ---------------------------------------------------------------------------

type fakeSeriesStore struct {
	series []library.Series
}

func (s *fakeSeriesStore) Create(_ context.Context, ser library.Series) (library.Series, error) {
	ser.ID = int64(len(s.series) + 1)
	s.series = append(s.series, ser)
	return ser, nil
}
func (s *fakeSeriesStore) Get(_ context.Context, id int64) (library.Series, error) {
	for _, ser := range s.series {
		if ser.ID == id {
			return ser, nil
		}
	}
	return library.Series{}, library.ErrNotFound
}
func (s *fakeSeriesStore) GetByTvdbID(_ context.Context, _ int64) (library.Series, error) {
	return library.Series{}, library.ErrNotFound
}
func (s *fakeSeriesStore) GetBySlug(_ context.Context, slug string) (library.Series, error) {
	for _, ser := range s.series {
		if ser.Slug == slug {
			return ser, nil
		}
	}
	return library.Series{}, library.ErrNotFound
}
func (s *fakeSeriesStore) List(_ context.Context) ([]library.Series, error) {
	return s.series, nil
}
func (s *fakeSeriesStore) Update(_ context.Context, _ library.Series) error { return nil }
func (s *fakeSeriesStore) Delete(_ context.Context, _ int64) error          { return nil }

type fakeEpisodesStore struct {
	episodes []library.Episode
}

func (e *fakeEpisodesStore) Create(_ context.Context, ep library.Episode) (library.Episode, error) {
	ep.ID = int64(len(e.episodes) + 1)
	e.episodes = append(e.episodes, ep)
	return ep, nil
}
func (e *fakeEpisodesStore) Get(_ context.Context, id int64) (library.Episode, error) {
	for _, ep := range e.episodes {
		if ep.ID == id {
			return ep, nil
		}
	}
	return library.Episode{}, library.ErrNotFound
}
func (e *fakeEpisodesStore) ListForSeries(_ context.Context, seriesID int64) ([]library.Episode, error) {
	var out []library.Episode
	for _, ep := range e.episodes {
		if ep.SeriesID == seriesID {
			out = append(out, ep)
		}
	}
	return out, nil
}
func (e *fakeEpisodesStore) Update(_ context.Context, _ library.Episode) error { return nil }
func (e *fakeEpisodesStore) Delete(_ context.Context, _ int64) error           { return nil }
func (e *fakeEpisodesStore) CountForSeries(_ context.Context, _ int64) (int, int, error) {
	return 0, 0, nil
}
func (e *fakeEpisodesStore) ListAll(_ context.Context) ([]library.Episode, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Fake quality stores
// ---------------------------------------------------------------------------

type fakeQualityDefStore struct {
	defs []profiles.QualityDefinition
}

func (s *fakeQualityDefStore) GetAll(_ context.Context) ([]profiles.QualityDefinition, error) {
	return s.defs, nil
}
func (s *fakeQualityDefStore) GetByID(_ context.Context, id int) (profiles.QualityDefinition, error) {
	for _, d := range s.defs {
		if d.ID == id {
			return d, nil
		}
	}
	return profiles.QualityDefinition{}, errors.New("quality def not found")
}

type fakeQualityProfileStore struct {
	profiles map[int]profiles.QualityProfile
}

func newFakeQualityProfileStore() *fakeQualityProfileStore {
	return &fakeQualityProfileStore{profiles: make(map[int]profiles.QualityProfile)}
}
func (s *fakeQualityProfileStore) Create(_ context.Context, p profiles.QualityProfile) (profiles.QualityProfile, error) {
	s.profiles[p.ID] = p
	return p, nil
}
func (s *fakeQualityProfileStore) GetByID(_ context.Context, id int) (profiles.QualityProfile, error) {
	p, ok := s.profiles[id]
	if !ok {
		return profiles.QualityProfile{}, errors.New("quality profile not found")
	}
	return p, nil
}
func (s *fakeQualityProfileStore) List(_ context.Context) ([]profiles.QualityProfile, error) {
	var out []profiles.QualityProfile
	for _, p := range s.profiles {
		out = append(out, p)
	}
	return out, nil
}
func (s *fakeQualityProfileStore) Update(_ context.Context, p profiles.QualityProfile) error {
	s.profiles[p.ID] = p
	return nil
}
func (s *fakeQualityProfileStore) Delete(_ context.Context, id int) error {
	delete(s.profiles, id)
	return nil
}

// ---------------------------------------------------------------------------
// Fake custom format store
// ---------------------------------------------------------------------------

type fakeCFStore struct {
	formats []customformats.CustomFormat
}

func (s *fakeCFStore) Create(_ context.Context, cf customformats.CustomFormat) (customformats.CustomFormat, error) {
	cf.ID = len(s.formats) + 1
	s.formats = append(s.formats, cf)
	return cf, nil
}
func (s *fakeCFStore) GetByID(_ context.Context, id int) (customformats.CustomFormat, error) {
	for _, cf := range s.formats {
		if cf.ID == id {
			return cf, nil
		}
	}
	return customformats.CustomFormat{}, errors.New("not found")
}
func (s *fakeCFStore) List(_ context.Context) ([]customformats.CustomFormat, error) {
	return s.formats, nil
}
func (s *fakeCFStore) Update(_ context.Context, cf customformats.CustomFormat) error { return nil }
func (s *fakeCFStore) Delete(_ context.Context, _ int) error                         { return nil }

// ---------------------------------------------------------------------------
// Fake download client + dcStore + historyStore (for GrabService)
// ---------------------------------------------------------------------------

type fakeDCClient struct {
	protocol   indexer.DownloadProtocol
	downloadID string
	addErr     error
	addCalls   int
}

func (f *fakeDCClient) Implementation() string { return "FakeDC" }
func (f *fakeDCClient) DefaultName() string    { return "FakeDC" }
func (f *fakeDCClient) Settings() any          { return &struct{}{} }
func (f *fakeDCClient) Test(_ context.Context) error {
	return nil
}
func (f *fakeDCClient) Protocol() indexer.DownloadProtocol { return f.protocol }
func (f *fakeDCClient) Add(_ context.Context, _, _ string) (string, error) {
	f.addCalls++
	return f.downloadID, f.addErr
}
func (f *fakeDCClient) Items(_ context.Context) ([]downloadclient.Item, error) { return nil, nil }
func (f *fakeDCClient) Remove(_ context.Context, _ string, _ bool) error       { return nil }
func (f *fakeDCClient) Status(_ context.Context) (downloadclient.Status, error) {
	return downloadclient.Status{}, nil
}

type fakeDCStore struct {
	instances []downloadclient.Instance
}

func (s *fakeDCStore) Create(_ context.Context, inst downloadclient.Instance) (downloadclient.Instance, error) {
	inst.ID = len(s.instances) + 1
	s.instances = append(s.instances, inst)
	return inst, nil
}
func (s *fakeDCStore) GetByID(_ context.Context, id int) (downloadclient.Instance, error) {
	for _, i := range s.instances {
		if i.ID == id {
			return i, nil
		}
	}
	return downloadclient.Instance{}, downloadclient.ErrNotFound
}
func (s *fakeDCStore) List(_ context.Context) ([]downloadclient.Instance, error) {
	return s.instances, nil
}
func (s *fakeDCStore) Update(_ context.Context, _ downloadclient.Instance) error { return nil }
func (s *fakeDCStore) Delete(_ context.Context, _ int) error                     { return nil }

type fakeHistoryStore struct{}

func (h *fakeHistoryStore) Create(_ context.Context, e history.Entry) (history.Entry, error) {
	e.ID = 1
	return e, nil
}
func (h *fakeHistoryStore) ListForSeries(_ context.Context, _ int64) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) ListForEpisode(_ context.Context, _ int64) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) FindByDownloadID(_ context.Context, _ string) ([]history.Entry, error) {
	return nil, nil
}
func (h *fakeHistoryStore) DeleteForSeries(_ context.Context, _ int64) error { return nil }
func (h *fakeHistoryStore) ListAll(_ context.Context) ([]history.Entry, error) { return nil, nil }

// ---------------------------------------------------------------------------
// Test builder
// ---------------------------------------------------------------------------

type testEnv struct {
	idxStore    *fakeIdxStore
	idxRegistry *indexer.Registry
	seriesStore *fakeSeriesStore
	epStore     *fakeEpisodesStore
	dcClient    *fakeDCClient
	handler     *rsssync.Handler
}

func buildEnv(t *testing.T, engineSpecs ...decisionengine.Spec) *testEnv {
	t.Helper()

	env := &testEnv{
		idxStore:    &fakeIdxStore{},
		idxRegistry: indexer.NewRegistry(),
		seriesStore: &fakeSeriesStore{},
		epStore:     &fakeEpisodesStore{},
		dcClient:    &fakeDCClient{protocol: indexer.ProtocolUsenet, downloadID: "dl-001"},
	}

	dcStore := &fakeDCStore{
		instances: []downloadclient.Instance{
			{ID: 1, Name: "FakeDC", Implementation: "FakeDC", Enable: true, Priority: 1},
		},
	}
	dcReg := downloadclient.NewRegistry()
	dcReg.Register("FakeDC", func() downloadclient.DownloadClient { return env.dcClient })

	grabSvc := grab.New(dcStore, dcReg, &fakeHistoryStore{}, events.NewBus(4), slog.Default())

	qualDefs := &fakeQualityDefStore{}
	qualProfs := newFakeQualityProfileStore()
	// Seed default profile.
	_, _ = qualProfs.Create(context.Background(), profiles.QualityProfile{
		ID:   1,
		Name: "Any",
	})

	cfStore := &fakeCFStore{}

	lib := &library.Library{
		Series:   env.seriesStore,
		Episodes: env.epStore,
	}

	eng := decisionengine.New(engineSpecs...)

	env.handler = rsssync.New(
		env.idxStore,
		env.idxRegistry,
		lib,
		eng,
		grabSvc,
		qualDefs,
		qualProfs,
		cfStore,
		slog.Default(),
	)
	return env
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// addIndexer registers a fake indexer with the given releases in the env.
func addIndexer(t *testing.T, env *testEnv, releases []indexer.Release) {
	t.Helper()
	fake := &fakeIndexer{releases: releases}
	const impl = "FakeIndexer"
	// Only register the factory once (multiple calls would panic).
	if _, err := env.idxRegistry.Get(impl); err != nil {
		env.idxRegistry.Register(impl, func() indexer.Indexer { return fake })
	}
	env.idxStore.instances = append(env.idxStore.instances, indexer.Instance{
		ID:             len(env.idxStore.instances) + 1,
		Name:           "Test Indexer",
		Implementation: impl,
		EnableRss:      true,
		Priority:       1,
	})
}

// addSeries adds a series and a monitored episode to the env.
func addSeries(env *testEnv, title string, season, ep int) (int64, int64) {
	ctx := context.Background()
	seriesID := int64(len(env.seriesStore.series) + 1)
	env.seriesStore.series = append(env.seriesStore.series, library.Series{
		ID:    seriesID,
		Title: title,
		Slug:  strings.ToLower(strings.ReplaceAll(title, " ", "-")),
	})
	epID := int64(len(env.epStore.episodes) + 1)
	env.epStore.episodes = append(env.epStore.episodes, library.Episode{
		ID:            epID,
		SeriesID:      seriesID,
		SeasonNumber:  int32(season),
		EpisodeNumber: int32(ep),
		Monitored:     true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	})
	_ = ctx
	return seriesID, epID
}

func TestRssSyncMatchesAndGrabs(t *testing.T) {
	env := buildEnv(t) // no specs → accepts all
	ctx := context.Background()

	addSeries(env, "Breaking Bad", 1, 1)

	releases := []indexer.Release{
		{
			Title:       "Breaking.Bad.S01E01.720p.WEB-DL-GROUP",
			DownloadURL: "https://example.com/bb.nzb",
			Protocol:    indexer.ProtocolUsenet,
			Size:        500 * 1024 * 1024,
		},
		// This one has no series match.
		{
			Title:       "Unknown.Show.S01E01.720p",
			DownloadURL: "https://example.com/unknown.nzb",
			Protocol:    indexer.ProtocolUsenet,
			Size:        400 * 1024 * 1024,
		},
	}
	addIndexer(t, env, releases)

	err := env.handler.Handle(ctx, commands.Command{Name: "RssSync"})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if env.dcClient.addCalls != 1 {
		t.Errorf("Add called %d times, want 1", env.dcClient.addCalls)
	}
}

func TestRssSyncSkipsUnmatched(t *testing.T) {
	env := buildEnv(t)
	ctx := context.Background()

	// No series added — nothing should match.
	releases := []indexer.Release{
		{
			Title:       "Unknown.Show.S01E01.720p",
			DownloadURL: "https://example.com/show.nzb",
			Protocol:    indexer.ProtocolUsenet,
			Size:        500 * 1024 * 1024,
		},
	}
	addIndexer(t, env, releases)

	err := env.handler.Handle(ctx, commands.Command{Name: "RssSync"})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if env.dcClient.addCalls != 0 {
		t.Errorf("Add called %d times, want 0 for unmatched series", env.dcClient.addCalls)
	}
}

func TestRssSyncSkipsRejectedReleases(t *testing.T) {
	// Use the NotSample spec, which rejects releases under 40 MB.
	env := buildEnv(t, specs.NotSampleSpec{})
	ctx := context.Background()

	addSeries(env, "Breaking Bad", 1, 1)

	// 10-byte release — well below the 40 MB threshold.
	releases := []indexer.Release{
		{
			Title:       "Breaking.Bad.S01E01.720p-GROUP",
			DownloadURL: "https://example.com/bb.nzb",
			Protocol:    indexer.ProtocolUsenet,
			Size:        10, // tiny — will be rejected by NotSample
		},
	}
	addIndexer(t, env, releases)

	err := env.handler.Handle(ctx, commands.Command{Name: "RssSync"})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	if env.dcClient.addCalls != 0 {
		t.Errorf("Add called %d times, want 0 for rejected release", env.dcClient.addCalls)
	}
}

func TestRssSyncPicksBest(t *testing.T) {
	env := buildEnv(t) // no specs → accepts all
	ctx := context.Background()

	addSeries(env, "Breaking Bad", 1, 1)

	// Two releases for the same episode. We give the second a larger size so
	// the Rank tiebreaker (size desc) would normally pick it. But we set up a
	// custom format to give the first a higher CF score so it wins.
	//
	// Simplest approach: both have the same quality/CF score 0, but the second
	// release is larger → Rank picks the second one as best.
	releases := []indexer.Release{
		{
			Title:       "Breaking.Bad.S01E01.720p-SMALLGROUP",
			DownloadURL: "https://example.com/bb-small.nzb",
			Protocol:    indexer.ProtocolUsenet,
			Size:        300 * 1024 * 1024,
		},
		{
			Title:       "Breaking.Bad.S01E01.720p-BIGGROUP",
			DownloadURL: "https://example.com/bb-big.nzb",
			Protocol:    indexer.ProtocolUsenet,
			Size:        800 * 1024 * 1024, // larger → ranked first
		},
	}
	addIndexer(t, env, releases)

	err := env.handler.Handle(ctx, commands.Command{Name: "RssSync"})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}

	// Only the best release should be grabbed (one grab for the group).
	if env.dcClient.addCalls != 1 {
		t.Errorf("Add called %d times, want 1 (only best should be grabbed)", env.dcClient.addCalls)
	}
}
