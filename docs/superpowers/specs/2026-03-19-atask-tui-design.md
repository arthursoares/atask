# atask TUI — Design Spec

**Date:** 2026-03-19
**Status:** Draft

## Overview

A terminal user interface for atask built with Bubbletea v2 and Bubbles. Three-pane layout (sidebar + list + detail) consuming the atask REST API as a thin client. Real-time updates via SSE.

### Success Criteria

- Full CRUD for all entities (tasks, projects, areas, tags, locations, checklist items, activity)
- Three-pane layout with hybrid keyboard navigation
- SSE-driven live updates
- Command palette for quick action access
- Single binary: `atask` (TUI default) / `atask serve` (headless API)

---

## CLI Interface

```
atask                              → starts TUI, connects to localhost:8080
atask --server http://host:8080    → TUI pointing at remote server
atask serve                        → headless API server
atask serve --addr :9090           → headless with custom address
atask serve --db /path/to/db       → custom database path
```

Default: `atask` with no args starts the TUI. If `--server` is not provided, connects to `localhost:8080`. The server must be running separately (via `atask serve` or Docker).

---

## Layout

Three-pane layout filling the terminal:

```
┌──────────────┬─────────────────────┬──────────────────────────────┐
│   Sidebar    │     List Pane       │        Detail Pane           │
│              │                     │                              │
│  VIEWS       │  Today — 5 tasks    │  Write email to Doctor       │
│  📥 Inbox  3 │                     │  📁 Health  🏷 waiting  📅 22│
│  ⭐ Today  5 │  ☐ Review PR #42    │                              │
│  📅 Upcoming │  ☐ Write email ←    │  [Notes] [Checklist] [Act.]  │
│  💤 Someday  │  ☐ Fix login bug    │                              │
│  📓 Logbook  │  ☐ Grocery shopping │  Need to reschedule my       │
│              │  ☐ Call dentist      │  March checkup to April.     │
│  AREAS       │                     │                              │
│  ▾ Work      │                     │  Ask about:                  │
│    📁 Q2     │                     │  - Blood test results        │
│    📁 Web    │                     │  - Prescription renewal      │
│  ▸ Personal  │                     │                              │
│  ▾ Health    │                     │  ─── Latest Activity ──────  │
│    📁 Marathon│                    │  🤖 agent — Draft ready.     │
│              │                     │                              │
│  TAGS        │  ✓ 3 completed      │                              │
│  🏷 urgent   │                     │                              │
│  🏷 waiting  │                     │                              │
└──────────────┴─────────────────────┴──────────────────────────────┘
 [Tab] panes  [j/k] navigate  [n] new  [x] complete  [?] help  [:] command
```

### Pane Proportions

- **Sidebar:** fixed width ~22 chars
- **List:** flexible, ~30% of remaining
- **Detail:** flexible, ~70% of remaining
- Adapts to terminal width. Below minimum width (~80 cols), detail pane collapses and Tab cycles only sidebar ↔ list. Selecting a task shows detail in a full-screen overlay instead. Above minimum width, detail pane reappears with cached content.

---

## Sidebar

### Sections

1. **Views** — fixed list: Inbox, Today, Upcoming, Someday, Logbook. Each shows a task count badge.
2. **Areas** — collapsible groups. Expanding an area shows its projects. Selecting an area shows all standalone tasks in that area. Selecting a project shows the project's tasks. Sections within a project are rendered as non-interactive visual dividers (dimmed uppercase headers between task groups). They are not focusable — `j/k` skips over them. Task indices are scoped per section.
3. **Tags** — flat list. Selecting a tag filters tasks by that tag across all views.

### Sidebar Actions

- `n` — create new area (when Areas section is focused) or new project (when inside an area)
- `e` — rename area or project
- `d` — delete (with confirmation)
- `Enter` — select view / expand-collapse area / select project
- `a` — archive/unarchive area

---

## List Pane

Shows tasks for the currently selected sidebar item.

### Display

Each task row shows:
- Checkbox status (☐ / ✓ / ✗)
- Title
- Right-aligned indicators: deadline (red if overdue), project name, location icon

### Task Actions

