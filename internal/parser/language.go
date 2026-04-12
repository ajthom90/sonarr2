package parser

// Language identifies a media language.
type Language string

const (
	LanguageEnglish Language = "english"
	LanguageUnknown Language = "unknown"
)

// ParseLanguage extracts language information from a release title.
// M4 stub: returns English. Full multi-language detection in a later milestone.
func ParseLanguage(title string) []Language {
	return []Language{LanguageEnglish}
}
