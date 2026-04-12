package specs

import (
	"context"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// UpgradeAllowedSpec rejects a release when an existing file is present but
// the profile does not permit upgrades.
type UpgradeAllowedSpec struct{}

// Name implements decisionengine.Spec.
func (s UpgradeAllowedSpec) Name() string { return "UpgradeAllowed" }

// Evaluate implements decisionengine.Spec.
func (s UpgradeAllowedSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, profile profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	// No existing file — upgrades are moot.
	if remote.ExistingFileQualityID == 0 {
		return decisionengine.Accept, nil
	}
	if profile.UpgradeAllowed {
		return decisionengine.Accept, nil
	}
	return decisionengine.Reject, []decisionengine.Rejection{
		{
			Type:   decisionengine.Permanent,
			Reason: "existing file and upgrading is not allowed by this profile",
			Spec:   s.Name(),
		},
	}
}
