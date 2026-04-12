package specs_test

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/decisionengine/specs"
	"github.com/ajthom90/sonarr2/internal/parser"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// ---- helpers ----------------------------------------------------------------

func mustAccept(t *testing.T, label string, s decisionengine.Spec, r decisionengine.RemoteEpisode, p profiles.QualityProfile) {
	t.Helper()
	d, rejs := s.Evaluate(context.Background(), r, p)
	if d != decisionengine.Accept {
		t.Errorf("%s: expected Accept, got Reject: %v", label, rejs)
	}
}

func mustReject(t *testing.T, label string, s decisionengine.Spec, r decisionengine.RemoteEpisode, p profiles.QualityProfile) {
	t.Helper()
	d, rejs := s.Evaluate(context.Background(), r, p)
	if d != decisionengine.Reject {
		t.Errorf("%s: expected Reject, got Accept", label)
	}
	if len(rejs) == 0 {
		t.Errorf("%s: expected at least one rejection reason", label)
	}
}

// ---- QualityAllowed ---------------------------------------------------------

func TestQualityAllowedAcceptsAllowed(t *testing.T) {
	s := specs.QualityAllowedSpec{}
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 5, Allowed: true},
		},
	}
	r := decisionengine.RemoteEpisode{QualityID: 5}
	mustAccept(t, "allowed quality", s, r, p)
}

func TestQualityAllowedRejectsDisallowed(t *testing.T) {
	s := specs.QualityAllowedSpec{}
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 5, Allowed: false},
		},
	}
	r := decisionengine.RemoteEpisode{QualityID: 5}
	mustReject(t, "disallowed quality", s, r, p)
}

func TestQualityAllowedRejectsUnknownQuality(t *testing.T) {
	s := specs.QualityAllowedSpec{}
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 5, Allowed: true},
		},
	}
	r := decisionengine.RemoteEpisode{QualityID: 99} // not in profile
	mustReject(t, "quality not in profile", s, r, p)
}

// ---- CustomFormatScore ------------------------------------------------------

func TestCustomFormatScoreAcceptsAboveMin(t *testing.T) {
	s := specs.CustomFormatScoreSpec{}
	p := profiles.QualityProfile{MinFormatScore: 50}
	r := decisionengine.RemoteEpisode{CFScore: 75}
	mustAccept(t, "score above min", s, r, p)
}

func TestCustomFormatScoreAcceptsAtMin(t *testing.T) {
	s := specs.CustomFormatScoreSpec{}
	p := profiles.QualityProfile{MinFormatScore: 50}
	r := decisionengine.RemoteEpisode{CFScore: 50}
	mustAccept(t, "score equal to min", s, r, p)
}

func TestCustomFormatScoreRejectsBelowMin(t *testing.T) {
	s := specs.CustomFormatScoreSpec{}
	p := profiles.QualityProfile{MinFormatScore: 50}
	r := decisionengine.RemoteEpisode{CFScore: 25}
	mustReject(t, "score below min", s, r, p)
}

// ---- UpgradeAllowed ---------------------------------------------------------

func TestUpgradeAllowedAcceptsFirstGrab(t *testing.T) {
	s := specs.UpgradeAllowedSpec{}
	p := profiles.QualityProfile{UpgradeAllowed: false}
	r := decisionengine.RemoteEpisode{ExistingFileQualityID: 0} // no existing file
	mustAccept(t, "no existing file, upgrades disabled", s, r, p)
}

func TestUpgradeAllowedAcceptsWhenEnabled(t *testing.T) {
	s := specs.UpgradeAllowedSpec{}
	p := profiles.QualityProfile{UpgradeAllowed: true}
	r := decisionengine.RemoteEpisode{ExistingFileQualityID: 5}
	mustAccept(t, "upgrade allowed by profile", s, r, p)
}

func TestUpgradeAllowedRejectsWhenDisabled(t *testing.T) {
	s := specs.UpgradeAllowedSpec{}
	p := profiles.QualityProfile{UpgradeAllowed: false}
	r := decisionengine.RemoteEpisode{ExistingFileQualityID: 5} // existing file present
	mustReject(t, "upgrade not allowed by profile", s, r, p)
}

// ---- Upgradable -------------------------------------------------------------

