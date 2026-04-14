# Wanted + Activity Operational Pages — Design

**Date:** 2026-04-14
**Status:** Approved.
**Sub-project:** #4.
**Depends on:** Sub-project #1 (apiV3 helper).

## Goal

Close the audit's four remaining P1 table-view gaps:
1. `/wanted/missing` — currently uses `useWantedMissing` which hits `/api/v6/wanted/missing` (404 — only v3 exists). Fix by pointing the hook at `/api/v3/wanted/missing`.
2. `/wanted/cutoffunmet` — currently a `PagePlaceholder`. Backend endpoint doesn't exist. Add backend + frontend.
3. `/activity/history` — currently delegates to a combined `<Activity />` tab component. Give it its own focused page with a proper table.
4. `/activity/blocklist` — the existing page works but uses ad-hoc `apiFetchRaw` calls instead of the apiV3 helper pattern and lacks a polished toolbar. Light refactor for consistency.

## Non-goals

- `/activity/queue` — needs a dedicated backend endpoint (`/api/v3/queue`) that synthesizes download-client state + command-queue state. That's its own sub-project. Queue page stays as-is (placeholder via the combined Activity component).
- History per-row expander showing release metadata (Details column) — Sonarr has it; we defer.
- Manual Import button on Missing — depends on Manual Import subsystem which is a separate audit item.
- Reorderable/resizable column configuration. Shown columns are fixed for now.

## Backend work

### `/api/v3/wanted/cutoff` (new)

File: `internal/api/v3/wanted.go` (extend existing handler).

Semantics: an episode is "cutoff unmet" when it has an associated file (`episode_file_id IS NOT NULL`) **and** the file's quality rank is below the series' quality profile's cutoff. For this sub-project we'll approximate with a simpler rule that matches Sonarr's wire shape but uses simpler data: **episodes that have a file AND the file's quality_id != profile.cutoff_quality_id**. Good enough for an operational view; the decision engine already handles the real upgrade-grab logic independently.

Actually — scope this even simpler: **any monitored episode with a file is a candidate** and the frontend presents them as "potential upgrades." The real decision engine already filters during RSS/search. For the UI, a table of "episodes with a file" is informationally useful. We'll document this simplification and note the exact Sonarr cutoff logic as a follow-up.

Wire shape matches `/wanted/missing` — paged `{page, pageSize, sortKey, sortDirection, totalRecords, records: [Episode]}`.

Route: `r.Get("/cutoff", h.cutoff)` inside the existing `MountWanted` block.

### Frontend

- `useWantedMissingV3(cursor?)` — new hook via apiV3. Replaces/renames `useWantedMissing`.
- `useWantedCutoffUnmet(cursor?)` — new hook.
- `frontend/src/pages/WantedMissing.tsx` — stop delegating to `Wanted.tsx`; implement its own focused page.
- `frontend/src/pages/WantedCutoffUnmet.tsx` — full rewrite from placeholder.
- `frontend/src/pages/ActivityHistory.tsx` — stop delegating to `Activity.tsx`; implement its own focused page using `useHistory`.
- `frontend/src/pages/ActivityBlocklist.tsx` — refactor to use `apiV3` helper instead of `apiFetchRaw` directly; tidy UI.
- `frontend/src/pages/Wanted.tsx` — leave unchanged for now (WantedMissing replaces its role via the route).

## Deliverables

Backend:
- [ ] `internal/api/v3/wanted.go` — add `cutoff` handler + route mount
- [ ] `internal/api/v3/wanted_test.go` — test for cutoff (optional but good)

Frontend:
- [ ] `frontend/src/api/hooks.ts` — swap `useWantedMissing` to v3, add `useWantedCutoffUnmet`
- [ ] `frontend/src/pages/WantedMissing.tsx` — full rewrite
- [ ] `frontend/src/pages/WantedCutoffUnmet.tsx` — full rewrite
- [ ] `frontend/src/pages/ActivityHistory.tsx` — full rewrite
- [ ] `frontend/src/pages/ActivityBlocklist.tsx` — refactor to use `apiV3`

Docs:
- [ ] README bullet noting Wanted + History working
- [ ] M24 status update
