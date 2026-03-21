# atask SwiftUI Desktop Client — Design Spec

> **Status:** Design. Lessons from v1 (Dioxus subagent chaos) and v2 (Dioxus reactivity fights) applied.

## Why SwiftUI

v1 and v2 used Dioxus (Rust + WebView). We hit:
- Signal reactivity bugs (reads outside rsx!, async writes not propagating)
- Keyboard event hijacking (WebView captures everything, no responder chain)
- Drag-and-drop limitations (HTML5 drag in WebView is janky)
- Context menus (custom JS, never native)
- State architecture (no centralized store, views desync)
- Constant framework fights that added zero user value

SwiftUI gives us for free:
- `@Observable` + `@State` — predictable, battle-tested state
- `keyboardShortcut()` — native, respects first responder (inputs don't get hijacked)
- `draggable()` / `dropDestination()` — OS-level drag-and-drop
- `.contextMenu {}` — native context menus in one line
- `NavigationSplitView` — three-pane layout natively
- Native macOS look & feel (vibrancy, traffic lights, sidebar style)

## What Carries Over From v2

### API (Go backend) — 100% reusable
- All endpoints tested (45 Playwright tests pass)
- Domain model validated (When model, view queries, etc.)
- SSE working with entity_id in payloads
- Checklist counts hydrated on GET /tasks/{id}
- Tag uniqueness enforced
- Project colors, section reorder, today-index, reopen — all done

### Design Specs — 100% reusable
- `docs/design_specs/DESIGN.md` — layout, components, views, keyboard shortcuts
- `docs/design_specs/CLAUDE.md` — aesthetic rules (Bone theme, no animations, etc.)
- `docs/design_specs/theme.css` — color tokens (translate to SwiftUI Color values)
- `docs/design_specs/atask-screens-validation.html` — visual reference

### Playwright Tests — 100% reusable
- `e2e/` — 45 API workflow tests, all pass against Go server
- These test the API, not the UI framework — work with any frontend

### Business Rules — documented
- When model (schedule × start_date × deadline)
- View query rules (inbox excludes dated, someday excludes dated, etc.)
- Full API endpoint reference in CLAUDE.md

## SwiftUI Architecture

### State: Single Observable TaskStore

The #1 lesson from Dioxus: **one source of truth for all tasks.**

```swift
@Observable
class TaskStore {
    var tasks: [TaskModel] = []         // ALL tasks, fetched on load
    var projects: [ProjectModel] = []
    var areas: [AreaModel] = []
    var tags: [TagModel] = []

    // Computed views — derived from tasks, never stored separately
    var inbox: [TaskModel] { tasks.filter { $0.schedule == .inbox && $0.startDate == nil && $0.status == .pending } }
    var today: [TaskModel] { tasks.filter { $0.schedule == .anytime && ($0.startDate == nil || $0.startDate! <= Date()) && $0.status == .pending } }
    var upcoming: [TaskModel] { tasks.filter { $0.startDate != nil && $0.startDate! > Date() && $0.schedule != .someday && $0.status == .pending } }
    var someday: [TaskModel] { tasks.filter { $0.schedule == .someday && $0.startDate == nil && $0.status == .pending } }
    var logbook: [TaskModel] { tasks.filter { $0.status == .completed || $0.status == .cancelled } }

    func tasksForProject(_ id: String) -> [TaskModel] { tasks.filter { $0.projectId == id && $0.status == .pending } }
}
```

When a task is completed, `tasks` is updated once → all computed views update automatically. No signal juggling, no SSE refetch gymnastics.

### API Client

Simple `URLSession` wrapper. Same patterns as the Rust client:
- GET views return bare JSON arrays
- Mutations return event envelopes
- Bearer token auth
- PascalCase JSON decoding via `JSONDecoder` with custom `CodingKeys`

### SSE

`URLSession` data task with streaming. Parse SSE protocol (same as Dioxus).
On event: refetch ALL tasks from the API and replace the store. Simple and correct.

Or better: parse the event, find the task by entity_id, fetch just that task via `GET /tasks/{id}`, and upsert into the store.

### Keyboard Shortcuts

SwiftUI's `keyboardShortcut()` modifier:
```swift
Button("Go to Inbox") { store.activeView = .inbox }
    .keyboardShortcut("1", modifiers: .command)
```

This automatically respects the first responder chain — typing in a TextField won't trigger shortcuts. The exact problem we couldn't solve in Dioxus.

### Drag and Drop

```swift
TaskRow(task: task)
    .draggable(task.id)

TaskList()
    .dropDestination(for: String.self) { taskIds, location in
        // Reorder or move tasks
    }
```

### Context Menus

```swift
TaskRow(task: task)
    .contextMenu {
        Button("Complete") { store.complete(task) }
        Button("Schedule for Today") { store.scheduleToday(task) }
        Divider()
        Button("Delete", role: .destructive) { store.delete(task) }
    }
```

## Project Structure

```
atask/
├── atask.xcodeproj
├── atask/
│   ├── ataskApp.swift              ← entry point, @main
│   ├── Models/
│   │   ├── TaskModel.swift         ← Codable struct matching Go API
│   │   ├── ProjectModel.swift
│   │   ├── AreaModel.swift
│   │   ├── TagModel.swift
│   │   ├── ChecklistItem.swift
│   │   └── Activity.swift
│   ├── Store/
│   │   ├── TaskStore.swift         ← @Observable, single source of truth
│   │   ├── APIClient.swift         ← URLSession wrapper
│   │   ├── SSEClient.swift         ← SSE streaming
│   │   └── Credentials.swift       ← Keychain token storage
│   ├── Views/
│   │   ├── ContentView.swift       ← NavigationSplitView (3-pane)
│   │   ├── Sidebar/
│   │   │   └── SidebarView.swift
│   │   ├── TaskList/
│   │   │   ├── TaskListView.swift  ← generic list used by all views
│   │   │   ├── TaskRow.swift       ← single task row (32px)
│   │   │   └── NewTaskRow.swift    ← inline creation
│   │   ├── Detail/
│   │   │   ├── TaskDetailView.swift
│   │   │   ├── ChecklistSection.swift
│   │   │   └── ActivitySection.swift
│   │   ├── CommandPalette/
│   │   │   └── CommandPaletteView.swift
│   │   └── Login/
│   │       └── LoginView.swift
│   └── Helpers/
│       ├── DateFormatting.swift     ← relative date logic (port from Rust)
│       └── Colors.swift            ← Bone theme colors
```

## What NOT To Do (lessons from v1/v2)

1. **Don't use separate state per view.** One `TaskStore`, computed views.
2. **Don't refetch after every mutation.** Optimistic update the local store. SSE confirms.
3. **Don't dispatch parallel agents for UI work.** One feature at a time, tested.
4. **Don't skip manual testing.** Every feature verified by running the app.
5. **Don't build all views at once.** Build Today first, end-to-end, then replicate.
6. **Don't fight the framework.** Use SwiftUI patterns, not React/Dioxus patterns.

## Implementation Phases

### Phase 1: Foundation (same scope as Plan 1)
- Xcode project + API client + models
- TaskStore with computed views
- Login + token storage (Keychain)
- Sidebar + toolbar
- Today view — end-to-end with real data
- Verification: create, complete, select tasks

### Phase 2: All Views + Detail Panel
- Inbox, Upcoming, Someday, Logbook, Project views
- Task detail panel (right side)
- All field editing (title, notes, schedule, dates, project, tags)
- Checklist + activity display

### Phase 3: Power Features
- Command palette (⌘K) — NSPanel or sheet overlay
- Keyboard shortcuts (native)
- Drag-and-drop (native)
- Context menus (native)
- Settings (server URL, auto-archive)

### Phase 4: Polish + Sync
- SSE real-time updates
- Local-first with Core Data or SQLite
- Offline support
- Dark mode (native)
