# atask

An AI-first task manager. The mental model and API are the product. Agents are first-class citizens. Human UIs are thin clients on top.

## The Thesis

Good software in the AI era is software built for agents to operate on. Cheap for tokens, good APIs, easy to extract and manipulate data. The UI is just one client — and probably not the most important one anymore.

```
1. Mental model   — opinionated domain design
2. Database       — event-sourced schema
3. API            — agent-friendly, semantic operations
4. UIs            — thin clients (TUI, web, native, agents)
```

## Quick Start

### Run locally

```bash
go build -o atask ./cmd/atask
./atask
```

The server starts on `:8080` by default. Configure with environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `:8080` | Server listen address |
| `DB_PATH` | `atask.db` | SQLite database file path |
| `JWT_SECRET` | `change-me-in-production` | Secret for JWT signing |

### Run with Docker

```bash
docker compose up -d
```

### Verify it works

```bash
# Health check
curl http://localhost:8080/health

# Register a user
curl -X POST http://localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"secret","name":"Your Name"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"secret"}' | jq -r .token)

# Create a task
curl -X POST http://localhost:8080/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Buy groceries"}'

# View your inbox
curl http://localhost:8080/views/inbox \
  -H "Authorization: Bearer $TOKEN"
```

## Architecture

```
cmd/atask/main.go       → entry point, wiring
internal/
  domain/                     → entities, validation, constraints (zero deps)
  store/                      → SQLite persistence, sqlc-generated queries
  event/                      → dual-stream events, pub/sub bus, SSE
  service/                    → business operations (validate → persist → emit)
  api/                        → HTTP handlers, middleware, response helpers
```

### Key Design Decisions

**Event sourcing with dual streams.** Every mutation produces two kinds of events:
- **Delta events** — field-level changes for sync/state reconstruction
- **Domain events** — semantic events (`task.completed`, `project.cancelled`) for agent intelligence

**Semantic API.** Operations have meaning: `POST /tasks/{id}/complete`, not `PATCH /tasks/{id} {status: "completed"}`. Every mutation response includes the event type: `{"event": "task.completed", "data": {...}}`.

**The "When" model.** Three layers control task visibility:
- **Schedule** (inbox / anytime / someday) — where it lives in your attention
- **Start date** — when you want to start seeing it
- **Deadline** — when it must be done

These produce computed views: inbox, today, upcoming, someday, logbook.

**Activity stream.** Tasks are collaboration surfaces. Humans and agents post comments, drafts, context requests, and artifacts directly on tasks — creating an auditable proof-of-work trail.

## API Reference

### Authentication

```
POST /auth/register              Register a new user
POST /auth/login                 Login, returns JWT token
GET  /auth/me                    Current user profile
PUT  /auth/me                    Update profile
```

**Agent API keys:**

```
POST   /auth/api-keys            Create API key (returns secret once)
GET    /auth/api-keys            List keys (metadata only)
PUT    /auth/api-keys/{id}       Rename key
DELETE /auth/api-keys/{id}       Revoke key
```

Authenticate with `Authorization: Bearer <jwt>` or `Authorization: ApiKey <key>`.

### Tasks

```
POST   /tasks                         Create task (lands in inbox)
GET    /tasks                         List all tasks
GET    /tasks/{id}                    Get task
DELETE /tasks/{id}                    Delete task

POST   /tasks/{id}/complete           Complete task
POST   /tasks/{id}/cancel             Cancel task
PUT    /tasks/{id}/title              Update title
PUT    /tasks/{id}/notes              Update notes (markdown)
PUT    /tasks/{id}/schedule           Set schedule (inbox/anytime/someday)
PUT    /tasks/{id}/start-date         Set start date
PUT    /tasks/{id}/deadline           Set deadline
PUT    /tasks/{id}/project            Move to project
PUT    /tasks/{id}/section            Move to section
PUT    /tasks/{id}/area               Move to area
PUT    /tasks/{id}/location           Set location
PUT    /tasks/{id}/recurrence         Set recurrence rule
PUT    /tasks/{id}/reorder            Reorder
POST   /tasks/{id}/tags/{tagId}       Add tag
DELETE /tasks/{id}/tags/{tagId}       Remove tag
POST   /tasks/{id}/links/{taskId}     Link to another task
DELETE /tasks/{id}/links/{taskId}     Remove link
```

### Checklist Items

```
POST   /tasks/{id}/checklist                          Add item
GET    /tasks/{id}/checklist                           List items
PUT    /tasks/{id}/checklist/{itemId}                  Update title
POST   /tasks/{id}/checklist/{itemId}/complete         Complete item
POST   /tasks/{id}/checklist/{itemId}/uncomplete       Uncomplete item
DELETE /tasks/{id}/checklist/{itemId}                  Remove item
```

### Activity (Proof-of-Work)

```
POST   /tasks/{id}/activity           Add activity entry
GET    /tasks/{id}/activity           List activity entries
```

Request body:
```json
{
  "actor_type": "agent",
  "type": "artifact",
  "content": "Email draft ready — check your inbox."
}
```

Activity types: `comment`, `context_request`, `reply`, `artifact`, `status_change`, `decomposition`.

### Projects

