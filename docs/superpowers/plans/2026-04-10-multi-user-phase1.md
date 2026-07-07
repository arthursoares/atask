# Multi-User Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-user data isolation to the Go backend with PocketBase-powered authentication, user-scoped sync/SSE, a basic web admin UI, and Tauri client login support.

**Architecture:** PocketBase is embedded as the auth engine (users, OAuth, tokens) and writes its own SQLite database under `${DATA_DIR}/pb_data/data.db`. The atask domain layer continues to write to its own SQLite database at `${DATA_DIR}/atask.db` — the two halves live side-by-side in the same data directory but never share a connection. All domain tables gain a `user_id` column scoped to PocketBase user record IDs (denormalized; no cross-database FK). A thin `AuthProvider` adapter interface isolates PocketBase internals from the rest of the codebase. The existing sqlc query layer, service layer, and event system are enhanced with per-user filtering at both the SQL and service layers.

**Estimate:** ~4.5 weeks (matching the spec's decisions log; the original ~3-week framing was found to be ~2x optimistic by Codex review).

**Tech Stack:** Go 1.25, PocketBase (embedded), sqlc, SQLite, Tauri 2 (Rust + React), `keyring` crate for OS keychain access

**Spec:** `docs/superpowers/specs/2026-04-10-multi-user-design.md`

**Review history:** This plan was revised on 2026-04-29 in response to a Codex adversarial design review that found three P0 issues (DB topology contradiction, missing legacy-table cleanup, orphan-data invisibility) and four P1 issues (router strategy unspecified, plaintext token storage drift, sync cursor not user-scoped, weak query scanner). All findings are addressed in the amendments below; see commit history for the diff.

---

## File Map

### New Files

| File | Responsibility |
|------|---------------|
| `internal/store/migrations/005_multi_user.sql` | Schema: add user_id to all domain tables + events + invites table |
| `internal/auth/adapter.go` | `AuthProvider` interface definition + `User` type |
| `internal/auth/pocketbase.go` | PocketBase implementation of `AuthProvider` |
| `internal/auth/adapter_test.go` | Integration tests for PocketBase adapter |
| `internal/api/admin.go` | Web admin handlers (Go templates) |
| `internal/api/admin_templates/` | Go HTML templates for admin UI |
| `internal/api/invite.go` | Invite CRUD handlers |
| `internal/store/queries/invites.sql` | Invite sqlc queries |
| `internal/config/config.go` | Typed config struct loaded from env |
| `internal/store/queries/query_scope_test.go` | Query scanning safety-net test |
| `atask-v4/src-tauri/src/auth.rs` | Tauri auth commands (login, refresh, logout, keychain) |
| `atask-v4/src-tauri/src/migrations/007_auth.sql` | Tauri local: auth settings columns |
| `atask-v4/src/store/auth.ts` | `$currentUser`, `$isAuthenticated`, login/logout mutations |
| `atask-v4/src/components/LoginPanel.tsx` | Login UI (email/password + OAuth buttons) |

### Modified Files

| File | Changes |
|------|---------|
| `go.mod` | Add `github.com/pocketbase/pocketbase` dependency |
| `cmd/atask/main.go` | Replace raw HTTP server with PocketBase app, register custom routes |
| `sqlc.yaml` | No changes needed (engine stays sqlite) |
| `internal/store/queries/tasks.sql` | Add `user_id` to all 29 queries |
| `internal/store/queries/projects.sql` | Add `user_id` to all 13 queries |
| `internal/store/queries/areas.sql` | Add `user_id` to all 7 queries |
| `internal/store/queries/tags.sql` | Add `user_id` to all 7 queries |
| `internal/store/queries/locations.sql` | Add `user_id` to all 6 queries |
| `internal/store/queries/sections.sql` | Add `user_id` to all 7 queries |
| `internal/store/queries/checklist_items.sql` | Add `user_id` to all 8 queries |
| `internal/store/queries/activities.sql` | Add `user_id` to all 2 queries |
| `internal/store/queries/task_tags.sql` | Add `user_id` to all queries |
| `internal/store/queries/task_links.sql` | Add `user_id` to all queries |
| `internal/store/queries/views.sql` | Add `user_id` to all 5 queries |
| `internal/store/queries/events.sql` | Add `user_id` to insert + filter queries |
| `internal/service/task_service.go` | Add `userID` param to all 31 methods |
| `internal/service/project_service.go` | Add `userID` param to all 15 methods |
| `internal/service/area_service.go` | Add `userID` param to all 8 methods |
| `internal/service/section_service.go` | Add `userID` param to all 7 methods |
| `internal/service/tag_service.go` | Add `userID` param to all 7 methods |
| `internal/service/location_service.go` | Add `userID` param to all 6 methods |
| `internal/service/checklist_service.go` | Add `userID` param to all 9 methods |
| `internal/service/activity_service.go` | Add `userID` param to all 2 methods |
| `internal/api/tasks.go` | Extract `userID` from context, pass to service |
| `internal/api/projects.go` | Extract `userID` from context, pass to service |
| `internal/api/areas.go` | Extract `userID` from context, pass to service |
| `internal/api/sections.go` | Extract `userID` from context, pass to service |
| `internal/api/tags.go` | Extract `userID` from context, pass to service |
| `internal/api/locations.go` | Extract `userID` from context, pass to service |
| `internal/api/checklist.go` | Extract `userID` from context, pass to service |
| `internal/api/activities.go` | Extract `userID` from context, pass to service |
| `internal/api/views.go` | Extract `userID` from context, pass to service |
| `internal/api/sync.go` | Add `user_id` filtering to delta query |
| `internal/api/auth.go` | Replace direct auth logic with AuthProvider adapter calls |
| `internal/api/router.go` | Restructure for PocketBase routing + admin routes |
| `internal/api/middleware.go` | Dual auth: PocketBase tokens + API keys |
| `internal/event/stream.go` | Add user_id to SSE stream filtering |
| `internal/event/event_store.go` | Add user_id param to InsertDeltaEvent |
| `Dockerfile` | Update for PocketBase data directory |
| `docker-compose.yml` | Add OAuth + BASE_URL env vars |
| `atask-v4/src-tauri/src/sync.rs` | Bearer token support + 401 handling |
| `atask-v4/src-tauri/src/sync_commands.rs` | Auth token in sync config |
| `atask-v4/src-tauri/src/lib.rs` | Register auth commands + keychain plugin |
| `atask-v4/src/views/SettingsView.tsx` | Expand with login/account UI |
| `atask-v4/src/store/ui.ts` | Add sync auth state atoms |

---

## Task 1: Schema Migration

**Files:**
- Create: `internal/store/migrations/005_multi_user.sql`

- [ ] **Step 1: Write migration file**

```sql
-- internal/store/migrations/005_multi_user.sql

-- +goose Up

-- user_id on all domain tables
ALTER TABLE tasks ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE areas ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE tags ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE locations ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE activities ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE sections ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE checklist_items ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE task_tags ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE project_tags ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE task_links ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

-- user_id on event tables
ALTER TABLE delta_events ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE domain_events ADD COLUMN user_id TEXT NOT NULL DEFAULT '';

-- Indexes for query performance
CREATE INDEX idx_tasks_user ON tasks(user_id);
CREATE INDEX idx_projects_user ON projects(user_id);
CREATE INDEX idx_areas_user ON areas(user_id);
CREATE INDEX idx_tags_user ON tags(user_id);
CREATE INDEX idx_locations_user ON locations(user_id);
CREATE INDEX idx_sections_user ON sections(user_id);
CREATE INDEX idx_delta_events_user ON delta_events(user_id);

-- Invite tokens for closed-registration flows
CREATE TABLE invites (
    id         TEXT NOT NULL PRIMARY KEY,
    email      TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT 'user',
    token      TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    claimed_at DATETIME,
    expires_at DATETIME NOT NULL
);

-- +goose Down
-- Down migrations for SQLite require table recreation; omitted for brevity.
-- In practice, rollback is handled by restoring from backup.
```

- [ ] **Step 2: Verify migration applies**

Run: `make test`
Expected: All existing tests pass (new columns have defaults, no breakage).

- [ ] **Step 3: Commit**

```bash
git add internal/store/migrations/005_multi_user.sql
git commit -m "feat(schema): add user_id to all domain and event tables (migration 005)"
```

---

## Task 1.5: Legacy Cleanup Migration (006)

**Why this task exists:** The original plan tried to add legacy-table cleanup and api_keys scope/expiry into migration 005 mid-stream (Task 16). That edits an already-committed migration file — brittle in any environment that ran 005 first. This task splits cleanup into a fresh migration 006 that runs sequentially after 005.

**Files:**
- Create: `internal/store/migrations/006_legacy_cleanup.sql`
- Modify: `internal/store/queries/auth.sql` (delete legacy user queries)

- [ ] **Step 1: Write migration 006**

```sql
-- internal/store/migrations/006_legacy_cleanup.sql

-- +goose Up

-- API keys: retarget user_id to PocketBase user record IDs and add scope/expiry.
-- SQLite cannot ALTER COLUMN to drop the FK reference, so we rebuild the table.
-- Existing rows are preserved; their user_id values become orphaned until
-- `atask admin assign-data --to <pb-user-id>` is run.
CREATE TABLE api_keys_new (
    id           TEXT NOT NULL PRIMARY KEY,
    user_id      TEXT NOT NULL DEFAULT '',
    name         TEXT,
    key_hash     TEXT UNIQUE,
    permissions  TEXT NOT NULL DEFAULT '[]',
    scope        TEXT NOT NULL DEFAULT 'read_write',
    expires_at   DATETIME,
    created_at   DATETIME,
    last_used_at DATETIME
);

INSERT INTO api_keys_new (id, user_id, name, key_hash, permissions, created_at, last_used_at)
SELECT id, COALESCE(user_id, ''), name, key_hash, permissions, created_at, last_used_at
FROM api_keys;

DROP TABLE api_keys;
ALTER TABLE api_keys_new RENAME TO api_keys;
CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);

-- Drop legacy users table; identity now lives in pb_data/data.db
DROP TABLE users;

-- +goose Down
-- Down migrations omitted (SQLite makes them painful and rollback is via backup).
```

- [ ] **Step 2: Delete legacy queries from auth.sql**

Open `internal/store/queries/auth.sql` and delete these four queries (they reference the dropped table):
- `CreateUser`
- `GetUserByEmail`
- `GetUserByID`
- `UpdateUser`

Keep all `api_keys`-related queries; they will be updated in a later step (Task 16 → folded into Task 11 below) to include `scope` and `expires_at`.

- [ ] **Step 3: Update CreateAPIKey query for new columns**

Replace the existing `CreateAPIKey` query in `auth.sql`:

```sql
-- name: CreateAPIKey :one
INSERT INTO api_keys (id, user_id, name, key_hash, permissions, scope, expires_at, created_at, last_used_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)
RETURNING *;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = ? AND (expires_at IS NULL OR expires_at > datetime('now'));
```

- [ ] **Step 4: Identify and stub callers of deleted queries**

Run: `grep -rn "CreateUser\|GetUserByEmail\|GetUserByID\b\|UpdateUser\b" internal/ cmd/`

For each match, the call site must either be deleted (legacy auth handler logic that will be replaced by the AuthProvider in Task 12) or temporarily stubbed to return `errors.New("legacy auth removed; use AuthProvider")`. Do not delete the call sites yet — Task 12 rewrites them properly. The point of this step is to make the build green after sqlc regeneration.

Expected matches: `internal/api/auth.go` (login, register handlers), and possibly `cmd/atask/main.go` if it has a bootstrap admin path.

- [ ] **Step 5: Regenerate sqlc and verify build**

Run: `make sqlc && go build ./...`
Expected: Compiles. The legacy `CreateUser` etc. methods are gone from generated code; api_keys queries have new params (`Scope`, `ExpiresAt`).

- [ ] **Step 6: Verify migration applies**

Run: `make migrate && make test`
Expected: 005 and 006 both apply cleanly; existing tests pass (auth tests may fail — that is fine; they will be rewritten in Task 12).

- [ ] **Step 7: Commit**

```bash
git add internal/store/migrations/006_legacy_cleanup.sql internal/store/queries/auth.sql internal/store/sqlc/ internal/api/auth.go
git commit -m "feat(schema): migration 006 — drop legacy users, retarget api_keys, add scope/expires_at"
```

---

## Task 2: Scope sqlc Queries — Tasks

**Files:**
- Modify: `internal/store/queries/tasks.sql`

- [ ] **Step 1: Add user_id to all task queries**

Every query gets `user_id` added. Pattern:
- `INSERT`: add `user_id` column + `?` placeholder
- `SELECT ... WHERE id = ?`: add `AND user_id = ?`
- `SELECT ... WHERE deleted = 0`: add `AND user_id = ?`
- `UPDATE ... WHERE id = ?`: add `AND user_id = ?`
- Cascade operations (e.g. `SoftDeleteTasksByProject`): add `AND user_id = ?`

```sql
-- name: CreateTask :one
INSERT INTO tasks (
    id, title, notes, status, schedule, start_date, deadline, completed_at,
    "index", today_index, project_id, section_id, area_id, location_id,
    recurrence_rule, deleted, deleted_at, created_at, updated_at, user_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, 0, NULL, ?, ?, ?
)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = ? AND user_id = ? AND deleted = 0;

-- name: ListTasks :many
SELECT * FROM tasks
WHERE user_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksByProject :many
SELECT * FROM tasks
WHERE project_id = ? AND user_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksByArea :many
SELECT * FROM tasks
WHERE area_id = ? AND user_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksBySection :many
SELECT * FROM tasks
WHERE section_id = ? AND user_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksByLocation :many
SELECT * FROM tasks
WHERE location_id = ? AND user_id = ? AND deleted = 0
ORDER BY "index";

-- name: ListTasksBySchedule :many
SELECT * FROM tasks
WHERE schedule = ? AND user_id = ? AND deleted = 0
ORDER BY "index";

-- name: UpdateTaskTitle :one
UPDATE tasks SET title = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskNotes :one
UPDATE tasks SET notes = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskStatus :one
UPDATE tasks SET status = ?, completed_at = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskSchedule :one
UPDATE tasks SET schedule = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskStartDate :one
UPDATE tasks SET start_date = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskDeadline :one
UPDATE tasks SET deadline = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskProject :one
UPDATE tasks SET project_id = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskSection :one
UPDATE tasks SET section_id = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskArea :one
UPDATE tasks SET area_id = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskLocation :one
UPDATE tasks SET location_id = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskRecurrence :one
UPDATE tasks SET recurrence_rule = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskIndex :one
UPDATE tasks SET "index" = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: UpdateTaskTodayIndex :one
UPDATE tasks SET today_index = ?, updated_at = ?
WHERE id = ? AND user_id = ? AND deleted = 0
RETURNING *;

-- name: SoftDeleteTask :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE id = ? AND user_id = ?;

-- name: SoftDeleteTasksByProject :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE project_id = ? AND user_id = ? AND deleted = 0;

-- name: OrphanTasksByArea :exec
UPDATE tasks SET area_id = NULL, updated_at = ?
WHERE area_id = ? AND user_id = ? AND deleted = 0;

-- name: OrphanTasksBySection :exec
UPDATE tasks SET section_id = NULL, updated_at = ?
WHERE section_id = ? AND user_id = ? AND deleted = 0;

-- name: CascadeDeleteTasksByArea :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE area_id = ? AND user_id = ? AND deleted = 0;

-- name: CascadeDeleteTasksBySection :exec
UPDATE tasks SET deleted = 1, deleted_at = ?, updated_at = ?
WHERE section_id = ? AND user_id = ? AND deleted = 0;

-- name: CompleteTasksByProject :exec
UPDATE tasks SET status = 1, completed_at = ?, updated_at = ?
WHERE project_id = ? AND user_id = ? AND status = 0 AND deleted = 0;

-- name: CancelTasksByProject :exec
UPDATE tasks SET status = 2, updated_at = ?
WHERE project_id = ? AND user_id = ? AND status = 0 AND deleted = 0;
```

- [ ] **Step 2: Run sqlc generate**

Run: `make sqlc`
Expected: Regenerated Go code in `internal/store/sqlc/` compiles. New params structs include `UserID string`.

- [ ] **Step 3: Commit**

```bash
git add internal/store/queries/tasks.sql internal/store/sqlc/
git commit -m "feat(queries): add user_id scoping to all task queries"
```

---

## Task 3: Scope sqlc Queries — All Other Entities

**Files:**
- Modify: `internal/store/queries/projects.sql`
- Modify: `internal/store/queries/areas.sql`
- Modify: `internal/store/queries/sections.sql`
- Modify: `internal/store/queries/tags.sql`
- Modify: `internal/store/queries/locations.sql`
- Modify: `internal/store/queries/checklist_items.sql`
- Modify: `internal/store/queries/activities.sql`
- Modify: `internal/store/queries/task_tags.sql`
- Modify: `internal/store/queries/task_links.sql`
- Modify: `internal/store/queries/views.sql`
- Modify: `internal/store/queries/events.sql`

Apply the same pattern as Task 2 to every remaining query file. The transformation rules are:

**For each query:**
- `INSERT ... VALUES (...)` → add `user_id` column and `?` placeholder at end
- `SELECT ... WHERE id = ?` → add `AND user_id = ?`
- `SELECT ... WHERE <filter> AND deleted = 0` → add `AND user_id = ?`
- `UPDATE ... WHERE id = ?` → add `AND user_id = ?`
- `DELETE` / soft-delete → add `AND user_id = ?`

- [ ] **Step 1: Scope projects.sql (13 queries)**

All queries follow the same pattern as tasks. Add `user_id` to `CreateProject` INSERT, add `AND user_id = ?` to all SELECT/UPDATE/DELETE WHERE clauses. `OrphanProjectsByArea` and `CascadeDeleteProjectsByArea` get `AND user_id = ?` appended.

- [ ] **Step 2: Scope areas.sql (7 queries)**

Add `user_id` to `CreateArea` INSERT. Add `AND user_id = ?` to `GetArea`, `ListAreas`, `ListAllAreas`, `UpdateAreaTitle`, `UpdateAreaArchived`, `SoftDeleteArea`.

- [ ] **Step 3: Scope sections.sql (7 queries)**

Add `user_id` to `CreateSection` INSERT. Add `AND user_id = ?` to all queries. `SoftDeleteSectionsByProject` gets `AND user_id = ?`.

- [ ] **Step 4: Scope tags.sql (7 queries)**

Same pattern. Add `user_id` to all 7 queries.

- [ ] **Step 5: Scope locations.sql (6 queries)**

Same pattern. Add `user_id` to all 6 queries.

- [ ] **Step 6: Scope checklist_items.sql (8 queries)**

Same pattern. Add `user_id` to all 8 queries.

- [ ] **Step 7: Scope activities.sql (2 queries)**

Add `user_id` to `CreateActivity` INSERT and `AND user_id = ?` to `ListActivitiesByTask`.

- [ ] **Step 8: Scope task_tags.sql + task_links.sql**

Add `user_id` to all join table queries: `AddTaskTag`, `RemoveTaskTag`, `ListTaskTags`, `RemoveAllTagReferences`, `AddProjectTag`, `RemoveProjectTag`, `ListProjectTags`, `RemoveAllProjectTagReferences`, `AddTaskLink`, `RemoveTaskLink`, `ListTaskLinks`.

- [ ] **Step 9: Scope views.sql (5 queries)**

Add `AND user_id = ?` to all 5 view queries (`ViewInbox`, `ViewToday`, `ViewUpcoming`, `ViewSomeday`, `ViewLogbook`).

- [ ] **Step 10: Scope events.sql**

```sql
-- name: InsertDeltaEvent :exec
INSERT INTO delta_events (
    entity_type, entity_id, action, field, old_value, new_value, actor_id, timestamp, user_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: ListDeltaEventsSince :many
SELECT * FROM delta_events
WHERE id > ? AND user_id = ?
ORDER BY id;

-- name: InsertDomainEvent :one
INSERT INTO domain_events (
    type, entity_type, entity_id, actor_id, payload, timestamp, user_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING id;

-- name: ListDomainEventsSince :many
SELECT * FROM domain_events
WHERE id > ? AND user_id = ?
ORDER BY id;

-- name: ListDomainEventsByTypeSince :many
SELECT * FROM domain_events
WHERE type = ? AND id > ? AND user_id = ?
ORDER BY id;

-- name: ListDomainEventsByEntitySince :many
SELECT * FROM domain_events
WHERE entity_type = ? AND entity_id = ? AND id > ? AND user_id = ?
ORDER BY id;
```

- [ ] **Step 11: Regenerate sqlc**

Run: `make sqlc`
Expected: All generated Go code compiles with new `UserID` params.

- [ ] **Step 12: Commit**

```bash
git add internal/store/queries/ internal/store/sqlc/
git commit -m "feat(queries): add user_id scoping to all domain, view, and event queries"
```

---

## Task 4: Query Scanning Safety-Net Test (Operation-Aware)

**Why this is operation-aware, not substring-based:** The original plan's scanner only checked that the substring `user_id` appeared somewhere in the query body. That false-passes any query whose `WHERE` clause omits `user_id` as long as the column appears anywhere — for example in a `SELECT user_id, ...` projection, a comment, or a sibling JOIN. The dangerous case Codex flagged: `ListTaskTags` and `ListProjectTags` JOIN through join tables to surface tag rows; the join table has `user_id` in its INSERT but the SELECT predicate may not constrain it. We need the test to *parse statements* and check that every `SELECT`/`UPDATE`/`DELETE` against a user-owned table has `user_id = ?` (or equivalent) in its `WHERE` predicate.

**Files:**
- Create: `internal/store/queries/query_scope_test.go`

- [ ] **Step 1: Write the operation-aware scanner**

```go
package queries_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// userOwnedTables: every read/write of these MUST include user_id in its WHERE.
var userOwnedTables = []string{
	"tasks", "projects", "areas", "sections", "tags",
	"locations", "checklist_items", "activities",
	"task_tags", "project_tags", "task_links",
	"delta_events", "domain_events",
}

// hasUserIDPredicate looks for `user_id` followed by a comparison operator
// somewhere in the body (after the first FROM/UPDATE/DELETE keyword).
// It is intentionally lenient about positioning — the goal is "is user_id
// constrained anywhere in this statement" — and strict about *form* (must
// be a predicate, not just a column reference).
var userIDPredicate = regexp.MustCompile(`(?i)user_id\s*(=|in\s*\()`)

// statementOpKind classifies the SQL statement.
func statementOpKind(body string) string {
	trim := strings.TrimSpace(strings.ToLower(body))
	switch {
	case strings.HasPrefix(trim, "insert"):
		return "insert"
	case strings.HasPrefix(trim, "select"):
		return "select"
	case strings.HasPrefix(trim, "update"):
		return "update"
	case strings.HasPrefix(trim, "delete"):
		return "delete"
	default:
		return "other"
	}
}

// touchesUserOwned: returns the matched table name, or "" if none.
func touchesUserOwned(body string) string {
	lower := strings.ToLower(body)
	for _, tbl := range userOwnedTables {
		// word-boundary match: " tasks " or " tasks\n" etc., not "tasks_x"
		idx := strings.Index(lower, tbl)
		for idx >= 0 {
			before := byte(' ')
			after := byte(' ')
			if idx > 0 {
				before = lower[idx-1]
			}
			if idx+len(tbl) < len(lower) {
				after = lower[idx+len(tbl)]
			}
			if !isIdentChar(before) && !isIdentChar(after) {
				return tbl
			}
			next := strings.Index(lower[idx+1:], tbl)
			if next < 0 {
				break
			}
			idx = idx + 1 + next
		}
	}
	return ""
}

func isIdentChar(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func TestAllUserOwnedQueriesScopeByUserID(t *testing.T) {
	skip := map[string]bool{
		"auth.sql":    true, // legacy users table queries are deleted in Task 1.5;
		                     // remaining api_keys queries scope by key_hash, not user_id
		"invites.sql": true, // invites are claimed by token, not scoped by user_id
	}

	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	for _, f := range files {
		if skip[f] {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		queries := strings.Split(string(data), "-- name:")
		for _, q := range queries[1:] {
			lines := strings.SplitN(q, "\n", 2)
			name := strings.TrimSpace(strings.Split(lines[0], ":")[0])
			if len(lines) < 2 {
				continue
			}
			body := lines[1]

			tbl := touchesUserOwned(body)
			if tbl == "" {
				continue
			}
			op := statementOpKind(body)

			switch op {
			case "insert":
				// INSERT must include user_id in the column list
				lower := strings.ToLower(body)
				colsStart := strings.Index(lower, "(")
				colsEnd := strings.Index(lower, ")")
				if colsStart < 0 || colsEnd < 0 || colsEnd < colsStart {
					t.Errorf("%s/%s: INSERT touches %q but column list unparseable", f, name, tbl)
					continue
				}
				cols := lower[colsStart:colsEnd]
				if !strings.Contains(cols, "user_id") {
					t.Errorf("%s/%s: INSERT into %q missing user_id column", f, name, tbl)
				}
			case "select", "update", "delete":
				if !userIDPredicate.MatchString(body) {
					t.Errorf("%s/%s: %s on %q has no `user_id =` or `user_id IN` predicate", f, name, strings.ToUpper(op), tbl)
				}
			}
		}
	}
}
```

- [ ] **Step 2: Add explicit failing-case tests for the JOIN-through queries**

These three tests exercise the scanner against synthetic query bodies that should fail. They guarantee the scanner catches the patterns Codex flagged, so future refactors of the scanner can't silently weaken it.

```go
func TestScannerCatchesJoinThroughWithoutPredicate(t *testing.T) {
	// Synthetic ListTaskTags-style query: joins tags via task_tags, but
	// the WHERE clause only filters by task_id — no user_id predicate.
	body := `
SELECT t.* FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id
WHERE tt.task_id = ?;`
	if userIDPredicate.MatchString(body) {
		t.Fatal("scanner should NOT find user_id predicate in this body")
	}
	if touchesUserOwned(body) == "" {
		t.Fatal("scanner should detect that this body touches a user-owned table")
	}
}

func TestScannerAcceptsScopedJoinThrough(t *testing.T) {
	body := `
SELECT t.* FROM tags t
JOIN task_tags tt ON tt.tag_id = t.id AND tt.user_id = ?
WHERE tt.task_id = ? AND tt.user_id = ?;`
	if !userIDPredicate.MatchString(body) {
		t.Fatal("scanner should accept this body as scoped")
	}
}

func TestScannerRejectsProjectionOnly(t *testing.T) {
	// `SELECT user_id, ...` mentions user_id as a column but does not constrain it.
	body := `SELECT user_id, id, title FROM tasks WHERE id = ?;`
	if userIDPredicate.MatchString(body) {
		t.Fatal("scanner should NOT count `SELECT user_id` as a predicate")
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd internal/store/queries && go test -run "TestAllUserOwnedQueriesScopeByUserID|TestScanner" -v`
Expected: All four tests PASS — the real query files include user_id, and the synthetic bad-pattern bodies are correctly flagged.

- [ ] **Step 4: Commit**

```bash
git add internal/store/queries/query_scope_test.go
git commit -m "test: operation-aware query scanner for user_id scoping (incl. JOIN-through)"
```

---

## Task 5: Thread userID Through Service Layer

**Files:**
- Modify: All 8 service files in `internal/service/`

Every public service method gains a `userID string` parameter (inserted after `ctx context.Context`). The `userID` is passed to every sqlc query call.

- [ ] **Step 1: Update TaskService (31 methods)**

Pattern for each method — example `Create`:

```go
// Before:
func (s *TaskService) Create(ctx context.Context, title, actorID string, opts ...string) (*domain.Task, error) {
    // ... calls s.q.CreateTask(ctx, sqlc.CreateTaskParams{...})

// After:
func (s *TaskService) Create(ctx context.Context, userID, title, actorID string, opts ...string) (*domain.Task, error) {
    // ... calls s.q.CreateTask(ctx, sqlc.CreateTaskParams{..., UserID: userID})
```

Example `Get`:
```go
// Before:
func (s *TaskService) Get(ctx context.Context, id string) (*domain.Task, error) {
    row, err := s.q.GetTask(ctx, id)

// After:
func (s *TaskService) Get(ctx context.Context, userID, id string) (*domain.Task, error) {
    row, err := s.q.GetTask(ctx, sqlc.GetTaskParams{ID: id, UserID: userID})
```

Apply to all 31 methods. Also update delta event emission to include `userID`:
```go
s.es.InsertDelta(ctx, event.Delta{
    EntityType: "task",
    EntityID:   task.ID,
    // ... existing fields ...
    UserID:     userID,  // NEW
})
```

- [ ] **Step 2: Update ProjectService (15 methods)**

Same pattern. Add `userID string` after `ctx` on all 15 public methods. Pass to all sqlc calls and delta emissions.

- [ ] **Step 3: Update AreaService (8 methods)**

Same pattern for all 8 methods.

- [ ] **Step 4: Update SectionService (7 methods)**

Same pattern for all 7 methods.

- [ ] **Step 5: Update TagService (7 methods)**

Same pattern for all 7 methods.

- [ ] **Step 6: Update LocationService (6 methods)**

Same pattern for all 6 methods.

- [ ] **Step 7: Update ChecklistService (9 methods)**

Same pattern for all 9 methods.

- [ ] **Step 8: Update ActivityService (2 methods)**

Same pattern for both methods.

- [ ] **Step 9: Update EventStore**

Modify `internal/event/event_store.go` — `InsertDelta` and `InsertDomainEvent` gain a `UserID` field in their params, passed to the sqlc `INSERT`.

- [ ] **Step 10: Add cross-entity ownership validation (Spec §2.4)**

**Why this step exists:** sqlc scoping prevents User A from reading or writing User B's *root* rows, but it does not stop User A from setting their own task's `project_id` to a UUID that belongs to User B. The SQL `UPDATE tasks SET project_id = ? WHERE id = ? AND user_id = ?` matches User A's task row regardless of who owns the project. Every service method that accepts a foreign-key reference must do an owner-scoped lookup before mutating. Codex flagged this; the spec §2.4 enumerates the full set.

For each method below, follow this pattern:

```go
// Before:
func (s *TaskService) UpdateProject(ctx context.Context, userID, taskID string, projectID *string) (*domain.Task, error) {
    // ... directly calls s.q.UpdateTaskProject(...)
}

// After:
func (s *TaskService) UpdateProject(ctx context.Context, userID, taskID string, projectID *string) (*domain.Task, error) {
    if projectID != nil {
        // Verify the user owns the target project. Returns ErrNotFound if not.
        if _, err := s.projects.Get(ctx, userID, *projectID); err != nil {
            return nil, err
        }
    }
    // ... existing UpdateTaskProject call
}
```

Apply to these specific methods (one TDD cycle per method — write a failing cross-user test first, then add the check):

- [ ] **TaskService.UpdateProject** — verify `GetProject(ctx, userID, *projectID)` succeeds
- [ ] **TaskService.UpdateArea** — verify `GetArea(ctx, userID, *areaID)` succeeds
- [ ] **TaskService.UpdateSection** — verify `GetSection(ctx, userID, *sectionID)` succeeds, and that the section's `project_id` matches the task's current `project_id`
- [ ] **TaskService.UpdateLocation** — verify `GetLocation(ctx, userID, *locationID)` succeeds
- [ ] **TaskService.AddLink** — verify `GetTask(ctx, userID, relatedID)` succeeds; reject self-link with 422
- [ ] **TaskService.AddTag** — verify `GetTag(ctx, userID, tagID)` succeeds
- [ ] **ProjectService.UpdateArea** — verify `GetArea(ctx, userID, *areaID)` succeeds
- [ ] **ProjectService.AddTag** — verify `GetTag(ctx, userID, tagID)` succeeds
- [ ] **SectionService.Create** — verify `GetProject(ctx, userID, projectID)` succeeds
- [ ] **ChecklistService.Add** — verify `GetTask(ctx, userID, taskID)` succeeds

For each method, the failing test pattern (which becomes a regression guard) is:

```go
// internal/service/task_service_ownership_test.go (or per-service file)
func TestTaskService_UpdateProject_RejectsCrossUserProject(t *testing.T) {
    svc := newTestSetup(t)
    // user-a owns task; user-b owns project
    taskA, _ := svc.tasks.Create(ctx, "user-a", "task A", "actor-a")
    projB, _ := svc.projects.Create(ctx, "user-b", "project B", "actor-b")
    // user-a tries to attach user-b's project to their task
    _, err := svc.tasks.UpdateProject(ctx, "user-a", taskA.ID, &projB.ID)
    if !errors.Is(err, domain.ErrNotFound) {
        t.Fatalf("expected ErrNotFound, got %v", err)
    }
}
```

- [ ] **Step 11: Verify compilation**

Run: `go build ./...`
Expected: Compilation errors in API handlers (they still pass old signatures). This is expected — Task 6 fixes them.

- [ ] **Step 12: Run service-layer ownership tests**

Run: `go test ./internal/service/ -run Ownership -v`
Expected: All ownership regression tests pass.

- [ ] **Step 13: Commit**

```bash
git add internal/service/ internal/event/
git commit -m "feat(service): thread userID + add cross-entity ownership validation (spec §2.4)"
```

---

## Task 6: Update API Handlers

**Files:**
- Modify: All handler files in `internal/api/`

Every handler extracts `userID` from context and passes it to service calls.

- [ ] **Step 1: Update all domain handlers**

Pattern for each handler method:

```go
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
    userID := UserIDFromContext(r.Context()) // already works via middleware
    // ... decode body ...
    task, err := h.tasks.Create(r.Context(), userID, body.Title, actorFromRequest(r), body.ID)
    // ...
}
```

Apply to all methods in:
- `tasks.go` — all handler methods
- `projects.go` — all handler methods
- `areas.go` — all handler methods
- `sections.go` — all handler methods
- `tags.go` — all handler methods
- `locations.go` — all handler methods
- `checklist.go` — all handler methods
- `activities.go` — all handler methods
- `views.go` — all view handlers

- [ ] **Step 2: Update sync handler**

Modify `internal/api/sync.go`:

```go
func (h *SyncHandler) Deltas(w http.ResponseWriter, r *http.Request) {
    userID := UserIDFromContext(r.Context())
    // ... parse cursor ...
    deltas, err := h.events.DeltasSince(r.Context(), cursor, userID)
    // ...
}
```

- [ ] **Step 3: Update SSE stream**

Modify `internal/event/stream.go` — `ServeHTTP` extracts `userID` from request context and only delivers events matching that user:

```go
func (sm *StreamManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID := api.UserIDFromContext(r.Context())
    // ... existing setup ...

    // Subscribe with user filter
    id := sm.bus.Subscribe(topic, func(e *domain.DomainEvent) {
        if e.UserID != "" && e.UserID != userID {
            return // skip events for other users
        }
        select {
        case ch <- e:
        default:
            slog.Warn("SSE event dropped: channel full")
        }
    })
    // ... rest unchanged ...
}
```

This requires `domain.DomainEvent` to have a `UserID` field. Add it:

```go
// internal/domain/events.go
type DomainEvent struct {
    // ... existing fields ...
    UserID     string `json:"userId"`
}
```

- [ ] **Step 4: Verify full build**

Run: `go build ./...`
Expected: Clean compilation.

- [ ] **Step 5: Run existing tests**

Run: `make test`
Expected: Tests fail where they don't pass `userID`. This is expected — test fixtures need updating.

- [ ] **Step 6: Fix test fixtures**

Every API test that doesn't currently authenticate needs a test user injected via context. The complete list of files (verified against the codebase 2026-04-29):

- `internal/api/patch_test.go` — PATCH endpoint integration tests
- `internal/api/decode_integration_test.go` — request decode/validation tests
- `internal/api/handler_regression_test.go` — recent-fix regression tests
- `internal/api/views_test.go` — Inbox/Today/Upcoming view handler tests
- `internal/api/sync_test.go` — delta sync handler tests
- `internal/api/areas_test.go` — area handler tests
- `internal/api/response_test.go` — response shape tests (only if it constructs a server, not pure helpers)

For each file, find the test setup function (`setupPatchTestServer`, `setupFullTestServer`, equivalents) and pass a default test user ID through the auth middleware. Add a helper:

```go
// internal/api/test_helpers.go (or wherever the existing setup helpers live)
func withTestUser(userID string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), ctxUserID, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Use it to wrap the handler under test. Default test user ID: `"test-user-1"`. Tests that need a second user (e.g., the cross-user isolation test in Task 7) construct their own wrapper.

- [ ] **Step 7: Run tests again**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/api/ internal/domain/ internal/event/
git commit -m "feat(api): extract userID from context in all handlers, scope sync + SSE"
```

---

## Task 7: Cross-User Isolation Test

**Files:**
- Create or modify: `internal/api/handler_regression_test.go`

- [ ] **Step 1: Write isolation test**

```go
func TestCrossUserIsolation(t *testing.T) {
    mux := setupFullTestServer(t)

    // Create tasks as two different users
    // (setupFullTestServer needs to support per-request user context)
    w := doJSONAsUser(t, mux, "user-a", http.MethodPost, "/tasks", `{"title":"A's task"}`)
    if w.Code != http.StatusCreated {
        t.Fatalf("create task A: %d", w.Code)
    }
    var respA struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
    json.NewDecoder(w.Body).Decode(&respA)

    w = doJSONAsUser(t, mux, "user-b", http.MethodPost, "/tasks", `{"title":"B's task"}`)
    if w.Code != http.StatusCreated {
        t.Fatalf("create task B: %d", w.Code)
    }

    // User A lists tasks — should only see their own
    w = doJSONAsUser(t, mux, "user-a", http.MethodGet, "/tasks", "")
    if w.Code != http.StatusOK {
        t.Fatalf("list tasks A: %d", w.Code)
    }
    var listA struct{ Data []struct{ Title string `json:"title"` } `json:"data"` }
    json.NewDecoder(w.Body).Decode(&listA)
    if len(listA.Data) != 1 || listA.Data[0].Title != "A's task" {
        t.Errorf("user A should see 1 task, got %d: %+v", len(listA.Data), listA.Data)
    }

    // User B cannot GET user A's task by ID
    w = doJSONAsUser(t, mux, "user-b", http.MethodGet, "/tasks/"+respA.Data.ID, "")
    if w.Code != http.StatusNotFound {
        t.Errorf("user B accessing A's task: expected 404, got %d", w.Code)
    }

    // User B cannot PATCH user A's task
    w = doJSONAsUser(t, mux, "user-b", http.MethodPatch, "/tasks/"+respA.Data.ID, `{"title":"hacked"}`)
    if w.Code != http.StatusNotFound {
        t.Errorf("user B patching A's task: expected 404, got %d", w.Code)
    }
}

// Horizontal-escalation case: User A tries to attach User B's project to their own task.
// SQL scoping passes (the task is A's), but service-layer ownership validation
// (Task 5 Step 10) must reject it. This test guards against that regression.
func TestCrossUserHorizontalEscalation_TaskProject(t *testing.T) {
    mux := setupFullTestServer(t)

    // User A creates a task
    w := doJSONAsUser(t, mux, "user-a", http.MethodPost, "/tasks", `{"title":"A's task"}`)
    if w.Code != http.StatusCreated {
        t.Fatalf("create task A: %d", w.Code)
    }
    var respA struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
    json.NewDecoder(w.Body).Decode(&respA)

    // User B creates a project
    w = doJSONAsUser(t, mux, "user-b", http.MethodPost, "/projects", `{"title":"B's project"}`)
    if w.Code != http.StatusCreated {
        t.Fatalf("create project B: %d", w.Code)
    }
    var respB struct{ Data struct{ ID string `json:"id"` } `json:"data"` }
    json.NewDecoder(w.Body).Decode(&respB)

    // User A tries to PATCH their own task to point at User B's project
    body := fmt.Sprintf(`{"projectId":%q}`, respB.Data.ID)
    w = doJSONAsUser(t, mux, "user-a", http.MethodPatch, "/tasks/"+respA.Data.ID, body)
    if w.Code != http.StatusNotFound && w.Code != http.StatusUnprocessableEntity {
        t.Errorf("horizontal escalation via projectId: expected 404 or 422, got %d", w.Code)
    }

    // Verify the task was NOT modified
    w = doJSONAsUser(t, mux, "user-a", http.MethodGet, "/tasks/"+respA.Data.ID, "")
    var task struct{ Data struct{ ProjectID *string `json:"projectId"` } `json:"data"` }
    json.NewDecoder(w.Body).Decode(&task)
    if task.Data.ProjectID != nil {
        t.Errorf("task projectId should remain unset after rejected escalation, got %v", *task.Data.ProjectID)
    }
}
```

The `doJSONAsUser` helper injects a user ID into the request context (similar to `doJSON` but with per-request auth context).

- [ ] **Step 2: Run isolation test**

Run: `go test ./internal/api/ -run TestCrossUserIsolation -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/api/handler_regression_test.go
git commit -m "test: add cross-user isolation regression test"
```

---

## Task 8: Configuration Module

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: Write config struct**

```go
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr             string
	DataDir          string
	BaseURL          string
	RegistrationOpen bool

	// OAuth (empty = disabled)
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string

	// Admin
	PocketBaseAdminUI bool
}

func Load() *Config {
	return &Config{
		Addr:               envOr("ADDR", ":8080"),
		DataDir:            envOr("DATA_DIR", "./pb_data"),
		BaseURL:            envOr("BASE_URL", "http://localhost:8080"),
		RegistrationOpen:   os.Getenv("REGISTRATION_OPEN") == "true",
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		PocketBaseAdminUI:  os.Getenv("POCKETBASE_ADMIN_UI") == "true",
	}
}

func (c *Config) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("DATA_DIR is required")
	}
	return nil
}

func (c *Config) EnabledProviders() map[string]bool {
	return map[string]bool{
		"email":  true,
		"google": c.GoogleClientID != "",
		"github": c.GitHubClientID != "",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add typed config module loaded from env"
```

---

## Task 9: PocketBase Adapter Interface

**Files:**
- Create: `internal/auth/adapter.go`

- [ ] **Step 1: Write the interface and types**

```go
package auth

import "time"

// User represents an authenticated user.
type User struct {
	ID        string
	Email     string
	Name      string
	Role      string // "user" or "admin"
	Disabled  bool
	AvatarURL string
	Verified  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AuthProvider abstracts the identity backend. The PocketBase implementation
// is the only concrete implementation, but the interface isolates upgrade risk.
type AuthProvider interface {
	// Token validation
	ValidateToken(token string) (userID string, err error)

	// User CRUD
	CreateUser(email, password, name, role string) (*User, error)
	FindUserByID(id string) (*User, error)
	FindUserByEmail(email string) (*User, error)
	UpdateUser(id string, updates map[string]any) error
	DisableUser(id string) error
	EnableUser(id string) error
	DeleteUser(id string) error
	ListUsers(filter string, page, perPage int) ([]*User, int, error)

	// Auth
	AuthWithPassword(email, password string) (token string, user *User, err error)
	RefreshToken(token string) (newToken string, err error)

	// Provider discovery
	EnabledProviders() []string
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/auth/adapter.go
git commit -m "feat(auth): define AuthProvider interface and User type"
```

---

## Task 10: PocketBase Integration + Main Rewrite

**Files:**
- Create: `internal/auth/pocketbase.go`
- Modify: `cmd/atask/main.go`
- Modify: `go.mod`

- [ ] **Step 1: Add PocketBase dependency**

Run: `go get github.com/pocketbase/pocketbase`

- [ ] **Step 2: Implement PocketBase adapter**

```go
package auth

import (
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// PBAdapter implements AuthProvider using PocketBase's Go API.
type PBAdapter struct {
	app *pocketbase.PocketBase
}

func NewPBAdapter(app *pocketbase.PocketBase) *PBAdapter {
	return &PBAdapter{app: app}
}

func (a *PBAdapter) ValidateToken(token string) (string, error) {
	record, err := a.app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	return record.Id, nil
}

func (a *PBAdapter) FindUserByID(id string) (*User, error) {
	record, err := a.app.FindRecordById("users", id)
	if err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

func (a *PBAdapter) FindUserByEmail(email string) (*User, error) {
	record, err := a.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

func (a *PBAdapter) CreateUser(email, password, name, role string) (*User, error) {
	collection, err := a.app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.SetPassword(password)
	record.Set("name", name)
	record.Set("role", role)
	if err := a.app.Save(record); err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

func (a *PBAdapter) AuthWithPassword(email, password string) (string, *User, error) {
	record, err := a.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		return "", nil, fmt.Errorf("user not found: %w", err)
	}
	if !record.ValidatePassword(password) {
		return "", nil, fmt.Errorf("invalid password")
	}
	token, err := record.NewAuthToken()
	if err != nil {
		return "", nil, err
	}
	return token, recordToUser(record), nil
}

func (a *PBAdapter) RefreshToken(token string) (string, error) {
	record, err := a.app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return "", err
	}
	newToken, err := record.NewAuthToken()
	if err != nil {
		return "", err
	}
	return newToken, nil
}

func (a *PBAdapter) UpdateUser(id string, updates map[string]any) error {
	record, err := a.app.FindRecordById("users", id)
	if err != nil {
		return err
	}
	for k, v := range updates {
		record.Set(k, v)
	}
	return a.app.Save(record)
}

func (a *PBAdapter) DisableUser(id string) error {
	return a.UpdateUser(id, map[string]any{"disabled": true})
}

func (a *PBAdapter) EnableUser(id string) error {
	return a.UpdateUser(id, map[string]any{"disabled": false})
}

func (a *PBAdapter) DeleteUser(id string) error {
	record, err := a.app.FindRecordById("users", id)
	if err != nil {
		return err
	}
	return a.app.Delete(record)
}

func (a *PBAdapter) ListUsers(filter string, page, perPage int) ([]*User, int, error) {
	records, err := a.app.FindRecordsByFilter("users", filter, "-created", perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	users := make([]*User, len(records))
	for i, r := range records {
		users[i] = recordToUser(r)
	}
	// Total count for pagination
	total, _ := a.app.CountRecords("users", filter)
	return users, int(total), nil
}

func (a *PBAdapter) EnabledProviders() []string {
	// This will be populated from config in the actual wiring
	return nil
}

func recordToUser(r *core.Record) *User {
	return &User{
		ID:        r.Id,
		Email:     r.Email(),
		Name:      r.GetString("name"),
		Role:      r.GetString("role"),
		Disabled:  r.GetBool("disabled"),
		AvatarURL: r.GetString("avatar"),
		Verified:  r.Verified(),
		CreatedAt: r.Created.Time(),
		UpdatedAt: r.Updated.Time(),
	}
}
```

- [ ] **Step 3: Rewrite main.go**

```go
package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	app := pocketbase.New()

	// Override PocketBase data dir
	app.RootCmd.SetArgs([]string{
		"serve",
		"--dir=" + cfg.DataDir,
		"--http=" + cfg.Addr,
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Open the domain database. Per spec §1, this is a SEPARATE SQLite file
		// in the same DATA_DIR as PocketBase's pb_data/data.db — they live
		// side-by-side but never share a connection.
		db, err := store.NewDB(cfg.DataDir + "/atask.db")
		if err != nil {
			return err
		}
		if err := db.Migrate(); err != nil {
			return err
		}

		// Auth adapter
		authProvider := auth.NewPBAdapter(app)

		// Event infrastructure
		bus := event.NewBus()
		eventStore := event.NewEventStore(db)

		// Services
		taskSvc := service.NewTaskService(db, eventStore, bus)
		projectSvc := service.NewProjectService(db, eventStore, bus)
		areaSvc := service.NewAreaService(db, eventStore, bus)
		sectionSvc := service.NewSectionService(db, eventStore, bus)
		tagSvc := service.NewTagService(db, eventStore, bus)
		locationSvc := service.NewLocationService(db, eventStore, bus)
		checklistSvc := service.NewChecklistService(db, eventStore, bus)
		activitySvc := service.NewActivityService(db, eventStore, bus)

		// Register custom routes on PocketBase's router
		apiKeySvc := service.NewAPIKeyService(db) // existing service, validates key + returns scope
		csrfStore := api.NewCSRFStore()           // see Task 14 Step 5
		api.RegisterRoutes(se, api.RoutesDeps{
			AuthProvider: authProvider,
			APIKeySvc:    apiKeySvc,
			CSRFStore:    csrfStore,
			Config:       cfg,
			TaskSvc:      taskSvc,
			ProjectSvc:   projectSvc,
			AreaSvc:      areaSvc,
			SectionSvc:   sectionSvc,
			TagSvc:       tagSvc,
			LocationSvc:  locationSvc,
			ChecklistSvc: checklistSvc,
			ActivitySvc:  activitySvc,
			EventStore:   eventStore,
			Bus:          bus,
		})

		// Disable PB admin UI if configured
		if !cfg.PocketBaseAdminUI {
			// PocketBase doesn't have a simple disable flag;
			// we restrict /_/ access via middleware
		}

		return se.Next()
	})

	// CLI subcommands (admin create-user, admin assign-data)
	registerAdminCommands(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
```

Note: The `api.RegisterRoutes` and `api.RoutesDeps` need to be created to replace `api.NewRouter`. This restructures handler registration to use PocketBase's router (`se.Router`) instead of `http.NewServeMux`.

- [ ] **Step 4: Verify it compiles**

Run: `go build ./cmd/atask`
Expected: Compiles (some wiring may need stubs initially).

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/auth/ cmd/atask/main.go
git commit -m "feat: embed PocketBase, implement auth adapter, rewrite main.go"
```

---

## Task 11: Auth Middleware + Router Restructure

**Files:**
- Modify: `internal/api/middleware.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Rewrite auth middleware for dual auth**

```go
// internal/api/middleware.go — updated requireAuth
func requireAuth(authProvider auth.AuthProvider, apiKeySvc APIKeyValidator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            header := r.Header.Get("Authorization")

            var userID string
            var err error

            if strings.HasPrefix(header, "ApiKey ") {
                key := strings.TrimPrefix(header, "ApiKey ")
                userID, _, err = apiKeySvc.ValidateAPIKey(r.Context(), key)
            } else if strings.HasPrefix(header, "Bearer ") {
                token := strings.TrimPrefix(header, "Bearer ")
                userID, err = authProvider.ValidateToken(token)
            } else {
                RespondError(w, http.StatusUnauthorized, "missing authorization header")
                return
            }

            if err != nil {
                RespondError(w, http.StatusUnauthorized, "invalid credentials")
                return
            }

            // Check if user is disabled
            user, err := authProvider.FindUserByID(userID)
            if err != nil || user.Disabled {
                RespondError(w, http.StatusForbidden, "account disabled")
                return
            }

            ctx := context.WithValue(r.Context(), ctxUserID, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

- [ ] **Step 2: Restructure router for PocketBase integration (single strategy: bridge wrapper)**

**Decision (Codex review, 2026-04-29):** Wrap existing `http.HandlerFunc`s as PocketBase `func(*core.RequestEvent) error` callbacks via a single bridge function. This avoids rewriting 14 handler files into PocketBase's request/response idiom and preserves all existing tests. The same wrapper is reused by Task 14's admin routes — no separate `*http.ServeMux` is reintroduced.

Add the bridge function at the top of `internal/api/router.go`:

```go
// bridge adapts an http.HandlerFunc to PocketBase's *core.RequestEvent callback.
// PocketBase's RequestEvent embeds the underlying http.ResponseWriter and *http.Request,
// so we can pass them through unchanged. Middleware that mutates the request context
// (auth, requestID) runs *before* this bridge and is preserved on the underlying request.
func bridge(h http.HandlerFunc) func(*core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        h.ServeHTTP(e.Response, e.Request)
        return nil
    }
}
```

Replace `NewRouter` with `RegisterRoutes`:

```go
type RoutesDeps struct {
    AuthProvider auth.AuthProvider
    APIKeySvc    APIKeyValidator   // satisfies ValidateAPIKey(ctx, key) (userID, scope, error)
    CSRFStore    *csrfStore         // session→csrf-token store; see Task 14 Step 5
    Config       *config.Config
    TaskSvc      *service.TaskService
    ProjectSvc   *service.ProjectService
    AreaSvc      *service.AreaService
    SectionSvc   *service.SectionService
    TagSvc       *service.TagService
    LocationSvc  *service.LocationService
    ChecklistSvc *service.ChecklistService
    ActivitySvc  *service.ActivityService
    EventStore   *event.EventStore
    Bus          *event.Bus
}

func RegisterRoutes(se *core.ServeEvent, deps RoutesDeps) {
    // Build handlers (existing constructors are unchanged)
    taskH := NewTaskHandler(deps.TaskSvc)
    projectH := NewProjectHandler(deps.ProjectSvc)
    // ... etc.

    authMW := requireAuth(deps.AuthProvider, deps.APIKeySvc)

    // Public routes
    se.Router.GET("/health", bridge(healthHandler))
    se.Router.POST("/auth/login", bridge(authH.Login))
    se.Router.POST("/auth/register", bridge(authH.Register))
    se.Router.POST("/auth/refresh", bridge(authH.Refresh))
    se.Router.GET("/auth/providers", bridge(authH.Providers))

    // Protected routes — wrap each bridge with auth middleware
    protect := func(h http.HandlerFunc) func(*core.RequestEvent) error {
        return bridge(func(w http.ResponseWriter, r *http.Request) {
            authMW(http.HandlerFunc(h)).ServeHTTP(w, r)
        })
    }

    se.Router.GET("/tasks", protect(taskH.List))
    se.Router.POST("/tasks", protect(taskH.Create))
    se.Router.GET("/tasks/{id}", protect(taskH.Get))
    se.Router.PATCH("/tasks/{id}", protect(taskH.Patch))
    se.Router.DELETE("/tasks/{id}", protect(taskH.Delete))
    // ... repeat for projects, areas, sections, tags, locations, checklist, activities, views, sync, sse
}
```

The wrapper preserves the entire existing handler surface — no handler signature changes, no test fixture rewrites beyond what Task 6 already covers. Admin routes in Task 14 use the same `bridge` and `protect` helpers (with `requireAdmin` substituted for `requireAuth`), so there is **only one routing primitive in the codebase**.

- [ ] **Step 3: Verify build + tests**

Run: `make check`
Expected: Build passes, tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/api/
git commit -m "feat(api): restructure router for PocketBase, dual auth middleware"
```

---

## Task 12: Auth Wrapper Endpoints

**Files:**
- Modify: `internal/api/auth.go`

- [ ] **Step 1: Rewrite auth handlers to use AuthProvider**

```go
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        RespondError(w, http.StatusBadRequest, "invalid body")
        return
    }

    token, user, err := h.auth.AuthWithPassword(body.Email, body.Password)
    if err != nil {
        RespondError(w, http.StatusUnauthorized, "invalid credentials")
        return
    }

    RespondJSON(w, http.StatusOK, map[string]any{
        "token": token,
        "user": map[string]any{
            "id":    user.ID,
            "email": user.Email,
            "name":  user.Name,
            "role":  user.Role,
        },
    })
}

func (h *AuthHandler) Providers(w http.ResponseWriter, r *http.Request) {
    RespondJSON(w, http.StatusOK, h.config.EnabledProviders())
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
    token := extractBearerToken(r)
    newToken, err := h.auth.RefreshToken(token)
    if err != nil {
        RespondError(w, http.StatusUnauthorized, "refresh failed")
        return
    }
    RespondJSON(w, http.StatusOK, map[string]string{"token": newToken})
}
```

- [ ] **Step 2: Add `/auth/providers` endpoint**

Register: `GET /auth/providers` → `h.Providers` (public, no auth required).

- [ ] **Step 3: Commit**

```bash
git add internal/api/auth.go
git commit -m "feat(auth): rewrite auth handlers to use PocketBase adapter"
```

---

## Task 13: Sync + SSE Scoping

**Files:**
- Modify: `internal/api/sync.go`
- Modify: `internal/event/stream.go`

- [ ] **Step 1: Scope delta sync endpoint**

Already partially done in Task 6. Verify the full flow:
1. `UserIDFromContext` extracts the authenticated user
2. `DeltasSince` takes `(cursor, userID)` and returns only that user's deltas
3. Client cursor advances through their own events only

- [ ] **Step 2: Scope SSE stream**

Already partially done in Task 6. Verify the domain event struct carries `UserID` and the SSE handler filters on it.

- [ ] **Step 3: Write integration test**

```go
func TestDeltaSyncIsolation(t *testing.T) {
    // Setup two users
    // User A creates a task → delta event with user_id = A
    // User B pulls deltas → should get 0 events
    // User A pulls deltas → should get 1 event
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/api/sync.go internal/event/ internal/api/sync_test.go
git commit -m "feat(sync): user-scoped delta sync and SSE stream filtering"
```

---

## Task 14: Web Admin UI

**Files:**
- Create: `internal/api/admin.go`
- Create: `internal/api/admin_templates/layout.html`
- Create: `internal/api/admin_templates/login.html`
- Create: `internal/api/admin_templates/dashboard.html`
- Create: `internal/api/admin_templates/users.html`
- Create: `internal/api/admin_templates/user_form.html`
- Create: `internal/api/admin_templates/user_edit.html`

- [ ] **Step 1: Write admin handler with template rendering**

```go
package api

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/atask/atask/internal/auth"
)

//go:embed admin_templates/*.html
var adminFS embed.FS

type AdminHandler struct {
	auth      auth.AuthProvider
	templates *template.Template
}

func NewAdminHandler(authProvider auth.AuthProvider) *AdminHandler {
	tmpl := template.Must(template.ParseFS(adminFS, "admin_templates/*.html"))
	return &AdminHandler{auth: authProvider, templates: tmpl}
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	users, total, _ := h.auth.ListUsers("", 1, 100)
	h.templates.ExecuteTemplate(w, "dashboard.html", map[string]any{
		"UserCount": total,
		"Users":     users,
	})
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, total, _ := h.auth.ListUsers("", 1, 50)
	h.templates.ExecuteTemplate(w, "users.html", map[string]any{
		"Users": users,
		"Total": total,
	})
}

func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.templates.ExecuteTemplate(w, "user_form.html", nil)
		return
	}
	// POST: create user via AuthProvider
	r.ParseForm()
	_, err := h.auth.CreateUser(
		r.FormValue("email"),
		r.FormValue("password"),
		r.FormValue("name"),
		r.FormValue("role"),
	)
	if err != nil {
		h.templates.ExecuteTemplate(w, "user_form.html", map[string]any{"Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}
```

- [ ] **Step 2: Write base layout template**

```html
{{/* admin_templates/layout.html */}}
{{define "layout"}}
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>atask admin</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; color: #1a1a1a; }
        nav { margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 1px solid #e0e0e0; }
        nav a { margin-right: 1rem; color: #0066cc; text-decoration: none; }
        table { width: 100%; border-collapse: collapse; margin: 1rem 0; }
        th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #e0e0e0; }
        .btn { display: inline-block; padding: 0.4rem 1rem; background: #0066cc; color: white; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; font-size: 0.9rem; }
        .btn-danger { background: #cc3333; }
        .btn-secondary { background: #666; }
        input, select { padding: 0.4rem; border: 1px solid #ccc; border-radius: 4px; width: 100%; margin-bottom: 0.5rem; }
        label { display: block; margin-top: 0.5rem; font-weight: 600; font-size: 0.9rem; }
        .error { color: #cc3333; margin-bottom: 1rem; }
        h1 { margin-bottom: 1rem; }
        .stat { font-size: 2rem; font-weight: 700; }
    </style>
</head>
<body>
    <nav>
        <strong>atask admin</strong>
        <a href="/admin/">Dashboard</a>
        <a href="/admin/users">Users</a>
        <a href="/admin/logout">Logout</a>
    </nav>
    {{template "content" .}}
</body>
</html>
{{end}}
```

- [ ] **Step 3: Write remaining templates (dashboard, users, user_form, user_edit, login)**

Each template extends `layout.html` via `{{template "layout" .}}` and defines `{{define "content"}}...{{end}}`. Keep them minimal — functional forms and tables.

- [ ] **Step 4: Register admin routes via the bridge helper from Task 11**

Use the same `bridge` and `protect` helpers introduced in Task 11 — the admin UI is just another set of `http.HandlerFunc`s. There is only one routing primitive in the codebase.

In `RegisterRoutes` (added in Task 11), append:

```go
adminMW := requireAdmin(deps.AuthProvider) // verifies cookie session + role=admin
adminProtect := func(h http.HandlerFunc) func(*core.RequestEvent) error {
    return bridge(func(w http.ResponseWriter, r *http.Request) {
        adminMW(http.HandlerFunc(h)).ServeHTTP(w, r)
    })
}

// Public admin login page
se.Router.GET("/admin/login", bridge(adminH.LoginPage))
se.Router.POST("/admin/login", bridge(adminH.LoginSubmit))
se.Router.GET("/admin/logout", bridge(adminH.Logout))

// Protected admin routes
se.Router.GET("/admin/", adminProtect(adminH.Dashboard))
se.Router.GET("/admin/users", adminProtect(adminH.ListUsers))
se.Router.GET("/admin/users/new", adminProtect(adminH.CreateUser))
se.Router.POST("/admin/users/new", adminProtect(adminH.CreateUser))
se.Router.GET("/admin/users/{id}", adminProtect(adminH.EditUser))
se.Router.POST("/admin/users/{id}", adminProtect(adminH.EditUser))
```

Login sets `HttpOnly`, `Secure`, `SameSite=Strict` cookie. CSRF protection is wired in Step 5 below.

- [ ] **Step 5: Implement CSRF protection (concrete)**

Spec §5.2 requires CSRF on all mutation forms. Implementation:

1. **Mint a CSRF token on login.** When `LoginSubmit` succeeds, generate a 32-byte random token, store it in a server-side session map keyed by the session ID (the cookie value), and surface it to templates via the request context.

```go
// internal/api/admin_csrf.go
package api

import (
    "crypto/rand"
    "encoding/hex"
    "net/http"
    "sync"
)

type csrfStore struct {
    mu     sync.RWMutex
    tokens map[string]map[string]struct{} // sessionID → set of valid CSRF tokens
}

func NewCSRFStore() *csrfStore { return &csrfStore{tokens: make(map[string]map[string]struct{})} }

// issue returns a fresh token AND adds it to the session's valid-token set.
// Multiple concurrent forms (e.g., two browser tabs) get distinct tokens that
// are all valid until consumed. This avoids the "second tab invalidates first"
// pitfall of single-token-per-session designs.
func (s *csrfStore) issue(sessionID string) string {
    buf := make([]byte, 32)
    rand.Read(buf)
    tok := hex.EncodeToString(buf)
    s.mu.Lock()
    if s.tokens[sessionID] == nil {
        s.tokens[sessionID] = make(map[string]struct{})
    }
    s.tokens[sessionID][tok] = struct{}{}
    s.mu.Unlock()
    return tok
}

// verify checks that the presented token is currently valid for this session
// AND consumes it (one-time use) so a captured token can't be replayed.
// Returns true on first valid presentation, false on any subsequent reuse.
func (s *csrfStore) verify(sessionID, presented string) bool {
    s.mu.Lock()
    defer s.mu.Unlock()
    set, ok := s.tokens[sessionID]
    if !ok {
        return false
    }
    if _, valid := set[presented]; !valid {
        return false
    }
    delete(set, presented) // consume — single-use
    return true
}

func (s *csrfStore) clear(sessionID string) {
    s.mu.Lock()
    delete(s.tokens, sessionID)
    s.mu.Unlock()
}
```

**Note on session-ID rotation (session-fixation defense):** The admin login handler must mint a *fresh* session ID after authenticating, not reuse a session ID the unauthenticated browser already had. This defeats session-fixation attacks where an attacker plants a session cookie before login and re-uses it after. Concretely, in `LoginSubmit`:

```go
func (h *AdminHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
    // ... validate email/password against AuthProvider ...

    // SECURITY: rotate the session ID at the auth boundary. If the unauthenticated
    // browser already had an admin_session cookie, discard it and mint a new one
    // tied to the now-authenticated identity.
    if oldSession, err := r.Cookie("admin_session"); err == nil {
        h.csrfStore.clear(oldSession.Value)
    }
    newSessionID := generateSessionID() // 32 bytes, base64url
    h.sessions.Set(newSessionID, userID) // server-side session→user map

    http.SetCookie(w, &http.Cookie{
        Name:     "admin_session",
        Value:    newSessionID,
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        Path:     "/admin/",
        MaxAge:   3600 * 8, // 8h
    })
    http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}
```

2. **Embed the token in every form template.** Add a hidden input to `user_form.html`, `user_edit.html`, and any future mutation form:

```html
<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
```

The `Dashboard`, `ListUsers`, `CreateUser` (GET), `EditUser` (GET) handlers must inject `CSRFToken` into the template data map by calling `store.issue(sessionID)` (or returning the existing token if already issued for this session).

3. **Verify on every mutation.** Add a middleware applied to all admin POST routes:

```go
func requireCSRF(store *csrfStore) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method != http.MethodPost {
                next.ServeHTTP(w, r)
                return
            }
            sessionID, err := r.Cookie("admin_session")
            if err != nil {
                http.Error(w, "no session", http.StatusForbidden)
                return
            }
            if err := r.ParseForm(); err != nil {
                http.Error(w, "bad form", http.StatusBadRequest)
                return
            }
            if !store.verify(sessionID.Value, r.FormValue("csrf_token")) {
                http.Error(w, "csrf mismatch", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

Wire it into `adminProtect`:

```go
adminProtect := func(h http.HandlerFunc) func(*core.RequestEvent) error {
    return bridge(func(w http.ResponseWriter, r *http.Request) {
        adminMW(requireCSRF(deps.CSRFStore)(http.HandlerFunc(h))).ServeHTTP(w, r)
    })
}
```

4. **Logout clears the CSRF tokens** for the session ID before destroying the cookie.

5. **Failure-path re-render must mint a fresh token.** Single-use tokens are consumed on `verify` regardless of whether the form's business logic ultimately succeeded — so any handler that displays a re-rendered form after a non-validation failure (e.g., `CreateUser` returning the form with an error message because the email is already taken) must call `csrfStore.issue(sessionID)` and inject the new token into the template, or the user's next submit will 403. The pattern:

```go
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        h.renderForm(w, r, nil) // helper mints a fresh token
        return
    }
    // POST: CSRF middleware already verified+consumed the token.
    if _, err := h.auth.CreateUser(...); err != nil {
        h.renderForm(w, r, &formErr{Message: err.Error()}) // mints a NEW token for the retry
        return
    }
    http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *AdminHandler) renderForm(w http.ResponseWriter, r *http.Request, formErr *formErr) {
    sessionID, _ := r.Cookie("admin_session")
    csrfToken := h.csrfStore.issue(sessionID.Value)
    h.templates.ExecuteTemplate(w, "user_form.html", map[string]any{
        "CSRFToken": csrfToken,
        "Error":     formErr,
    })
}
```

Note: this does not protect against browser back-button-then-resubmit of the *previous* form (the token in the back-cached HTML is already consumed). That trade-off is intentional — single-use tokens are the security primitive — but the user-facing "your session expired, please retry" message in `requireCSRF`'s 403 response should make this clear.

- [ ] **Step 6: Test CSRF protection**

```go
// internal/api/admin_csrf_test.go
func TestAdminCSRF_RejectsPostWithoutToken(t *testing.T) {
    mux := setupAdminTestServer(t, "admin-user")
    // login to get a session cookie
    sessionCookie := loginAsAdmin(t, mux, "admin@test.com", "pw")

    // POST to /admin/users/new without csrf_token
    body := strings.NewReader("email=foo@bar.com&name=Foo&role=user")
    req := httptest.NewRequest(http.MethodPost, "/admin/users/new", body)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(sessionCookie)
    w := httptest.NewRecorder()
    mux.ServeHTTP(w, req)
    if w.Code != http.StatusForbidden {
        t.Errorf("expected 403, got %d", w.Code)
    }
}

