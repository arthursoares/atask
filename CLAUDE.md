# CLAUDE.md — atask

## Project Overview

atask is an AI-first task manager. Go backend with SQLite, event-sourced, semantic REST API. Tauri 2 desktop app with React 19 + nanostores. Agents are first-class citizens.

## Quick Commands

### Go Backend
```bash
make build          # Build binary
make run            # Run server (default :8080)
make test           # Run tests with race detector
make lint           # golangci-lint
make check          # fmt + vet + lint + test (run before committing)
make sqlc           # Regenerate sqlc code after changing .sql files
make migrate        # Run migrations against atask.db
```

### Tauri Desktop App (`atask-v4/`)
```bash
cd atask-v4
npm run dev             # Dev server (Vite + Tauri)
npm run build           # Build frontend
npx tauri build --debug # Build debug .app bundle
npx tsc --noEmit        # Type check
npx wdio run wdio.conf.ts  # Run E2E tests (requires built .app)
npm run storybook       # Design system explorer (port 6006)
```

## Architecture

### Go Backend
```
cmd/atask/main.go       → entry point; embeds PocketBase, wires services + `admin` CLI
internal/
  config/               → typed env config (DATA_DIR, BASE_URL, REGISTRATION_OPEN, OAuth…)
  auth/                 → AuthProvider interface + PocketBase adapter (users, tokens, OAuth)
  domain/               → pure Go types (all with json tags for camelCase), validation, zero deps
  store/                → SQLite, migrations (001-008), sqlc queries, stats.go, orphan_check.go
  event/                → delta events, domain events, pub/sub bus, SSE (all user-scoped)
  service/              → business logic (validate → persist → emit); every method takes userID
  api/                  → HTTP handlers, dual-auth middleware, web admin UI (admin.go + templates)
                          PATCH endpoints for tasks/projects/areas (pre-validated, atomic)
```

**Multi-user (PocketBase hybrid).** PocketBase is embedded as the auth engine and
owns `${DATA_DIR}/pb_data/data.db` (users, tokens, OAuth). The domain layer keeps
its own `${DATA_DIR}/atask.db` — side by side, never a shared connection. Every
domain table has a denormalized `user_id` (the PocketBase record ID); all
queries/services/handlers/sync/SSE filter by it. The `AuthProvider` interface
(`internal/auth`) isolates PocketBase internals from the rest of the code. Auth
middleware accepts `Bearer <pb-token>` or `ApiKey <key>` (keys carry a
read/read_write/admin scope + optional expiry). Web admin UI at `/admin` (cookie
session, role=admin, CSRF-protected). Bootstrap the first admin with
`atask admin create-user`. Design: `docs/superpowers/specs/2026-04-10-multi-user-design.md`.

**modernc.org/sqlite gotcha:** the driver does NOT round-trip a bound `time.Time`
through SQLite date functions (`date()`, `datetime()`), and `MAX(timestamp)`
aggregates break scanning into `time.Time`. Do date bucketing / max-timestamp
work in Go over raw rows (see `store/stats.go`), and enforce API-key expiry in Go
(see `service/auth_service.go`), not via SQL predicates.

### Tauri Desktop App
```
atask-v4/
  src-tauri/src/
    commands.rs         → Tauri IPC commands (all CRUD + queue_pending_op for sync)
    sync.rs             → Sync worker (pending ops flush + delta pull + relationship sync)
    sync_commands.rs    → Sync Tauri commands (trigger_sync, test_connection, initial_sync)
    auth.rs             → Login/logout/refresh; access token in OS keychain (never SQLite)
    db.rs               → Database struct (Arc<Mutex<Connection>>), migrations (001-007)
    models.rs           → Rust model structs (Task, Project, Area, Section, Tag, Activity,
                          Location, TaskLink, TaskTag, ProjectTag, ChecklistItem)
    lib.rs              → App setup, plugin registration, system menus
  src/
    store/              → Nanostores state management
      mutations.ts      → All async Tauri-calling actions + sync notification + activity logging
      selectors.ts      → Cross-domain computed views (useInbox, useTodayMorning, etc.)
      ui.ts             → UI state atoms + drag state ($taskPointerDrag)
      activities.ts     → $activities atom + useActivitiesForTask hook
      locations.ts      → $locations atom
      taskLinks.ts      → $taskLinks atom + $linksByTaskId computed map
      tasks.ts, projects.ts, areas.ts, sections.ts, tags.ts, checklist.ts
    components/         → React components
      task-edit/        → Shared task edit components (useTaskDraft, useTaskPickers, fields)
      task-row/         → TaskRow + DropSlot primitives
      task-inline-editor/ → Split inline editor (EditorAttributeBar, EditorNotesField)
      sidebar/          → SidebarParts (NavItem, ProjectItem, SidebarRow) + SidebarIcons
      ActivityFeed.tsx  → Mutation log + comment input (wired to activities store)
      DragOverlay.tsx   → Floating drag overlay for pointer reorder
    ui/                 → Design system primitives (Button, Field, MenuList, PopoverPanel,
                          ProgressBar, SectionHeader, TagPill, EmptyState, Surface)
                          + Storybook stories for each component
    views/              → View components (Inbox, Today, Project, Area, etc.)
      area-view/        → AreaProjectList, AreaTaskList (with pointer reorder)
      project-view/     → ProjectTaskList, ProjectSectionBlock (with cross-project drag)
    hooks/
      useKeyboard.ts    → Global keyboard shortcuts
      useSync.ts        → Sync triggers (event-driven reload via "store-changed" event)
      usePointerReorder.ts → Pointer/mouse drag reorder with cross-list drop support
      useTauri.ts       → All Tauri invoke wrappers
    lib/dates.ts        → todayLocal(), tomorrowLocal() — always use local timezone
  tests/e2e/            → 29 WebDriverIO E2E test suites (~200 tests)
                          All tests use resetDatabase() + waitForAppReady() in beforeEach
```

