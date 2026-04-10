# Multi-User Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add multi-user data isolation to the Go backend with PocketBase-powered authentication, user-scoped sync/SSE, a basic web admin UI, and Tauri client login support.

**Architecture:** PocketBase is embedded as the auth engine (users, OAuth, tokens). All domain tables gain a `user_id` column scoped to PocketBase user record IDs. A thin `AuthProvider` adapter interface isolates PocketBase internals from the rest of the codebase. The existing sqlc query layer, service layer, and event system are enhanced with per-user filtering.

**Tech Stack:** Go 1.25, PocketBase (embedded), sqlc, SQLite, Tauri 2 (Rust + React), `tauri-plugin-keychain`

**Spec:** `docs/superpowers/specs/2026-04-10-multi-user-design.md`

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

## Task 4: Query Scanning Safety-Net Test

**Files:**
- Create: `internal/store/queries/query_scope_test.go`

- [ ] **Step 1: Write the test**

```go
package queries_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// domainTables lists tables that MUST have user_id scoping in every query.
var domainTables = []string{
	"tasks", "projects", "areas", "sections", "tags",
	"locations", "checklist_items", "activities",
	"task_tags", "project_tags", "task_links",
	"delta_events", "domain_events",
}

func TestAllDomainQueriesIncludeUserID(t *testing.T) {
	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	for _, f := range files {
		if f == "auth.sql" || f == "invites.sql" {
			continue // auth queries don't need user_id scoping
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		content := string(data)

		// Split by query name comments
		queries := strings.Split(content, "-- name:")
		for _, q := range queries[1:] { // skip preamble
			lines := strings.SplitN(q, "\n", 2)
			name := strings.TrimSpace(strings.Split(lines[0], ":")[0])
			body := ""
			if len(lines) > 1 {
				body = strings.ToLower(lines[1])
			}

			// Check if query touches a domain table
			touchesDomain := false
			for _, tbl := range domainTables {
				if strings.Contains(body, tbl) {
					touchesDomain = true
					break
				}
			}
			if !touchesDomain {
				continue
			}

			if !strings.Contains(body, "user_id") {
				t.Errorf("query %q in %s touches a domain table but does not include user_id", name, f)
			}
		}
	}
}
```

- [ ] **Step 2: Run the test**

Run: `cd internal/store/queries && go test -run TestAllDomainQueriesIncludeUserID -v`
Expected: PASS — all queries include `user_id`.

- [ ] **Step 3: Commit**

```bash
git add internal/store/queries/query_scope_test.go
git commit -m "test: add query scanning safety-net for user_id scoping"
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

- [ ] **Step 10: Verify compilation**

Run: `go build ./...`
Expected: Compilation errors in API handlers (they still pass old signatures). This is expected — Task 6 fixes them.

- [ ] **Step 11: Commit**

```bash
git add internal/service/ internal/event/
git commit -m "feat(service): thread userID through all service and event methods"
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

