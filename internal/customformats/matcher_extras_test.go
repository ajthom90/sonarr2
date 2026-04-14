package customformats_test

import (
	"testing"

	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/parser"
)

func TestMatchLanguageSpec(t *testing.T) {
	cf := customformats.CustomFormat{
		Specifications: []customformats.Specification{
			{Implementation: "LanguageSpecification", Value: "English"},
		},
	}
	if !customformats.Match(parser.ParsedEpisodeInfo{Languages: []string{"English", "French"}}, cf) {
		t.Error("expected match when English is present")
	}
	if customformats.Match(parser.ParsedEpisodeInfo{Languages: []string{"French"}}, cf) {
		t.Error("expected no match when English absent")
	}
}

func TestMatchIndexerFlagSpec(t *testing.T) {
	cf := customformats.CustomFormat{
		Specifications: []customformats.Specification{
			{Implementation: "IndexerFlagSpecification", Value: "Freeleech"},
		},
	}
	if !customformats.Match(parser.ParsedEpisodeInfo{IndexerFlags: []string{"Freeleech"}}, cf) {
		t.Error("expected match")
	}
	if customformats.Match(parser.ParsedEpisodeInfo{IndexerFlags: []string{"Internal"}}, cf) {
		t.Error("expected no match")
	}
}

func TestMatchSizeSpec(t *testing.T) {
	cf := customformats.CustomFormat{
		Specifications: []customformats.Specification{
			{Implementation: "SizeSpecification", Value: "2-10"},
		},
	}
	if !customformats.Match(parser.ParsedEpisodeInfo{Size: 5 * (1 << 30)}, cf) {
		t.Error("expected match for 5GB in 2-10GB range")
	}
	if customformats.Match(parser.ParsedEpisodeInfo{Size: 1 * (1 << 30)}, cf) {
		t.Error("expected no match for 1GB below range")
	}
	if customformats.Match(parser.ParsedEpisodeInfo{Size: 15 * (1 << 30)}, cf) {
		t.Error("expected no match for 15GB above range")
	}
}

func TestMatchReleaseTypeSpec(t *testing.T) {
	cf := customformats.CustomFormat{
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseTypeSpecification", Value: "season"},
		},
	}
	if !customformats.Match(parser.ParsedEpisodeInfo{ReleaseType: "season"}, cf) {
		t.Error("expected match for season pack")
	}
	if customformats.Match(parser.ParsedEpisodeInfo{ReleaseType: "single"}, cf) {
		t.Error("expected no match for single episode")
	}
}
