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
```

## Architecture

### Go Backend
```
cmd/atask/main.go       → entry point, wires everything
internal/
  domain/               → pure Go types, validation, zero dependencies
  store/                → SQLite, migrations, sqlc-generated queries
  event/                → delta events, domain events, pub/sub bus, SSE
  service/              → business logic (validate → persist → emit events)
  api/                  → HTTP handlers, middleware, response helpers
```

### Tauri Desktop App
```
atask-v4/
  src-tauri/src/
    commands.rs         → Tauri IPC commands (all CRUD + queue_pending_op for sync)
    sync.rs             → Sync worker (pending ops flush + delta pull)
    sync_commands.rs    → Sync Tauri commands (trigger_sync, test_connection)
    db.rs               → Database struct (Arc<Mutex<Connection>>)
    models.rs           → Rust model structs
    lib.rs              → App setup, plugin registration, system menus
  src/
    store/              → Nanostores state management
      mutations.ts      → All async Tauri-calling actions + sync notification
      selectors.ts      → Cross-domain computed views (useInbox, useTodayMorning, etc.)
      ui.ts             → UI state atoms
      tasks.ts, projects.ts, areas.ts, sections.ts, tags.ts, checklist.ts
    components/         → React components
    views/              → View components (Inbox, Today, Project, Area, etc.)
    hooks/              → useKeyboard, useSync, useDragReorder, useTauri
    lib/dates.ts        → todayLocal() — always use local timezone, never UTC
  tests/e2e/            → 28 WebDriverIO E2E test suites (~190 tests)
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

### Sync Engine
- Every mutation inserts a `pendingOps` row when sync is enabled
- `trigger_sync`: flush pending ops → pull deltas → upsert local SQLite
- Triggers: after mutations (debounced 1s), window focus, view change, 5-min fallback
- Conflict resolution: server wins (newer `updatedAt`)
- All Create endpoints accept client-provided `id` for consistent UUIDs

### Auth (Go)
- JWT: `Authorization: Bearer <token>` / API keys: `Authorization: ApiKey <key>`
- Only `/health`, `/auth/login`, `/auth/register` are public

## Testing

### Go: `go test -race ./...`
### Tauri E2E: `npx wdio run wdio.conf.ts` (from `atask-v4/`)
- Uses `browser.execute()` with raw DOM queries
- Sync integration tests need running Go server: `DB_PATH=/tmp/test.db ./bin/atask`

## Domain Model

- Projects don't nest. Sections only exist inside projects.
- Areas are permanent life categories (can be archived, not completed).
- Checklist items are not tasks (no dates, tags, or nesting).
- No explicit priorities — ordering within lists is the priority.
- Use `todayLocal()` for dates, never `new Date().toISOString().slice(0,10)`.
