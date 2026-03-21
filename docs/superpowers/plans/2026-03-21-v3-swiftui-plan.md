# atask SwiftUI v3 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a complete, local-first macOS task manager in SwiftUI that works offline and optionally syncs with the atask Go API.

**Architecture:** Local SQLite (GRDB.swift) is the source of truth. Single `@Observable TaskStore` with computed views. SwiftUI for all UI — native keyboard shortcuts, drag-and-drop, context menus. Server sync is optional, built last.

**Tech Stack:** Swift, SwiftUI (macOS 15), GRDB.swift, URLSession, Atkinson Hyperlegible font

**Design Spec:** `docs/superpowers/specs/2026-03-21-v3-swiftui-design.md`
**Visual Reference:** `docs/design_specs/atask-screens-validation.html`
**API Reference:** `CLAUDE.md` (comprehensive endpoint list + business rules)

**Working Directory:** `/Users/arthur.soares/Github/openthings/.worktrees/native-client-v3`

**Lessons applied:**
1. Local-first — all mutations are local, instant. No API dependency for UI.
2. One store, computed views — no separate state per view.
3. One feature at a time, tested by running the app.
4. SwiftUI native patterns — no framework fights.
5. Things-compatible keyboard shortcuts.

---

## Go API Prerequisites

Before starting the SwiftUI app, add the `time_slot` field to the Go API:

### Task 0: Add time_slot to Go API

**Files:**
- Create: `internal/store/migrations/004_add_time_slot.sql`
- Modify: `internal/domain/task.go`
- Modify: `internal/store/queries/tasks.sql`
- Modify: `internal/store/queries/views.sql`
- Modify: `internal/service/task_service.go`
- Modify: `internal/api/tasks.go`
- Modify: `internal/domain/event.go`

- [ ] **Step 1: Create migration**

```sql
-- +goose Up
ALTER TABLE tasks ADD COLUMN time_slot TEXT;

-- +goose Down
ALTER TABLE tasks DROP COLUMN time_slot;
```

- [ ] **Step 2: Add TimeSlot to domain Task**

In `internal/domain/task.go`, add to Task struct:
```go
TimeSlot *string // nil, "morning", "evening"
```

- [ ] **Step 3: Add event type**

In `internal/domain/event.go`:
```go
TaskTimeSlotSet EventType = "task.time_slot_set"
```

- [ ] **Step 4: Add sqlc query**

In `internal/store/queries/tasks.sql`:
```sql
-- name: UpdateTaskTimeSlot :one
UPDATE tasks SET time_slot = ?, updated_at = ?
WHERE id = ? AND deleted = 0
RETURNING *;
```

- [ ] **Step 5: Update ViewToday query**

In `internal/store/queries/views.sql`, update ViewToday to order by time_slot:
```sql
-- name: ViewToday :many
SELECT * FROM tasks
WHERE schedule = 1 AND status = 0 AND deleted = 0
  AND (start_date IS NULL OR start_date <= ?)
ORDER BY
  CASE WHEN time_slot = 'evening' THEN 1 ELSE 0 END,
  COALESCE(today_index, 999999),
  "index";
```

- [ ] **Step 6: Run make sqlc, add service + handler**

Follow the standard pattern (service method + API handler). Register `PUT /tasks/{id}/time-slot`.

- [ ] **Step 7: Run tests**

```bash
make sqlc && make test
```

- [ ] **Step 8: Add Playwright test**

```typescript
test('set time slot', async ({ request }) => {
    const { data: task } = await (await request.post('/tasks', { headers, data: { title: 'Evening task' } })).json();
    await request.put(`/tasks/${task.ID}/schedule`, { headers, data: { schedule: 'anytime' } });
    await request.put(`/tasks/${task.ID}/time-slot`, { headers, data: { time_slot: 'evening' } });

    const today = await (await request.get('/views/today', { headers })).json();
    // Evening tasks should be at the end
    const idx = today.findIndex((t: any) => t.ID === task.ID);
    expect(idx).toBeGreaterThan(-1);
});
```

