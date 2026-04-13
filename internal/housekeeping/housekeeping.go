// Package housekeeping provides periodic database cleanup operations.
package housekeeping

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// HistoryTrimmer deletes old history entries.
type HistoryTrimmer interface {
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// EpisodeFileLister lists and deletes episode files.
type EpisodeFileLister interface {
	ListForSeries(ctx context.Context, seriesID int64) ([]EpisodeFileInfo, error)
	Delete(ctx context.Context, id int64) error
}

// EpisodeFileInfo is the subset of episode file data needed.
type EpisodeFileInfo struct {
	ID           int64
	SeriesID     int64
	RelativePath string
}

// SeriesLister lists all series for iteration.
type SeriesLister interface {
	ListAll(ctx context.Context) ([]SeriesInfo, error)
}

// SeriesInfo is the subset of series data needed.
type SeriesInfo struct {
	ID   int64
	Path string
}

// StatsRecomputer recalculates statistics for a series.
type StatsRecomputer interface {
	Recompute(ctx context.Context, seriesID int64) error
}

// Vacuumer compacts the database.
type Vacuumer interface {
	Vacuum(ctx context.Context) error
}

// Options configures the housekeeping Runner.
type Options struct {
	History          HistoryTrimmer
	EpisodeFiles     EpisodeFileLister
	Series           SeriesLister
	Stats            StatsRecomputer
	DB               Vacuumer
	Log              *slog.Logger
	HistoryRetention time.Duration
}

// Runner performs all housekeeping operations.
type Runner struct {
	opts Options
}

// New creates a Runner.
func New(opts Options) *Runner {
	if opts.HistoryRetention <= 0 {
		opts.HistoryRetention = 90 * 24 * time.Hour
	}
	return &Runner{opts: opts}
}

// Run executes all housekeeping operations. Errors are logged but non-fatal.
func (r *Runner) Run(ctx context.Context) {
	r.trimHistory(ctx)
	r.cleanOrphanEpisodeFiles(ctx)
	r.recalculateStatistics(ctx)
	r.vacuumDatabase(ctx)
}

func (r *Runner) trimHistory(ctx context.Context) {
	cutoff := time.Now().Add(-r.opts.HistoryRetention)
	n, err := r.opts.History.DeleteBefore(ctx, cutoff)
	if err != nil {
		r.opts.Log.Error("housekeeping: trim history", slog.String("err", err.Error()))
		return
	}
	if n > 0 {
		r.opts.Log.Info("housekeeping: trimmed history", slog.Int64("deleted", n))
	}
}

func (r *Runner) cleanOrphanEpisodeFiles(ctx context.Context) {
	series, err := r.opts.Series.ListAll(ctx)
	if err != nil {
		r.opts.Log.Error("housekeeping: list series", slog.String("err", err.Error()))
		return
	}
	var deleted int
	for _, s := range series {
		files, err := r.opts.EpisodeFiles.ListForSeries(ctx, s.ID)
		if err != nil {
			r.opts.Log.Error("housekeeping: list episode files",
				slog.Int64("seriesID", s.ID), slog.String("err", err.Error()))
			continue
		}
		for _, f := range files {
			fullPath := filepath.Join(s.Path, f.RelativePath)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				if err := r.opts.EpisodeFiles.Delete(ctx, f.ID); err != nil {
					r.opts.Log.Error("housekeeping: delete orphan file",
						slog.Int64("id", f.ID), slog.String("err", err.Error()))
					continue
				}
				deleted++
			}
		}
	}
	if deleted > 0 {
		r.opts.Log.Info("housekeeping: cleaned orphan episode files", slog.Int("deleted", deleted))
	}
}

func (r *Runner) recalculateStatistics(ctx context.Context) {
	series, err := r.opts.Series.ListAll(ctx)
	if err != nil {
		r.opts.Log.Error("housekeeping: list series for stats", slog.String("err", err.Error()))
		return
	}
	for _, s := range series {
		if err := r.opts.Stats.Recompute(ctx, s.ID); err != nil {
			r.opts.Log.Error("housekeeping: recompute stats",
				slog.Int64("seriesID", s.ID), slog.String("err", err.Error()))
		}
	}
}

func (r *Runner) vacuumDatabase(ctx context.Context) {
	if err := r.opts.DB.Vacuum(ctx); err != nil {
		r.opts.Log.Error("housekeeping: vacuum", slog.String("err", err.Error()))
	}
}