func TestUpgradableAcceptsFirstGrab(t *testing.T) {
	s := specs.UpgradableSpec{}
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 1, Allowed: true},
			{QualityID: 2, Allowed: true},
		},
	}
	r := decisionengine.RemoteEpisode{QualityID: 2, ExistingFileQualityID: 0}
	mustAccept(t, "no existing file", s, r, p)
}

func TestUpgradableAcceptsBetterQuality(t *testing.T) {
	s := specs.UpgradableSpec{}
	// Profile items: quality 1 (index 0) is better than quality 2 (index 1)
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 1, Allowed: true}, // index 0 = highest preference
			{QualityID: 2, Allowed: true}, // index 1
		},
	}
	// Existing is quality 2 (index 1), release is quality 1 (index 0) — this is an upgrade
	r := decisionengine.RemoteEpisode{QualityID: 1, ExistingFileQualityID: 2}
	mustAccept(t, "release is better quality than existing", s, r, p)
}

func TestUpgradableRejectsLowerQuality(t *testing.T) {
	s := specs.UpgradableSpec{}
	// Profile: quality 1 at index 0 = best, quality 2 at index 1
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 1, Allowed: true},
			{QualityID: 2, Allowed: true},
		},
	}
	// Existing is quality 1 (best), release is quality 2 (worse) — not an upgrade
	r := decisionengine.RemoteEpisode{QualityID: 2, ExistingFileQualityID: 1}
	mustReject(t, "release is worse quality than existing", s, r, p)
}

func TestUpgradableRejectsSameQuality(t *testing.T) {
	s := specs.UpgradableSpec{}
	p := profiles.QualityProfile{
		Items: []profiles.QualityProfileItem{
			{QualityID: 5, Allowed: true},
		},
	}
	r := decisionengine.RemoteEpisode{QualityID: 5, ExistingFileQualityID: 5}
	mustReject(t, "same quality as existing", s, r, p)
}

// ---- AcceptableSize ---------------------------------------------------------

func defs(id int, minMB, maxMB float64) []profiles.QualityDefinition {
	return []profiles.QualityDefinition{
		{ID: id, Name: "TestQuality", MinSize: minMB, MaxSize: maxMB},
	}
}

func TestAcceptableSizeAcceptsUnknownSize(t *testing.T) {
	s := specs.AcceptableSizeSpec{QualityDefs: defs(1, 10, 100)}
	r := decisionengine.RemoteEpisode{QualityID: 1}
	r.Release.Size = 0
	mustAccept(t, "unknown size", s, r, profiles.QualityProfile{})
}

func TestAcceptableSizeAcceptsWithinRange(t *testing.T) {
	s := specs.AcceptableSizeSpec{QualityDefs: defs(1, 10, 100)}
	r := decisionengine.RemoteEpisode{QualityID: 1}
	r.Release.Size = 50 * 1024 * 1024 // 50 MB
	mustAccept(t, "size within range", s, r, profiles.QualityProfile{})
}

func TestAcceptableSizeRejectsTooSmall(t *testing.T) {
	s := specs.AcceptableSizeSpec{QualityDefs: defs(1, 10, 100)}
	r := decisionengine.RemoteEpisode{QualityID: 1}
	r.Release.Size = 5 * 1024 * 1024 // 5 MB — below 10 MB min
	mustReject(t, "size too small", s, r, profiles.QualityProfile{})
}

func TestAcceptableSizeRejectsTooLarge(t *testing.T) {
	s := specs.AcceptableSizeSpec{QualityDefs: defs(1, 10, 100)}
	r := decisionengine.RemoteEpisode{QualityID: 1}
	r.Release.Size = 200 * 1024 * 1024 // 200 MB — above 100 MB max
	mustReject(t, "size too large", s, r, profiles.QualityProfile{})
}

func TestAcceptableSizeAcceptsUnlimitedMax(t *testing.T) {
	s := specs.AcceptableSizeSpec{QualityDefs: defs(1, 10, 0)} // MaxSize=0 means unlimited
	r := decisionengine.RemoteEpisode{QualityID: 1}
	r.Release.Size = 10000 * 1024 * 1024
	mustAccept(t, "unlimited max size", s, r, profiles.QualityProfile{})
}

// ---- NotSample --------------------------------------------------------------

func TestNotSampleAcceptsLargeRelease(t *testing.T) {
	s := specs.NotSampleSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Release.Size = 500 * 1024 * 1024 // 500 MB
	mustAccept(t, "large release", s, r, profiles.QualityProfile{})
}

