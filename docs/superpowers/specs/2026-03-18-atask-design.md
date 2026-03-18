# atask — AI-First Task Manager Design Spec

**Date:** 2026-03-18
**Status:** Draft

## Overview

atask is a task manager built for the AI era. The mental model and API are the product. Agents are first-class citizens. Human UIs are thin clients on top.

The inverted pyramid:
1. **Mental model** — opinionated domain design
2. **Database** — event-sourced schema encoding the mental model
3. **API** — agent-friendly, semantic operations, small payloads
4. **UIs** — thin clients (TUI, web, native, agents) — all equal consumers of the API

### Success Criteria (v0)

- A working API + event store that agents can operate against
- An agent workflow end-to-end: triage inbox, schedule tasks into Today, collaborate on tasks via activity stream
- Documentation sufficient to build opinionated UIs on top (TUI with Bubbletea, native macOS client, web app)

### What This Is Not

- Not a JIRA/Linear replacement — no sprints, no story points, no team workflows (yet)
- Not an AI task planner — the AI monitors and assists, the human decides
- Not a project management tool — it's a personal task manager that agents can operate

---

## Tech Stack

### Backend (Go)

- **Language:** Go 1.22+ with modern idioms
- **HTTP:** `net/http` with Go 1.22 routing patterns (`GET /tasks/{id}`) — evaluate whether `chi` is needed
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- **Query layer:** `sqlc` — write SQL, generate type-safe Go
- **Migrations:** `goose`
- **Logging:** `log/slog` (structured logging)
- **Linting:** `golangci-lint` (includes `go vet`, `staticcheck`, `errcheck`, `gocritic`)
- **Formatting:** `go fmt` / `goimports` enforced
- **Auth:** JWT for users, API keys for agents

### Project Standards

- `go fmt` and `go vet` pass on all code
- `golangci-lint` with strict configuration
- Context propagation throughout
- Error wrapping with `%w`
- No external framework dependencies where the standard library suffices

---

## Domain Model

### Core Entities

#### Task
The atomic unit of work.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Title | string | The task name |
| Notes | string | Markdown content |
| Status | enum | `pending`, `completed`, `cancelled` |
| Schedule | enum | `inbox`, `anytime`, `someday` |
| StartDate | date? | When to start showing this task |
| Deadline | date? | Hard due date |
| CompletedAt | timestamp? | When it was completed/cancelled |
| CreatedAt | timestamp | Creation time |
| UpdatedAt | timestamp | Last modified |
| Index | int | Sort order in parent context |
| TodayIndex | int? | Sort order in Today view (separate from project order) |
| ProjectID | uuid? | Parent project |
| SectionID | uuid? | Section within project (task always knows its project too) |
| AreaID | uuid? | Parent area (for standalone tasks) |
| LocationID | uuid? | Where this should happen |
| RecurrenceRule | rule? | Repeat configuration |
| Tags | []uuid | Cross-cutting labels |

#### Project
A completable container of tasks — not a folder. Projects can be completed or cancelled. Completing a project marks all open tasks as completed. Cancelling marks them as cancelled.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Title | string | The project name |
| Notes | string | Markdown content |
| Status | enum | `pending`, `completed`, `cancelled` |
| Schedule | enum | `inbox`, `anytime`, `someday` |
| StartDate | date? | When to start showing this project |
| Deadline | date? | Hard due date |
| CompletedAt | timestamp? | When it was completed/cancelled |
| CreatedAt | timestamp | Creation time |
| UpdatedAt | timestamp | Last modified |
| Index | int | Sort order in area or top-level |
| AreaID | uuid? | Parent area |
| Tags | []uuid | Cross-cutting labels |
| AutoComplete | bool | Complete project when all tasks are done |

#### Section
Lightweight grouping label within a project.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Title | string | Section name |
| ProjectID | uuid | Always belongs to a project |
| Index | int | Sort order within project |

#### Area
A permanent responsibility domain. Areas never complete — they represent ongoing life categories ("Work", "Family", "Health"). Areas can be archived (hidden from active views) but history is preserved.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Title | string | Area name |
| Index | int | Sort order |
| Archived | bool | Hidden from active views |

#### Tag
Cross-cutting label, optionally hierarchical.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Title | string | Tag name |
| ParentID | uuid? | Parent tag for hierarchy |
| Shortcut | string? | Keyboard shortcut |
| Index | int | Sort order |

