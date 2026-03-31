# Sync Engine Design

**Date:** 2026-03-29
**Scope:** Bidirectional sync between atask v4 Tauri app and Go backend API

## Problem

The Tauri app works offline-only. Users need their tasks synced across devices via the existing Go backend, which already has REST endpoints, SSE event streaming, and JWT/API key auth.

## Architecture

```
React (nanostores)
  ↕ Tauri events ("store-changed")
Rust sync worker (background thread)
  ↕ HTTP + SSE
Go backend (REST API + SSE stream)
```

## First-Connect Dialog

When sync is enabled and no prior sync has occurred (`lastSyncCursor` absent), show a modal with three options:

1. **"Fresh sync from server"** — Wipe local DB, pull all entities from server. For new devices.
2. **"Merge with server"** — Fetch all server entities, upsert by ID (newer `updatedAt` wins). Push local-only records to server. For devices with offline work.
3. **"Push local to server"** — Push all local entities to server, overwriting server conflicts. For when local is the source of truth.

After first sync completes, store `lastSyncCursor` (latest delta event ID) in the settings table. Subsequent syncs are incremental.

## Pending Ops Queue

Every mutation command in `commands.rs` inserts a row into the existing `pendingOps` table after writing to SQLite:

```sql
INSERT INTO pendingOps (method, path, body, createdAt, synced)
VALUES ('POST', '/tasks', '{"title":"..."}', '2026-03-29T12:00:00Z', 0);
```

Mapping: `create_task` → POST /tasks, `update_task` → PUT /tasks/{id}, `delete_task` → DELETE /tasks/{id}, etc. for all entity types.

## Rust Background Sync Worker

Spawned on app startup when `syncEnabled && serverUrl && apiKey` are all set.

### Outbound flush (every 30s)

1. Query `SELECT * FROM pendingOps WHERE synced=0 ORDER BY id`
2. For each op: make HTTP request to Go API with `Authorization: ApiKey {key}`
3. On 2xx: `UPDATE pendingOps SET synced=1 WHERE id=?`
4. On 4xx (client error): discard the op (mark synced=1) — conflict lost, server wins
5. On network error: exponential backoff (2s → 4s → 8s → ... → 60s max)
6. Stop after 3 consecutive failures, retry on next 30s cycle

### Inbound SSE connection

1. Connect to `GET /events/stream?topics=*` with `Authorization: ApiKey {key}`
2. On event received:
   - Parse event type and entity_id from SSE data
   - Fetch full entity from Go API: `GET /{entity_type}s/{entity_id}`
   - Compare `updatedAt` — if server is newer, upsert local SQLite
   - If local has a pending op for the same entity, skip (local op will sync later)
   - Emit Tauri event `"store-changed"` to React
3. On disconnect: reconnect with exponential backoff (5s → 10s → 20s → 40s → 60s max)
4. On reconnect: pull missed deltas via `GET /sync/deltas?since={lastSyncCursor}` before resuming SSE

## Conflict Resolution

Server wins. When both local and remote have changed the same entity:
- Compare `updatedAt` timestamps
- If server is newer, overwrite local
- If local is newer, the pending op will push to server on next flush
- Pending ops that fail with 4xx are discarded (server state is authoritative)

## React Integration

### useSync hook

```typescript
// src/hooks/useSync.ts
// - Listens for Tauri "store-changed" events
// - Calls loadAll() to refresh all nanostores atoms
// - Exposes sync status for UI
```

Called from `App.tsx` alongside `useKeyboard()`.

### Sync status atom

```typescript
// Added to store/ui.ts
export const $syncStatus = atom<{
  isSyncing: boolean;
  lastSyncAt: string | null;
  lastError: string | null;
  pendingOpsCount: number;
}>({ isSyncing: false, lastSyncAt: null, lastError: null, pendingOpsCount: 0 });
```

### SyncStatusIndicator

Small icon in the Toolbar:
- Spinner during active sync
- Green dot when synced and connected
- Orange dot with pending count when ops are queued
- Red triangle on error (click to see details)

### InitialSyncDialog

Modal shown once on first sync enable. Three buttons for the three modes. Blocks UI during sync with a progress indicator.

## Tauri Commands

```rust
configure_sync(server_url: String, api_key: String) → ()
  // Start or restart the sync worker with new credentials

trigger_sync() → ()
  // Force immediate outbound flush (called after settings save)

get_sync_status() → SyncStatus { is_syncing, last_sync_at, last_error, pending_ops_count }
  // Polled by React for status display

initial_sync(mode: String) → ()
  // mode: "fresh" | "merge" | "push"
  // Performs first-connect sync, sets lastSyncCursor
```

## Files

### New (Rust)
- `src-tauri/src/sync.rs` — Sync worker: outbound flusher, SSE client, initial sync logic
- `src-tauri/src/sync_commands.rs` — Tauri command handlers for sync operations

### New (TypeScript)
- `src/hooks/useSync.ts` — Tauri event listener, loadAll trigger
- `src/components/SyncStatusIndicator.tsx` — Toolbar icon
- `src/components/InitialSyncDialog.tsx` — First-connect modal

### Modified (Rust)
- `src-tauri/src/commands.rs` — Add pendingOps inserts to all mutation commands
- `src-tauri/src/lib.rs` — Register sync commands, spawn worker on setup
- `src-tauri/Cargo.toml` — Add reqwest, reqwest-eventsource, tokio

### Modified (TypeScript)
- `src/hooks/useTauri.ts` — Add sync command wrappers
- `src/store/ui.ts` — Add $syncStatus atom
- `src/views/SettingsView.tsx` — Working test connection, sync status, initial sync trigger
- `src/components/Toolbar.tsx` — Render SyncStatusIndicator

## Dependencies (Rust)

- `reqwest` — HTTP client for Go API calls
- `reqwest-eventsource` — SSE client built on reqwest
- `tokio` — Async runtime (Tauri 2 already uses it)

## Auth

All HTTP requests use `Authorization: ApiKey {key}` header. The API key is stored in the local settings table. No JWT login flow — the user pastes their API key in Settings.

## Local-Only Guard

When `syncEnabled` is false (or serverUrl/apiKey are empty):
- Sync worker does not spawn
- No pendingOps are inserted on mutations
- No SSE connection
- App works fully offline as it does today