| Key | Action |
|-----|--------|
| `n` | New task (in current context) |
| `x` | Complete task |
| `X` | Cancel task |
| `d` | Delete task (with confirmation) |
| `e` | Edit title (replaces row with text input, Enter saves, Esc cancels) |
| `s` | Schedule picker (inbox / today / someday + start date) |
| `m` | Move to project (fuzzy picker) |
| `t` | Assign/remove tag (fuzzy picker) |
| `l` | Set location (fuzzy picker) |
| `Enter` | Focus detail pane |

### Ordering

Tasks are displayed in their `index` order. In the Today view, ordered by `today_index`.

---

## Detail Pane

Shows full details for the selected task. Has three tabs navigable with `1`, `2`, `3` or `Tab` within the pane.

### Header

Always visible:
- Task title (editable with `e`)
- Metadata line: project, tags, deadline, location, recurrence

### Tab 1: Notes

- Rendered markdown in a scrollable viewport
- `e` opens notes in `$EDITOR` (falls back to inline editing)
- Saved on editor close

### Tab 2: Checklist

- List of checklist items with status
- `x` toggles item completion
- `n` adds new item
- `e` edits item title
- `d` removes item
- Tab label shows progress: "Checklist (2/4)"

### Tab 3: Activity

- Chronological stream of activity entries
- Each entry shows: actor type icon (👤 human / 🤖 agent), actor name, type badge, content, timestamp
- `a` adds a new comment (opens text input)
- Scrollable viewport
- Tab label shows count: "Activity (3)"

### Activity Preview

Below the tabs, a persistent "Latest Activity" line shows the most recent entry without switching to the Activity tab. This surfaces agent work immediately. Updated via SSE — when an `activity.added` event arrives for the selected task, the preview updates in place (no debouncing needed, activity events are infrequent).

---

## Command Palette

Triggered by `:` or `Ctrl+P`. A fuzzy-searchable overlay listing all available actions.

```
┌─────────────────────────────────────┐
│ > schedule today                    │
│                                     │
│   Schedule → Today                  │
│   Schedule → Someday                │
│   Schedule → Inbox                  │
│   Set Start Date                    │
│   Set Deadline                      │
│   Search Tasks                      │
│   ...                               │
└─────────────────────────────────────┘
```

### Available Commands

**Task operations:** New Task, Complete, Cancel, Delete, Edit Title, Edit Notes, Schedule (Today/Someday/Inbox), Set Start Date, Set Deadline, Move to Project, Move to Section, Move to Area, Set Location, Set Recurrence, Add Tag, Remove Tag, Add Link, Add Checklist Item, Add Activity Comment

**Project operations:** New Project, Complete Project, Cancel Project, Delete Project, Edit Title, Set Deadline, Move to Area

**Area operations:** New Area, Rename, Archive, Unarchive, Delete

**Tag operations:** New Tag, Rename, Delete

**Location operations:** New Location, Rename, Delete

**Navigation:** Go to Inbox, Go to Today, Go to Upcoming, Go to Someday, Go to Logbook, Search

**System:** Refresh, Quit, Help

Commands are context-aware based on the **focused pane and selected item**:
- Sidebar focused → area/project/tag/navigation commands
- List focused with task selected → all task operations + navigation
- Detail focused → edit notes, checklist, activity commands
- All panes → navigation and system commands always visible

Unavailable commands are hidden, not grayed out.

---

## Search / Filter

Triggered by `/`. Opens a search input at the top of the list pane.

- Filters the current list pane contents by title substring (client-side). "Current list" means whatever tasks are displayed — if viewing a project, filters that project's tasks; if viewing Today, filters today's tasks.
- `Esc` clears filter and returns to normal view
- Results update as you type

---

## SSE Integration

The TUI subscribes to `GET /events/stream?topics=*` on startup. Domain events arrive as `tea.Msg` and trigger targeted refreshes:

| Event | Action |
|-------|--------|
| `task.*` | Refresh list pane if affected task is in current view. Update detail pane if affected task is selected. |
| `project.*` | Refresh sidebar project tree and list if viewing that project. |
| `area.*` | Refresh sidebar area tree. |
| `checklist.*` | Refresh detail pane checklist tab if viewing affected task. |
| `activity.added` | Refresh detail pane activity tab + latest activity preview if viewing affected task. |
| `tag.*`, `location.*`, `section.*` | Refresh relevant UI components. |

On SSE reconnect, the client sends `Last-Event-ID` to catch up on missed events.

---

## Navigation Model

### Pane Focus

