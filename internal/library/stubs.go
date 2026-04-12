package library

// The following interfaces are placeholders filled in by later M2 tasks:
//   SeasonsStore       — Task 4 (seasons)
//   EpisodesStore      — Task 5 (episodes)
//   EpisodeFilesStore  — Task 6 (episode files)
//   SeriesStatsStore   — Task 7 (statistics)
//
// Leaving them empty here lets library.go's Library struct reference them
// so the package compiles during Tasks 3-6, and the corresponding task
// replaces the stub with the real interface definition.

// SeasonsStore is a placeholder; the real interface is defined in Task 4's
// seasons.go. Until then, Library.Seasons is always nil.
type SeasonsStore interface{}

// EpisodesStore is a placeholder; real interface in Task 5's episodes.go.
type EpisodesStore interface{}

// EpisodeFilesStore is a placeholder; real interface in Task 6's episodefiles.go.
type EpisodeFilesStore interface{}

// SeriesStatsStore is a placeholder; real interface in Task 7's stats.go.
type SeriesStatsStore interface{}
