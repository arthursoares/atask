# atask SwiftUI Desktop Client — v3 Design Spec

> **Status:** Design. Incorporates all lessons from v1/v2 and merges ALL planned features into a single coherent architecture.

## Why v3 / SwiftUI

v1 (Dioxus + subagents) and v2 (Dioxus + manual fixes) both failed on framework-level issues: signal reactivity bugs, keyboard hijacking, WebView drag-and-drop limitations, context menu hacks. SwiftUI eliminates these — keyboard shortcuts, drag-and-drop, context menus, and three-pane layout are all native primitives.

## Core Architecture: Local-First

**The app works without a server.** Local SwiftData/SQLite is the source of truth. The server is optional — you configure it in settings, and sync happens in the background.

```
┌──────────────────────────────┐
│        SwiftUI Views         │  ← reads from local store
│   (Sidebar, TaskList, Detail) │
└──────────────┬───────────────┘
               │ @Observable
┌──────────────▼───────────────┐
│         TaskStore            │  ← single source of truth
│   (in-memory, backed by DB)  │
└──────────────┬───────────────┘
               │ read/write
┌──────────────▼───────────────┐
│     Local SQLite / SwiftData │  ← persistent, works offline
│   ~/Library/.../atask.sqlite │
└──────────────┬───────────────┘
               │ background sync
┌──────────────▼───────────────┐
│         SyncEngine           │  ← optional, when server configured
│  Outbound: queue + POST/PUT  │
│  Inbound: SSE → upsert local │
│  Conflict: server wins       │
└──────────────┬───────────────┘
               │ HTTP + SSE
┌──────────────▼───────────────┐
│       Go API Server          │  ← authoritative when connected
│    (already built, tested)   │
└──────────────────────────────┘
```

### Flow: Creating a Task
1. User types title, presses Enter
2. TaskStore creates task in local SQLite (instant, no network)
3. UI updates immediately (observable)
4. If server configured: SyncEngine queues the mutation
5. Background: POST to API, mark as synced
6. If offline: stays in queue, syncs when connection restored

### Flow: Incoming SSE Event
1. Server sends `task.completed` event
2. SyncEngine receives it
3. Fetch updated task via `GET /tasks/{entity_id}`
4. Upsert into local SQLite
5. TaskStore refreshes → UI updates

## What Carries Over

### API (Go backend) — 100% done
- 45 Playwright API tests pass
- All endpoints: CRUD, views, schedule, dates, tags, checklist, activity, SSE
- Domain model: When model (schedule × start_date × deadline)
- New endpoints: today-index, reopen, project color, section reorder
- Tag uniqueness enforced (migration 003)

### Design Specs — translate CSS to SwiftUI
- `docs/design_specs/DESIGN.md` — layout, components, views
- `docs/design_specs/CLAUDE.md` — aesthetic rules
- `docs/design_specs/theme.css` — color tokens → Swift `Color` constants
- `docs/design_specs/atask-screens-validation.html` — visual reference

### Playwright Tests — reusable
- `e2e/` — test the API, not the UI framework

### Business Rules — documented in CLAUDE.md
- View query logic (inbox/today/upcoming/someday/logbook)
- Status integers: 0=pending, 1=completed, 2=cancelled
- Schedule integers: 0=inbox, 1=anytime, 2=someday
- Completion behavior: strikethrough, visible until next day
- Projects nested under areas in sidebar

## Feature Inventory (ALL features, no phasing)

Everything below ships in v3. No "deferred" features.

### Layout
- [x] Three-pane: `NavigationSplitView` (sidebar 240pt, content flex, detail 340pt)
- [x] Sidebar: nav items (Inbox/Today/Upcoming/Someday/Logbook), areas with nested projects, + buttons
- [x] Toolbar: view title + icon + date subtitle, search + new task buttons
- [x] Detail panel: appears when task selected, 340pt right side

### Task Row (32pt, single line — collapsed state)
- [x] Circular checkbox (20pt) — amber border in Today view
- [x] Title — truncates with ellipsis
- [x] Metadata (right-aligned): project pill (colored dot + name), deadline, today badge, checklist count ("3/5"), agent indicator
- [x] Grip handle on hover (drag affordance)
- [x] Completion: instant strikethrough, stays visible until next day
- [x] Context menu (right-click): Complete, Schedule Today, Defer, Move to Project →, Set Date, Delete

### Inline Task Editor (Things-style expanded card)

When a task is clicked OR a new task is created, the row expands into an inline editor card. This is the PRIMARY editing surface — not the detail panel.

