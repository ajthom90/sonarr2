package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/parser"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// RepackSpec enforces that a repack must come from the same release group as
// the existing file. This prevents grabbing a repack from a different group
// when the original release group issued the repack for their own quality
// reasons.
//
// If the release is not a repack, or if there is no existing file, the spec
// always accepts.
type RepackSpec struct{}

// Name implements decisionengine.Spec.
func (s RepackSpec) Name() string { return "Repack" }

// Evaluate implements decisionengine.Spec.
func (s RepackSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	if remote.Quality.Modifier != parser.ModifierRepack {
		return decisionengine.Accept, nil
	}
	// No existing file — a repack for a first grab is fine.
	if remote.ExistingFileQualityID == 0 {
		return decisionengine.Accept, nil
	}
	if remote.ParsedInfo.ReleaseGroup == remote.ExistingFileReleaseGroup {
		return decisionengine.Accept, nil
	}
	return decisionengine.Reject, []decisionengine.Rejection{
		{
			Type: decisionengine.Permanent,
			Reason: fmt.Sprintf(
				"repack release group %q does not match existing file release group %q",
				remote.ParsedInfo.ReleaseGroup,
				remote.ExistingFileReleaseGroup,
			),
			Spec: s.Name(),
		},
	}
}