Update all test helpers in `internal/api/patch_test.go`, `internal/api/handler_regression_test.go`, and any service tests to pass a test `userID` (e.g., `"test-user-1"`). For `setupPatchTestServer` and `setupFullTestServer`, the auth middleware can be configured to inject a default test user.

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
		// Open our domain database (shares the PocketBase SQLite file)
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
		api.RegisterRoutes(se, api.RoutesDeps{
			AuthProvider: authProvider,
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

- [ ] **Step 2: Restructure router for PocketBase integration**

Replace `NewRouter` with `RegisterRoutes` that works with PocketBase's router:

```go
type RoutesDeps struct {
    AuthProvider auth.AuthProvider
    Config       *config.Config
    TaskSvc      *service.TaskService
    // ... all services ...
}

func RegisterRoutes(se *core.ServeEvent, deps RoutesDeps) {
    // Health (public)
    se.Router.GET("/health", func(e *core.RequestEvent) error {
        return e.JSON(200, map[string]string{"status": "ok"})
    })

    // Auth wrapper endpoints (public)
    registerAuthRoutes(se, deps)

    // Protected domain routes — use middleware group
    protected := se.Router.Group("")
    protected.BindFunc(requireAuthMiddleware(deps.AuthProvider))

    // Register all domain handlers on the protected group
    registerTaskRoutes(protected, deps)
    registerProjectRoutes(protected, deps)
    // ... etc
}
```

Note: This is a significant restructure from `http.ServeMux` to PocketBase's router. The handlers need to be adapted to PocketBase's `RequestEvent` pattern or wrapped.

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

- [ ] **Step 4: Register admin routes with requireAdmin middleware**

```go
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /admin/", h.Dashboard)
    mux.HandleFunc("GET /admin/users", h.ListUsers)
    mux.HandleFunc("GET /admin/users/new", h.CreateUser)
    mux.HandleFunc("POST /admin/users/new", h.CreateUser)
    mux.HandleFunc("GET /admin/users/{id}", h.EditUser)
    mux.HandleFunc("POST /admin/users/{id}", h.EditUser)
    mux.HandleFunc("GET /admin/login", h.LoginPage)
    mux.HandleFunc("POST /admin/login", h.LoginSubmit)
    mux.HandleFunc("GET /admin/logout", h.Logout)
}
```

Admin routes use cookie-based auth with CSRF. Login sets `HttpOnly`, `Secure`, `SameSite=Strict` cookie.

- [ ] **Step 5: Commit**

```bash
git add internal/api/admin.go internal/api/admin_templates/
git commit -m "feat(admin): web admin UI with Go templates (login, users, dashboard)"
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

## Task 16: API Key Scope Enhancement

**Files:**
- Modify: `internal/store/migrations/005_multi_user.sql`
- Modify: `internal/store/queries/auth.sql`

- [ ] **Step 1: Add scope + expires_at to api_keys (in migration 005)**

Append to migration 005:

```sql
ALTER TABLE api_keys ADD COLUMN scope TEXT NOT NULL DEFAULT 'read_write';
ALTER TABLE api_keys ADD COLUMN expires_at DATETIME;
```

- [ ] **Step 2: Update auth.sql queries**

Add `scope` and `expires_at` to `CreateAPIKey` INSERT. Add expiry check to `GetAPIKeyByHash`:

```sql
-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = ? AND (expires_at IS NULL OR expires_at > datetime('now'));
```

- [ ] **Step 3: Enforce scope in middleware**

In the auth middleware (Task 11), after validating an API key, check:
- `read` scope → reject non-GET requests
- `read_write` scope → allow all domain endpoints
- `admin` scope → allow admin API endpoints too

- [ ] **Step 4: Regenerate sqlc + commit**

Run: `make sqlc`

```bash
git add internal/store/migrations/005_multi_user.sql internal/store/queries/auth.sql internal/store/sqlc/ internal/api/middleware.go
git commit -m "feat(auth): add scope + expiry to API keys with middleware enforcement"
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

## Task 19: Tauri Auth — Rust Commands + Keychain

**Files:**
- Create: `atask-v4/src-tauri/src/auth.rs`
- Create: `atask-v4/src-tauri/src/migrations/007_auth.sql`
- Modify: `atask-v4/src-tauri/src/lib.rs`
- Modify: `atask-v4/src-tauri/src/sync.rs`
- Modify: `atask-v4/src-tauri/src/sync_commands.rs`

- [ ] **Step 1: Add auth migration for Tauri local DB**

```sql
-- atask-v4/src-tauri/src/migrations/007_auth.sql
-- Auth settings columns (stored alongside existing settings)
-- Settings table already uses key-value pairs; these are new keys:
-- 'auth_token' (cached, refreshed on launch)
-- 'user_id', 'user_email', 'user_name' (profile cache)
-- No schema change needed — settings table is key-value.
```

- [ ] **Step 2: Write auth.rs with Tauri commands**

```rust
// atask-v4/src-tauri/src/auth.rs
use serde::{Deserialize, Serialize};
use tauri::State;
use crate::db::Database;

#[derive(Serialize, Deserialize, Clone)]
pub struct AuthState {
    pub token: Option<String>,
    pub user_id: Option<String>,
    pub user_email: Option<String>,
    pub user_name: Option<String>,
    pub server_url: Option<String>,
}

#[tauri::command]
pub fn login(
    db: State<Database>,
    server_url: String,
    email: String,
    password: String,
) -> Result<AuthState, String> {
    // POST to {server_url}/auth/login with email/password
    // Store token in OS keychain via keyring crate
    // Cache user profile in settings table
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
    let token = body["token"].as_str().ok_or("missing token")?.to_string();
    let user_id = body["user"]["id"].as_str().unwrap_or("").to_string();
    let user_email = body["user"]["email"].as_str().unwrap_or("").to_string();
    let user_name = body["user"]["name"].as_str().unwrap_or("").to_string();

    // Store in OS keychain
    let entry = keyring::Entry::new("atask", &user_email).map_err(|e| e.to_string())?;
    entry.set_password(&token).map_err(|e| e.to_string())?;

    // Cache in local settings
    let conn = db.conn.lock().unwrap();
    conn.execute("INSERT OR REPLACE INTO settings (key, value) VALUES ('auth_token', ?1)", [&token]).ok();
    conn.execute("INSERT OR REPLACE INTO settings (key, value) VALUES ('user_id', ?1)", [&user_id]).ok();
    conn.execute("INSERT OR REPLACE INTO settings (key, value) VALUES ('user_email', ?1)", [&user_email]).ok();
    conn.execute("INSERT OR REPLACE INTO settings (key, value) VALUES ('user_name', ?1)", [&user_name]).ok();
    conn.execute("INSERT OR REPLACE INTO settings (key, value) VALUES ('server_url', ?1)", [&server_url]).ok();

    Ok(AuthState {
        token: Some(token),
        user_id: Some(user_id),
        user_email: Some(user_email),
        user_name: Some(user_name),
        server_url: Some(server_url),
    })
}

#[tauri::command]
pub fn logout(db: State<Database>) -> Result<(), String> {
    let conn = db.conn.lock().unwrap();

    // Clear keychain
    let email: Option<String> = conn
        .query_row("SELECT value FROM settings WHERE key = 'user_email'", [], |r| r.get(0))
        .ok();
    if let Some(email) = email {
        if let Ok(entry) = keyring::Entry::new("atask", &email) {
            entry.delete_credential().ok();
        }
    }

    // Clear auth settings
    for key in &["auth_token", "user_id", "user_email", "user_name"] {
        conn.execute("DELETE FROM settings WHERE key = ?1", [key]).ok();
    }

    // Wipe local domain data
    for table in &[
        "tasks", "projects", "areas", "sections", "tags",
        "locations", "checklist_items", "activities",
        "task_tags", "project_tags", "task_links",
    ] {
        conn.execute(&format!("DELETE FROM {}", table), []).ok();
    }

    // Reset sync state
    conn.execute("DELETE FROM settings WHERE key = 'sync_cursor'", []).ok();
    conn.execute("DELETE FROM pending_ops", []).ok();

    Ok(())
}

#[tauri::command]
pub fn get_auth_state(db: State<Database>) -> Result<AuthState, String> {
    let conn = db.conn.lock().unwrap();
    let get = |key: &str| -> Option<String> {
        conn.query_row("SELECT value FROM settings WHERE key = ?1", [key], |r| r.get(0)).ok()
    };
    Ok(AuthState {
        token: get("auth_token"),
        user_id: get("user_id"),
        user_email: get("user_email"),
        user_name: get("user_name"),
        server_url: get("server_url"),
    })
}
```

- [ ] **Step 3: Update sync.rs for Bearer token + 401 handling**

In the `flush_pending_ops` function, replace the fixed `ApiKey` header with the dual-auth logic:

```rust
fn auth_header(config: &SyncConfig) -> String {
    if let Some(ref token) = config.auth_token {
        format!("Bearer {}", token)
    } else if !config.api_key.is_empty() {
        format!("ApiKey {}", config.api_key)
    } else {
        String::new()
    }
}
```

For 401 handling in the sync loop:

```rust
if resp.status() == reqwest::StatusCode::UNAUTHORIZED {
    // Do NOT mark op as synced — preserve it
    // Attempt token refresh
    if let Some(new_token) = refresh_token(config) {
        config.auth_token = Some(new_token);
        continue; // retry this op
    } else {
        // Refresh failed — pause sync
        set_last_sync_error(conn, "Authentication expired. Please sign in again.");
        return;
    }
}
```

- [ ] **Step 4: Register auth commands in lib.rs**

Add `login`, `logout`, `get_auth_state` to the Tauri command registration.

- [ ] **Step 5: Commit**

```bash
git add atask-v4/src-tauri/src/auth.rs atask-v4/src-tauri/src/sync.rs atask-v4/src-tauri/src/lib.rs
git commit -m "feat(tauri): auth commands, keychain storage, 401 handling in sync"
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

export interface AuthState {
  token: string | null;
  userId: string | null;
  userEmail: string | null;
  userName: string | null;
  serverUrl: string | null;
}

export const $authState = atom<AuthState>({
  token: null,
  userId: null,
  userEmail: null,
  userName: null,
  serverUrl: null,
});

export const $isAuthenticated = computed($authState, (s) => !!s.token);
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