func TestAdminCSRF_AcceptsPostWithValidToken(t *testing.T) {
    // GET /admin/users/new → parse hidden csrf_token from HTML
    // POST /admin/users/new with the token → expect 303 redirect
}

func TestAdminCSRF_RejectsTokenReuse(t *testing.T) {
    // Tokens are single-use (defends against captured-token replay).
    // First POST with a valid token → 303
    // Second POST with the SAME token → 403
}

func TestAdminCSRF_ConcurrentTabsBothWork(t *testing.T) {
    // Tab A opens user_form.html → token_A
    // Tab B opens user_form.html → token_B (different)
    // POST from Tab A with token_A → 303
    // POST from Tab B with token_B → 303 (still valid)
}

func TestAdminLogin_RotatesSessionID(t *testing.T) {
    // Plant an admin_session cookie before login.
    // POST /admin/login with valid creds.
    // The Set-Cookie response MUST issue a different admin_session value
    // (defeats session-fixation).
}
```

Run: `go test ./internal/api/ -run TestAdminCSRF -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/api/admin.go internal/api/admin_csrf.go internal/api/admin_csrf_test.go internal/api/admin_templates/
git commit -m "feat(admin): web admin UI with Go templates + CSRF protection"
```

---

## Task 15: CLI Bootstrap Commands

**Files:**
- Modify: `cmd/atask/main.go`

- [ ] **Step 1: Add admin create-user command**

```go
func registerAdminCommands(app *pocketbase.PocketBase) {
    adminCmd := &cobra.Command{Use: "admin", Short: "Admin commands"}

    createUserCmd := &cobra.Command{
        Use:   "create-user",
        Short: "Create a new user",
        RunE: func(cmd *cobra.Command, args []string) error {
            email, _ := cmd.Flags().GetString("email")
            name, _ := cmd.Flags().GetString("name")
            role, _ := cmd.Flags().GetString("role")

            // Prompt for password
            fmt.Print("Password: ")
            pw, _ := term.ReadPassword(int(os.Stdin.Fd()))
            fmt.Println()

            adapter := auth.NewPBAdapter(app)
            user, err := adapter.CreateUser(email, string(pw), name, role)
            if err != nil {
                return err
            }
            fmt.Printf("Created user: %s (%s) role=%s\n", user.Email, user.ID, user.Role)
            return nil
        },
    }
    createUserCmd.Flags().String("email", "", "User email (required)")
    createUserCmd.Flags().String("name", "", "User name")
    createUserCmd.Flags().String("role", "user", "User role (user or admin)")
    createUserCmd.MarkFlagRequired("email")

    assignDataCmd := &cobra.Command{
        Use:   "assign-data",
        Short: "Assign orphaned data to a user",
        RunE: func(cmd *cobra.Command, args []string) error {
            userID, _ := cmd.Flags().GetString("to")
            // Open domain DB, UPDATE all rows WHERE user_id = '' SET user_id = userID
            // across all 11 domain tables + 2 event tables
            fmt.Printf("Assigned all orphaned data to user %s\n", userID)
            return nil
        },
    }
    assignDataCmd.Flags().String("to", "", "Target user ID (required)")
    assignDataCmd.MarkFlagRequired("to")

    adminCmd.AddCommand(createUserCmd, assignDataCmd)
    app.RootCmd.AddCommand(adminCmd)
}
```

- [ ] **Step 2: Commit**

```bash
git add cmd/atask/main.go
git commit -m "feat(cli): add admin create-user and assign-data commands"
```

---

## Task 16: API Key Scope Enforcement (Middleware Only)

**Note (Codex review, 2026-04-29):** The original Task 16 added `scope`/`expires_at` columns to `api_keys` by editing the already-committed migration 005. That work has moved to Task 1.5 (migration 006), where it can land cleanly. This task is now narrowly scoped to *middleware enforcement* of the columns added in 1.5.

**Files:**
- Modify: `internal/api/middleware.go`

- [ ] **Step 1: Enforce scope in the API key auth path**

In the auth middleware from Task 11, after validating an API key, branch on the loaded `scope`:

```go
// scope check: api_keys.scope is loaded by ValidateAPIKey
switch scope {
case "read":
    if r.Method != http.MethodGet {
        RespondError(w, http.StatusForbidden, "api key has read-only scope")
        return
    }
case "read_write":
    // domain endpoints OK; admin endpoints rejected below
case "admin":
    // all endpoints OK
default:
    RespondError(w, http.StatusForbidden, "unknown scope")
    return
}

