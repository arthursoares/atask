# atask — SwiftUI Design Package (v3)

**Target:** macOS 15+ native app (SwiftUI, Xcode)
**Future:** iOS/iPadOS via same codebase (NavigationSplitView adapts)
**Font:** Atkinson Hyperlegible (bundled .ttf)
**Theme:** "Bone" — ivory canvas, desaturated blue accent
**Motion:** None. All state changes instant.
**Architecture:** Local-first. SQLite (GRDB.swift) is source of truth. Server sync optional.

---

## Table of Contents

1. [Architecture](#1-architecture)
2. [Project Structure](#2-project-structure)
3. [Design Tokens (Swift)](#3-design-tokens-swift)
4. [Typography](#4-typography)
5. [Layout](#5-layout)
6. [Component Specs](#6-component-specs)
7. [View Specs](#7-view-specs)
8. [Command Palette](#8-command-palette)
9. [Keyboard Shortcuts (Things-compatible)](#9-keyboard-shortcuts)
10. [Motion Policy](#10-motion-policy)
11. [Data Layer](#11-data-layer)
12. [Sync Engine (optional)](#12-sync-engine)

---

## 1. Architecture

### Local-First

The app works without a server. Local SQLite (GRDB.swift) is the source of truth. All mutations are local and instant — no network round-trip for any UI action.

```
┌──────────────────────────────┐
│        SwiftUI Views         │  ← reads from TaskStore
└──────────────┬───────────────┘
               │ @Observable
┌──────────────▼───────────────┐
│         TaskStore            │  ← single source of truth
│   (in-memory + computed)     │
└──────────────┬───────────────┘
               │ read/write
┌──────────────▼───────────────┐
│     Local SQLite (GRDB)      │  ← persistent, works offline
│   ~/Library/.../atask.sqlite │
└──────────────┬───────────────┘
               │ background sync (optional)
┌──────────────▼───────────────┐
│         SyncEngine           │  ← only when server configured
│  Outbound: pendingOps queue  │
│  Inbound: SSE → upsert local │
│  Conflict: server wins       │
└──────────────┬───────────────┘
               │ HTTP + SSE
┌──────────────▼───────────────┐
│       Go API Server          │  ← authoritative when connected
└──────────────────────────────┘
```

### One Store, Computed Views

A single `@Observable TaskStore` holds all data in memory (backed by SQLite). Every view (inbox, today, upcoming, etc.) is a computed property — not a separate state object.

---

## 2. Project Structure

```
atask-app/
├── atask.xcodeproj
├── atask/
│   ├── ataskApp.swift
│   ├── Models/
│   │   ├── TaskModel.swift             ← GRDB record
│   │   ├── ProjectModel.swift
│   │   ├── AreaModel.swift
│   │   ├── SectionModel.swift
│   │   ├── TagModel.swift
│   │   ├── ChecklistItem.swift
│   │   └── Activity.swift
│   ├── Store/
│   │   ├── LocalDatabase.swift         ← GRDB DatabaseQueue + migrations
│   │   ├── TaskStore.swift             ← @Observable, computed views, mutations
│   │   ├── APIClient.swift             ← optional
│   │   ├── SSEClient.swift             ← optional
│   │   └── SyncEngine.swift            ← optional
│   ├── Views/
│   │   ├── ContentView.swift           ← NavigationSplitView shell
│   │   ├── Sidebar/SidebarView.swift
│   │   ├── TaskList/
│   │   │   ├── TaskRow.swift           ← 32pt collapsed
│   │   │   ├── TaskInlineEditor.swift  ← expanded card
│   │   │   ├── TaskMetaView.swift
│   │   │   ├── CheckboxView.swift
│   │   │   ├── WhenPicker.swift
│   │   │   ├── TagPickerView.swift
│   │   │   ├── ProjectPickerView.swift
│   │   │   ├── NewTaskRow.swift
│   │   │   └── SectionHeaderView.swift
│   │   ├── Detail/
│   │   │   ├── TaskDetailView.swift
│   │   │   ├── ChecklistSection.swift
│   │   │   └── ActivitySection.swift
│   │   ├── TodayView.swift
│   │   ├── InboxView.swift
│   │   ├── UpcomingView.swift
│   │   ├── SomedayView.swift
│   │   ├── LogbookView.swift
│   │   └── ProjectView.swift
│   ├── Overlays/
│   │   ├── CommandPaletteView.swift
│   │   └── SettingsView.swift
│   ├── Theme/
│   │   ├── Colors.swift                ← enum Theme
│   │   ├── Typography.swift
│   │   └── Spacing.swift
│   └── Helpers/
│       ├── DateFormatting.swift
│       └── KeyboardShortcuts.swift
├── ataskTests/
│   ├── TaskStoreTests.swift
│   └── DateFormattingTests.swift
└── Assets/Fonts/AtkinsonHyperlegible-*.ttf
```

---

## 3. Design Tokens (Swift)

### Colors.swift

```swift
import SwiftUI

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet(charactersIn: "#"))
        let scanner = Scanner(string: hex)
        var rgbValue: UInt64 = 0
        scanner.scanHexInt64(&rgbValue)
        self.init(
            red: Double((rgbValue & 0xFF0000) >> 16) / 255.0,
            green: Double((rgbValue & 0x00FF00) >> 8) / 255.0,
            blue: Double(rgbValue & 0x0000FF) / 255.0
        )
    }
}

enum Theme {
    static let canvas         = Color(hex: "#f6f5f2")
    static let canvasElevated = Color(hex: "#fefefe")
    static let canvasSunken   = Color(hex: "#eceae7")
    static let inkPrimary     = Color(hex: "#222120")
    static let inkSecondary   = Color(hex: "#686664")
    static let inkTertiary    = Color(hex: "#a09e9a")
    static let inkQuaternary  = Color(hex: "#c8c6c2")
    static let accent         = Color(hex: "#4670a0")
    static let accentHover    = Color(hex: "#3a5f8a")
    static let todayStar      = Color(hex: "#c88c30")
    static let somedayTint    = Color(hex: "#8878a0")
    static let deadlineRed    = Color(hex: "#c04848")
    static let success        = Color(hex: "#4a8860")
    static let agentTint      = Color(hex: "#7868a8")

    static let sidebarHover    = Color.black.opacity(0.04)
    static let sidebarActive   = Color.black.opacity(0.06)
    static let sidebarSelected = accent.opacity(0.10)
    static let accentSubtle    = accent.opacity(0.10)
    static let todayBg         = todayStar.opacity(0.08)
    static let deadlineBg      = deadlineRed.opacity(0.08)
    static let successBg       = success.opacity(0.08)
    static let agentBg         = agentTint.opacity(0.07)
    static let agentBorder     = agentTint.opacity(0.20)
    static let border          = Color.black.opacity(0.06)
    static let borderStrong    = Color.black.opacity(0.12)
    static let separator       = Color.black.opacity(0.05)
}
```

### Typography.swift

```swift
import SwiftUI

extension Font {
    static func atkinson(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        weight == .bold
            ? .custom("AtkinsonHyperlegible-Bold", size: size)
            : .custom("AtkinsonHyperlegible-Regular", size: size)
    }

    static let viewTitle     = atkinson(20, weight: .bold)
    static let sectionHeader = atkinson(17, weight: .bold)
    static let taskTitle     = atkinson(14)
    static let detailBody    = atkinson(15)
    static let metadata      = atkinson(12, weight: .bold)
    static let groupLabel    = atkinson(11, weight: .bold)
}
```

### Spacing.swift

```swift
enum Spacing {
    static let sp1:  CGFloat = 4
    static let sp2:  CGFloat = 8
    static let sp3:  CGFloat = 12
    static let sp4:  CGFloat = 16
    static let sp5:  CGFloat = 20
    static let sp6:  CGFloat = 24
    static let sp8:  CGFloat = 32

    static let sidebarWidth:  CGFloat = 240
    static let detailWidth:   CGFloat = 340
    static let toolbarHeight: CGFloat = 52
    static let taskRowHeight: CGFloat = 32
}

enum Radius {
    static let xs: CGFloat = 4
    static let sm: CGFloat = 6
    static let md: CGFloat = 8
    static let lg: CGFloat = 12
    static let xl: CGFloat = 16
}
```

---

## 4. Typography

| Role | Size | Weight | Usage |
|------|------|--------|-------|
| View title | 20 | Bold | "Today", "Inbox" |
| Section header | 17 | Bold | Section dividers |
| Task title | 14 | Regular | Row text |
| Inline editor title | 16 | Bold | Expanded card title |
| Inline notes | 13 | Regular | Notes in expanded card |
| Detail body | 15 | Regular | Detail panel notes |
| Metadata | 12 | Bold | Badges, timestamps, pills |
| Group label | 11 | Bold + uppercase + tracking | Sidebar area headers |

---

## 5. Layout

```swift
struct ContentView: View {
    @State var store: TaskStore

    var body: some View {
        NavigationSplitView {
            SidebarView(store: store)
        } content: {
            MainContentView(store: store)
                .background(Theme.canvas)
        } detail: {
            if store.selectedTaskId != nil {
                TaskDetailView(store: store)
            }
        }
        .navigationSplitViewStyle(.balanced)
    }
}
```

---

## 6. Component Specs

### 6.1 Sidebar

Areas are non-selectable headers. Projects nest under their area. Standalone projects (areaId == nil) appear after areas.

```swift
struct SidebarView: View {
    @Bindable var store: TaskStore

    var body: some View {
        List(selection: /* binding */) {
            Section {
                Label("Inbox", systemImage: "tray").tag("inbox").badge(store.inbox.count)
                Label("Today", systemImage: "star.fill").tag("today").badge(store.today.count)
                    .foregroundColor(Theme.todayStar)
                Label("Upcoming", systemImage: "calendar").tag("upcoming")
                Label("Someday", systemImage: "clock").tag("someday")
                Label("Logbook", systemImage: "archivebox").tag("logbook")
            }

            ForEach(store.areas) { area in
                Section(area.title) {
                    ForEach(store.projects.filter { $0.areaId == area.id }) { project in
                        Label(project.title, systemImage: "circle.fill")
                            .tag("project:\(project.id)")
                            .foregroundColor(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                    }
                }
            }

            let orphans = store.projects.filter { $0.areaId == nil }
            if !orphans.isEmpty {
                Section {
                    ForEach(orphans) { project in
                        Label(project.title, systemImage: "circle.fill").tag("project:\(project.id)")
                    }
                }
            }
        }
        .listStyle(.sidebar)
    }
}
```

### 6.2 CheckboxView

Circular 20pt, amber border in Today. Instant fill.

```swift
struct CheckboxView: View {
    let isChecked: Bool
    let isToday: Bool
    let onToggle: () -> Void

    var body: some View {
        Button(action: onToggle) {
            Circle()
                .strokeBorder(borderColor, lineWidth: 1.5)
                .background(Circle().fill(isChecked ? Theme.accent : .clear))
                .overlay {
                    if isChecked {
                        Image(systemName: "checkmark")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundColor(.white)
                    }
                }
                .frame(width: 20, height: 20)
        }
        .buttonStyle(.plain)
    }

    private var borderColor: Color {
        if isChecked { return Theme.accent }
        if isToday { return Theme.todayStar }
        return Theme.inkQuaternary
    }
}
```

### 6.3 TaskRow (32pt collapsed)

```swift
struct TaskRow: View {
    let task: TaskModel
    let isToday: Bool
    @Bindable var store: TaskStore

    private var isExpanded: Bool { store.expandedTaskId == task.id }

    var body: some View {
        if isExpanded {
            TaskInlineEditor(task: task, store: store)
        } else {
            HStack(spacing: Spacing.sp3) {
                CheckboxView(isChecked: task.isCompleted, isToday: isToday, onToggle: { store.completeTask(task.id) })

                Text(task.title)
                    .font(.taskTitle)
                    .foregroundColor(task.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                    .strikethrough(task.isCompleted, color: Theme.inkQuaternary)
                    .lineLimit(1)

                Spacer(minLength: Spacing.sp2)

                TaskMetaView(task: task, store: store)
            }
            .frame(height: Spacing.taskRowHeight)
            .padding(.horizontal, Spacing.sp4)
            .contentShape(Rectangle())
            .onTapGesture { store.expandedTaskId = task.id }
            .contextMenu { TaskContextMenu(task: task, store: store) }
        }
    }
}
```

**Only one card expanded at a time.** Setting `store.expandedTaskId` collapses any other.

### 6.4 Inline Editor (Expanded Card)

The **primary editing surface.** Title + notes (auto-growing, URL detection) + bottom action bar. Closer to Things than a simple attribute row.

```
┌─────────────────────────────────────────────────────────────────┐
│  ○  Task title (editable, 16pt bold)                            │
│                                                                 │
│     Notes area (13pt, auto-grows, URLs clickable)               │
│                                                                 │
│  ★ Today ✕                          🏷 Tags  ☰ Checklist  📅 When │
│                                     ○ Project Browser >         │
└─────────────────────────────────────────────────────────────────┘
```

```swift
struct TaskInlineEditor: View {
    let task: TaskModel
    @Bindable var store: TaskStore
    @State private var title: String
    @State private var notes: String
    @FocusState private var titleFocused: Bool

    init(task: TaskModel, store: TaskStore) {
        self.task = task; self.store = store
        _title = State(initialValue: task.title)
        _notes = State(initialValue: task.notes)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Top: checkbox + title
            HStack(spacing: Spacing.sp3) {
                CheckboxView(isChecked: task.isCompleted, isToday: task.isToday, onToggle: { store.completeTask(task.id) })
                TextField("", text: $title)
                    .font(.atkinson(16, weight: .bold))
                    .textFieldStyle(.plain)
                    .focused($titleFocused)
                    .onSubmit { save() }
            }
            .padding(.bottom, Spacing.sp2)

            // Middle: notes
            TextField("Notes", text: $notes, axis: .vertical)
                .font(.atkinson(13))
                .foregroundColor(Theme.inkSecondary)
                .textFieldStyle(.plain)
                .lineLimit(1...10)
                .padding(.leading, 32)

            // Bottom: action bar
            HStack {
                if task.isToday {
                    TagPill(label: "★ Today", variant: .today, removable: true) { store.setSchedule(task.id, 0) }
                }
                Spacer()
                // 🏷 Tags, ☰ Checklist, 📅 When buttons
                // Each opens a .popover
            }
            .padding(.top, Spacing.sp2)
            .padding(.leading, 32)

            // Project selector
            HStack { Spacer(); Text("○ \(store.projectFor(task)?.title ?? "No Project") >").font(.metadata).foregroundColor(Theme.inkTertiary) }
            .padding(.leading, 32)
        }
        .padding(Spacing.sp4)
        .background(Theme.canvasElevated)
        .clipShape(RoundedRectangle(cornerRadius: Radius.md))
        .shadow(color: .black.opacity(0.06), radius: 8, y: 2)
        .onAppear { titleFocused = true }
        .onExitCommand { save(); store.expandedTaskId = nil }
    }

    private func save() { store.updateTitle(task.id, title); store.updateNotes(task.id, notes) }
}
```

**Behaviors:** Click row → expand. Escape / ⌘Return → save + collapse. Return in title → focus notes. All saves instant to local SQLite.

### 6.5 "When" Popover

Things-style. ★ Today / 🌙 This Evening / calendar grid / 📦 Someday / Clear.

| Selection | Mutation |
|-----------|---------|
| Today | `schedule = 1, timeSlot = nil` |
| This Evening | `schedule = 1, timeSlot = "evening"` |
| Calendar date | `schedule = 1, startDate = date` |
| Someday | `schedule = 2` |
| Clear | `schedule = 1, startDate = nil, timeSlot = nil` |

Width: 260pt. Triggered by 📅 button or `⌘S`.

### 6.6 Project Picker / 6.7 Tag Picker

Searchable popovers. Projects grouped by area. Tags with checkmarks + inline create.

### 6.8 Fast Task Creation

`⌘N` → creates task in local DB with context defaults → opens inline editor with cursor in title.

Context: Today → schedule=1, Inbox → schedule=0, Project → projectId set, Someday → schedule=2.

`Space` (when list focused, NOT in text field) → same as ⌘N but inserts below selection.

Rapid capture: `⌘N` → type → Escape → `⌘N` → type → Escape.

### 6.9 TaskDetail (Right Panel)

340pt. The **deep-dive** — triggered by `Enter` or double-click. Full fields, notes, checklist, activity. Most daily operations happen via inline editor (§6.4).

### 6.10–6.13 SectionHeader, TagPill, ActivityEntry, ChecklistItemRow

Same specs as visual reference. SectionHeader collapsible. TagPill with variant colors from `Theme.*`. ActivityEntry with human/agent avatars. ChecklistItemRow with square 16pt checkbox.

---

## 7. View Specs

### 7.1 Today

Data: `store.todayMorning` + `store.todayEvening`

Morning tasks first (timeSlot != "evening"). "This Evening" section header only if `todayEvening` is non-empty. Amber checkboxes. Completed tasks visible until next day.

### 7.2 Inbox

Data: `store.inbox` (pending, schedule=0, no startDate). Standard checkboxes. Triage via shortcuts: `⌘T` Today, `⌘E` Evening, `⌘O` Someday.

### 7.3 Upcoming

Data: `store.upcoming`. Grouped by startDate. Relative date headers.

### 7.4 Someday

Data: `store.someday`. Flat list.

### 7.5 Logbook

Data: `store.logbook`. Grouped by completedAt. Completed=checked+struck. Cancelled=✕+struck.

### 7.6 Project

Data: `store.tasksForProject(id)`. Sections with headers. Progress bar. + Add Section.

---

## 8. Command Palette

**Trigger: `⇧⌘O`** (NOT ⌘K — that's Complete Task).

560pt overlay. Fuzzy search on commands + task titles. Categories: Navigation, Task Actions, Creation.

---

## 9. Keyboard Shortcuts (Things-compatible)

### Create

| Shortcut | Action |
|----------|--------|
| `⌘N` | New task (opens inline editor) |
| `Space` | New task below selection (list focused only) |
| `⌥⌘N` | New project |
| `⇧⌘N` | New section |
| `⇧⌘C` | New checklist in open task |

### Edit

| Shortcut | Action |
|----------|--------|
| `Return` | Open inline editor |
| `⌘Return` | Save and close editor |
| `⌘K` | **Complete task** |
| `⌥⌘K` | Cancel task |
| `⌘D` | Duplicate |
| `⌫` | Delete |

### Schedule

| Shortcut | Action |
|----------|--------|
| `⌘S` | Show When picker |
| `⌘T` | Start Today |
| `⌘E` | Start This Evening |
| `⌘R` | Start Anytime |
| `⌘O` | Start Someday |
| `⇧⌘D` | Set Deadline |
| `Ctrl+]` | Start date +1 day |
| `Ctrl+[` | Start date -1 day |

### Move

| Shortcut | Action |
|----------|--------|
| `⇧⌘M` | Move to project |
| `⌘↑/⌘↓` | Reorder |

### Navigate

| Shortcut | Action |
|----------|--------|
| `⇧⌘O` | Command palette |
| `⌘1`–`⌘6` | Inbox, Today, Upcoming, Anytime, Someday, Logbook |
| `⌘F` | Search |
| `⌘/` | Toggle sidebar |
| `⌘,` | Settings |
| `Escape` | Close editor / palette / deselect |

### Tags

| Shortcut | Action |
|----------|--------|
| `⇧⌘T` | Edit tags |

---

## 10. Motion Policy

No animations. Never `withAnimation`. Never `.animation()`. Never `.transition()`. Suppress implicit animation with `.animation(.none, value:)`.

---

## 11. Data Layer

### Models (GRDB)

Int-based status/schedule. Conforms to `Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable`.

```swift
struct TaskModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable {
    static let databaseTableName = "tasks"
    var id: String
    var title: String
    var notes: String
    var status: Int         // 0=pending, 1=completed, 2=cancelled
    var schedule: Int       // 0=inbox, 1=anytime, 2=someday
    var startDate: String?
    var deadline: String?
    var completedAt: String?
    var index: Int
    var todayIndex: Int?
    var timeSlot: String?   // nil, "morning", "evening"
    var projectId: String?
    var sectionId: String?
    var areaId: String?
    var createdAt: String
    var updatedAt: String
    var syncStatus: Int     // 0=local, 1=synced

    var isPending: Bool { status == 0 }
    var isCompleted: Bool { status == 1 }
    var isCancelled: Bool { status == 2 }
    var isToday: Bool { schedule == 1 }

    static func newTask(title: String) -> TaskModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return TaskModel(id: UUID().uuidString, title: title, notes: "", status: 0, schedule: 0,
            startDate: nil, deadline: nil, completedAt: nil, index: 0, todayIndex: nil, timeSlot: nil,
            projectId: nil, sectionId: nil, areaId: nil, createdAt: now, updatedAt: now, syncStatus: 0)
    }
}
```

### TaskStore

Single `@Observable`. Computed views. Local-first mutations.

```swift
@Observable
class TaskStore {
    private let db: LocalDatabase
    private(set) var tasks: [TaskModel] = []
    private(set) var projects: [ProjectModel] = []
    private(set) var areas: [AreaModel] = []
    // ...

    var selectedTaskId: String?
    var expandedTaskId: String?
    var activeView: ActiveView = .today

    var inbox: [TaskModel] { tasks.filter { $0.isPending && $0.schedule == 0 && $0.startDate == nil }.sorted { $0.index < $1.index } }

    var today: [TaskModel] {
        let todayStr = DateFormatting.todayString()
        return tasks.filter { $0.schedule == 1 && ($0.isPending || $0.isCompleted) && ($0.startDate == nil || $0.startDate! <= todayStr) }
            .sorted { a, b in
                let aSlot = a.timeSlot == "evening" ? 1 : 0
                let bSlot = b.timeSlot == "evening" ? 1 : 0
                if aSlot != bSlot { return aSlot < bSlot }
                return (a.todayIndex ?? 999999) < (b.todayIndex ?? 999999)
            }
    }

    var todayMorning: [TaskModel] { today.filter { $0.timeSlot != "evening" } }
    var todayEvening: [TaskModel] { today.filter { $0.timeSlot == "evening" } }

    func createTask(title: String) -> TaskModel {
        var task = TaskModel.newTask(title: title)
        switch activeView {
        case .today: task.schedule = 1
        case .someday: task.schedule = 2
        case .project(let pid): task.projectId = pid
        default: break
        }
        try? db.dbQueue.write { db in try task.insert(db) }
        tasks.append(task)
        return task
    }

    // completeTask, reopenTask, updateTitle, updateNotes, setSchedule, setTimeSlot, deleteTask...
    // All follow: mutate in-memory array → write to SQLite → done
}

enum ActiveView: Equatable {
    case inbox, today, upcoming, someday, logbook
    case project(String)
}
```

---

## 12. Sync Engine (Optional)

Active only when server URL configured in Settings.

**Outbound:** Local mutations → `pendingOps` table → background drain → POST/PUT to API → mark synced.
**Inbound:** SSE `GET /events/stream` → fetch entity → upsert local → TaskStore reloads.
**Conflict:** Server wins.
**Bootstrap:** On first connect, fetch all from API.

---

## Appendix: Design Decisions

| Decision | Rationale |
|----------|-----------|
| Bone ivory (#f6f5f2) | Warm but not Anthropic-beige |
| `enum Theme` not `Color` extensions | Explicit namespace |
| Things-compatible shortcuts | ⌘K=complete, ⌘T=today, ⇧⌘O=palette |
| Local-first (GRDB) | Instant mutations, works offline |
| Single TaskStore, computed views | One truth, no view-specific state |
| Int-based status/schedule | Matches SQLite, no enum complexity |
| `time_slot` field | "This Evening" is domain, not just UI |
| Inline editor = expanded card | Title + notes + actions. Primary surface. |
| One card expanded at a time | `expandedTaskId` |
| No animations | `withAnimation` is banned |
| `pendingOps` queue | Offline mutations sync in background |
