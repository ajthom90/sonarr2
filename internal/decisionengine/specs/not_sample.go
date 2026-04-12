package specs

import (
	"context"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

const sampleThreshold = 40 * 1024 * 1024 // 40 MB in bytes

// NotSampleSpec rejects releases that are suspiciously small — a strong signal
// that the file is a sample or trailer rather than a full episode.
// Releases with size 0 (unknown) are accepted.
type NotSampleSpec struct{}

// Name implements decisionengine.Spec.
func (s NotSampleSpec) Name() string { return "NotSample" }

// Evaluate implements decisionengine.Spec.
func (s NotSampleSpec) Evaluate(_ context.Context, remote decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	if remote.Release.Size > 0 && remote.Release.Size < sampleThreshold {
		return decisionengine.Reject, []decisionengine.Rejection{
			{
				Type:   decisionengine.Permanent,
				Reason: fmt.Sprintf("release size %d bytes is below 40 MB sample threshold", remote.Release.Size),
				Spec:   s.Name(),
			},
		}
	}
	return decisionengine.Accept, nil
}