- [ ] **Step 9: Commit**

```bash
git add internal/ e2e/
git commit -m "feat: add time_slot field (morning/evening) for Today sub-sections"
```

---

## SwiftUI App Tasks

### Task 1: Xcode Project + GRDB Setup

**Files:**
- Create: `atask-app/` Xcode project via command line or Xcode
- Create: `atask-app/atask/ataskApp.swift`
- Create: `atask-app/atask/Store/LocalDatabase.swift`
- Create: `atask-app/Package dependencies` (GRDB.swift)

- [ ] **Step 1: Create Xcode project**

```bash
# Create macOS App project targeting macOS 15
mkdir -p atask-app
cd atask-app
# Use Xcode to create project, OR use swift package init + manual xcodeproj
```

Alternatively, create manually:
- Open Xcode → New Project → macOS → App
- Product Name: atask
- Interface: SwiftUI
- Language: Swift
- Minimum Deployment: macOS 15.0
- Save in the worktree's `atask-app/` directory

- [ ] **Step 2: Add GRDB.swift dependency**

In Xcode: File → Add Package Dependencies → `https://github.com/groue/GRDB.swift` → branch `master`

Or in Package.swift if using SPM:
```swift
dependencies: [
    .package(url: "https://github.com/groue/GRDB.swift", from: "7.0.0")
]
```

- [ ] **Step 3: Create LocalDatabase.swift**

```swift
import GRDB

class LocalDatabase {
    let dbQueue: DatabaseQueue

    init() throws {
        let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
        let dbDir = appSupport.appendingPathComponent("atask", isDirectory: true)
        try FileManager.default.createDirectory(at: dbDir, withIntermediateDirectories: true)
        let dbPath = dbDir.appendingPathComponent("atask.sqlite").path
        dbQueue = try DatabaseQueue(path: dbPath)
        try migrate()
    }

    // In-memory for testing
    init(inMemory: Bool) throws {
        dbQueue = try DatabaseQueue()
        try migrate()
    }

    private func migrate() throws {
        var migrator = DatabaseMigrator()

        migrator.registerMigration("v1_schema") { db in
            try db.create(table: "tasks") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("notes", .text).defaults(to: "")
                t.column("status", .integer).defaults(to: 0)
                t.column("schedule", .integer).defaults(to: 0)
                t.column("startDate", .text)
                t.column("deadline", .text)
                t.column("completedAt", .text)
                t.column("index", .integer).defaults(to: 0)
                t.column("todayIndex", .integer)
                t.column("timeSlot", .text)
                t.column("projectId", .text).references("projects")
                t.column("sectionId", .text).references("sections")
                t.column("areaId", .text).references("areas")
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
                t.column("syncStatus", .integer).defaults(to: 0) // 0=local, 1=synced
            }

            try db.create(table: "projects") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("notes", .text).defaults(to: "")
                t.column("status", .integer).defaults(to: 0)
                t.column("color", .text).defaults(to: "")
                t.column("areaId", .text).references("areas")
                t.column("index", .integer).defaults(to: 0)
                t.column("completedAt", .text)
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
            }

            try db.create(table: "areas") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("index", .integer).defaults(to: 0)
                t.column("archived", .boolean).defaults(to: false)
            }

            try db.create(table: "sections") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("projectId", .text).notNull().references("projects")
                t.column("index", .integer).defaults(to: 0)
            }

            try db.create(table: "tags") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).notNull().unique()
                t.column("index", .integer).defaults(to: 0)
            }

            try db.create(table: "taskTags") { t in
                t.column("taskId", .text).notNull().references("tasks")
                t.column("tagId", .text).notNull().references("tags")
                t.primaryKey(["taskId", "tagId"])
            }

            try db.create(table: "checklistItems") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("status", .integer).defaults(to: 0)
                t.column("taskId", .text).notNull().references("tasks")
                t.column("index", .integer).defaults(to: 0)
            }

            try db.create(table: "pendingOps") { t in
                t.autoIncrementedPrimaryKey("id")
                t.column("method", .text).notNull()
                t.column("path", .text).notNull()
                t.column("body", .text)
                t.column("createdAt", .text).notNull()
                t.column("synced", .boolean).defaults(to: false)
            }
        }

        try migrator.migrate(dbQueue)
    }
}
```

