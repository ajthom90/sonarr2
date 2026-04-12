package parser

import "testing"

func TestParseLanguageReturnsEnglishByDefault(t *testing.T) {
	langs := ParseLanguage("Some.Show.S01E01.720p.HDTV-GROUP")
	if len(langs) != 1 || langs[0] != LanguageEnglish {
		t.Errorf("ParseLanguage() = %v, want [english]", langs)
	}
}