#### ChecklistItem
Lightweight sub-step within a task. Intentionally limited — no dates, no tags, no nesting.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Title | string | Item text |
| Status | enum | `pending`, `completed` |
| TaskID | uuid | Always belongs to a task |
| Index | int | Sort order within task |

#### Location
A reusable place, structured for geofencing by mobile clients.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| Name | string | Location name |
| Latitude | float64? | GPS latitude |
| Longitude | float64? | GPS longitude |
| Radius | int? | Geofence radius in meters |
| Address | string? | Human-readable address |

#### Activity
Collaboration entry on a task — the proof-of-work. The activity stream is where humans and agents collaborate on task execution, creating an auditable trail of value delivered.

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| TaskID | uuid | Always scoped to a task |
| ActorID | string | User or agent identifier |
| ActorType | enum | `human`, `agent` |
| Type | enum | `comment`, `context_request`, `reply`, `artifact`, `status_change`, `decomposition` |
| Content | string | Markdown content |
| CreatedAt | timestamp | Creation time |

### Entity Relationships

```
Area (permanent, archivable)
 ├── Project (completable, cancellable)
 │    ├── Section (organizational label)
 │    │    └── Task (actionable item)
 │    └── Task (no section)
 │         ├── ChecklistItem (lightweight sub-step)
 │         └── Activity (collaboration entry)
 └── Task (standalone, not in a project)

Location (reusable, referenced by tasks)

Tag (hierarchical, referenced by tasks and projects)

Task ···soft link···> Task (advisory relationship, not enforced)
```

**Key structural rule:** A task always knows its `project_id` even when it belongs to a section. The section is organizational grouping, not structural hierarchy. `GET /tasks?project_id=X` returns all tasks in the project regardless of section.

### Ordering Model

Two index spaces coexist within a project:
- **Project-level:** Sections and sectionless tasks share one `Index` sequence
- **Within-section:** Tasks inside a section have their own `Index` sequence scoped to that section

Sectionless tasks appear at the top of the project, before any sections.

The **Today view** has its own ordering via `TodayIndex` — separate from project ordering. This answers "what order do I work in today?" independently of how tasks are organized in their projects.

### Constraint Map

These are design opinions — places where atask deliberately limits what you can do:

| Constraint | Rationale |
|-----------|-----------|
| Projects don't nest | A project contains tasks and sections, not other projects. No infinite hierarchy. |
| Sections only inside projects | No top-level sections or sections inside areas. |
| Checklist items are not tasks | No dates, no tags, no nesting. Lightweight and intentionally limited. |
| Areas don't complete | Permanent life categories. Can be archived, not completed. |
| No explicit priorities | Ordering within lists is the priority. Today IS your priority list. |
| No hard dependencies | Soft links only — advisory, not enforced. |
| One area per task/project | Zero or one. Forces you to decide where something lives. |
| No project nesting | Keeps the hierarchy flat and manageable. |

### The "When" Model

Three layers control when a task appears:

1. **Schedule** (`inbox`, `anytime`, `someday`) — where the task lives in your attention
2. **StartDate** — when you want to START seeing this task
3. **Deadline** — when this MUST be done

These combine to produce views:
- **Inbox**: `schedule = inbox` (unprocessed, not yet decided)
- **Today**: `schedule = anytime` AND (`start_date` is null OR `start_date <= today`) — tasks with no start date and `schedule = anytime` are immediately visible in Today (ordered by `TodayIndex`)
- **Upcoming**: tasks with `start_date > today` (grouped by date)
- **Someday**: `schedule = someday` (deferred, hidden from daily view)
- **Logbook**: `status = completed | cancelled` (history)

### Recurrence

Tasks can repeat with two modes:
- **Fixed schedule:** Repeat every N days/weeks/months (next instance spawns on the fixed date regardless of completion)
- **After completion:** Repeat N days/weeks/months after the task is completed

When triggered, the system spawns a new task instance from the recurrence rule.

Recurrence rules are stored as structured JSON:
```json
{
  "mode": "fixed" | "after_completion",
  "interval": 1,
  "unit": "day" | "week" | "month",
  "end": null | { "date": "2026-12-31" } | { "count": 10 }
}
```

### Soft Links

