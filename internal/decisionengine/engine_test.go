package decisionengine_test

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// ---- helpers ----------------------------------------------------------------

func profile(items ...profiles.QualityProfileItem) profiles.QualityProfile {
	return profiles.QualityProfile{Items: items}
}

func item(qualityID int, allowed bool) profiles.QualityProfileItem {
	return profiles.QualityProfileItem{QualityID: qualityID, Allowed: allowed}
}

func remote(qualityID int, cfScore int, size int64) decisionengine.RemoteEpisode {
	r := decisionengine.RemoteEpisode{}
	r.QualityID = qualityID
	r.CFScore = cfScore
	r.Release.Size = size
	return r
}

// ---- Task 4 basic engine tests ----------------------------------------------

func TestEngineNoSpecsAcceptsEverything(t *testing.T) {
	eng := decisionengine.New()
	d, rejections := eng.Evaluate(context.Background(), decisionengine.RemoteEpisode{}, profiles.QualityProfile{})
	if d != decisionengine.Accept {
		t.Fatalf("expected Accept, got %v", d)
	}
	if len(rejections) != 0 {
		t.Fatalf("expected no rejections, got %v", rejections)
	}
}

func TestEngineCollectsAllRejections(t *testing.T) {
	// An engine with two always-reject specs should surface both rejections.
	eng := decisionengine.New(alwaysReject("spec-a", "reason-a"), alwaysReject("spec-b", "reason-b"))
	d, rejections := eng.Evaluate(context.Background(), decisionengine.RemoteEpisode{}, profiles.QualityProfile{})
	if d != decisionengine.Reject {
		t.Fatalf("expected Reject, got %v", d)
	}
	if len(rejections) != 2 {
		t.Fatalf("expected 2 rejections, got %d: %v", len(rejections), rejections)
	}
}

// alwaysReject is a test-only Spec that always rejects.
type alwaysRejectSpec struct {
	name   string
	reason string
}

func alwaysReject(name, reason string) decisionengine.Spec {
	return &alwaysRejectSpec{name: name, reason: reason}
}

func (s *alwaysRejectSpec) Name() string { return s.name }

func (s *alwaysRejectSpec) Evaluate(_ context.Context, _ decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	return decisionengine.Reject, []decisionengine.Rejection{
		{Type: decisionengine.Permanent, Reason: s.reason, Spec: s.name},
	}
}

// ---- Task 6 ranking tests ---------------------------------------------------

// TestRankStableOrder verifies that releases that compare equal (same CF score,
// same quality, same size) preserve their original insertion order — confirming
// that Rank uses sort.SliceStable, not sort.Slice.
func TestRankStableOrder(t *testing.T) {
	eng := decisionengine.New()
	p := profile(item(1, true))
	// Mark each release with a unique protocol so we can distinguish them.
	r1 := remote(1, 10, 500*1024*1024)
	r1.Release.Protocol = "first"
	r2 := remote(1, 10, 500*1024*1024)
	r2.Release.Protocol = "second"
	r3 := remote(1, 10, 500*1024*1024)
	r3.Release.Protocol = "third"
	ranked := eng.Rank([]decisionengine.RemoteEpisode{r1, r2, r3}, p)
	wantOrder := []string{"first", "second", "third"}
	for i, want := range wantOrder {
		if ranked[i].Release.Protocol != want {
			t.Errorf("position %d: want %q, got %q (stable sort violated)", i, want, ranked[i].Release.Protocol)
		}
	}
}

func TestRankByCustomFormatScore(t *testing.T) {
	eng := decisionengine.New()
	p := profile(item(1, true), item(2, true), item(3, true))
	remotes := []decisionengine.RemoteEpisode{
		remote(1, 50, 1000),
		remote(1, 100, 1000),
		remote(1, 25, 1000),
	}
	ranked := eng.Rank(remotes, p)
	wantScores := []int{100, 50, 25}
	for i, want := range wantScores {
		if ranked[i].CFScore != want {
			t.Errorf("position %d: want CFScore %d, got %d", i, want, ranked[i].CFScore)
		}
	}
}

func TestRankByQualityWhenCFEqual(t *testing.T) {
	eng := decisionengine.New()
	// Profile items order: qualityID 3 > 2 > 1 (lower index = higher preference).
	p := profile(item(3, true), item(2, true), item(1, true))
	// All have same CF score but different quality IDs.
	remotes := []decisionengine.RemoteEpisode{
		remote(1, 10, 1000), // worst quality (index 2)
		remote(3, 10, 1000), // best quality (index 0)
		remote(2, 10, 1000), // middle quality (index 1)
	}
	ranked := eng.Rank(remotes, p)
	wantQualities := []int{3, 2, 1}
	for i, want := range wantQualities {
		if ranked[i].QualityID != want {
			t.Errorf("position %d: want QualityID %d, got %d", i, want, ranked[i].QualityID)
		}
	}
}

func TestRankTiebreaker(t *testing.T) {
	eng := decisionengine.New()
	p := profile(item(1, true))
	// All same CF score and quality — ranked by size descending.
	remotes := []decisionengine.RemoteEpisode{
		remote(1, 10, 500*1024*1024),
		remote(1, 10, 2000*1024*1024),
		remote(1, 10, 1000*1024*1024),
	}
	ranked := eng.Rank(remotes, p)
	wantSizes := []int64{2000 * 1024 * 1024, 1000 * 1024 * 1024, 500 * 1024 * 1024}
	for i, want := range wantSizes {
		if ranked[i].Release.Size != want {
			t.Errorf("position %d: want size %d, got %d", i, want, ranked[i].Release.Size)
		}
	}
}
