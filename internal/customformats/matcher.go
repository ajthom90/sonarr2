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
//
// Specification identifiers match Sonarr byte-for-byte
// (src/NzbDrone.Core/CustomFormats/Specifications/).
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

	case "LanguageSpecification":
		// Value is a language ID (per Sonarr's Language enum) or name. We
		// support name-based match since sonarr2 doesn't track language IDs
		// in ParsedEpisodeInfo yet — caller can pre-populate info.Languages
		// and the spec matches if any language in the slice equals the value.
		for _, l := range info.Languages {
			if strings.EqualFold(l, spec.Value) {
				return true
			}
		}
		return false

	case "IndexerFlagSpecification":
		// IndexerFlags is a bitfield in Sonarr; here we match a flag name
		// against the parsed info's IndexerFlag slice (e.g. "Freeleech").
		for _, f := range info.IndexerFlags {
			if strings.EqualFold(f, spec.Value) {
				return true
			}
		}
		return false

	case "SizeSpecification":
		// Value is "minGB-maxGB" (e.g. "2-10") in Sonarr. Parse and compare
		// against info.Size (in bytes).
		lo, hi := parseSizeRange(spec.Value)
		if info.Size == 0 {
			return false
		}
		return info.Size >= lo && (hi == 0 || info.Size <= hi)

	case "ReleaseTypeSpecification":
		// Matches release type: single|season|multi (derived from episode
		// range). Sonarr surfaces this as an enum; we compare string.
		return strings.EqualFold(info.ReleaseType, spec.Value)

	default:
		// Unknown implementation — treat as matching so unknown specs don't
		// break existing formats when new implementations are added later.
		return true
	}
}

// parseSizeRange parses a size range string "loGB-hiGB" into bytes. Returns
// (lo=0, hi=0) on parse error so the spec fails safely.
func parseSizeRange(s string) (int64, int64) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	lo := parseGB(strings.TrimSpace(parts[0]))
	hi := parseGB(strings.TrimSpace(parts[1]))
	return lo, hi
}

func parseGB(s string) int64 {
	// Accept integer or float-as-string; return bytes.
	f := float64(0)
	var sign int64 = 1
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = s[1:]
	}
	beforeDot := true
	divisor := float64(1)
	for _, c := range s {
		if c == '.' && beforeDot {
			beforeDot = false
			continue
		}
		if c < '0' || c > '9' {
			return 0
		}
		if beforeDot {
			f = f*10 + float64(c-'0')
		} else {
			divisor *= 10
			f = f + float64(c-'0')/divisor
		}
	}
	return int64(f * (1 << 30)) * sign
}