// Admin web UI is cookie-based — bearer/api-key auth does not apply.
// Admin *API* endpoints (future) check for scope == "admin".
```

- [ ] **Step 2: Test scope enforcement**

```go
func TestAPIKeyScope_ReadOnlyRejectsPost(t *testing.T) {
    // mint key with scope=read; POST /tasks → expect 403
}

func TestAPIKeyScope_ReadWriteAllowsPostAndGet(t *testing.T) {
    // mint key with scope=read_write; POST /tasks → 201; GET /tasks → 200
}

func TestAPIKeyScope_ExpiredRejected(t *testing.T) {
    // mint key with expires_at in the past; any request → 401
    // (expiry is enforced inside ValidateAPIKey via the SQL predicate from Task 1.5 Step 3)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/api/middleware.go internal/api/middleware_scope_test.go
git commit -m "feat(auth): API key scope enforcement (read / read_write / admin)"
```

---

## Task 17: Invite Flow

**Files:**
- Create: `internal/store/queries/invites.sql`
- Create: `internal/api/invite.go`

- [ ] **Step 1: Write invite queries**

```sql
-- internal/store/queries/invites.sql

-- name: CreateInvite :one
INSERT INTO invites (id, email, role, token, created_by, created_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetInviteByToken :one
SELECT * FROM invites
WHERE token = ? AND claimed_at IS NULL AND expires_at > datetime('now');

-- name: ClaimInvite :exec
UPDATE invites SET claimed_at = ? WHERE id = ?;

-- name: ListInvites :many
SELECT * FROM invites ORDER BY created_at DESC;
```

- [ ] **Step 2: Write invite handler**

```go
// internal/api/invite.go
func (h *AuthHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
    // Admin-only: create invite token for an email
    // Returns invite URL: {BASE_URL}/invite/{token}
}

func (h *AuthHandler) ClaimInvite(w http.ResponseWriter, r *http.Request) {
    // Public: validate invite token, create user (or link OAuth), mark claimed
}
```

Register endpoint is gated on invite token when `REGISTRATION_OPEN=false`:
```go
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    if !h.config.RegistrationOpen {
        inviteToken := body.InviteToken
        if inviteToken == "" {
            RespondError(w, 403, "registration closed — invite required")
            return
        }
        // Validate invite token...
    }
    // ... create user via AuthProvider
}
```

- [ ] **Step 3: Regenerate sqlc + commit**

```bash
git add internal/store/queries/invites.sql internal/api/invite.go internal/store/sqlc/
git commit -m "feat(auth): invite flow for closed-registration servers"
```

---

## Task 18: Docker + Deployment Config (was Task 16)

**Files:**
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`
- Create: `.env.example`

- [ ] **Step 1: Update Dockerfile**

Replace `DB_PATH` assumption with `DATA_DIR`. PocketBase manages its own files inside this directory.

```dockerfile
# Only change: ENTRYPOINT args
ENTRYPOINT ["/app/atask", "serve"]
```

- [ ] **Step 2: Update docker-compose.yml**

Replace with the version from the spec (Section 7.1): `DATA_DIR`, `REGISTRATION_OPEN`, OAuth env vars, `BASE_URL`.

- [ ] **Step 3: Create .env.example**

```bash
# Required
DATA_DIR=/app/data
BASE_URL=https://your-server.com

# Auth
REGISTRATION_OPEN=false

# OAuth (optional)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
```

- [ ] **Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml .env.example
git commit -m "feat(deploy): update Docker config for PocketBase + multi-user"
```

---

## Task 19: Tauri Auth — Rust Commands + Keychain (Hardened)

**Why this task was reworked (Codex review, 2026-04-29):** The original draft (a) wrote `auth_token` to the unencrypted SQLite settings table, contradicting spec §4.2's explicit "no token in plaintext" decision; (b) only fixed 401 handling on the *flush* path, leaving the *pull* and *entity-fetch* paths to silently advance the cursor on 401 and lose inbound changes; (c) used a single global `sync_cursor` key, which replays/skips deltas across account switches; (d) duplicated auth-header selection logic per call site, inviting drift. This rewrite addresses all four.

**Hard rules enforced by this task:**
1. **No auth token of any kind in Tauri SQLite.** Refresh token in OS keychain only; access token in an in-memory `Mutex<Option<String>>` held by Tauri `State`.
2. **All three sync paths share one `auth_header()` helper** — no duplication.
3. **No sync path advances the cursor on a 401.** The cursor advances only after a successful pull *and* successful entity fetches.
4. **Sync cursor is keyed by `(server_url, user_id)`** so account switching does not corrupt cursors.

**Files:**
- Create: `atask-v4/src-tauri/src/auth.rs`
- Modify: `atask-v4/src-tauri/src/lib.rs` (register state + commands)
- Modify: `atask-v4/src-tauri/src/sync.rs` (auth_header, 401 handling on all 3 paths, cursor key)
- Modify: `atask-v4/src-tauri/src/sync_commands.rs` (read auth from State)

(No new SQL migration is needed — the existing `settings` table is key-value and stores only `user_id`, `user_email`, `user_name`, `server_url` strings as profile cache. Token columns are explicitly forbidden.)

- [ ] **Step 1: Define in-memory AuthState held by Tauri State**

```rust
// atask-v4/src-tauri/src/auth.rs
use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use tauri::State;
use crate::db::Database;

/// Tokens held only in memory. Lost on app restart by design — a refresh on
/// launch re-derives the access token from the keychain-stored refresh token.
///
/// `refresh_in_progress` is the single-flight coordinator required by spec §4.4:
/// if multiple sync paths see a 401 simultaneously, only the first acquires the
/// refresh lock and rotates the token; the rest park on the same lock and read
/// the rotated token without sending a duplicate (and now invalid) refresh call.
#[derive(Default)]
pub struct AuthTokens {
    pub access_token: Mutex<Option<String>>,
    pub refresh_in_progress: tokio::sync::Mutex<()>,
}

/// Profile cache, persisted to SQLite settings (no token material).
#[derive(Serialize, Deserialize, Clone)]
pub struct AuthState {
    pub user_id: Option<String>,
    pub user_email: Option<String>,
    pub user_name: Option<String>,
    pub server_url: Option<String>,
    /// Bool indicator only — the token itself is never serialized to the frontend.
    pub authenticated: bool,
}

const KEYRING_SERVICE: &str = "atask-refresh-token";
```

- [ ] **Step 2: Implement login (no token in SQLite)**

```rust
#[tauri::command]
pub fn login(
    db: State<Database>,
    tokens: State<AuthTokens>,
    server_url: String,
    email: String,
    password: String,
) -> Result<AuthState, String> {
    let client = reqwest::blocking::Client::new();
    let resp = client
        .post(format!("{}/auth/login", server_url))
        .json(&serde_json::json!({"email": email, "password": password}))
        .send()
        .map_err(|e| e.to_string())?;

    if !resp.status().is_success() {
        return Err(format!("Login failed: {}", resp.status()));
    }

    let body: serde_json::Value = resp.json().map_err(|e| e.to_string())?;
    // PocketBase issues a single auth token (not a separate access/refresh pair).
    // Per spec §4.2, the keychain holds the canonical token; the in-memory cache
    // holds a copy to avoid keychain reads on every sync request. Both are kept
    // in sync; the 401 handler rotates both.
    let auth_token = body["token"].as_str().ok_or("missing token")?.to_string();
    let user_id = body["user"]["id"].as_str().unwrap_or("").to_string();
    let user_email = body["user"]["email"].as_str().unwrap_or("").to_string();
    let user_name = body["user"]["name"].as_str().unwrap_or("").to_string();

    // Token → OS keychain (canonical) AND in-memory cache (copy).
    let entry = keyring::Entry::new(KEYRING_SERVICE, &user_email).map_err(|e| e.to_string())?;
    entry.set_password(&auth_token).map_err(|e| e.to_string())?;
    *tokens.access_token.lock().unwrap() = Some(auth_token);

    // Profile cache → SQLite (NO TOKEN MATERIAL)
    let conn = db.conn.lock().unwrap();
    for (key, value) in &[
        ("user_id", &user_id),
        ("user_email", &user_email),
        ("user_name", &user_name),
        ("server_url", &server_url),
    ] {
        conn.execute(
            "INSERT OR REPLACE INTO settings (key, value) VALUES (?1, ?2)",
            [&key.to_string(), *value],
        ).ok();
    }

    Ok(AuthState {
        user_id: Some(user_id),
        user_email: Some(user_email),
        user_name: Some(user_name),
        server_url: Some(server_url),
        authenticated: true,
    })
}
```

- [ ] **Step 3: Implement refresh_on_launch and logout**

```rust
#[tauri::command]
pub fn refresh_on_launch(
    db: State<Database>,
    tokens: State<AuthTokens>,
) -> Result<AuthState, String> {
    let conn = db.conn.lock().unwrap();
    let get = |k: &str| -> Option<String> {
        conn.query_row("SELECT value FROM settings WHERE key = ?1", [k], |r| r.get(0)).ok()
    };
    let server_url = match get("server_url") { Some(v) => v, None => return Ok(AuthState::default()) };
    let user_email = match get("user_email") { Some(v) => v, None => return Ok(AuthState::default()) };

    // Read current token from keychain
    let entry = keyring::Entry::new(KEYRING_SERVICE, &user_email).map_err(|e| e.to_string())?;
    let current_token = match entry.get_password() {
        Ok(t) => t,
        Err(_) => return Ok(AuthState::default()), // not signed in
    };
    drop(conn);

    // Hit /auth/refresh — PocketBase rotates the token: returns a new one and
    // invalidates the old one. We must persist the new token to the keychain
    // immediately, otherwise subsequent launches will use a now-invalid token.
    let client = reqwest::blocking::Client::new();
    let resp = client
        .post(format!("{}/auth/refresh", server_url))
        .header("Authorization", format!("Bearer {}", current_token))
        .send()
        .map_err(|e| e.to_string())?;

    if !resp.status().is_success() {
        return Ok(AuthState::default()); // surfaces "please sign in again" in UI
    }
    let body: serde_json::Value = resp.json().map_err(|e| e.to_string())?;
    let new_token = body["token"].as_str().ok_or("missing token")?.to_string();
    // Write the rotated token back to the keychain BEFORE updating the in-memory
    // cache, so a crash between the two leaves the keychain canonical.
    entry.set_password(&new_token).map_err(|e| e.to_string())?;
    *tokens.access_token.lock().unwrap() = Some(new_token);

    let conn = db.conn.lock().unwrap();
    Ok(AuthState {
        user_id: get_from(&conn, "user_id"),
        user_email: Some(user_email),
        user_name: get_from(&conn, "user_name"),
        server_url: Some(server_url),
        authenticated: true,
    })
}

fn get_from(conn: &rusqlite::Connection, key: &str) -> Option<String> {
    conn.query_row("SELECT value FROM settings WHERE key = ?1", [key], |r| r.get(0)).ok()
}

#[tauri::command]
pub fn logout(db: State<Database>, tokens: State<AuthTokens>) -> Result<(), String> {
    let conn = db.conn.lock().unwrap();

    // Clear keychain
    if let Some(email) = get_from(&conn, "user_email") {
        if let Ok(entry) = keyring::Entry::new(KEYRING_SERVICE, &email) {
            entry.delete_credential().ok();
        }
    }

    // Clear in-memory access token
    *tokens.access_token.lock().unwrap() = None;

    // Clear profile cache from settings
    for key in &["user_id", "user_email", "user_name", "server_url"] {
        conn.execute("DELETE FROM settings WHERE key = ?1", [key]).ok();
    }

    // Clear ALL per-user/per-server cursor keys (they are namespaced; see Step 5)
    conn.execute("DELETE FROM settings WHERE key LIKE 'sync_cursor:%'", []).ok();

    // Wipe local domain data
    for table in &[
        "tasks", "projects", "areas", "sections", "tags",
        "locations", "checklist_items", "activities",
        "task_tags", "project_tags", "task_links",
    ] {
        conn.execute(&format!("DELETE FROM {}", table), []).ok();
    }
    conn.execute("DELETE FROM pending_ops", []).ok();

    Ok(())
}
```

- [ ] **Step 4: Add a single auth_header() helper in sync.rs**

Replace any per-path Authorization header construction with this single helper:

```rust
// atask-v4/src-tauri/src/sync.rs
fn auth_header(tokens: &AuthTokens, api_key: &str) -> Option<String> {
    if let Some(ref t) = *tokens.access_token.lock().unwrap() {
        return Some(format!("Bearer {}", t));
    }
    if !api_key.is_empty() {
        return Some(format!("ApiKey {}", api_key));
    }
    None
}
```

Apply in all three sync paths: `flush_pending_ops`, `pull_deltas`, `fetch_entity`. No path constructs its own header.

- [ ] **Step 5: Cursor key namespacing**

The current `sync_cursor` key is global. Replace with per-user-per-server keys:

```rust
fn cursor_key(server_url: &str, user_id: &str) -> String {
    format!("sync_cursor:{}:{}", server_url, user_id)
}

fn read_cursor(conn: &rusqlite::Connection, server_url: &str, user_id: &str) -> i64 {
    let key = cursor_key(server_url, user_id);
    conn.query_row("SELECT value FROM settings WHERE key = ?1", [&key], |r| r.get::<_, String>(0))
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(0)
}

fn write_cursor(conn: &rusqlite::Connection, server_url: &str, user_id: &str, cursor: i64) {
    let key = cursor_key(server_url, user_id);
    conn.execute(
        "INSERT OR REPLACE INTO settings (key, value) VALUES (?1, ?2)",
        [&key, &cursor.to_string()],
    ).ok();
}
```

When `user_id` is unknown (anonymous local-only mode), use the empty string — that key remains stable for unauthenticated state.

- [ ] **Step 6: 401 handling on all three paths**

The previous draft only handled 401 in `flush_pending_ops`. The current `sync.rs` has three paths that hit the server:
1. `flush_pending_ops` (POST/PATCH outbound)
2. `pull_deltas` (GET delta cursor)
3. `fetch_entity` (GET entity body, called for each delta)

Each must use the same 401 contract: **never advance the cursor; never mark an op as synced; attempt single-flight refresh; pause on refresh failure**. Implement a single guard:

```rust
async fn refresh_access_token(
    tokens: &AuthTokens,
    server_url: &str,
    user_email: &str,
) -> Result<(), String> {
    // Single-flight: acquire the refresh lock. Concurrent callers (e.g. flush,
    // pull, and entity-fetch all hitting 401 at once) park here and proceed
    // serially. The first holder rotates; subsequent holders detect the rotation
    // via the cache/keychain comparison below and short-circuit.
    let _guard = tokens.refresh_in_progress.lock().await;

    let entry = keyring::Entry::new(KEYRING_SERVICE, user_email).map_err(|e| e.to_string())?;
    let keychain_token = entry.get_password().map_err(|e| e.to_string())?;

    // Has the keychain token changed while we were waiting for the lock?
    // If yes, another caller already rotated. Sync the cache from the keychain
    // and return — no need to send a second /auth/refresh call.
    let cache_matches_keychain = {
        let cache = tokens.access_token.lock().unwrap();
        match &*cache {
            Some(cached) => *cached == keychain_token,
            None => false, // empty cache always counts as "needs sync"
        }
    };
    if !cache_matches_keychain {
        // Sync the cache from the keychain. Some other caller already did the
        // network rotation; we just need to pick up its result.
        *tokens.access_token.lock().unwrap() = Some(keychain_token);
        return Ok(());
    }

    // Cache and keychain agree — we are the rotator. PocketBase rotates: passing
    // the old token yields a new token and invalidates the old.
    let resp = reqwest::Client::new()
        .post(format!("{}/auth/refresh", server_url))
        .header("Authorization", format!("Bearer {}", keychain_token))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    if !resp.status().is_success() {
        return Err(format!("refresh failed: {}", resp.status()));
    }
    let body: serde_json::Value = resp.json().await.map_err(|e| e.to_string())?;
    let new_token = body["token"].as_str().ok_or("missing token")?.to_string();
    // Persist to keychain BEFORE updating the in-memory cache. If we crash
    // between the two writes, the keychain remains the source of truth and
    // the next launch will pick up the rotated token.
    entry.set_password(&new_token).map_err(|e| e.to_string())?;
    *tokens.access_token.lock().unwrap() = Some(new_token);
    Ok(())
    // _guard drops here, releasing the single-flight lock.
}

/// Returns Ok(true) if the caller should retry; Ok(false) if not authenticated;
/// Err if refresh failed and sync should pause.
async fn handle_401(
    tokens: &AuthTokens,
    server_url: &str,
    user_email: Option<&str>,
) -> Result<bool, String> {
    let email = match user_email {
        Some(e) => e,
        None => return Ok(false), // anonymous local mode — nothing to refresh
    };
    refresh_access_token(tokens, server_url, email).await.map(|_| true)
}
```

In each of the three paths:

```rust
// flush_pending_ops:
if resp.status() == reqwest::StatusCode::UNAUTHORIZED {
    match handle_401(&tokens, &server_url, user_email.as_deref()).await {
        Ok(true) => continue,                         // retry the op (do NOT mark synced)
        Ok(false) => return,                          // anonymous; nothing to do
        Err(e) => { set_last_sync_error(conn, &format!("Authentication expired: {e}")); return; }
    }
}

// pull_deltas:
if resp.status() == reqwest::StatusCode::UNAUTHORIZED {
    match handle_401(&tokens, &server_url, user_email.as_deref()).await {
        Ok(true) => continue,                         // retry the pull (do NOT advance cursor)
        Ok(false) => return,
        Err(e) => { set_last_sync_error(conn, &format!("Authentication expired: {e}")); return; }
    }
}

// fetch_entity (inside the per-delta loop):
if resp.status() == reqwest::StatusCode::UNAUTHORIZED {
    match handle_401(&tokens, &server_url, user_email.as_deref()).await {
        Ok(true) => continue,                         // retry this fetch (do NOT advance cursor past this delta)
        Ok(false) => return,
        Err(e) => { set_last_sync_error(conn, &format!("Authentication expired: {e}")); return; }
    }
}
```

**Critical:** The cursor advancement at the end of `pull_deltas` must be moved *inside* the per-delta loop's success arm — only advance after each entity is successfully fetched and applied. A 401 mid-batch must leave the cursor at the last fully-applied delta, never beyond it.

- [ ] **Step 7: Register state + commands in lib.rs**

```rust
// atask-v4/src-tauri/src/lib.rs
.manage(crate::auth::AuthTokens::default())
.invoke_handler(tauri::generate_handler![
    // ... existing commands ...
    crate::auth::login,
    crate::auth::logout,
    crate::auth::refresh_on_launch,
])
```

- [ ] **Step 8: Verify the SQLite-token guard via grep**

Run: `grep -n "auth_token" atask-v4/src-tauri/src/`
Expected: zero matches in `db.rs`, `sync.rs`, `auth.rs` settings writes. The only references should be field names on the in-memory `AuthTokens` struct, never SQL.

- [ ] **Step 9: Run Rust tests**

Run: `cargo test --manifest-path atask-v4/src-tauri/Cargo.toml`
Expected: All tests pass; new tests for `auth_header()`, `cursor_key()`, and `handle_401()` cover the contract.

- [ ] **Step 10: Commit**

```bash
git add atask-v4/src-tauri/src/auth.rs atask-v4/src-tauri/src/sync.rs atask-v4/src-tauri/src/lib.rs
git commit -m "feat(tauri): keychain-only refresh token, in-memory access, 3-path 401 handling, namespaced cursor"
```

---

## Task 20: Tauri Login UI (Settings Page)

**Files:**
- Create: `atask-v4/src/store/auth.ts`
- Create: `atask-v4/src/components/LoginPanel.tsx`
- Modify: `atask-v4/src/views/SettingsView.tsx`
- Modify: `atask-v4/src/hooks/useTauri.ts`

- [ ] **Step 1: Create auth store**

```typescript
// atask-v4/src/store/auth.ts
import { atom, computed } from 'nanostores';

// IMPORTANT: No token field. The auth token never crosses the Tauri IPC
// boundary into the frontend — it lives only in the Rust-side AuthTokens
// state and the OS keychain. The frontend tracks identity via `authenticated`
// (a boolean) and the user profile fields, which are safe to expose.
export interface AuthState {
  authenticated: boolean;
  userId: string | null;
  userEmail: string | null;
  userName: string | null;
  serverUrl: string | null;
}

export const $authState = atom<AuthState>({
  authenticated: false,
  userId: null,
  userEmail: null,
  userName: null,
  serverUrl: null,
});

export const $isAuthenticated = computed($authState, (s) => s.authenticated);
export const $currentUser = computed($authState, (s) =>
  s.userId ? { id: s.userId, email: s.userEmail, name: s.userName } : null,
);
```

- [ ] **Step 2: Add Tauri invoke wrappers**

```typescript
// In useTauri.ts — add:
export async function login(serverUrl: string, email: string, password: string) {
  return invoke<AuthState>('login', { serverUrl, email, password });
}
export async function logout() {
  return invoke<void>('logout');
}
export async function getAuthState() {
  return invoke<AuthState>('get_auth_state');
}
export async function getProviders(serverUrl: string) {
  const resp = await fetch(`${serverUrl}/auth/providers`);
  return resp.json() as Promise<Record<string, boolean>>;
}
```

- [ ] **Step 3: Create LoginPanel component**

```tsx
// atask-v4/src/components/LoginPanel.tsx
import { useState } from 'react';
import { login } from '../hooks/useTauri';
import { $authState } from '../store/auth';
import { Button, Field } from '../ui';

interface Props {
  serverUrl: string;
}

export default function LoginPanel({ serverUrl }: Props) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const state = await login(serverUrl, email, password);
      $authState.set(state);
    } catch (err: any) {
      setError(err.toString());
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="login-panel">
      <Field label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
      <Field label="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
      {error && <div className="login-error">{error}</div>}
      <Button variant="primary" type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign In'}
      </Button>
    </form>
  );
}
```

- [ ] **Step 4: Expand SettingsView with auth section**

The existing SettingsView shows sync config. Expand it:
- If not authenticated: show server URL field + LoginPanel
- If authenticated: show user profile, connected status, sync settings, sign out button

- [ ] **Step 5: Load auth state on app startup**

In the app's root component or a `useEffect` in `App.tsx`, call `getAuthState()` on mount and populate `$authState`.

- [ ] **Step 6: Commit**

```bash
git add atask-v4/src/store/auth.ts atask-v4/src/components/LoginPanel.tsx atask-v4/src/views/SettingsView.tsx atask-v4/src/hooks/useTauri.ts
git commit -m "feat(ui): login panel, auth store, expanded settings view"
```

---

## Task 21: End-to-End Integration Test

**Files:**
- Run existing test suites with multi-user context

- [ ] **Step 1: Run Go backend tests**

Run: `make check`
Expected: All tests pass with user_id scoping.

- [ ] **Step 2: Run Rust tests**

Run: `cargo test --manifest-path atask-v4/src-tauri/Cargo.toml`
Expected: All 37+ existing tests pass. Auth commands compile.

- [ ] **Step 3: Run TypeScript tests**

Run: `cd atask-v4 && npx vitest run`
Expected: All unit tests pass.

- [ ] **Step 4: Manual smoke test**

1. `docker-compose up --build`
2. `atask admin create-user --email admin@test.com --role admin`
3. Open Tauri app → Settings → enter server URL → sign in with email/password
4. Create a task → verify it syncs
5. Create a second user → verify user isolation (user 2 can't see user 1's tasks)
6. Open `/admin/` in browser → login as admin → verify user list shows both users

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "test: multi-user integration verification pass"
```