```
POST   /projects                      Create project
GET    /projects                      List projects
GET    /projects/{id}                 Get project
DELETE /projects/{id}                 Delete (cascades to tasks + sections)

POST   /projects/{id}/complete        Complete (cascades: marks all tasks complete)
POST   /projects/{id}/cancel          Cancel (cascades: marks all tasks cancelled)
PUT    /projects/{id}/title           Update title
PUT    /projects/{id}/notes           Update notes
PUT    /projects/{id}/deadline        Set deadline
PUT    /projects/{id}/area            Move to area
POST   /projects/{id}/tags/{tagId}    Add tag
DELETE /projects/{id}/tags/{tagId}    Remove tag
```

### Sections

```
POST   /projects/{id}/sections        Create section
GET    /projects/{id}/sections        List sections
PUT    /projects/{id}/sections/{sid}  Rename
DELETE /projects/{id}/sections/{sid}  Delete (?cascade=true to delete tasks)
```

### Areas, Tags, Locations

```
POST   /areas                         Create
GET    /areas                         List (non-archived)
GET    /areas/{id}                    Get
PUT    /areas/{id}                    Rename
DELETE /areas/{id}                    Delete (?cascade=true)
POST   /areas/{id}/archive            Archive
POST   /areas/{id}/unarchive          Unarchive

POST   /tags                          Create
GET    /tags                          List
GET    /tags/{id}                     Get
PUT    /tags/{id}                     Rename
DELETE /tags/{id}                     Delete

POST   /locations                     Create
GET    /locations                     List
GET    /locations/{id}                Get
PUT    /locations/{id}                Rename
DELETE /locations/{id}                Delete
```

### Views (Computed, Read-Only)

```
GET /views/inbox       Tasks where schedule=inbox
GET /views/today       Tasks for today (schedule=anytime, start_date <= today)
GET /views/upcoming    Tasks with future start dates
GET /views/someday     Tasks where schedule=someday
GET /views/logbook     Completed and cancelled tasks
```

### Event Stream (SSE)

```
GET /events/stream?topics=task.*,project.completed&since=0
```

Server-Sent Events with topic filtering. Supports wildcards (`task.*`, `*`).

SSE format:
```
event: task.completed
data: {"entity_id":"abc","title":"Buy groceries"}
id: 42
```

Reconnect with `Last-Event-ID` header to resume from where you left off.

### Sync (Delta Events)

```
GET /sync/deltas?since=0
```

Returns all field-level change events since the given cursor for full state reconstruction.

## Domain Events

Every mutation emits a semantic domain event. Subscribe via SSE to react in real-time.

**Task:** `task.created`, `task.completed`, `task.cancelled`, `task.deleted`, `task.title_changed`, `task.notes_changed`, `task.scheduled_today`, `task.deferred`, `task.moved_to_inbox`, `task.start_date_set`, `task.deadline_set`, `task.deadline_removed`, `task.moved_to_project`, `task.removed_from_project`, `task.moved_to_section`, `task.moved_to_area`, `task.tag_added`, `task.tag_removed`, `task.location_set`, `task.location_removed`, `task.link_added`, `task.link_removed`, `task.recurrence_set`, `task.recurrence_removed`, `task.reordered`

**Project:** `project.created`, `project.completed`, `project.cancelled`, `project.deleted`, `project.title_changed`, `project.notes_changed`, `project.tag_added`, `project.tag_removed`, `project.moved_to_area`, `project.deadline_set`, `project.deadline_removed`

**Checklist:** `checklist.item_added`, `checklist.item_completed`, `checklist.item_uncompleted`, `checklist.item_title_changed`, `checklist.item_removed`

**Activity:** `activity.added`

**Area:** `area.created`, `area.deleted`, `area.renamed`, `area.archived`, `area.unarchived`

**Tag:** `tag.created`, `tag.deleted`, `tag.renamed`, `tag.shortcut_changed`

**Section:** `section.created`, `section.deleted`, `section.renamed`

**Location:** `location.created`, `location.deleted`, `location.renamed`

## Building Clients

atask is designed to be consumed by any client. The API is the product.

### For TUI / CLI clients

1. Authenticate once, store the JWT token
2. Use `/views/*` endpoints for the main screens
3. Use semantic mutation endpoints for actions
4. Optionally subscribe to SSE for live updates

### For AI agents

1. Create an API key via `/auth/api-keys` — each agent gets its own key
2. Use `Authorization: ApiKey <key>` for all requests
3. Subscribe to relevant domain events via SSE (`/events/stream?topics=task.*`)
4. Use the activity stream to collaborate on tasks
5. Every action you take is attributed to your API key in the audit trail

### For web / mobile clients

1. Use JWT auth flow (login → token → Bearer header)
2. `/views/*` endpoints map directly to UI screens
3. SSE for real-time updates
4. Location entities support geofencing (lat/lng/radius)

## Development

```bash
make build          # Build binary
make run            # Run server
make test           # Run tests with race detector
make lint           # Run golangci-lint
make fmt            # Format code
make check          # fmt + vet + lint + test
make migrate        # Run database migrations
make sqlc           # Regenerate sqlc code
make docker-build   # Build Docker image
make docker-up      # Start with docker compose
```

## Tech Stack

- **Go 1.25** with standard library HTTP routing
- **SQLite** via modernc.org/sqlite (pure Go, no CGo)
- **sqlc** for type-safe SQL query generation
- **goose** for database migrations
- **log/slog** for structured logging
- **bcrypt** for password hashing
- **JWT** for user authentication
- **SHA256** for API key hashing

## License

MIT
