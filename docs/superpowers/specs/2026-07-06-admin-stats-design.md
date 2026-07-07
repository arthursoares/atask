# Admin Statistics & Account Overview — Design

**Date:** 2026-07-06
**Status:** Approved (brainstorming)
**Depends on:** Multi-user Phase 1 (PocketBase auth, admin UI Task 14, orphan-check Task 22)

## Goal

Add statistics and a per-account overview to the web admin UI so an operator can
see, at a glance: how much data each account holds, who is active vs dormant, a
running creation/activity log, and growth trends over time. All from data that
already exists (`domain_events` + the domain entity tables) — no new tracking.

## Approach

Server-rendered, consistent with the existing admin UI (Go `html/template`,
inline CSS, no JS, no new dependencies). A new raw-SQL stats module mirrors the
established `internal/store/orphan_check.go` pattern. Growth is drawn as
server-computed inline-SVG bars. Rejected alternatives: a JSON API + JS charting
lib (big divergence from the dependency-free admin UI), and a live SSE feed
(unneeded complexity for an overview).

## Architecture

### Data layer — `internal/store/stats.go` (new)

Raw SQL over `atask.db`, typed functions returning plain structs/maps. Each is
independently callable and testable. All entity counts filter `deleted = 0`
where applicable and are scoped by `user_id` for the per-user variants.

- `SystemStats(ctx, db) (SystemStats, error)` — system-wide totals: tasks,
  projects, areas, tags, and total `domain_events`.
- `PerUserCounts(ctx, db) (map[string]EntityCounts, error)` — tasks/projects/
  areas/tags grouped by `user_id`.
- `UserActivity(ctx, db) (map[string]Activity, error)` — per `user_id`:
  `LastActiveAt` (`max(timestamp)`) and `EventCount` from `domain_events`.
- `RecentEvents(ctx, db, limit int) ([]Event, error)` — most recent
  `domain_events` (type, entity_type, entity_id, actor_id, user_id, timestamp),
  newest first.
- `RecentEventsByUser(ctx, db, userID string, limit int) ([]Event, error)` —
  same, filtered to one user.
- `CreationGrowth(ctx, db, days int) ([]DayBucket, error)` — `domain_events`
  counted per `date(timestamp)` for the last `days` (default 30), for the trend
  bars. Zero-fills empty days in Go so the chart has a continuous axis.

Types: `SystemStats{Tasks, Projects, Areas, Tags, Events int}`;
`EntityCounts{Tasks, Projects, Areas, Tags int}`;
`Activity{LastActiveAt *time.Time, EventCount int}`;
`Event{Type, EntityType, EntityID, ActorID, UserID string, Timestamp time.Time}`;
`DayBucket{Date string, Count int}`.

Table-name interpolation, where needed, uses only hardcoded constants (same
`#nosec G201` discipline as `orphan_check.go`); the `--`/user-supplied values
(`userID`, `limit`, `days`) are always bound parameters.

### Schema — migration `008_domain_events_user_index.sql` (new)

`CREATE INDEX idx_domain_events_user_ts ON domain_events(user_id, timestamp);`
Migration 005 added the `user_id` column but no index; the per-user and
time-bucketed stats queries want it as the event log grows. Down section is a
comment (matching the 005–007 convention).

### Handlers — `internal/api/admin.go` (modify)

- `Dashboard` gains: `SystemStats`, `PerUserCounts` + `UserActivity` (merged into
  the user rows), `RecentEvents(20)`, and `CreationGrowth(30)`, passed into the
  template. Each stats call is non-fatal: on error it logs (`slog`) and omits
  that panel, exactly like the existing orphan-count block — a stats failure
  never breaks the dashboard or the user list.
- `EditUser` (the existing `GET /admin/users/{id}` page) gains an **overview
  section rendered above the existing edit form**: the user's join date (from
  the PocketBase record `CreatedAt`), their `EntityCounts`, `Activity`
  (last-active + total events), and `RecentEventsByUser(id, 20)`. No new route.

Correlation: users come from PocketBase (`h.auth.ListUsers` / `FindUserByID`),
domain data from `atask.db` keyed by `user_id` = the PocketBase record ID. The
handlers already hold both `h.auth` and `h.db`; the merge happens in Go.

### Templates (modify)

- `dashboard.html`: a stat-tile row (users/tasks/projects/areas/events), a
  growth panel (inline-SVG bars, one series for entity creation, computed
  server-side from `CreationGrowth`), a recent-activity table
  (actor · action · entity · relative time), and two extra user-row columns
  (item count, last active).
- `user_edit.html`: an overview block above the form (join date, count tiles,
  last active, per-user recent-activity table).
- Reuse existing admin CSS classes; add minimal styles for tiles/bars in the
  layout's `<style>`. All dynamic values through `html/template` auto-escaping.

## Error handling

Every stats query is best-effort at the presentation layer: a failure logs and
degrades to omitting that one panel (never a 500, never a blank dashboard),
consistent with the orphan-count precedent. The data-layer functions themselves
return errors normally; the handler decides to soften them.

## Testing

- `internal/store/stats_test.go`: seed a migrated in-memory DB with entities +
  `domain_events` for two users; assert `SystemStats`, `PerUserCounts`,
  `UserActivity`, `RecentEvents(ByUser)`, and `CreationGrowth` (including
  zero-fill of empty days and the per-user isolation of counts).
- `internal/api/admin_test.go`: via the real PB admin harness, assert the
  dashboard body renders the stat tiles + a seeded recent event, and that the
  user-detail page renders the overview block for a user with data.

## Out of scope (possible follow-ups)

Cross-user event visibility (admin sees all events — acceptable for an admin
tool); caching the aggregate COUNT queries (fine at current scale); charting
beyond simple bars; CSV export.