---

## Task 22: Orphan-Data Startup Guard

**Why this task exists (Codex review, 2026-04-29):** After migration 005, every existing single-user-mode row carries `user_id = ''`. After Task 6, every domain query filters by the authenticated user's ID. Result: an admin who upgrades a single-user deployment and signs in *sees an empty task list* until they run `atask admin assign-data --to <user-id>`. The plan acknowledges this exists but offers no guardrail. This task adds an explicit startup check that warns operators before the data appears to be lost.

**Files:**
- Modify: `cmd/atask/main.go` (call the check after migrations)
- Create: `internal/store/orphan_check.go`
- Create: `internal/store/orphan_check_test.go`
- Modify: `internal/api/admin.go` (surface the count on the dashboard)

- [ ] **Step 1: Write the orphan check**

```go
// internal/store/orphan_check.go
package store

import (
    "context"
    "database/sql"
    "fmt"
)

// orphanedTables: every table that migration 005 added a user_id column to.
// Each is checked for rows with user_id = '' since those become invisible
// after Task 6 enforces user_id filtering. The list mirrors migration 005:
// 11 domain tables (incl. join tables) + 2 event tables.
var orphanedTables = []string{
    // Root domain tables
    "tasks", "projects", "areas", "sections", "tags",
    "locations", "checklist_items", "activities",
    // Join tables (orphaned rows here mean tags/links survive but their
    // ownership relationship is invisible — equally bad)
    "task_tags", "project_tags", "task_links",
    // Event tables (orphaned events would otherwise replay to no one)
    "delta_events", "domain_events",
}

// OrphanCounts returns the row count per domain table where user_id = ''.
// A non-zero count for any table indicates pre-multi-user data that has not
// been claimed via `atask admin assign-data`.
func OrphanCounts(ctx context.Context, db *sql.DB) (map[string]int, error) {
    out := make(map[string]int, len(orphanedTables))
    for _, t := range orphanedTables {
        var n int
        // #nosec G201: table names come from a constant whitelist
        q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE user_id = ''`, t)
        if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
            return nil, fmt.Errorf("count %s: %w", t, err)
        }
        if n > 0 {
            out[t] = n
        }
    }
    return out, nil
}
```

- [ ] **Step 2: Test the check**

```go
// internal/store/orphan_check_test.go
func TestOrphanCounts_EmptyDB(t *testing.T) {
    db := newTestDB(t) // applies migrations 001-006
    counts, err := OrphanCounts(context.Background(), db.Raw())
    if err != nil { t.Fatal(err) }
    if len(counts) != 0 {
        t.Errorf("expected zero orphans on fresh DB, got %v", counts)
    }
}