## Key Patterns

### Store pattern (nanostores)
```typescript
const tasks = useStore($tasks);              // Subscribe to atom
import { createTask } from '../store';       // Import mutation
const id = $selectedTaskId.get();            // Imperative read
$activeView.set('inbox');                    // Direct write
```

### Adding a Tauri feature
1. Add command in `src-tauri/src/commands.rs` (with `queue_pending_op` for sync)
2. Add invoke wrapper in `src/hooks/useTauri.ts`
3. Add mutation in `src/store/mutations.ts` (with `notifySync()`)
4. If the entity syncs from server: add `upsert_*` in `sync.rs` + entity type in fetch/delete maps

### Sync Engine
- Every mutation inserts a `pendingOps` row when sync is enabled
- `trigger_sync`: flush pending ops → pull deltas → upsert local SQLite
- Triggers: after mutations (debounced 1s), window focus, view change, 5-min fallback
- Reload: single `loadAll()` via `"store-changed"` event (no double-reload)
- Conflict resolution: server wins (newer `updatedAt`)
- All Create endpoints accept client-provided `id` for consistent UUIDs
- Create ops queue POST + follow-up PATCH to sync all initial fields
- Update ops use `PATCH /entity/{id}` (Go pre-validates, then applies atomically)
- Relationship sync: `upsert_task` syncs taskTags + taskLinks; `upsert_project` syncs projectTags
- Pending op field names must match Go handler's expected JSON keys exactly

### Drag & Drop (Pointer Reorder)
- `usePointerReorder` hook replaces native HTML5 drag for task/project/area reordering
- 8px movement threshold + 150ms hold delay before drag activates
- `setPointerCapture` captures all pointer events → sidebar highlights use `document.elementFromPoint` + `$taskPointerDrag.hoverTargetId` (not pointerenter/leave)
- Cross-list drops detected via `data-sidebar-item-id` / `data-sidebar-item-kind` attributes
- All cross-list handlers must clear `sectionId: null` alongside project/area/schedule changes

### Activities
- Mutation log: auto-generated `status_change` entries for complete/cancel/reopen/schedule/project changes
- Title/notes changes debounced (one activity per editing session via `useTaskDraft`)
- Comments: user-authored `comment` type activities via ActivityFeed input
- Agent activities arrive via server sync (agents call Go API directly)
- `createMutationActivity` does NOT call `notifySync()` — called from within mutations that already do

### Auth (Go)
- Dual auth: `Authorization: Bearer <pb-token>` (PocketBase) or `ApiKey <key>`. Middleware rejects empty/disabled/expired principals; API-key scope (read/read_write/admin) is enforced per request. `ValidateToken`/`RefreshToken` only accept the `users` collection (a `_superusers` token is rejected).
- Public routes: `/health`, `/auth/login`, `/auth/register`, `/auth/refresh`, `/auth/providers`, and the admin login page. Everything else is protected. `/admin/*` uses a separate cookie session + CSRF (not Bearer).
- `AuthWithPassword` returns an identical error for unknown-email vs wrong-password (no user enumeration). Registration is invite-gated unless `REGISTRATION_OPEN=true`.

## Testing

### Go: `go test -race ./...`
- PATCH endpoint integration tests in `internal/api/patch_test.go`
- Decode validation tests in `internal/api/decode_integration_test.go`

### Tauri E2E: `npx wdio run wdio.conf.ts` (from `atask-v4/`)
- Uses `browser.execute()` with raw DOM queries
- `resetDatabase()` Tauri command clears all tables between tests
- Sync integration tests need running Go server: `DATA_DIR=/tmp/atask-test ./bin/atask` (then `atask admin create-user` to make an account, and sign in from the client)

### Rust: `cargo test --manifest-path atask-v4/src-tauri/Cargo.toml`
- Delta planning tests in sync.rs

## Domain Model

- Projects don't nest. Sections only exist inside projects.
- Areas are permanent life categories (can be archived, not completed).
- Checklist items are not tasks (no dates, tags, or nesting).
- Tasks can link to other tasks (bidirectional, via `taskLinks` table).
- Locations are named places assignable to tasks.
- Tags apply to both tasks and projects (separate join tables).
- No explicit priorities — ordering within lists is the priority.
- Use `todayLocal()` for dates, never `new Date().toISOString().slice(0,10)`.
- Use `tomorrowLocal()` when scheduling tasks for Upcoming (needs a future start date).
