package customformats

import (
	"regexp"
	"strings"

	"github.com/ajthom90/sonarr2/internal/parser"
)

// Match reports whether the parsed release info satisfies every specification
// in the custom format (AND logic). Negated specifications invert their match
// result. Unknown implementation types are treated as matching (accepted by
// default) so that adding new spec types doesn't break existing formats.
func Match(info parser.ParsedEpisodeInfo, cf CustomFormat) bool {
	for _, spec := range cf.Specifications {
		matched := matchSpec(info, spec)
		if spec.Negate {
			matched = !matched
		}
		if !matched {
			return false
		}
	}
	return true
}

// matchSpec evaluates a single specification against the parsed info.
// Returns true if the spec matches (before negation is applied).
// Invalid regex patterns silently fail to match.
func matchSpec(info parser.ParsedEpisodeInfo, spec Specification) bool {
	switch spec.Implementation {
	case "ReleaseTitleSpecification":
		re, err := regexp.Compile(spec.Value)
		if err != nil {
			return false
		}
		return re.MatchString(info.ReleaseTitle)

	case "SourceSpecification":
		return string(info.Quality.Source) == strings.ToLower(spec.Value)

	case "ResolutionSpecification":
		return string(info.Quality.Resolution) == strings.ToLower(spec.Value)

	case "ReleaseGroupSpecification":
		re, err := regexp.Compile(spec.Value)
		if err != nil {
			return false
		}
		return re.MatchString(info.ReleaseGroup)

	default:
		// Unknown implementation — treat as matching so unknown specs don't
		// break existing formats when new implementations are added later.
		return true
	}
}
