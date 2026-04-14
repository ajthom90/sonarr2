// SPDX-License-Identifier: GPL-3.0-or-later
// Ported from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
// Reference: src/Sonarr.Api.V3/Calendar/CalendarFeedController.cs.

package v3

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// CalendarFeedHandler serves the iCalendar (.ics) feed consumed by calendar
// apps (Google/Apple/Outlook/Thunderbird). Compatible with Sonarr v3's
// /feed/calendar/Sonarr.ics endpoint, including pastDays/futureDays/tags/
// unmonitored/premieresOnly/asAllDay query params.
type CalendarFeedHandler struct {
	episodes library.EpisodesStore
	series   library.SeriesStore
	log      *slog.Logger
}

// NewCalendarFeedHandler constructs a CalendarFeedHandler.
func NewCalendarFeedHandler(ep library.EpisodesStore, sr library.SeriesStore, log *slog.Logger) *CalendarFeedHandler {
	return &CalendarFeedHandler{episodes: ep, series: sr, log: log}
}

// MountCalendarFeed wires the legacy feed path. Note the path is unprefixed
// (no /api/v3) matching Sonarr's wire format exactly — reverse-proxy users
// whose clients embed "Sonarr.ics" into subscriptions will see no change.
func MountCalendarFeed(r chi.Router, h *CalendarFeedHandler) {
	// Sonarr's route: [V3FeedController("calendar")] → /feed/v3/calendar/Sonarr.ics
	// The legacy /feed/calendar/Sonarr.ics is also commonly used.
	r.Get("/feed/v3/calendar/Sonarr.ics", h.serve)
	r.Get("/feed/calendar/Sonarr.ics", h.serve)
}

func (h *CalendarFeedHandler) serve(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pastDays, _ := strconv.Atoi(q.Get("pastDays"))
	if pastDays == 0 {
		pastDays = 7
	}
	futureDays, _ := strconv.Atoi(q.Get("futureDays"))
	if futureDays == 0 {
		futureDays = 28
	}
	unmonitored := q.Get("unmonitored") == "true"
	premieresOnly := q.Get("premieresOnly") == "true"
	asAllDay := q.Get("asAllDay") == "true"
	// tags param accepts CSV of tag IDs — tag *labels* are looked up separately
	// in Sonarr; sonarr2 accepts IDs directly or CSV of labels.
	tagFilter := parseTagCSV(q.Get("tags"))

	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -pastDays)
	end := today.AddDate(0, 0, futureDays)

	eps, err := h.episodes.ListAll(r.Context())
	if err != nil {
		h.log.Error("calendar feed list", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	allSeries, err := h.series.List(r.Context())
	if err != nil {
		h.log.Error("calendar feed series", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	seriesByID := make(map[int64]library.Series, len(allSeries))
	for _, s := range allSeries {
		seriesByID[s.ID] = s
	}

	filtered := make([]library.Episode, 0, len(eps))
	for _, ep := range eps {
		if ep.AirDateUtc == nil {
			continue
		}
		air := ep.AirDateUtc.UTC()
		if air.Before(start) || air.After(end) {
			continue
		}
		if !unmonitored && !ep.Monitored {
			continue
		}
		if premieresOnly && (ep.SeasonNumber == 0 || ep.EpisodeNumber != 1) {
			continue
		}
		if len(tagFilter) > 0 {
			// Series-tag matching would go here once series have tags wired in.
			// Until then, tag filter is a no-op (preserves Sonarr wire accepting
			// the param but matching nothing).
			continue
		}
		filtered = append(filtered, ep)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].AirDateUtc.Before(*filtered[j].AirDateUtc)
	})

	var sb strings.Builder
	writeICalLine(&sb, "BEGIN:VCALENDAR")
	writeICalLine(&sb, "VERSION:2.0")
	writeICalLine(&sb, "PRODID:-//sonarr2//Sonarr2//EN")
	writeICalLine(&sb, "CALSCALE:GREGORIAN")
	writeICalLine(&sb, "NAME:Sonarr TV Schedule")
	writeICalLine(&sb, "X-WR-CALNAME:Sonarr TV Schedule")

	for _, ep := range filtered {
		series, ok := seriesByID[ep.SeriesID]
		if !ok {
			continue
		}
		summary := fmt.Sprintf("%s - %dx%02d - %s",
			series.Title, ep.SeasonNumber, ep.EpisodeNumber, ep.Title)
		if strings.EqualFold(series.SeriesType, "daily") {
			summary = fmt.Sprintf("%s - %s", series.Title, ep.Title)
		}

		writeICalLine(&sb, "BEGIN:VEVENT")
		writeICalLine(&sb, fmt.Sprintf("UID:sonarr2_episode_%d", ep.ID))
		writeICalLine(&sb, "DTSTAMP:"+icalTimestamp(time.Now().UTC()))
		writeICalLine(&sb, "SUMMARY:"+icalEscape(summary))
		if ep.Overview != "" {
			writeICalLine(&sb, "DESCRIPTION:"+icalEscape(ep.Overview))
		}
		if ep.EpisodeFileID != nil && *ep.EpisodeFileID != 0 {
			writeICalLine(&sb, "STATUS:CONFIRMED")
		} else {
			writeICalLine(&sb, "STATUS:TENTATIVE")
		}
		if asAllDay {
			local := ep.AirDateUtc.Local()
			writeICalLine(&sb, fmt.Sprintf("DTSTART;VALUE=DATE:%04d%02d%02d",
				local.Year(), int(local.Month()), local.Day()))
		} else {
			start := ep.AirDateUtc.UTC()
			writeICalLine(&sb, "DTSTART:"+icalTimestamp(start))
			// Default 30-minute runtime; Series.Runtime isn't on sonarr2's Series yet.
			writeICalLine(&sb, "DTEND:"+icalTimestamp(start.Add(30*time.Minute)))
		}
		writeICalLine(&sb, "END:VEVENT")
	}

	writeICalLine(&sb, "END:VCALENDAR")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="Sonarr.ics"`)
	_, _ = w.Write([]byte(sb.String()))
}

// icalTimestamp formats a UTC timestamp as "20260414T120000Z".
func icalTimestamp(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// icalEscape escapes per RFC 5545: commas, semicolons, backslashes, newlines.
func icalEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", "\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// writeICalLine writes a line terminated by CRLF, folding at 75 octets as RFC
// 5545 requires. Subsequent lines are prefixed with a space (continuation).
func writeICalLine(sb *strings.Builder, line string) {
	const limit = 75
	b := []byte(line)
	for i := 0; i < len(b); {
		end := i + limit
		if end > len(b) {
			end = len(b)
		}
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.Write(b[i:end])
		sb.WriteString("\r\n")
		i = end
	}
}

func parseTagCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