- [ ] **Step 4: Create minimal ataskApp.swift**

```swift
import SwiftUI

@main
struct ataskApp: App {
    var body: some Scene {
        WindowGroup {
            Text("atask v3 — scaffold works")
        }
    }
}
```

- [ ] **Step 5: Build and run**

```bash
xcodebuild -project atask-app/atask.xcodeproj -scheme atask build
# Or open in Xcode and press ⌘R
```

Expected: Window opens showing "atask v3 — scaffold works".

- [ ] **Step 6: Commit**

```bash
git add atask-app/
git commit -m "feat: scaffold Xcode project with GRDB.swift local database"
```

---

### Task 2: Models + TaskStore

**Files:**
- Create: `atask-app/atask/Models/TaskModel.swift`
- Create: `atask-app/atask/Models/ProjectModel.swift`
- Create: `atask-app/atask/Models/AreaModel.swift`
- Create: `atask-app/atask/Models/TagModel.swift`
- Create: `atask-app/atask/Models/SectionModel.swift`
- Create: `atask-app/atask/Models/ChecklistItem.swift`
- Create: `atask-app/atask/Store/TaskStore.swift`
- Create: `atask-app/ataskTests/TaskStoreTests.swift`

- [ ] **Step 1: Create GRDB record types**

Each model conforms to `Codable`, `FetchableRecord`, `PersistableRecord`:

```swift
// TaskModel.swift
import GRDB
import Foundation

struct TaskModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable {
    static let databaseTableName = "tasks"

    var id: String
    var title: String
    var notes: String
    var status: Int       // 0=pending, 1=completed, 2=cancelled
    var schedule: Int     // 0=inbox, 1=anytime, 2=someday
    var startDate: String?
    var deadline: String?
    var completedAt: String?
    var index: Int
    var todayIndex: Int?
    var timeSlot: String? // nil, "morning", "evening"
    var projectId: String?
    var sectionId: String?
    var areaId: String?
    var createdAt: String
    var updatedAt: String
    var syncStatus: Int   // 0=local, 1=synced

    var isPending: Bool { status == 0 }
    var isCompleted: Bool { status == 1 }
    var isCancelled: Bool { status == 2 }
    var isToday: Bool { schedule == 1 }

    static func newTask(title: String) -> TaskModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return TaskModel(
            id: UUID().uuidString,
            title: title, notes: "", status: 0, schedule: 0,
            startDate: nil, deadline: nil, completedAt: nil,
            index: 0, todayIndex: nil, timeSlot: nil,
            projectId: nil, sectionId: nil, areaId: nil,
            createdAt: now, updatedAt: now, syncStatus: 0
        )
    }
}
```

Similar for ProjectModel, AreaModel, TagModel, SectionModel, ChecklistItem.

- [ ] **Step 2: Create TaskStore**