Tasks can reference other tasks as advisory relationships. Not enforced — the system won't block you from completing tasks out of order. Agents can read these links to reason about task ordering and suggest sequencing.

Stored as a join table: `task_id`, `related_task_id`, `relationship_type` (default: `related`).

---

## Event System

Dual-stream architecture: delta events for persistence/sync, domain events for intelligence.

### Delta Events (Persistence Layer)

Every mutation is recorded as immutable delta events. These are the sync primitive — a client says "give me everything since cursor N."

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Auto-increment cursor |
| EntityType | enum | `task`, `project`, `section`, `area`, `tag`, `checklist_item`, `location`, `activity` |
| EntityID | uuid | Which entity changed |
| Action | enum | `created`, `modified`, `deleted` |
| Field | string? | Which field changed (null for create/delete) |
| OldValue | json? | Previous value |
| NewValue | json? | New value |
| ActorID | string | Who made the change |
| Timestamp | timestamp | When it happened |

### Domain Events (Intelligence Layer)

Semantic events emitted at write time by the domain logic. One domain event per semantic API operation — no gaps. These are what agents subscribe to.

| Field | Type | Description |
|-------|------|-------------|
| ID | int64 | Auto-increment cursor |
| Type | string | Event type (e.g. `task.completed`) |
| EntityType | enum | Which entity type |
| EntityID | uuid | Which entity |
| ActorID | string | Who triggered it |
| Payload | json | Semantic context |
| Timestamp | timestamp | When it happened |

### Domain Event Catalog

**Task events:**
- `task.created` / `task.deleted`
- `task.completed` / `task.cancelled`
- `task.title_changed` / `task.notes_changed`
- `task.scheduled_today` / `task.deferred` / `task.moved_to_inbox`
- `task.start_date_set` / `task.deadline_set` / `task.deadline_removed`
- `task.moved_to_project` / `task.removed_from_project`
- `task.moved_to_section` / `task.removed_from_section`
- `task.moved_to_area` / `task.removed_from_area`
- `task.tag_added` / `task.tag_removed`
- `task.location_set` / `task.location_removed`
- `task.link_added` / `task.link_removed`
- `task.recurrence_set` / `task.recurrence_removed`
- `task.reordered`

**Project events:**
- `project.created` / `project.deleted`
- `project.completed` / `project.cancelled`
- `project.title_changed` / `project.notes_changed`
- `project.tag_added` / `project.tag_removed`
- `project.moved_to_area` / `project.removed_from_area`
- `project.deadline_set` / `project.deadline_removed`

**Checklist events:**
- `checklist.item_added` / `checklist.item_removed`
- `checklist.item_completed` / `checklist.item_uncompleted`
- `checklist.item_title_changed`

**Activity events:**
- `activity.added`

**Section / Area / Tag / Location events:**
- `{entity}.created` / `{entity}.deleted` / `{entity}.renamed`
- `area.archived` / `area.unarchived`
- `tag.shortcut_changed`

### Deletion Model (Tombstones)

All entities support soft-delete via tombstones. A deletion event is recorded, the entity is marked as deleted, but history is preserved. This applies uniformly to tasks, projects, areas, tags, locations, sections, and checklist items.

**Cascade behavior on deletion:**
- **Project deleted** → all tasks and sections within it are also tombstoned
- **Area deleted** → by default, projects and standalone tasks are orphaned (`area_id` set to null). With `?cascade=true`, all contents are tombstoned.
- **Section deleted** → by default, tasks are moved to project top-level (`section_id` set to null). With `?cascade=true`, all tasks in the section are tombstoned.
- **Tag deleted** → tag references are removed from all entities
- **Location deleted** → location references are removed from all tasks

---

## API Design

### Principles

- **Semantic operations** — `POST /tasks/{id}/complete` not `PATCH /tasks/{id} {status: "completed"}`
- **Small response payloads** — return what changed + entity ID, not the full entity
- **Consistent envelope** — `{ "data": ..., "event": "task.completed" }` — every mutation tells you what event it emitted
- **Agent-friendly** — obvious semantics, minimal token cost, clear schemas

### Endpoints

#### Tasks

