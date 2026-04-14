package v3

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
	"github.com/ajthom90/sonarr2/internal/rootfolder"
)

// libraryImportConcurrencyCap bounds the number of concurrent TVDB
// SearchSeries calls issued by a single scan. 8 keeps us well under
// typical upstream rate limits while still parallelising the common
// "100-show first import" case.
const libraryImportConcurrencyCap = 8

// yearParensRe matches a trailing " (YYYY)" in a folder name so we can
// strip it before searching TVDB. TVDB's search indexes the bare title,
// and folder conventions like "Breaking Bad (2008)" otherwise confuse
// the search ranker.
var yearParensRe = regexp.MustCompile(`\s*\((\d{4})\)\s*$`)

// libraryImportMatch is the TVDB hit attached to a scanned folder, or
// null when no match was found (or preview-only skipped the lookup).
type libraryImportMatch struct {
	TvdbID   int64  `json:"tvdbId"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Overview string `json:"overview,omitempty"`
}

// libraryImportEntry is one row in the scan response — a subfolder under
// the root, optionally enriched with a TVDB match and an "already
// imported" flag so the UI can suppress the add-series affordance.
type libraryImportEntry struct {
	FolderName      string              `json:"folderName"`
	RelativePath    string              `json:"relativePath"`
	AbsolutePath    string              `json:"absolutePath"`
	TvdbMatch       *libraryImportMatch `json:"tvdbMatch"`
	AlreadyImported bool                `json:"alreadyImported"`
}

// libraryImportHandler serves GET /api/v3/libraryimport/scan. It walks
// one root folder one level deep, filters dotfolders and non-dirs, and
// enriches each remaining subfolder with a TVDB search result (unless
// previewOnly=true, in which case TVDB lookups are skipped entirely).
type libraryImportHandler struct {
	rf         rootfolder.Store
	series     library.SeriesStore
	hostConfig hostconfig.Store
	source     metadatasource.MetadataSource
	log        *slog.Logger
}

// MountLibraryImport registers GET /api/v3/libraryimport/scan on r.
func MountLibraryImport(
	r chi.Router,
	rf rootfolder.Store,
	series library.SeriesStore,
	hc hostconfig.Store,
	source metadatasource.MetadataSource,
) {
	h := &libraryImportHandler{
		rf:         rf,
		series:     series,
		hostConfig: hc,
		source:     source,
		log:        slog.Default(),
	}
	r.Get("/api/v3/libraryimport/scan", h.scan)
}

// scan walks the named root folder and returns one entry per direct
// subfolder. Dotfolders and files are skipped. The handler resolves each
// subfolder name against TVDB unless previewOnly=true, rate-limited by
// a buffered-channel semaphore of size libraryImportConcurrencyCap.
func (h *libraryImportHandler) scan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get("rootFolderId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "rootFolderId is required")
		return
	}
	previewOnly := r.URL.Query().Get("previewOnly") == "true"

	rf, err := h.rf.Get(ctx, id)
	if errors.Is(err, rootfolder.ErrNotFound) {
		writeError(w, http.StatusNotFound, "root folder not found")
		return
	}
	if err != nil {
		h.log.Error("libraryimport: rootfolder get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if _, err := os.Stat(rf.Path); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"message": "root folder " + rf.Path + " is not readable",
		})
		return
	}

	if !previewOnly {
		hc, err := h.hostConfig.Get(ctx)
		if err != nil && !errors.Is(err, hostconfig.ErrNotFound) {
			h.log.Error("libraryimport: hostconfig get", slog.String("err", err.Error()))
			writeError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if hc.TvdbApiKey == "" {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"message": "TVDB API key is not configured",
				"fixPath": "/settings/general",
			})
			return
		}
	}

	entries, err := os.ReadDir(rf.Path)
	if err != nil {
		h.log.Error("libraryimport: readdir", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "read dir: "+err.Error())
		return
	}

	// Build an in-memory set of imported series paths so the per-folder
	// "already imported?" check is O(1) instead of O(series) per folder.
	allSeries, err := h.series.List(ctx)
	if err != nil {
		h.log.Error("libraryimport: series list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "list series: "+err.Error())
		return
	}
	imported := make(map[string]struct{}, len(allSeries))
	for _, s := range allSeries {
		imported[s.Path] = struct{}{}
	}

	subfolders := make([]libraryImportEntry, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		abs := filepath.Join(rf.Path, e.Name())
		entry := libraryImportEntry{
			FolderName:   e.Name(),
			RelativePath: e.Name(),
			AbsolutePath: abs,
		}
		if _, yes := imported[abs]; yes {
			entry.AlreadyImported = true
		}
		subfolders = append(subfolders, entry)
	}

	if !previewOnly {
		h.resolveMatches(ctx, subfolders)
	}

	writeJSON(w, http.StatusOK, subfolders)
}

// resolveMatches issues a TVDB SearchSeries call for each non-already-
// imported entry, in parallel but capped at libraryImportConcurrencyCap.
// Per-folder errors are SOFT: the corresponding tvdbMatch stays nil and
// the scan as a whole still succeeds — one flaky lookup must not fail
// the entire library-import preview.
func (h *libraryImportHandler) resolveMatches(ctx context.Context, entries []libraryImportEntry) {
	sem := make(chan struct{}, libraryImportConcurrencyCap)
	var wg sync.WaitGroup
	for i := range entries {
		if entries[i].AlreadyImported {
			continue
		}
		i := i
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			term := strings.TrimSpace(yearParensRe.ReplaceAllString(entries[i].FolderName, ""))
			results, err := h.source.SearchSeries(ctx, term)
			if err != nil {
				h.log.Debug("libraryimport: tvdb search failed (soft)",
					slog.String("folder", entries[i].FolderName),
					slog.String("err", err.Error()))
				return
			}
			if len(results) == 0 {
				return
			}
			top := results[0]
			entries[i].TvdbMatch = &libraryImportMatch{
				TvdbID:   top.TvdbID,
				Title:    top.Title,
				Year:     top.Year,
				Overview: top.Overview,
			}
		}()
	}
	wg.Wait()
}
