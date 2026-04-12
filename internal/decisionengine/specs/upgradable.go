package specs

import (
	"context"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// UpgradableSpec rejects a release when the existing file's quality is at least
// as good as (or better than) the candidate release's quality, as determined
// by the order of qualities in the profile's Items list.
// Lower index in Items = higher preference. If the existing quality has a lower
// (or equal) index than the release, there is no upgrade to be had.
type UpgradableSpec struct{}

// Name implements decisionengine.Spec.
func (s UpgradableSpec) Name() string { return "Upgradable" }

// Evaluate implements decisionengine.Spec.
func (s UpgradableSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, profile profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	// No existing file — any release is a valid grab.
	if remote.ExistingFileQualityID == 0 {
		return decisionengine.Accept, nil
	}

	existingIndex := profileIndex(remote.ExistingFileQualityID, profile)
	releaseIndex := profileIndex(remote.QualityID, profile)

	// existingIndex <= releaseIndex means existing is same or higher preference
	// (remember: lower index = better). If it's already at least as good,
	// the candidate is not an upgrade.
	if existingIndex <= releaseIndex {
		return decisionengine.Reject, []decisionengine.Rejection{
			{
				Type:   decisionengine.Permanent,
				Reason: "existing file quality is equal or better than release quality",
				Spec:   s.Name(),
			},
		}
	}
	return decisionengine.Accept, nil
}

// profileIndex returns the index of qualityID in profile.Items.
// If not found, returns len(profile.Items) (worst rank).
func profileIndex(qualityID int, profile profiles.QualityProfile) int {
	for i, it := range profile.Items {
		if it.QualityID == qualityID {
			return i
		}
	}
	return len(profile.Items)
}