```
POST   /tasks                              → task.created
GET    /tasks                              → list (filter: project, area, schedule, tag, location)
GET    /tasks/{id}                         → single task with checklist + recent activity
DELETE /tasks/{id}                         → task.deleted

POST   /tasks/{id}/complete                → task.completed
POST   /tasks/{id}/cancel                  → task.cancelled
PUT    /tasks/{id}/title                   → task.title_changed
PUT    /tasks/{id}/notes                   → task.notes_changed
PUT    /tasks/{id}/schedule                → task.scheduled_today | task.deferred | task.moved_to_inbox
PUT    /tasks/{id}/start-date              → task.start_date_set
PUT    /tasks/{id}/deadline                → task.deadline_set | task.deadline_removed
PUT    /tasks/{id}/project                 → task.moved_to_project | task.removed_from_project
PUT    /tasks/{id}/section                 → task.moved_to_section
PUT    /tasks/{id}/area                    → task.moved_to_area
PUT    /tasks/{id}/location                → task.location_set | task.location_removed
PUT    /tasks/{id}/recurrence              → task.recurrence_set | task.recurrence_removed
POST   /tasks/{id}/tags/{tag_id}           → task.tag_added
DELETE /tasks/{id}/tags/{tag_id}           → task.tag_removed
POST   /tasks/{id}/links/{task_id}         → task.link_added
DELETE /tasks/{id}/links/{task_id}         → task.link_removed
PUT    /tasks/{id}/reorder                 → task.reordered
```

#### Checklist Items

```
POST   /tasks/{id}/checklist               → checklist.item_added
PUT    /tasks/{id}/checklist/{item_id}     → checklist.item_title_changed
POST   /tasks/{id}/checklist/{item_id}/complete    → checklist.item_completed
POST   /tasks/{id}/checklist/{item_id}/uncomplete  → checklist.item_uncompleted
DELETE /tasks/{id}/checklist/{item_id}     → checklist.item_removed
```

#### Activity (Proof-of-Work)

```
POST   /tasks/{id}/activity                → activity.added
GET    /tasks/{id}/activity                → list activity entries
```

#### Projects

```
POST   /projects                           → project.created
GET    /projects                           → list all projects
GET    /projects/{id}                      → project with sections + task counts
DELETE /projects/{id}                      → project.deleted

POST   /projects/{id}/complete             → project.completed (cascades to tasks)
POST   /projects/{id}/cancel               → project.cancelled (cascades to tasks)
PUT    /projects/{id}/title                → project.title_changed
PUT    /projects/{id}/notes                → project.notes_changed
PUT    /projects/{id}/deadline             → project.deadline_set
PUT    /projects/{id}/area                 → project.moved_to_area
POST   /projects/{id}/tags/{tag_id}        → project.tag_added
DELETE /projects/{id}/tags/{tag_id}        → project.tag_removed
```

#### Sections

```
POST   /projects/{id}/sections             → section.created
PUT    /projects/{id}/sections/{sid}       → section.renamed
DELETE /projects/{id}/sections/{sid}       → section.deleted
```

#### Areas

```
POST   /areas                              → area.created
GET    /areas                              → list all areas
GET    /areas/{id}                         → area with its projects + standalone tasks
PUT    /areas/{id}                         → area.renamed
DELETE /areas/{id}                         → area.deleted
POST   /areas/{id}/archive                 → area.archived
POST   /areas/{id}/unarchive               → area.unarchived
```

#### Tags

```
POST   /tags                               → tag.created
GET    /tags                               → list all tags (with hierarchy)
GET    /tags/{id}                          → tag + entities using it
PUT    /tags/{id}                          → tag.renamed
DELETE /tags/{id}                          → tag.deleted
```

#### Locations

```
POST   /locations                          → location.created
GET    /locations                          → list all locations
GET    /locations/{id}                     → location + tasks at this location
PUT    /locations/{id}                     → location.renamed
DELETE /locations/{id}                     → location.deleted
```

#### Views (Read-Only, Computed)

```
GET    /views/inbox                        → tasks where schedule=inbox
GET    /views/today                        → tasks for today, ordered by today_index
GET    /views/upcoming                     → tasks with future start_date, grouped by date
GET    /views/someday                      → tasks where schedule=someday
GET    /views/logbook                      → completed/cancelled tasks
```

#### Event Stream (SSE)

```
GET    /events/stream                      → Server-Sent Events
  Query params:
    topics    → comma-separated: task.*, project.completed, activity.added
    scope     → filter: project:{id}, area:{id}
    since     → cursor (Last-Event-ID also supported via header)
```