```swift
// TaskStore.swift
import SwiftUI
import GRDB

@Observable
class TaskStore {
    private let db: LocalDatabase
    private(set) var tasks: [TaskModel] = []
    private(set) var projects: [ProjectModel] = []
    private(set) var areas: [AreaModel] = []
    private(set) var tags: [TagModel] = []
    private(set) var sections: [SectionModel] = []

    var selectedTaskId: String?
    var expandedTaskId: String?  // inline editor
    var activeView: ActiveView = .today

    init(db: LocalDatabase) {
        self.db = db
        reload()
    }

    func reload() {
        do {
            try db.dbQueue.read { db in
                tasks = try TaskModel.fetchAll(db)
                projects = try ProjectModel.fetchAll(db)
                areas = try AreaModel.fetchAll(db)
                tags = try TagModel.fetchAll(db)
                sections = try SectionModel.fetchAll(db)
            }
        } catch {
            print("Failed to load: \(error)")
        }
    }

    // MARK: - Computed Views

    var inbox: [TaskModel] {
        tasks.filter { $0.isPending && $0.schedule == 0 && $0.startDate == nil }
            .sorted { $0.index < $1.index }
    }

    var today: [TaskModel] {
        let todayStr = DateFormatting.todayString()
        return tasks.filter { task in
            task.schedule == 1 && (task.isPending || task.isCompleted) &&
            (task.startDate == nil || task.startDate! <= todayStr)
        }.sorted { a, b in
            let aSlot = a.timeSlot == "evening" ? 1 : 0
            let bSlot = b.timeSlot == "evening" ? 1 : 0
            if aSlot != bSlot { return aSlot < bSlot }
            return (a.todayIndex ?? 999999) < (b.todayIndex ?? 999999)
        }
    }

    var todayMorning: [TaskModel] { today.filter { $0.timeSlot != "evening" } }
    var todayEvening: [TaskModel] { today.filter { $0.timeSlot == "evening" } }

    var upcoming: [TaskModel] {
        let todayStr = DateFormatting.todayString()
        return tasks.filter { $0.isPending && $0.startDate != nil && $0.startDate! > todayStr && $0.schedule != 2 }
            .sorted { ($0.startDate ?? "") < ($1.startDate ?? "") }
    }

    var someday: [TaskModel] {
        tasks.filter { $0.isPending && $0.schedule == 2 && $0.startDate == nil }
            .sorted { $0.index < $1.index }
    }

    var logbook: [TaskModel] {
        tasks.filter { $0.isCompleted || $0.isCancelled }
            .sorted { ($0.completedAt ?? "") > ($1.completedAt ?? "") }
    }

    func tasksForProject(_ projectId: String) -> [TaskModel] {
        tasks.filter { $0.projectId == projectId && $0.isPending }
            .sorted { $0.index < $1.index }
    }

    // MARK: - Mutations (local-first)

    func createTask(title: String) -> TaskModel {
        var task = TaskModel.newTask(title: title)
        // Set schedule based on active view
        switch activeView {
        case .today: task.schedule = 1
        case .someday: task.schedule = 2
        case .project(let pid): task.projectId = pid
        default: break // inbox
        }
        do {
            try db.dbQueue.write { db in try task.insert(db) }
            tasks.append(task)
        } catch { print("Create failed: \(error)") }
        return task
    }

    func completeTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 1
        tasks[idx].completedAt = ISO8601DateFormatter().string(from: Date())
        save(tasks[idx])
    }

    func reopenTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 0
        tasks[idx].completedAt = nil
        save(tasks[idx])
    }

    func updateTitle(_ id: String, _ title: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].title = title
        save(tasks[idx])
    }

    func updateNotes(_ id: String, _ notes: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].notes = notes
        save(tasks[idx])
    }

    func setSchedule(_ id: String, _ schedule: Int) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].schedule = schedule
        save(tasks[idx])
    }

    func setTimeSlot(_ id: String, _ slot: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].timeSlot = slot
        save(tasks[idx])
    }

    func deleteTask(_ id: String) {
        tasks.removeAll { $0.id == id }
        do { try db.dbQueue.write { db in try TaskModel.deleteOne(db, id: id) } }
        catch { print("Delete failed: \(error)") }
    }

    private func save(_ task: TaskModel) {
        do { try db.dbQueue.write { db in try task.update(db) } }
        catch { print("Save failed: \(error)") }
    }
}

enum ActiveView: Equatable {
    case inbox, today, upcoming, someday, logbook
    case project(String)
}
```

- [ ] **Step 3: Write unit tests**