**Layout (expanded card):**
```
┌─────────────────────────────────────────────────────────────────────┐
│  ○  Task title (editable, large)                                    │
│                                                                     │
│     Notes area (editable, smaller text, auto-grows)                 │
│     URLs detected and shown as clickable links                      │
│                                                                     │
│  ★ Today ✕                              🏷 Tags  ☰ Checklist  🗓 Deadline  │
│                                         ○ Project Browser >         │
└─────────────────────────────────────────────────────────────────────┘
```

**Components:**
- Top: checkbox + editable title (TextField, large font)
- Middle: notes (TextEditor, auto-grows, detects URLs as clickable links)
- Bottom-left: schedule badge (★ Today with ✕ to remove, or schedule indicator)
- Bottom-right: action icons — Tags, Checklist, Deadline/Date, then Project selector below

**Bottom bar action icons:**
- 🏷 Tags icon — opens tag picker popover (searchable dropdown, dark theme)
- ☰ Checklist icon — toggles checklist section visibility within the card
- 🗓 Deadline icon — opens "When" picker popover

**"When" picker popover (Things-style):**
```
┌─────────────────────────┐
│  When                   │  ← searchable field
├─────────────────────────┤
│  ★ Today          ✓     │  ← current selection highlighted
│  🌙 This Evening        │
├─────────────────────────┤
│  Mon Tue Wed Thu Fri Sat Sun │  ← calendar grid
│  23  24  25  26  27  28  29  │
│  30  31   1   2   3   4   5  │
│  ...                         │
├─────────────────────────┤
│  📦 Someday              │
│  + Add Reminder          │
├─────────────────────────┤
│  [ Clear ]               │  ← clears schedule/date
└─────────────────────────┘
```

Selecting "Today" sets schedule=anytime. Selecting a date sets start_date. Selecting "Someday" sets schedule=someday. "Clear" removes schedule and date.

**Tag picker popover:**
- Dark-themed dropdown
- Search field at top ("Tags")
- List of all tags with icons
- Click to toggle tag on/off

**Behavior:**
- Click task row → expands inline (other expanded cards collapse)
- Click outside or press Escape → collapse back to single-line row
- ⌘N → creates new task and opens inline editor immediately
- All changes save locally instantly (local-first)
- Only ONE card expanded at a time

**New task creation flow:**
1. Click "+ New Task" or press ⌘N
2. New task created in local store
3. Inline editor opens with cursor in title field
4. Type title, optionally set tags/date/project via bottom bar
5. Click outside or Escape → card collapses to normal row

### Task Detail Panel (340pt right side — full view)

The detail panel is for FULL task view when you want to see everything. It appears when you explicitly open it (Enter key or double-click). It shows ALL fields including activity stream.

- [x] Editable title (large, ghost input style)
- [x] Schedule picker (Inbox / Today / Someday)
- [x] Project picker (dropdown, colored dots)
- [x] Start date picker (native DatePicker)
- [x] Deadline picker
- [x] Tag pills + add/remove
- [x] Notes (TextEditor, markdown)
- [x] Checklist (square checkboxes, add item, toggle)
- [x] Activity stream (human + agent entries)

### Views
- [x] Inbox: tasks with schedule=inbox AND no start_date, + New Task
- [x] Today: schedule=anytime AND (no date OR date ≤ today), amber checkboxes
- [x] Upcoming: date-grouped by start_date, section headers with relative dates
- [x] Someday: flat list, schedule=someday AND no start_date
- [x] Logbook: completed/cancelled grouped by date, reopen on checkbox click
- [x] Project: sections with tasks, progress bar, + Add Section

### Inbox Triage
- [x] Hover quick-actions: ★ Today, 💤 Someday, 📁 Project
- [x] Completing/triaging removes from inbox immediately

### Command Palette (⌘K)
- [x] 560pt overlay, centered
- [x] Search: fuzzy match on commands AND task titles
- [x] Categories: Navigation, Task Actions (context-aware), Creation
- [x] Keyboard: ↑↓ navigate, Enter execute, Escape close

### Keyboard Shortcuts (Things-compatible, native SwiftUI)

**Create:**
| Shortcut | Action |
|----------|--------|
| ⌘N | New task (opens inline editor) |
| Space | New task below selection (only when task list focused, NOT in text fields) |
| ⌥⌘N | New project |
| ⇧⌘N | New heading/section |
| ⇧⌘C | New checklist in open task |

**Edit:**
| Shortcut | Action |
|----------|--------|
| Return | Open selected task (inline editor) |
| ⌘Return | Save and close inline editor |
| ⌘K | Complete selected task |
| ⌥⌘K | Cancel selected task |
| ⌘D | Duplicate task |
| ⌫ | Delete task |

