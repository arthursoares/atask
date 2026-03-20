# Local-First Sync Architecture — Design Spec

> **Status:** Design only. To be implemented in Plan 3.

## Problem

The current Dioxus client treats the Go API as the source of truth. Every mutation requires an API call, and the UI waits for the response before updating. This creates visible latency and makes the app unusable offline.

## Architecture

```
┌─────────────────────────────────────────────┐
│               Dioxus UI                      │
│  (signals read from local SQLite, not API)   │
└──────────────┬──────────────────────────────┘
               │ reads/writes
┌──────────────▼──────────────────────────────┐
│           Local SQLite                       │
│  (mirrors server schema, single file)        │
│  ~/.config/atask/local.db                    │
└──────────────┬──────────────────────────────┘
               │ sync
┌──────────────▼──────────────────────────────┐
│           Sync Engine                        │
│                                              │
│  Outbound:                                   │
│    - Queue local mutations as pending ops    │
│    - POST/PUT to API in background           │
│    - Mark ops as synced on success            │
│    - Retry on failure (exponential backoff)   │
│                                              │
│  Inbound:                                    │
│    - SSE stream from /events/stream           │
│    - On event: update local SQLite            │
│    - Conflict resolution: server wins         │
│                                              │
│  Bootstrap:                                  │
│    - GET /sync/deltas?since=last_cursor       │
│    - Apply all deltas to local DB             │
│    - Then switch to SSE for live updates      │
└──────────────┬──────────────────────────────┘
               │ HTTP + SSE
┌──────────────▼──────────────────────────────┐
│           Go API Server                      │
│  (authoritative, event-sourced)              │
└─────────────────────────────────────────────┘
```

## Local Database

Mirror the server schema in a local SQLite file:

```sql
-- Same tables: tasks, projects, sections, areas, tags, checklist_items
-- Plus sync metadata:
CREATE TABLE sync_state (
    key TEXT PRIMARY KEY,
    value TEXT
);
-- Stores: last_delta_cursor, last_sse_id

CREATE TABLE pending_ops (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    method TEXT NOT NULL,        -- "POST", "PUT", "DELETE"
    path TEXT NOT NULL,          -- "/tasks/{id}/title"
    body TEXT,                   -- JSON body
    created_at DATETIME NOT NULL,
    synced INTEGER NOT NULL DEFAULT 0
);
```

## Data Flow

### Read (instant)
```
UI signal ← read from local SQLite ← no network
```

### Write (optimistic)
```
1. Write to local SQLite immediately
2. Update Dioxus signal (UI updates instantly)
3. Insert into pending_ops queue
4. Background task processes queue → API calls
5. On success: mark op as synced
6. On failure: retry with backoff
```

### Inbound sync (SSE)
```
1. SSE event arrives (entity_type, entity_id, event_type)
2. Fetch updated entity from API (GET /tasks/{id})
3. Upsert into local SQLite
4. Update Dioxus signal (UI updates)
5. If conflict with pending_op: server wins, discard local change
```

### Bootstrap (first load / reconnect)
```
1. GET /sync/deltas?since=last_cursor
2. Apply each delta to local SQLite
3. Update last_cursor
4. Start SSE stream for live updates
```

## Conflict Resolution

Simple strategy: **server wins**. If a pending local mutation conflicts with an incoming SSE event for the same entity, the SSE event overwrites the local state. The pending op is discarded.

This is acceptable because:
- Single-user app (no collaborative editing)
- Conflicts are rare (same user, one device)
- Server is authoritative

## Dependencies

- `rusqlite` — embedded SQLite for Rust
- Existing `reqwest` + SSE infrastructure
- The Go API's `/sync/deltas` endpoint (already exists)

## Migration from Current Architecture

1. Add `rusqlite` to Cargo.toml
2. Create `src/state/local_db.rs` — local SQLite wrapper
3. Create `src/state/sync.rs` — sync engine (outbound queue + inbound SSE)
4. Change all signals to read from local DB instead of API responses
5. Change all mutations to write to local DB + queue for sync
6. Remove all direct API calls from components/views
7. The data loader effect in main.rs becomes a bootstrap sync

## What This Enables

- **Instant UI** — all reads/writes are local, no network latency
- **Offline support** — app works without server, syncs when reconnected
- **Background sync** — mutations queue up and sync without blocking UI
- **Multi-device** — SSE + delta sync keeps devices in sync (future)