```swift
// TaskStoreTests.swift
import XCTest
@testable import atask

final class TaskStoreTests: XCTestCase {
    func makeStore() throws -> TaskStore {
        let db = try LocalDatabase(inMemory: true)
        return TaskStore(db: db)
    }

    func testCreateTask() throws {
        let store = try makeStore()
        let task = store.createTask(title: "Test task")
        XCTAssertEqual(store.inbox.count, 1)
        XCTAssertEqual(store.inbox.first?.title, "Test task")
    }

    func testCompleteTask() throws {
        let store = try makeStore()
        let task = store.createTask(title: "Complete me")
        store.completeTask(task.id)
        XCTAssertEqual(store.inbox.count, 0)
        XCTAssertEqual(store.logbook.count, 1)
    }

    func testScheduleToday() throws {
        let store = try makeStore()
        let task = store.createTask(title: "Today task")
        store.setSchedule(task.id, 1) // anytime
        XCTAssertEqual(store.inbox.count, 0)
        XCTAssertEqual(store.today.count, 1)
    }

    func testInboxExcludesDated() throws {
        let store = try makeStore()
        var task = store.createTask(title: "Dated task")
        // Manually set start date
        if let idx = store.tasks.firstIndex(where: { $0.id == task.id }) {
            store.tasks[idx].startDate = "2026-04-01"
            // save...
        }
        XCTAssertEqual(store.inbox.count, 0) // should not appear
    }

    func testTodayEvening() throws {
        let store = try makeStore()
        let task = store.createTask(title: "Evening task")
        store.setSchedule(task.id, 1)
        store.setTimeSlot(task.id, "evening")
        XCTAssertEqual(store.todayEvening.count, 1)
        XCTAssertEqual(store.todayMorning.count, 0)
    }
}
```

- [ ] **Step 4: Run tests**

```bash
xcodebuild test -project atask-app/atask.xcodeproj -scheme atask -destination 'platform=macOS'
```

- [ ] **Step 5: Commit**

```bash
git add atask-app/
git commit -m "feat: add GRDB models, TaskStore with computed views, unit tests"
```

---

### Task 3: Theme + Date Formatting

**Files:**
- Create: `atask-app/atask/Theme/Colors.swift`
- Create: `atask-app/atask/Theme/Typography.swift`
- Create: `atask-app/atask/Theme/Spacing.swift`
- Create: `atask-app/atask/Helpers/DateFormatting.swift`
- Create: `atask-app/ataskTests/DateFormattingTests.swift`
- Add: Atkinson Hyperlegible font files to Assets

- [ ] **Step 1: Create Colors.swift**

Port all CSS tokens from `docs/design_specs/theme.css`:

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
    static let canvas = Color(hex: "#f6f5f2")
    static let canvasElevated = Color(hex: "#fefefe")
    static let canvasSunken = Color(hex: "#eceae7")
    static let inkPrimary = Color(hex: "#222120")
    static let inkSecondary = Color(hex: "#686664")
    static let inkTertiary = Color(hex: "#a09e9a")
    static let inkQuaternary = Color(hex: "#c8c6c2")
    static let accent = Color(hex: "#4670a0")
    static let accentHover = Color(hex: "#3a5f8a")
    static let todayStar = Color(hex: "#c88c30")
    static let somedayTint = Color(hex: "#8878a0")
    static let deadlineRed = Color(hex: "#c04848")
    static let success = Color(hex: "#4a8860")
    static let agentTint = Color(hex: "#7868a8")
}
```

- [ ] **Step 2: Create DateFormatting.swift**

Port from the verified Rust implementation (12 unit tests passed):

```swift
import Foundation

enum DateFormatting {
    static func todayString() -> String {
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd"
        return fmt.string(from: Date())
    }