func TestNotSampleAcceptsUnknownSize(t *testing.T) {
	s := specs.NotSampleSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Release.Size = 0
	mustAccept(t, "unknown size", s, r, profiles.QualityProfile{})
}

func TestNotSampleRejectsTiny(t *testing.T) {
	s := specs.NotSampleSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Release.Size = 10 * 1024 * 1024 // 10 MB — below 40 MB threshold
	mustReject(t, "tiny sample-sized release", s, r, profiles.QualityProfile{})
}

func TestNotSampleRejectsExactlyBeforeThreshold(t *testing.T) {
	s := specs.NotSampleSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Release.Size = 40*1024*1024 - 1 // just under 40 MB
	mustReject(t, "just under sample threshold", s, r, profiles.QualityProfile{})
}

func TestNotSampleAcceptsExactlyAtThreshold(t *testing.T) {
	s := specs.NotSampleSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Release.Size = 40 * 1024 * 1024 // exactly 40 MB is not a sample
	mustAccept(t, "exactly at threshold", s, r, profiles.QualityProfile{})
}

// ---- Repack -----------------------------------------------------------------

func TestRepackAcceptsNonRepack(t *testing.T) {
	s := specs.RepackSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Quality.Modifier = parser.ModifierNone
	mustAccept(t, "non-repack release", s, r, profiles.QualityProfile{})
}

func TestRepackAcceptsFirstGrabRepack(t *testing.T) {
	s := specs.RepackSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Quality.Modifier = parser.ModifierRepack
	r.ExistingFileQualityID = 0 // no existing file
	mustAccept(t, "repack with no existing file", s, r, profiles.QualityProfile{})
}

func TestRepackAcceptsMatchingGroup(t *testing.T) {
	s := specs.RepackSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Quality.Modifier = parser.ModifierRepack
	r.ExistingFileQualityID = 5
	r.ParsedInfo.ReleaseGroup = "GroupA"
	r.ExistingFileReleaseGroup = "GroupA"
	mustAccept(t, "repack from same group", s, r, profiles.QualityProfile{})
}

func TestRepackRejectsNonMatchingGroup(t *testing.T) {
	s := specs.RepackSpec{}
	r := decisionengine.RemoteEpisode{}
	r.Quality.Modifier = parser.ModifierRepack
	r.ExistingFileQualityID = 5
	r.ParsedInfo.ReleaseGroup = "GroupB"
	r.ExistingFileReleaseGroup = "GroupA"
	mustReject(t, "repack from different group", s, r, profiles.QualityProfile{})
}

// ---- AlreadyImported --------------------------------------------------------

func TestAlreadyImportedAlwaysAccepts(t *testing.T) {
	s := specs.AlreadyImportedSpec{}
	r := decisionengine.RemoteEpisode{}
	mustAccept(t, "stub always accepts", s, r, profiles.QualityProfile{})
}

// ---- Full pipeline test -----------------------------------------------------

func TestEngineEvaluateFullPipeline(t *testing.T) {
	qualityDefs := []profiles.QualityDefinition{
		{ID: 10, Name: "WEBDL-1080p", Source: "webdl", Resolution: "1080p", MinSize: 3, MaxSize: 400},
	}

	allSpecs := []decisionengine.Spec{
		specs.QualityAllowedSpec{},
		specs.CustomFormatScoreSpec{},
		specs.UpgradeAllowedSpec{},
		specs.UpgradableSpec{},
		specs.AcceptableSizeSpec{QualityDefs: qualityDefs},
		specs.NotSampleSpec{},
		specs.RepackSpec{},
		specs.AlreadyImportedSpec{},
	}
	eng := decisionengine.New(allSpecs...)

	p := profiles.QualityProfile{
		UpgradeAllowed: true,
		MinFormatScore: 0,
		Items: []profiles.QualityProfileItem{
			{QualityID: 10, Allowed: true},
		},
	}

	remote := decisionengine.RemoteEpisode{
		QualityID:             10,
		CFScore:               0, // meets MinFormatScore of 0
		ExistingFileQualityID: 0, // first grab — no existing file
	}
	remote.Release.Size = 200 * 1024 * 1024 // 200 MB — well within 3-400 MB for 1080p
	remote.Quality.Modifier = parser.ModifierNone

	d, rejections := eng.Evaluate(context.Background(), remote, p)
	if d != decisionengine.Accept {
		t.Fatalf("full pipeline: expected Accept, got Reject: %v", rejections)
	}
}