- `Tab` / `Shift+Tab` — cycle focus: sidebar → list → detail
- Focused pane has a highlighted border
- `Esc` from detail returns focus to list; from list returns to sidebar

### Within Panes

- `j` / `k` or `↓` / `↑` — navigate items
- `Enter` — select / expand / activate
- `g` / `G` — go to top / bottom of list

### Global Keys

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle pane focus |
| `:` or `Ctrl+P` | Command palette |
| `/` | Search / filter |
| `?` | Help overlay |
| `q` | Quit |
| `r` | Manual refresh |

---

## HTTP Client

A Go client wrapping the atask REST API. Lives in `internal/client/`.

```go
type Client struct {
    baseURL    string
    token      string
    httpClient *http.Client
}

func New(baseURL, token string) *Client
```

Methods mirror the API:
- `ListInbox()`, `ListToday()`, `ListUpcoming()`, `ListSomeday()`, `ListLogbook()`
- `CreateTask(title)`, `CompleteTask(id)`, `UpdateTaskTitle(id, title)`, etc.
- `ListAreas()`, `ListProjects()`, `ListTags()`, `ListLocations()`
- `SubscribeEvents(topics) <-chan DomainEvent` (SSE client)

The client handles auth headers, JSON encoding/decoding, and error mapping.

---

## Project Structure

```
cmd/atask/main.go                    → CLI entry point (TUI or serve)
internal/
  client/
    client.go                        → HTTP API client
    sse.go                           → SSE subscription client
  tui/
    app.go                           → root model, pane coordination
    sidebar.go                       → sidebar model (views, areas, tags)
    list.go                          → list pane model (task/project lists)
    detail.go                        → detail pane model (tabs, content)
    palette.go                       → command palette overlay
    search.go                        → search/filter overlay
    help.go                          → help overlay
    keys.go                          → key bindings definition
    styles.go                        → lipgloss styles (colors, borders)
    messages.go                      → custom tea.Msg types
```

### Dependencies

- `github.com/charmbracelet/bubbletea/v2` — TUI framework
- `github.com/charmbracelet/bubbles/v2` — text input, viewport, list components
- `github.com/charmbracelet/lipgloss/v2` — styling and layout
- `github.com/spf13/cobra` — CLI subcommands (`serve` vs TUI)

---

## Styling

Use lipgloss for all styling. Dark theme matching the mockup:

- Background: terminal default
- Focused pane: highlighted border (cyan/blue)
- Selected item: inverse or highlighted background
- Completed tasks: dimmed/strikethrough
- Overdue deadlines: red
- Agent activity: purple accent
- Section headers in lists: dimmed, uppercase

---

## Key Binding Matrix

| Key | Global | Sidebar | List | Detail |
|-----|--------|---------|------|--------|
| `Tab`/`Shift+Tab` | cycle panes | — | — | — |
| `j`/`k` | — | navigate items | navigate tasks | scroll content |
| `Enter` | — | select/expand | focus detail | — |
| `Esc` | — | — | focus sidebar | focus list |
| `n` | — | new area/project | new task | new checklist item |
| `e` | — | rename | edit title | edit notes ($EDITOR) |
| `x` | — | — | complete task | toggle checklist item |
| `X` | — | — | cancel task | — |
| `d` | — | delete | delete task | delete checklist item |
| `s` | — | — | schedule picker | — |
| `m` | — | — | move to project | — |
| `t` | — | — | assign tag | — |
| `l` | — | — | set location | — |
| `a` | — | archive area | — | add comment |
| `1`/`2`/`3` | — | — | — | switch tabs |
| `/` | search | — | — | — |
| `:` or `Ctrl+P` | command palette | — | — | — |
| `?` | help overlay | — | — | — |
| `r` | refresh | — | — | — |
| `q` | quit | — | — | — |

---

## Error Handling

- Network errors show a status bar message at the bottom: "Connection lost — retrying..."
- SSE reconnection is automatic with exponential backoff
- Invalid operations (e.g., complete an already-completed task) show a flash message
- Server errors show the error message briefly, don't crash

---

## Future Considerations (Not in v0)

- Inline task creation without popup (just start typing in the list)
- Drag-and-drop reordering (if terminal supports mouse)
- Multiple selection for batch operations
- Split views (two list panes side by side)
- Notification badge for agent activity
- Theme customization