SSE format:
```
event: task.completed
data: {"entity_id":"xyz","title":"Write proposal","project":"Q2 Launch"}
id: 44
```

Reconnection: client sends `Last-Event-ID: 44` header, receives all events since cursor 44.

#### Sync (Delta Events)

```
GET    /sync/deltas?since={cursor}         → delta events since cursor for full state reconstruction
```

#### Auth

```
POST   /auth/login                         → returns JWT
POST   /auth/refresh                       → refresh JWT
POST   /auth/logout                        → invalidate token
GET    /auth/me                            → current user profile
PUT    /auth/me                            → update profile

GET    /auth/api-keys                      → list keys (metadata only, never the secret)
POST   /auth/api-keys                      → create key (returns secret ONCE)
PUT    /auth/api-keys/{id}                 → rename / update permissions
DELETE /auth/api-keys/{id}                 → revoke key
```

---

## Auth Model

### User Authentication
- JWT tokens issued via login
- Bearer token in HTTP headers

### Agent Authentication
- API keys created by the user, one per agent
- Each key has a name/label identifying the agent
- The key serves as `ActorID` in events and activity entries

### API Key Entity

| Field | Type | Description |
|-------|------|-------------|
| ID | uuid | Unique identifier |
| UserID | uuid | The user who created it |
| Name | string | Agent label (e.g. "Claude daily planner") |
| KeyHash | string | Hashed key — plaintext shown once on creation |
| Permissions | []string | Scoped access (full access for v0) |
| CreatedAt | timestamp | When created |
| LastUsedAt | timestamp | Last used timestamp |

---

## Project Structure

```
atask/
├── cmd/
│   └── atask/
│       └── main.go                        → entry point, wiring
├── internal/
│   ├── domain/                            → entities, validation, constraints
│   │   ├── task.go
│   │   ├── project.go
│   │   ├── area.go
│   │   ├── tag.go
│   │   ├── section.go
│   │   ├── checklist.go
│   │   ├── location.go
│   │   ├── activity.go
│   │   └── event.go                       → domain event types
│   ├── api/                               → HTTP handlers
│   │   ├── router.go
│   │   ├── tasks.go
│   │   ├── projects.go
│   │   ├── views.go
│   │   ├── events.go                      → SSE handler
│   │   ├── auth.go
│   │   └── middleware.go
│   ├── store/                             → SQLite persistence
│   │   ├── queries/                       → .sql files for sqlc
│   │   ├── migrations/                    → goose migration files
│   │   └── db.go                          → sqlc-generated code
│   ├── event/                             → event store + domain event bus
│   │   ├── store.go                       → delta event persistence
│   │   ├── bus.go                         → in-process pub/sub for domain events
│   │   └── stream.go                      → SSE subscription manager
│   └── sync/                              → cursor-based sync engine
│       └── sync.go
├── sqlc.yaml
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile                             → multi-stage build, single binary
└── docker-compose.yml                     → local dev + deployment
```

### Deployment

- **Dockerfile:** Multi-stage build — build stage compiles the Go binary, final stage is `scratch` or `alpine` with just the binary and migrations
- **docker-compose.yml:** Single service for the API server, with a volume mount for the SQLite database file (persistence across restarts)
- **Makefile targets:** `build`, `run`, `test`, `lint`, `migrate`, `docker-build`, `docker-up`

SQLite runs embedded in the Go process — no separate database container needed. The Docker volume ensures the `.db` file persists.

### Dependency Flow

```
cmd/atask → api → domain → event → store
                              ↘ store
```

- `domain` has zero dependencies on HTTP or SQL — pure Go types and business rules
- `api` translates HTTP requests into domain operations
- `event` sits between domain and persistence — every mutation flows through it
- `store` is generated by sqlc, only knows SQL

---

## Future Considerations (Not in v0)

These are explicitly out of scope for v0 but the architecture should not block them:

- **Multi-user** — events carry `ActorID`, auth is token-based, entities can gain ownership
- **MCP wrapper** — thin layer over the REST API, just another client
- **Webhooks** — complement SSE for server-to-server integrations
- **OAuth / SSO** — extend auth beyond email/password
- **Fine-grained permissions** — scoped API keys (`read:tasks`, `write:tasks`)
- **Mobile clients** — location geofencing is structured, ready for native consumption
- **File attachments** — artifacts in activity stream currently hold markdown text