**Schedule (When):**
| Shortcut | Action |
|----------|--------|
| ⌘S | Show When picker |
| ⌘T | Start Today |
| ⌘E | Start This Evening |
| ⌘R | Start Anytime (schedule=anytime) |
| ⌘O | Start Someday |
| ⇧⌘D | Set/edit Deadline |
| Ctrl+] | Start date +1 day |
| Ctrl+[ | Start date -1 day |

**Move:**
| Shortcut | Action |
|----------|--------|
| ⇧⌘M | Move to another list/project |
| ⌘↑ | Move item up |
| ⌘↓ | Move item down |

**Navigate:**
| Shortcut | Action |
|----------|--------|
| ⇧⌘O | Command palette / Quick Find (Things calls this "navigation popover") |
| ⌘1 | Inbox |
| ⌘2 | Today |
| ⌘3 | Upcoming |
| ⌘4 | Anytime |
| ⌘5 | Someday |
| ⌘6 | Logbook |
| ⌘F | Search |
| ⌘/ | Toggle sidebar |
| ↑↓ | Navigate task list |
| ⌘← | Go back |

**Tags:**
| Shortcut | Action |
|----------|--------|
| ⇧⌘T | Edit tags for selected task |

**Other:**
| Shortcut | Action |
|----------|--------|
| Escape | Close inline editor / command palette / deselect |
| ⌘, | Settings |

### Drag and Drop (native SwiftUI)
- [x] Task rows draggable in Today/Someday/Project views
- [x] Drop gap indicator (32pt space opens at drop position)
- [x] Cross-section drag in project view
- [x] Drag projects between areas in sidebar

### Context Menus (native SwiftUI)
- [x] Task: Complete, Schedule Today, Defer, Move to Project →, Dates, Tags, Delete
- [x] Project (sidebar): Rename, Set Color, Complete, Delete
- [x] Section (header): Rename, Delete
- [x] Area (sidebar): Rename, Archive, Delete

### Settings (⌘,)
- [x] Server URL (default: none — offline only)
- [x] Auto-archive threshold (Never / 1 day / 1 week / 1 month)
- [x] Stored in UserDefaults or local config

### Date Formatting (relative)
- [x] Today, Tomorrow, Yesterday, weekday name, "Last Monday", "Mar 25", "Mar 25, 2027"
- [x] Deadline: "Due Tomorrow", "Due Today" (amber), "Overdue · Mar 18" (red)

### Sync (when server configured)
- [x] SSE for inbound events → upsert local
- [x] Outbound mutation queue → POST/PUT in background
- [x] Retry with exponential backoff on failure
- [x] Conflict resolution: server wins
- [x] Bootstrap: fetch all data from API on first connect

## Project Structure

```
atask/
├── atask.xcodeproj (or Package.swift for SPM)
├── atask/
│   ├── ataskApp.swift                  ← @main, WindowGroup
│   │
│   ├── Models/                         ← Codable structs + SwiftData models
│   │   ├── TaskModel.swift
│   │   ├── ProjectModel.swift
│   │   ├── AreaModel.swift
│   │   ├── TagModel.swift
│   │   ├── SectionModel.swift
│   │   ├── ChecklistItem.swift
│   │   └── Activity.swift
│   │
│   ├── Store/                          ← Business logic + state
│   │   ├── TaskStore.swift             ← @Observable, computed views, mutations
│   │   ├── LocalDatabase.swift         ← SwiftData/SQLite wrapper
│   │   ├── APIClient.swift             ← URLSession, all endpoints
│   │   ├── SSEClient.swift             ← EventSource streaming
│   │   ├── SyncEngine.swift            ← Outbound queue + inbound merge
│   │   └── Credentials.swift           ← Keychain storage
│   │
│   ├── Views/
│   │   ├── ContentView.swift           ← NavigationSplitView shell
│   │   ├── Sidebar/
│   │   │   └── SidebarView.swift       ← Nav items, areas with nested projects
│   │   ├── TaskList/
│   │   │   ├── TaskListView.swift      ← Generic list, used by all views
│   │   │   ├── TaskRow.swift           ← 32pt row with checkbox + title + meta
│   │   │   ├── TaskMetaView.swift      ← Right-aligned pills (project, deadline, checklist)
│   │   │   ├── NewTaskRow.swift        ← Inline creation (+ New Task → TextField)
│   │   │   └── SectionHeaderView.swift ← Bold title + count + line
│   │   ├── Detail/
│   │   │   ├── TaskDetailView.swift    ← Right panel, all fields editable
│   │   │   ├── SchedulePicker.swift
│   │   │   ├── ProjectPicker.swift
│   │   │   ├── TagPicker.swift
│   │   │   ├── ChecklistSection.swift
│   │   │   └── ActivitySection.swift
│   │   ├── Overlays/
│   │   │   ├── CommandPaletteView.swift
│   │   │   └── SettingsView.swift
│   │   └── Login/
│   │       └── LoginView.swift         ← Only if server configured + needs auth
│   │
│   ├── Theme/
│   │   ├── Colors.swift                ← Bone theme colors as Swift constants
│   │   ├── Typography.swift            ← Font definitions (Atkinson Hyperlegible)
│   │   └── Spacing.swift               ← 4px-based spacing constants
│   │
│   └── Helpers/
│       ├── DateFormatting.swift         ← Relative date logic (port unit tests from Rust)
│       └── KeyboardShortcuts.swift     ← Commands + shortcut definitions
│
├── ataskTests/
│   ├── DateFormattingTests.swift
│   ├── TaskStoreTests.swift
│   ├── APIClientTests.swift
│   └── SyncEngineTests.swift
│
├── e2e/                                ← Playwright API tests (from v2)
│   ├── package.json
│   ├── playwright.config.ts
│   └── tests/
│       ├── helpers.ts
│       ├── today-view.spec.ts
│       ├── tag-flow.spec.ts
│       ├── missing-flows.spec.ts
│       ├── full-coverage.spec.ts
│       ├── checklist-counts.spec.ts
│       └── keyboard-workflows.spec.ts
│
└── Assets/
    └── Fonts/
        └── AtkinsonHyperlegible-*.ttf
```

## Color Tokens (CSS → Swift)

```swift
enum Theme {
    // Canvas
    static let canvas = Color(hex: "#f6f5f2")
    static let canvasElevated = Color(hex: "#fefefe")
    static let canvasSunken = Color(hex: "#eceae7")

    // Ink
    static let inkPrimary = Color(hex: "#222120")
    static let inkSecondary = Color(hex: "#686664")
    static let inkTertiary = Color(hex: "#a09e9a")
    static let inkQuaternary = Color(hex: "#c8c6c2")

    // Accent
    static let accent = Color(hex: "#4670a0")
    static let accentHover = Color(hex: "#3a5f8a")

    // Semantic
    static let todayStar = Color(hex: "#c88c30")
    static let somedayTint = Color(hex: "#8878a0")
    static let deadlineRed = Color(hex: "#c04848")
    static let success = Color(hex: "#4a8860")
    static let agentTint = Color(hex: "#7868a8")
}
```

## What NOT To Do

1. **Don't treat the API as source of truth.** Local DB is source of truth. API is sync target.
2. **Don't refetch after mutations.** Write locally, sync in background.
3. **Don't build separate state per view.** One TaskStore, computed views.
4. **Don't dispatch parallel agents for UI.** One feature at a time, tested.
5. **Don't skip manual testing.** Every feature verified by running the app.
6. **Don't use UIKit patterns in SwiftUI.** Use `@Observable`, `NavigationSplitView`, `.contextMenu`, `.draggable`.
7. **Don't defer "hard" features.** Keyboard shortcuts, drag-and-drop, context menus are EASIER in SwiftUI — build them from day one.
8. **Don't animate state changes.** The design spec says instant everything. Use `.animation(.none)` where SwiftUI adds implicit animation.

## Implementation Approach

**No phases.** Build the full app feature-by-feature, in this order:

1. **Models + Local DB** — SwiftData models, CRUD operations, unit tests
2. **TaskStore** — @Observable with computed views, mutations, unit tests
3. **Theme + helpers** — colors, typography, date formatting (with unit tests)
4. **App shell** — NavigationSplitView, sidebar (areas + nested projects), toolbar
5. **Task row + inline editor** — collapsed row (32pt) + expanded inline card (Things-style). This is THE core interaction — clicking expands, editing saves locally, click outside collapses. Includes "When" picker, tag picker, checklist toggle.
6. **Today view** — first working view with task rows, amber checkboxes, inline creation via ⌘N
7. **Remaining views** — inbox (with triage actions), upcoming (date-grouped), someday, logbook (with reopen), project (with sections)
8. **Detail panel** — full task view (right side), all fields, checklist, activity
9. **Keyboard shortcuts** — native SwiftUI (⌘K, ⌘1-5, ⌘N, ⌘⇧C, ⌘T, etc.)
10. **Command palette** — ⌘K overlay with fuzzy search on commands + tasks
11. **Drag and drop** — native SwiftUI (.draggable, .dropDestination), gap indicator
12. **Context menus** — native SwiftUI (.contextMenu) on tasks, projects, sections, areas
13. **Settings** — server URL, auto-archive threshold
14. **API client + Sync engine** — connect to server, SSE inbound, outbound queue
15. **Login** — only when server configured + needs auth

Each step: build → test → verify in simulator → commit. No subagents.