func TestOrphanCounts_DetectsPreMultiUserData(t *testing.T) {
    db := newTestDB(t)
    _, _ = db.Raw().Exec(`INSERT INTO tasks (id, user_id, title, "index", today_index, created_at, updated_at) VALUES ('t1', '', 'orphan task', 0, 0, datetime('now'), datetime('now'))`)
    counts, err := OrphanCounts(context.Background(), db.Raw())
    if err != nil { t.Fatal(err) }
    if counts["tasks"] != 1 {
        t.Errorf("expected 1 orphaned task, got %v", counts["tasks"])
    }
}
```

Run: `go test ./internal/store/ -run TestOrphan -v`
Expected: PASS.

- [ ] **Step 3: Wire the check into startup**

In `cmd/atask/main.go`, immediately after `db.Migrate()`, log a structured warning when orphans are present:

```go
if counts, err := store.OrphanCounts(ctx, db.Raw()); err != nil {
    slog.Warn("orphan check failed", "err", err)
} else if len(counts) > 0 {
    total := 0
    for _, n := range counts { total += n }
    slog.Warn(
        "orphaned single-user data detected",
        "tables", counts,
        "total_rows", total,
        "remediation", "atask admin assign-data --to <user-id>",
    )
}
```

- [ ] **Step 4: Surface the count on the admin dashboard**

In `internal/api/admin.go`'s `Dashboard` handler, call `store.OrphanCounts` and pass the result into the template data. Update `dashboard.html` to render a yellow banner when total > 0:

```html
{{if .OrphanTotal}}
<div style="background: #fff3cd; border: 1px solid #ffc107; padding: 1rem; margin-bottom: 1rem; border-radius: 4px;">
    <strong>{{.OrphanTotal}} orphaned rows detected.</strong>
    Pre-multi-user data is not visible to any user. Run
    <code>atask admin assign-data --to &lt;user-id&gt;</code> to claim it.
</div>
{{end}}
```

- [ ] **Step 5: Test the dashboard banner**

```go
func TestAdminDashboard_ShowsOrphanBanner(t *testing.T) {
    // setup: insert a task with user_id = ''
    // GET /admin/ as admin
    // assert response body contains "orphaned rows detected"
}
```

Run: `go test ./internal/api/ -run TestAdminDashboard -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/orphan_check.go internal/store/orphan_check_test.go cmd/atask/main.go internal/api/admin.go internal/api/admin_templates/dashboard.html
git commit -m "feat: orphan-data startup warning + admin dashboard banner"
```
