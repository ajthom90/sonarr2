// Package autotag implements Sonarr's Auto Tagging feature: rule-based
// automatic application of tags to series based on series attributes.
//
// A Rule has a Name, a list of Tags to apply, an optional
// RemoveTagsAutomatically flag (removes the tags when the rule stops
// matching), and a list of Specifications. Each specification matches
// against a property of the series (Genre, Status, SeriesType, Root Folder,
// Year, Network, Original Language).
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/AutoTagging/).
package autotag

import (
	"context"
	"errors"
	"regexp"
	"strings"
)

// ErrNotFound is returned for unknown rule IDs.
var ErrNotFound = errors.New("autotag: not found")

// Specification is one filter condition. Implementation names match Sonarr
// byte-for-byte so config JSON imports cleanly.
type Specification struct {
	Name           string `json:"name"`
	Implementation string `json:"implementation"` // e.g. "GenreSpecification"
	Negate         bool   `json:"negate"`
	Required       bool   `json:"required"`
	Value          string `json:"value"`
}

// Rule is one auto-tagging rule.
type Rule struct {
	ID                      int             `json:"id"`
	Name                    string          `json:"name"`
	RemoveTagsAutomatically bool            `json:"removeTagsAutomatically"`
	Tags                    []int           `json:"tags"`
	Specifications          []Specification `json:"specifications"`
}

// SeriesAttr is the subset of a Series relevant to spec matching.
type SeriesAttr struct {
	Title           string
	Genres          []string
	Status          string
	SeriesType      string
	Network         string
	Year            int
	OriginalLang    string
	RootFolderPath  string
}

// Store persists auto-tag rules.
type Store interface {
	Create(ctx context.Context, r Rule) (Rule, error)
	GetByID(ctx context.Context, id int) (Rule, error)
	List(ctx context.Context) ([]Rule, error)
	Update(ctx context.Context, r Rule) error
	Delete(ctx context.Context, id int) error
}

// Matches reports whether attr satisfies every required spec in rule and at
// least one non-required spec (matching Sonarr's behavior where required=true
// are AND, non-required are OR).
func Matches(rule Rule, attr SeriesAttr) bool {
	if len(rule.Specifications) == 0 {
		return false
	}
	anyOptionalMatch := false
	haveOptional := false
	for _, s := range rule.Specifications {
		match := matchSpec(s, attr)
		if s.Negate {
			match = !match
		}
		if s.Required {
			if !match {
				return false
			}
			continue
		}
		haveOptional = true
		if match {
			anyOptionalMatch = true
		}
	}
	if !haveOptional {
		return true
	}
	return anyOptionalMatch
}

// ApplyTags returns the tags list with rule's Tags merged in (de-duplicated).
func ApplyTags(existing []int, rule Rule) []int {
	seen := make(map[int]struct{}, len(existing))
	for _, t := range existing {
		seen[t] = struct{}{}
	}
	out := append([]int{}, existing...)
	for _, t := range rule.Tags {
		if _, ok := seen[t]; ok {
			continue
		}
		out = append(out, t)
		seen[t] = struct{}{}
	}
	return out
}

// RemoveTags returns tags minus rule's Tags. Used when a rule no longer
// matches and RemoveTagsAutomatically is set.
func RemoveTags(existing []int, rule Rule) []int {
	drop := make(map[int]struct{}, len(rule.Tags))
	for _, t := range rule.Tags {
		drop[t] = struct{}{}
	}
	out := make([]int, 0, len(existing))
	for _, t := range existing {
		if _, ok := drop[t]; ok {
			continue
		}
		out = append(out, t)
	}
	return out
}

func matchSpec(s Specification, a SeriesAttr) bool {
	switch s.Implementation {
	case "GenreSpecification":
		for _, g := range a.Genres {
			if strings.EqualFold(g, s.Value) {
				return true
			}
		}
		return false
	case "SeriesStatusSpecification":
		return strings.EqualFold(a.Status, s.Value)
	case "SeriesTypeSpecification":
		return strings.EqualFold(a.SeriesType, s.Value)
	case "NetworkSpecification":
		re, err := regexp.Compile("(?i)" + s.Value)
		if err != nil {
			return strings.Contains(strings.ToLower(a.Network), strings.ToLower(s.Value))
		}
		return re.MatchString(a.Network)
	case "OriginalLanguageSpecification":
		return strings.EqualFold(a.OriginalLang, s.Value)
	case "YearSpecification":
		// Value is a single year or a range "2020-2024".
		if strings.Contains(s.Value, "-") {
			parts := strings.SplitN(s.Value, "-", 2)
			lo, hi := parseInt(parts[0]), parseInt(parts[1])
			return a.Year >= lo && a.Year <= hi
		}
		return a.Year == parseInt(s.Value)
	case "RootFolderSpecification":
		return strings.EqualFold(a.RootFolderPath, s.Value)
	default:
		return false
	}
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