    static func formatRelative(_ dateStr: String) -> String {
        guard let date = parseDate(dateStr) else { return dateStr }
        let cal = Calendar.current
        let today = cal.startOfDay(for: Date())
        let target = cal.startOfDay(for: date)
        let days = cal.dateComponents([.day], from: today, to: target).day ?? 0

        switch days {
        case 0: return "Today"
        case 1: return "Tomorrow"
        case -1: return "Yesterday"
        case 2...6:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE"
            return fmt.string(from: date)
        case -6...(-2):
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE"
            return "Last \(fmt.string(from: date))"
        default:
            let fmt = DateFormatter()
            if cal.component(.year, from: date) == cal.component(.year, from: Date()) {
                fmt.dateFormat = "MMM d"
            } else {
                fmt.dateFormat = "MMM d, yyyy"
            }
            return fmt.string(from: date)
        }
    }

    static func formatDeadline(_ dateStr: String) -> (String, DeadlineVariant) {
        guard let date = parseDate(dateStr) else { return (dateStr, .normal) }
        let cal = Calendar.current
        let today = cal.startOfDay(for: Date())
        let target = cal.startOfDay(for: date)
        let days = cal.dateComponents([.day], from: today, to: target).day ?? 0

        switch days {
        case ..<0:
            let fmt = DateFormatter()
            fmt.dateFormat = "MMM d"
            return ("Overdue · \(fmt.string(from: date))", .overdue)
        case 0: return ("Due Today", .today)
        case 1: return ("Due Tomorrow", .normal)
        case 2...6:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE"
            return ("Due \(fmt.string(from: date))", .normal)
        default:
            let fmt = DateFormatter()
            fmt.dateFormat = "MMM d"
            return ("Due \(fmt.string(from: date))", .normal)
        }
    }

    enum DeadlineVariant {
        case normal, today, overdue
    }

    private static func parseDate(_ str: String) -> Date? {
        // Try YYYY-MM-DD
        let fmt1 = DateFormatter()
        fmt1.dateFormat = "yyyy-MM-dd"
        if let d = fmt1.date(from: str) { return d }
        // Try ISO8601 (from Go API)
        let fmt2 = ISO8601DateFormatter()
        fmt2.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = fmt2.date(from: str) { return d }
        fmt2.formatOptions = [.withInternetDateTime]
        if let d = fmt2.date(from: str) { return d }
        return nil
    }
}
```

- [ ] **Step 3: Port date formatting unit tests**

```swift
// DateFormattingTests.swift
import XCTest
@testable import atask

final class DateFormattingTests: XCTestCase {
    func testToday() {
        XCTAssertEqual(DateFormatting.formatRelative(DateFormatting.todayString()), "Today")
    }

    func testTomorrow() {
        let tomorrow = Calendar.current.date(byAdding: .day, value: 1, to: Date())!
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd"
        XCTAssertEqual(DateFormatting.formatRelative(fmt.string(from: tomorrow)), "Tomorrow")
    }

    func testYesterday() {
        let yesterday = Calendar.current.date(byAdding: .day, value: -1, to: Date())!
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd"
        XCTAssertEqual(DateFormatting.formatRelative(fmt.string(from: yesterday)), "Yesterday")
    }

    func testDeadlineToday() {
        let (label, variant) = DateFormatting.formatDeadline(DateFormatting.todayString())
        XCTAssertEqual(label, "Due Today")
        XCTAssertEqual(variant, .today)
    }

    func testDeadlineOverdue() {
        let past = Calendar.current.date(byAdding: .day, value: -3, to: Date())!
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd"
        let (label, variant) = DateFormatting.formatDeadline(fmt.string(from: past))
        XCTAssertTrue(label.starts(with: "Overdue"))
        XCTAssertEqual(variant, .overdue)
    }

    func testISODateTimeParsing() {
        // Go API format
        let iso = "\(DateFormatting.todayString())T00:00:00Z"
        XCTAssertEqual(DateFormatting.formatRelative(iso), "Today")
    }
}
```

- [ ] **Step 4: Add Atkinson Hyperlegible font**

Copy TTF files from `docs/design_specs/` or v2 assets into `atask-app/atask/Assets/Fonts/`. Register in Info.plist under `ATSApplicationFontsPath`.

- [ ] **Step 5: Create Typography.swift**

```swift
import SwiftUI

