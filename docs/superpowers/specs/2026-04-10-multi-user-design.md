# Multi-User Support — Design Spec

**Date:** 2026-04-10
**Status:** Draft
**Approach:** PocketBase embedded (auth engine) + custom Go domain layer (sqlc)

---

## 1. Architecture Overview

Single Go binary with PocketBase embedded as a library. PocketBase owns identity and auth. The custom Go layer owns all domain data, event sourcing, and sync.

```
Single Go binary (PocketBase framework)
│
├── PocketBase (auth engine)
│   ├── _users, _externalAuths tables (PocketBase-managed)
│   ├── OAuth flows: Google, GitHub, Apple (PKCE, per-provider handling)
│   ├── Token lifecycle: issuance, refresh, verification
│   ├── /_/ dashboard (developer-only, locked down in production)
│   └── Password hashing, email verification
│
├── Custom routes (registered via OnServe hook)
│   ├── /auth/*        → thin wrappers delegating to PocketBase Go API
│   ├── /tasks/*       → existing handlers (sqlc-backed, user-scoped)
│   ├── /projects/*    → existing handlers (user-scoped)
│   ├── /areas/*       → ...
│   ├── /sync/deltas   → delta sync (user-scoped via user_id on events)
│   ├── /events/stream → SSE (user-scoped via user_id on events)
│   ├── /admin/*       → web admin UI (Go templates)
│   └── /views/*       → inbox, today, upcoming, etc.
│
├── Domain layer
│   ├── internal/store/   → sqlc queries + SQLite (domain tables)
│   ├── internal/service/ → business logic (all methods take userID)
│   └── internal/event/   → event sourcing, delta events, domain events
│
└── Single SQLite file
    ├── PocketBase tables: _users, _externalAuths, _admins, ...
    └── Domain tables: tasks, projects, areas, sections, tags, ...
```

**Key boundary:** PocketBase manages identity. Custom code manages domain data. They share one SQLite file but own different tables. The bridge is `user_id` — PocketBase's user record ID becomes the `user_id` foreign key in all domain tables.

---

## 2. Data Layer

### 2.1 Schema Migration (005)

`user_id TEXT NOT NULL DEFAULT ''` added to **all 11 domain tables**. No JOIN-based scoping for children — every table carries its own `user_id` for defense-in-depth.

| Table | Type | Gets `user_id` |
|-------|------|----------------|
| tasks | root | Yes |
| projects | root | Yes |
| areas | root | Yes |
| tags | root | Yes |
| locations | root | Yes |
| activities | root | Yes |
| sections | child of project | Yes |
| checklist_items | child of task | Yes |
| task_tags | join | Yes |
| project_tags | join | Yes |
| task_links | join | Yes |

```sql
-- Migration 005: multi-user data scoping
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

-- Indexes
CREATE INDEX idx_tasks_user ON tasks(user_id);
CREATE INDEX idx_projects_user ON projects(user_id);
CREATE INDEX idx_areas_user ON areas(user_id);
CREATE INDEX idx_tags_user ON tags(user_id);
CREATE INDEX idx_locations_user ON locations(user_id);
CREATE INDEX idx_sections_user ON sections(user_id);

-- Users: PocketBase manages the _users collection (email, password,
-- name, avatar, verified, etc.). Additional fields needed by atask
-- (role, disabled) are added as custom fields on the PocketBase
-- _users collection via migration or PocketBase settings.
-- The legacy `users` table from migration 001 is dropped;
-- PocketBase's _users replaces it entirely.
-- The `api_keys` table's user_id FK is updated to reference
-- PocketBase user record IDs.

-- OAuth accounts: handled by PocketBase's _externalAuths table.
-- No custom oauth_accounts table needed — PocketBase manages
-- (provider, provider_user_id) → user mapping internally.

-- Invite tokens (for closed-registration OAuth)
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
```

### 2.2 Event Tables

`user_id` added to `delta_events` at write time. Events are self-contained — no ownership lookup needed at sync time.

```sql
ALTER TABLE delta_events ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
ALTER TABLE domain_events ADD COLUMN user_id TEXT NOT NULL DEFAULT '';
```

When a service method emits a delta event, it includes the `user_id` of the entity owner. For delete events, the `user_id` is captured before the delete occurs. This makes deltas replayable without consulting current entity state.

### 2.3 Query Scoping

All sqlc queries include `WHERE user_id = ?` (or `AND user_id = ?` for by-ID lookups):

