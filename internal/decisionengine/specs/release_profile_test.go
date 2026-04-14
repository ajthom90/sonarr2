package specs_test

import (
	"context"
	"testing"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/decisionengine/specs"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/releaseprofile"
)

func TestReleaseProfileSpecAccepts(t *testing.T) {
	spec := specs.ReleaseProfileSpec{
		ProfilesFn: func(context.Context, int64, string) ([]releaseprofile.Profile, error) {
			return []releaseprofile.Profile{{
				Name: "Require WEB-DL", Enabled: true,
				Required: []string{"WEB-DL"}, Ignored: []string{},
			}}, nil
		},
	}
	good := decisionengine.RemoteEpisode{
		Release: decisionengine.Release{Title: "Show.S01E01.1080p.WEB-DL"},
	}
	d, _ := spec.Evaluate(context.Background(), good, profiles.QualityProfile{})
	if d != decisionengine.Accept {
		t.Error("expected Accept for matching release")
	}
}

func TestReleaseProfileSpecRejectsIgnored(t *testing.T) {
	spec := specs.ReleaseProfileSpec{
		ProfilesFn: func(context.Context, int64, string) ([]releaseprofile.Profile, error) {
			return []releaseprofile.Profile{{
				Name: "No CAM", Enabled: true, Ignored: []string{"CAM", "TS"},
			}}, nil
		},
	}
	bad := decisionengine.RemoteEpisode{
		Release: decisionengine.Release{Title: "Show.S01E01.CAM.x264"},
	}
	d, rej := spec.Evaluate(context.Background(), bad, profiles.QualityProfile{})
	if d != decisionengine.Reject {
		t.Errorf("expected Reject, got %v", d)
	}
	if len(rej) != 1 {
		t.Errorf("expected 1 rejection, got %d", len(rej))
	}
}

func TestReleaseProfileSpecDisabledProfileIgnored(t *testing.T) {
	spec := specs.ReleaseProfileSpec{
		ProfilesFn: func(context.Context, int64, string) ([]releaseprofile.Profile, error) {
			return []releaseprofile.Profile{{
				Name: "Disabled", Enabled: false, Ignored: []string{"CAM"},
			}}, nil
		},
	}
	// Even though title contains CAM, disabled profile shouldn't reject.
	rel := decisionengine.RemoteEpisode{Release: decisionengine.Release{Title: "Show.CAM"}}
	d, _ := spec.Evaluate(context.Background(), rel, profiles.QualityProfile{})
	if d != decisionengine.Accept {
		t.Error("disabled profile should not affect decision")
	}
}

func TestReleaseProfileSpecNilFn(t *testing.T) {
	spec := specs.ReleaseProfileSpec{ProfilesFn: nil}
	d, _ := spec.Evaluate(context.Background(), decisionengine.RemoteEpisode{}, profiles.QualityProfile{})
	if d != decisionengine.Accept {
		t.Error("nil ProfilesFn should Accept")
	}
}