extension Font {
    static let atkinsonRegular = Font.custom("AtkinsonHyperlegible-Regular", size: 14)
    static let atkinsonBold = Font.custom("AtkinsonHyperlegible-Bold", size: 14)
    // Size variants
    static func atkinson(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        weight == .bold
            ? .custom("AtkinsonHyperlegible-Bold", size: size)
            : .custom("AtkinsonHyperlegible-Regular", size: size)
    }
}
```

- [ ] **Step 6: Run tests + build**

```bash
xcodebuild test -project atask-app/atask.xcodeproj -scheme atask -destination 'platform=macOS'
```

- [ ] **Step 7: Commit**

```bash
git add atask-app/
git commit -m "feat: add theme colors, date formatting with tests, Atkinson Hyperlegible font"
```

---

### Task 4: App Shell — NavigationSplitView + Sidebar

**Files:**
- Create: `atask-app/atask/Views/ContentView.swift`
- Create: `atask-app/atask/Views/Sidebar/SidebarView.swift`
- Modify: `atask-app/atask/ataskApp.swift`

- [ ] **Step 1: Create ContentView with NavigationSplitView**

```swift
import SwiftUI

struct ContentView: View {
    @State var store: TaskStore

    var body: some View {
        NavigationSplitView {
            SidebarView(store: store)
        } content: {
            Text("Task list will go here")
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .background(Theme.canvas)
        } detail: {
            if store.selectedTaskId != nil {
                Text("Detail panel will go here")
            } else {
                Text("Select a task")
                    .foregroundColor(Theme.inkTertiary)
            }
        }
        .navigationSplitViewStyle(.balanced)
    }
}
```

- [ ] **Step 2: Create SidebarView**

Areas with nested projects, nav items with badges:

```swift
struct SidebarView: View {
    @Bindable var store: TaskStore

    var body: some View {
        List(selection: Binding(
            get: { viewToSelection(store.activeView) },
            set: { if let v = $0 { store.activeView = selectionToView(v) } }
        )) {
            Section {
                Label("Inbox", systemImage: "tray")
                    .tag("inbox")
                    .badge(store.inbox.count)
                Label("Today", systemImage: "star.fill")
                    .tag("today")
                    .badge(store.today.count)
                    .foregroundColor(Theme.todayStar)
                Label("Upcoming", systemImage: "calendar")
                    .tag("upcoming")
                Label("Someday", systemImage: "clock")
                    .tag("someday")
                Label("Logbook", systemImage: "archivebox")
                    .tag("logbook")
            }

            // Areas with nested projects
            ForEach(store.areas) { area in
                Section(area.title) {
                    ForEach(store.projects.filter { $0.areaId == area.id }) { project in
                        Label(project.title, systemImage: "circle.fill")
                            .tag("project:\(project.id)")
                            .foregroundColor(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                    }
                }
            }

            // Orphan projects (no area)
            let orphans = store.projects.filter { $0.areaId == nil }
            if !orphans.isEmpty {
                Section {
                    ForEach(orphans) { project in
                        Label(project.title, systemImage: "circle.fill")
                            .tag("project:\(project.id)")
                    }
                }
            }
        }
        .listStyle(.sidebar)
    }
}
```

- [ ] **Step 3: Wire into ataskApp.swift**

```swift
@main
struct ataskApp: App {
    @State var store: TaskStore

    init() {
        let db = try! LocalDatabase()
        _store = State(initialValue: TaskStore(db: db))
    }