- **Root entity queries (~55):** Direct `WHERE user_id = ?`
- **Child entity queries (~16):** Direct `WHERE user_id = ?` (denormalized, no JOINs needed)
- **Join table queries (~8):** Direct `WHERE user_id = ?`
- **View queries (5):** `WHERE user_id = ?`
- **Event queries:** `WHERE user_id = ?` on delta pulls
- **Total: ~84 queries modified**

Every `INSERT` includes `user_id` in its VALUES clause.

### 2.4 Cross-Entity Validation

Service methods that reference other entities scope the lookup to the current user:

| Operation | Validation |
|-----------|-----------|
| Set task's project | `GetProject(projectID, userID)` |
| Set task's area | `GetArea(areaID, userID)` |
| Set task's section | `GetSection(sectionID, userID)` |
| Set task's location | `GetLocation(locationID, userID)` |
| Add task link | `GetTask(relatedID, userID)` |
| Create section in project | `GetProject(projectID, userID)` |
| Add tag to task/project | `GetTag(tagID, userID)` |

This prevents horizontal privilege escalation (User A moving their task into User B's project by guessing a UUID).

### 2.5 Query Scanning Test

A Go test that parses all `.sql` files in `internal/store/queries/` and verifies:
- Every `SELECT` on a domain table includes `user_id` in its WHERE clause
- Every `INSERT` on a domain table includes `user_id` in its columns
- Every `UPDATE`/`DELETE` includes `user_id` in its WHERE clause

This compensates for the lack of database-level RLS. Catches missed filters at test time.

### 2.6 Data Ownership Bootstrap

No auto-promotion. Explicit commands only:

```bash
# Create first admin user
atask admin create-user --email admin@example.com --role admin

# Assign orphaned data from single-user deployment upgrade
atask admin assign-data --to <user-id>
# Updates all rows WHERE user_id = '' to the specified user
```

---

## 3. Authentication

### 3.1 PocketBase Auth Engine

PocketBase handles (via its Go API, not HTTP):
- User record storage (`_users` collection)
- Password hashing (bcrypt) and validation
- OAuth2 flows with PKCE for Google, GitHub, Apple
- Per-provider identity: uses `(provider, sub)` as identity, not email
- Token issuance and refresh
- Email verification (optional, enabled via PocketBase settings)
- External auth linking (`_externalAuths` table)

### 3.2 Auth Middleware Bridge

Translates PocketBase auth into the existing service layer:

```go
func requireAuth(app *pocketbase.PocketBase) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractToken(r) // "Bearer <token>" or "ApiKey <key>"

            if isAPIKey(token) {
                // Existing API key validation (enhanced with scopes)
                userID, scope, err := validateAPIKey(token)
                // ... check scope, check disabled, store in context
            } else {
                // PocketBase token validation
                record, err := app.FindAuthRecordByToken(token, "auth")
                // ... check disabled, store record.Id in context
            }

            ctx := context.WithValue(r.Context(), ctxUserID, userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

The existing `UserIDFromContext(ctx)` continues working. Services and handlers don't know PocketBase exists.

### 3.3 Auth Wrapper Endpoints

Thin wrappers at the existing `/auth/` paths. The Tauri client never talks to PocketBase's API directly.

```
POST /auth/register       → creates PocketBase user record (if registration open OR valid invite)
POST /auth/login          → delegates to PocketBase, returns token
POST /auth/refresh        → delegates to PocketBase token refresh
GET  /auth/me             → returns user profile from PocketBase record
PUT  /auth/me             → updates user profile
GET  /auth/providers      → returns enabled auth methods (see 3.5)
GET  /auth/oauth/{provider}          → builds OAuth URL (via PocketBase), redirects
GET  /auth/oauth/{provider}/callback → exchanges code (via PocketBase), issues token

GET    /auth/api-keys     → list API keys for current user
POST   /auth/api-keys     → create API key (with scope + expiry)
PUT    /auth/api-keys/{id}→ rename API key
DELETE /auth/api-keys/{id}→ delete API key
```

### 3.4 API Keys for Agents

Existing `api_keys` table, enhanced:

```sql
ALTER TABLE api_keys ADD COLUMN scope TEXT NOT NULL DEFAULT 'read_write';
ALTER TABLE api_keys ADD COLUMN expires_at DATETIME;
```

Scopes: `read`, `read_write`, `admin`.

Middleware enforcement:
- `read` scope: only `GET` requests allowed
- `read_write` scope: `GET`, `POST`, `PATCH`, `DELETE` on domain endpoints
- `admin` scope: all of the above + admin endpoints

Agent keys with `admin` scope cannot reach `/admin/` web UI (which uses cookie-based auth, not bearer tokens). Admin API endpoints are separate from the admin web UI.

API key `user_id` references a PocketBase user record ID. The existing `api_keys` table's `user_id` column stores PocketBase record IDs (not the legacy `users` table IDs). During migration, the `users` table is replaced by PocketBase's `_users` collection — the `api_keys` FK needs updating accordingly. When a user is disabled via PocketBase, their API keys are also effectively disabled (middleware checks user disabled status on API key auth).

### 3.5 Provider Discovery

```
GET /auth/providers
→ { "email": true, "google": true, "github": false, "apple": false }
```

Derived from server config: if `GOOGLE_CLIENT_ID` is set, Google is enabled. The Tauri client and admin login page call this to render only available login options.

### 3.6 Registration Control + Invite Flow

```env
REGISTRATION_OPEN=false   # default for self-hosted
```

When `REGISTRATION_OPEN=false`:
- `POST /auth/register` requires a valid invite token
- OAuth login for unknown users requires a valid invite token (passed as state parameter)
- Admin creates invites via admin UI → generates a link like `https://server/invite/{token}`
- Invite link can be opened in browser (sets up password) or pasted into Tauri app (initiates OAuth or password registration)

When `REGISTRATION_OPEN=true`:
- `POST /auth/register` works without invite
- OAuth login auto-creates user on first sign-in

Invite schema:
- `email`: must match the registering user's email
- `role`: assigned on claim (default: `user`)
- `expires_at`: 7 days from creation
- `claimed_at`: set when used, prevents reuse

### 3.7 Account Linking

A logged-in user can link additional OAuth providers. This is a separate flow from login:

1. User clicks "Link Google Account" in Tauri settings
2. Tauri opens browser to `/auth/oauth/google?link=true` (includes current auth token)
3. Server verifies existing auth, initiates OAuth
4. On callback, links the OAuth identity to the existing user (no new user created)

PocketBase's `_externalAuths` table handles the `(provider, provider_user_id) → user_id` mapping.

No email-based auto-linking. Identity is always `(provider, sub)`.

---

## 4. Tauri Client

### 4.1 Login Is Opt-In

The app launches to Inbox with no auth gate. Settings page gets an expanded "Account & Sync" section:

**Not connected state:**
- Server URL field
- "Sign in with Email" → expands to email/password form
- "Sign in with Google" / "Sign in with GitHub" (dynamic, from `/auth/providers`)
- "I have an invite" → paste invite token, then choose auth method

**Connected state:**
- User name, email display
- Connected OAuth accounts (with link/unlink)
- API Keys management (list, create with scope, delete)
- Sync toggle + status
- "Sign Out" button

### 4.2 Token Storage

| Token | Storage | Rationale |
|-------|---------|-----------|
| Access token (short-lived) | In-memory (`Arc<Mutex<Option<String>>>`) | Lost on restart, refreshed on launch |
| Refresh token (long-lived) | OS keychain (`tauri-plugin-keychain`) | Survives restart, OS-level encryption |
| Server URL | Tauri SQLite settings | Non-sensitive config |
| User profile cache | Tauri SQLite settings (`user_id`, `user_email`, `user_name`) | Display when offline |

### 4.3 Sync Auth

`sync.rs` auth header selection:

```rust
fn auth_header(config: &SyncConfig) -> String {
    if let Some(token) = &config.auth_token {
        format!("Bearer {}", token)
    } else if !config.api_key.is_empty() {
        format!("ApiKey {}", config.api_key)
    } else {
        String::new()
    }
}
```

### 4.4 401 Handling

The sync worker currently treats all 4xx as permanent failure and discards pending ops. Revised behavior for 401:

1. On 401 response: **do not discard** the pending op
2. Attempt single-flight token refresh (one retry)
3. If refresh succeeds: retry the failed op with new token
4. If refresh fails: pause sync, set sync status to "auth expired", preserve all pending ops
5. Surface "Please sign in again" in the sync status UI

All other 4xx responses keep existing behavior (log and skip).

### 4.5 Account Switching

When a user signs out:

1. Revoke token server-side (PocketBase API call)
2. Clear OS keychain (refresh token)
3. Clear in-memory access token
4. Wipe local Tauri SQLite domain data (tasks, projects, etc.)
5. Reset sync cursor to 0
6. Clear pending ops queue
7. Clear user profile cache from settings
8. UI returns to "not connected" state

When a new user signs in, initial sync pulls their data fresh.

### 4.6 Pre-Login Local Data

The app is local-first and works without login. Data created before signing in lives only in local SQLite with `user_id = ''`.

**On first sign-in:** Local data created before login is **not uploaded**. The initial sync pulls the user's server-side data. Pre-login local data remains in the local database but is invisible once the user authenticates (queries filter by the authenticated `user_id`).

**Rationale:** Merging anonymous local data with server-side data requires conflict resolution that adds significant complexity. For a task manager, it's simpler to treat pre-login as a "try it out" mode. If a user needs to preserve local tasks, they can manually recreate them after login.

**Future option:** An explicit "Import local data" button that assigns `user_id` to orphaned local rows and queues them as pending ops.

---

## 5. Web Admin UI

### 5.1 Pages

Go `html/template` served at `/admin/`:

| Path | Function |
|------|----------|
| `/admin/login` | Email/password + OAuth buttons (same providers as Tauri) |
| `/admin/` | Dashboard: user count, recent registrations, sync activity |
| `/admin/users` | User list with search, filter by role/status |
| `/admin/users/new` | Create user form (name, email, temp password, role) |
| `/admin/users/{id}` | Edit user: name, email, role, disable/enable, delete |
| `/admin/invites` | Invite list, create new invite |

### 5.2 Authentication

Cookie-based sessions for the admin web UI (not bearer tokens in localStorage):

- Login sets an `HttpOnly`, `Secure`, `SameSite=Strict` session cookie
- CSRF token on all mutation forms
- Session stored server-side (PocketBase token in cookie, validated on each request)
- Logout clears cookie

This is separate from the bearer-token auth used by the Tauri client and API keys.

### 5.3 Authorization

`requireAdmin` middleware wraps all `/admin/` routes:

```go
func requireAdmin(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userID := UserIDFromContext(r.Context())
        user, _ := app.FindRecordById("users", userID)
        if user.GetString("role") != "admin" {
            http.Error(w, "forbidden", 403)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 5.4 PocketBase `/_/` Dashboard

Exists but locked down:
- Disabled via config in production (`POCKETBASE_ADMIN_UI=false`)
- Available in development for debugging
- Separate from `/admin/` — different auth, different purpose

### 5.5 Implementation

- Single base layout template with nav
- Minimal CSS, no JS framework
- Optional: HTMX for interactive bits (disable user without page reload)
- ~6 templates total
- All user operations through PocketBase's Go API (`app.FindAuthRecordByEmail`, `app.Save`, etc.)

---

## 6. Sync + SSE Scoping

### 6.1 Delta Sync

`delta_events` carries `user_id` at write time. The sync endpoint filters by authenticated user:

```sql
-- Modified delta query
SELECT * FROM delta_events
WHERE id > ? AND user_id = ?
ORDER BY id ASC;
```

Delete events include the `user_id` of the entity owner, captured before the delete. Cursor advancement is per-user (the client's sync cursor advances through their own events only).

### 6.2 SSE Stream

Events tagged with `user_id` at publish time. Each SSE connection is associated with an authenticated user. Only matching events are delivered:

```go
type userStream struct {
    userID string
    ch     chan event.DomainEvent
}

func (sm *StreamManager) Publish(evt event.DomainEvent) {
    for _, s := range sm.streams {
        if s.userID == evt.UserID {
            s.ch <- evt
        }
    }
}
```

No runtime ownership lookups. Events are self-contained.

---

## 7. Docker + Configuration

### 7.1 docker-compose.yml

```yaml
services:
  api:
    build: .
    image: atask:latest
    ports:
      - "8080:8080"
    environment:
      ADDR: ":8080"
      DATA_DIR: /app/data

      # Auth
      REGISTRATION_OPEN: "false"

      # OAuth (optional — omit to disable provider)
      GOOGLE_CLIENT_ID: "${GOOGLE_CLIENT_ID:-}"
      GOOGLE_CLIENT_SECRET: "${GOOGLE_CLIENT_SECRET:-}"
      GITHUB_CLIENT_ID: "${GITHUB_CLIENT_ID:-}"
      GITHUB_CLIENT_SECRET: "${GITHUB_CLIENT_SECRET:-}"

      # Server identity
      BASE_URL: "${BASE_URL:-http://localhost:8080}"

      # Admin UI
      POCKETBASE_ADMIN_UI: "false"
    volumes:
      - app_data:/app/data
    restart: unless-stopped

volumes:
  app_data:
```

### 7.2 Configuration

```env
# Required
DATA_DIR=/app/data              # PocketBase data directory

# Auth
REGISTRATION_OPEN=false         # true = open registration, false = invite-only
BASE_URL=https://your-server.com # for OAuth redirect URIs

# OAuth (all optional — omit to disable)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
APPLE_CLIENT_ID=
APPLE_TEAM_ID=
APPLE_KEY_ID=
APPLE_KEY_PATH=

# Admin
POCKETBASE_ADMIN_UI=false       # expose /_/ dashboard
```

`JWT_SECRET` is removed — PocketBase manages its own signing keys.

### 7.3 CLI Commands

```bash
atask serve                                    # start server (default)
atask admin create-user --email X --role admin # bootstrap admin
atask admin assign-data --to <user-id>         # claim orphaned data
atask admin create-invite --email X            # generate invite token
```

---

## 8. PocketBase Adapter Layer

To manage PocketBase upgrade risk (pre-1.0, breaking changes possible), all PocketBase interactions go through a thin adapter:

```go
// internal/auth/adapter.go
type AuthProvider interface {
    ValidateToken(token string) (userID string, err error)
    CreateUser(email, password, name string) (userID string, err error)
    FindUserByID(id string) (*User, error)
    FindUserByEmail(email string) (*User, error)
    UpdateUser(id string, updates map[string]any) error
    DisableUser(id string) error
    ListUsers(filter string, page, perPage int) ([]*User, error)
    InitiateOAuth(provider string, redirectURI string) (authURL string, err error)
    CompleteOAuth(provider string, code string) (userID string, token string, err error)
    LinkOAuth(userID string, provider string, code string) error
}
```

The adapter wraps PocketBase's Go API. If PocketBase introduces breaking changes, only the adapter needs updating. Services, handlers, and the admin UI depend on the interface, not PocketBase internals.

---

## 9. Phasing

### Phase 1: Multi-User Foundation (~3 weeks)

- Schema migration (user_id on all tables)
- Query scoping (~84 queries)
- Service layer threading (userID parameter on all methods)
- PocketBase integration + auth middleware bridge
- Email/password login (via PocketBase)
- API key scope enforcement
- Sync + SSE user scoping
- Web admin UI (Go templates): login, user list, create user
- CLI bootstrap commands
- Docker config updates
- Tauri settings: email/password login, token storage, 401 handling, account switching
- Testing: query scanning test, cross-user isolation tests, auth integration tests

**Outcome:** Working multi-user with email/password auth. Admin creates users. Self-hosted deployable.

### Phase 2: OAuth + Invites (~1.5 weeks)

- OAuth wrapper endpoints (Google, GitHub, Apple via PocketBase)
- Invite flow (admin creates invite → user claims via OAuth or password)
- Tauri: OAuth browser redirect handler, provider buttons in settings
- Admin UI: invite management page
- Account linking (connect additional OAuth providers)
- Testing: OAuth flow tests, invite claim tests

**Outcome:** Full OAuth support. Closed-registration servers use invite flow.

### Phase 3: Polish (1 week, optional)

- "Import local data" for pre-login tasks
- Admin dashboard stats
- API key expiry enforcement + rotation reminders
- PocketBase /_/ lockdown hardening

---

## 10. Decisions Log

| Decision | Chosen | Alternative Considered | Rationale |
|----------|--------|----------------------|-----------|
| Auth engine | PocketBase embedded | Full DIY | Codex review found 9 auth issues; PocketBase solves 6 for free |
| Data scoping | user_id on all 11 tables | JOIN-based scoping for children | Codex: "scoped by JOIN is fail-open" — denormalization cost is low |
| Delta event scoping | user_id on events at write time | Dynamic filtering at sync time | Codex: deleted entities have no ownership to look up |
| Admin UI | Go templates at /admin/ | PocketBase /_/ dashboard | Want control over UX, not tied to PocketBase's generic admin |
| Admin web auth | Cookie-based with CSRF | Bearer token in localStorage | Security: HttpOnly cookies prevent XSS token theft |
| OAuth + closed registration | Invite flow | Auto-provision on first OAuth login | Codex: auto-provision contradicts closed registration |
| Pre-login data on sign-in | Keep local, invisible after auth | Merge/upload | Merge requires conflict resolution; explicit import as future option |
| Email auto-linking on OAuth | Disabled | Link by matching email | Codex: account takeover vector; link only via authenticated flow |
| Token storage in Tauri | OS keychain (refresh), in-memory (access) | SQLite settings table | Codex: plaintext SQLite is insecure for long-lived tokens |
| PocketBase boundary | Thin adapter interface | Direct PB API calls everywhere | Isolates upgrade risk; pre-1.0 breaking changes affect only adapter |
| Per-user DBs | Rejected | One DB with user_id | Sharing requires cross-user queries; per-user DBs make that hard |
| Estimate | ~4.5 weeks (phased) | 16-19 days (original) | Codex: original estimate low by ~2x; phasing de-risks |
