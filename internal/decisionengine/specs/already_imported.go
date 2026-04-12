package specs

import (
	"context"

	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// AlreadyImportedSpec is a stub that always accepts. The full implementation
// requires a history store (added in M8) to check whether the exact release
// has already been imported.
type AlreadyImportedSpec struct{}

// Name implements decisionengine.Spec.
func (s AlreadyImportedSpec) Name() string { return "AlreadyImported" }

// Evaluate implements decisionengine.Spec.
func (s AlreadyImportedSpec) Evaluate(_ context.Context, _ decisionengine.RemoteEpisode, _ profiles.QualityProfile) (decisionengine.Decision, []decisionengine.Rejection) {
	return decisionengine.Accept, nil
}