    var body: some Scene {
        WindowGroup {
            ContentView(store: store)
        }
    }
}
```

- [ ] **Step 4: Build and run**

Expected: Three-pane window with sidebar (nav items + areas + projects from local DB — initially empty).

- [ ] **Step 5: Commit**

```bash
git add atask-app/
git commit -m "feat: add NavigationSplitView shell with sidebar"
```

---

### Task 5: Task Row + Inline Editor

This is the core interaction. Build both the collapsed row (32pt) and the expanded inline editor card.

**Files:**
- Create: `atask-app/atask/Views/TaskList/TaskRow.swift`
- Create: `atask-app/atask/Views/TaskList/TaskInlineEditor.swift`
- Create: `atask-app/atask/Views/TaskList/TaskMetaView.swift`
- Create: `atask-app/atask/Views/TaskList/CheckboxView.swift`
- Create: `atask-app/atask/Views/TaskList/WhenPicker.swift`
- Create: `atask-app/atask/Views/TaskList/TagPickerView.swift`

This task is large — implement incrementally:

- [ ] **Step 1: CheckboxView** — circular, 20pt, amber variant for Today
- [ ] **Step 2: TaskMetaView** — right-aligned pills (project, deadline, checklist count)
- [ ] **Step 3: TaskRow** — collapsed 32pt row (checkbox + title + meta)
- [ ] **Step 4: WhenPicker** — popover with Today/Evening/calendar/Someday/Clear
- [ ] **Step 5: TagPickerView** — dark popover with searchable tag list
- [ ] **Step 6: TaskInlineEditor** — expanded card with title, notes, bottom bar actions
- [ ] **Step 7: Wire expand/collapse** — click row → expand, click outside → collapse
- [ ] **Step 8: Context menu on TaskRow** — `.contextMenu` with Complete, Schedule, Delete, etc.
- [ ] **Step 9: Build and verify** — create tasks, expand, edit, collapse
- [ ] **Step 10: Commit**

---

### Tasks 6-15: Remaining Features

Each follows the same pattern: create view/component → wire into ContentView → test → commit.

| Task | Feature | Key File |
|------|---------|----------|
| 6 | Today view with morning/evening sections | `Views/TaskList/TodayView.swift` |
| 7 | Remaining views (inbox, upcoming, someday, logbook, project) | `Views/TaskList/*.swift` |
| 8 | Detail panel (full view, right side) | `Views/Detail/TaskDetailView.swift` |
| 9 | Keyboard shortcuts (native SwiftUI `.keyboardShortcut`) | `Helpers/KeyboardShortcuts.swift` |
| 10 | Command palette (⇧⌘O overlay) | `Views/Overlays/CommandPaletteView.swift` |
| 11 | Drag and drop (`.draggable`, `.dropDestination`) | Modify TaskRow + views |
| 12 | Context menus (already added in Task 5, extend) | Modify TaskRow, SidebarView |
| 13 | Settings (⌘, — server URL, auto-archive) | `Views/Overlays/SettingsView.swift` |
| 14 | API client + Sync engine | `Store/APIClient.swift`, `Store/SyncEngine.swift` |
| 15 | Login (when server configured) | `Views/Login/LoginView.swift` |

Each task follows the same structure:
1. Write the code
2. Build and run
3. Verify the feature works
4. Write tests where applicable
5. Commit

---

## Summary

| Task | What | Verification |
|------|------|-------------|
| 0 | time_slot in Go API | `make test` + Playwright |
| 1 | Xcode + GRDB setup | Window opens |
| 2 | Models + TaskStore | Unit tests pass |
| 3 | Theme + dates | Date tests pass |
| 4 | App shell + sidebar | Three-pane renders |
| 5 | Task row + inline editor | Create, expand, edit, collapse |
| 6 | Today view | Morning/evening sections |
| 7 | All other views | Navigate, verify data |
| 8 | Detail panel | Full task view with activity |
| 9 | Keyboard shortcuts | ⌘K complete, ⌘T today, etc. |
| 10 | Command palette | ⇧⌘O search + execute |
| 11 | Drag and drop | Reorder tasks |
| 12 | Context menus | Right-click actions |
| 13 | Settings | Server URL config |
| 14 | API + Sync | Connect to Go server |
| 15 | Login | Auth when server configured |

**Total tests after completion:** Existing 45 Playwright (API) + new Swift unit tests (TaskStore, DateFormatting, SyncEngine)
