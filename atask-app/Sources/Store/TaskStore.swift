import SwiftUI
import GRDB

@Observable
class TaskStore {
    let db: LocalDatabase

    private(set) var tasks: [TaskModel] = []
    private(set) var projects: [ProjectModel] = []
    private(set) var areas: [AreaModel] = []
    private(set) var tags: [TagModel] = []
    private(set) var sections: [SectionModel] = []

    var selectedTaskId: String?
    var expandedTaskId: String?
    var activeView: ActiveView = .today
    var sidebarSelection: SidebarItem? = .today
    var showCommandPalette = false
    var showWhenPicker = false

    /// Called on every mutation — sync engine hooks this to enqueue outbound ops.
    var onMutation: ((_ method: String, _ path: String, _ body: String?) -> Void)?

    init(db: LocalDatabase) {
        self.db = db
        reload()
    }

    // MARK: - Load from DB

    func reload() {
        do {
            try db.dbQueue.read { db in
                self.tasks = try TaskModel.fetchAll(db)
                self.projects = try ProjectModel.fetchAll(db)
                self.areas = try AreaModel.fetchAll(db)
                self.tags = try TagModel.fetchAll(db)
                self.sections = try SectionModel.fetchAll(db)
            }
        } catch {
            print("[TaskStore] Reload failed: \(error)")
        }
    }

    // MARK: - Computed Views

    var inbox: [TaskModel] {
        tasks.filter { ($0.isPending || completedToday($0)) && $0.schedule == 0 && $0.startDate == nil }
            .sorted { $0.index < $1.index }
    }

    var today: [TaskModel] {
        let todayStr = DateFormatting.todayString()
        return tasks.filter { task in
            task.schedule == 1 &&
            (task.isPending || task.isCompleted) &&
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
        tasks.filter { ($0.isPending || completedToday($0)) && $0.schedule == 2 && $0.startDate == nil }
            .sorted { $0.index < $1.index }
    }

    var logbook: [TaskModel] {
        tasks.filter { $0.isCompleted || $0.isCancelled }
            .sorted { ($0.completedAt ?? "") > ($1.completedAt ?? "") }
    }

    func tasksForProject(_ projectId: String) -> [TaskModel] {
        tasks.filter { $0.projectId == projectId && ($0.isPending || completedToday($0)) }
            .sorted { $0.index < $1.index }
    }

    /// Returns true if the task was completed today (stays visible with strikethrough)
    private func completedToday(_ task: TaskModel) -> Bool {
        guard task.isCompleted, let completedAt = task.completedAt else { return false }
        // Parse the ISO date and check if it's today in local timezone
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        guard let date = iso.date(from: completedAt) else {
            // Fallback: try prefix match
            return completedAt.hasPrefix(DateFormatting.todayString())
        }
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd"
        return fmt.string(from: date) == DateFormatting.todayString()
    }

    // MARK: - Task Mutations (local-first)

    /// Create task with context-aware defaults based on activeView.
    /// Today → schedule=1, Someday → schedule=2, Project → projectId set, else inbox.
    @discardableResult
    func createTask(title: String) -> TaskModel {
        var task = TaskModel.create(title: title)
        switch activeView {
        case .today:
            task.schedule = 1
        case .someday:
            task.schedule = 2
        case .project(let pid):
            task.projectId = pid
        default:
            break // inbox (schedule=0)
        }
        persist(task)
        tasks.append(task)
        let scheduleStr = ["inbox", "anytime", "someday"][task.schedule]
        onMutation?("POST", "/tasks", "{\"title\":\"\(task.title)\",\"schedule\":\"\(scheduleStr)\"}")
        return task
    }

    func completeTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 1
        tasks[idx].completedAt = ISO8601DateFormatter().string(from: Date())
        tasks[idx].touch()
        persist(tasks[idx])
        onMutation?("POST", "/tasks/\(id)/complete", nil)
    }

    func cancelTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 2
        tasks[idx].completedAt = ISO8601DateFormatter().string(from: Date())
        tasks[idx].touch()
        persist(tasks[idx])
        onMutation?("POST", "/tasks/\(id)/cancel", nil)
    }

    func reopenTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 0
        tasks[idx].completedAt = nil
        tasks[idx].touch()
        persist(tasks[idx])
        onMutation?("POST", "/tasks/\(id)/reopen", nil)
    }

    func updateTitle(_ id: String, _ title: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].title = title
        tasks[idx].touch()
        persist(tasks[idx])
        onMutation?("PUT", "/tasks/\(id)/title", "{\"title\":\"\(title)\"}")
    }

    func updateNotes(_ id: String, _ notes: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].notes = notes
        tasks[idx].touch()
        persist(tasks[idx])
        onMutation?("PUT", "/tasks/\(id)/notes", "{\"notes\":\"\(notes)\"}")
    }

    func setSchedule(_ id: String, _ schedule: Int) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].schedule = schedule
        tasks[idx].touch()
        persist(tasks[idx])
        let scheduleStr = ["inbox", "anytime", "someday"][schedule]
        onMutation?("PUT", "/tasks/\(id)/schedule", "{\"schedule\":\"\(scheduleStr)\"}")
    }

    func setTimeSlot(_ id: String, _ slot: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].timeSlot = slot
        tasks[idx].touch()
        persist(tasks[idx])
        let json = slot.map { "{\"time_slot\":\"\($0)\"}" } ?? "{\"time_slot\":null}"
        onMutation?("PUT", "/tasks/\(id)/time-slot", json)
    }

    func setStartDate(_ id: String, _ date: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].startDate = date
        tasks[idx].touch()
        persist(tasks[idx])
        let json = date.map { "{\"start_date\":\"\($0)\"}" } ?? "{\"start_date\":null}"
        onMutation?("PUT", "/tasks/\(id)/start-date", json)
    }

    func setDeadline(_ id: String, _ date: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].deadline = date
        tasks[idx].touch()
        persist(tasks[idx])
        let json = date.map { "{\"deadline\":\"\($0)\"}" } ?? "{\"deadline\":null}"
        onMutation?("PUT", "/tasks/\(id)/deadline", json)
    }

    func moveToProject(_ id: String, _ projectId: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].projectId = projectId
        if projectId == nil { tasks[idx].sectionId = nil }
        tasks[idx].touch()
        persist(tasks[idx])
        let json = projectId.map { "{\"project_id\":\"\($0)\"}" } ?? "{\"project_id\":null}"
        onMutation?("PUT", "/tasks/\(id)/project", json)
    }

    // MARK: - Reorder

    /// Reorder a task within its current view list by moving it to a new position.
    func reorderTask(_ id: String, toIndex newIndex: Int) {
        let list = reorderableList()
        guard let currentIdx = list.firstIndex(where: { $0.id == id }) else { return }
        guard newIndex >= 0 && newIndex < list.count else { return }
        guard currentIdx != newIndex else { return }

        // Build the ordered ID list, move the task
        var ids = list.map(\.id)
        ids.remove(at: currentIdx)
        ids.insert(id, at: min(newIndex, ids.count))

        // Update index/todayIndex for all tasks in the list
        for (i, taskId) in ids.enumerated() {
            guard let idx = tasks.firstIndex(where: { $0.id == taskId }) else { continue }
            if activeView == .today {
                tasks[idx].todayIndex = i
            } else {
                tasks[idx].index = i
            }
            tasks[idx].touch()
            persist(tasks[idx])
        }
    }

    func moveTaskUp(_ id: String) {
        let list = reorderableList()
        guard let idx = list.firstIndex(where: { $0.id == id }) else { return }
        reorderTask(id, toIndex: idx - 1)
    }

    func moveTaskDown(_ id: String) {
        let list = reorderableList()
        guard let idx = list.firstIndex(where: { $0.id == id }) else { return }
        reorderTask(id, toIndex: idx + 1)
    }

    /// Returns the task list for the current view that supports reordering.
    private func reorderableList() -> [TaskModel] {
        switch activeView {
        case .inbox: return inbox
        case .today: return today
        case .someday: return someday
        case .project(let id): return tasksForProject(id)
        default: return []
        }
    }

    func deleteTask(_ id: String) {
        tasks.removeAll { $0.id == id }
        do {
            try db.dbQueue.write { db in
                _ = try TaskModel.deleteOne(db, id: id)
            }
        } catch {
            print("[TaskStore] Delete failed: \(error)")
        }
        onMutation?("DELETE", "/tasks/\(id)", nil)
    }

    // MARK: - Project Mutations

    @discardableResult
    func createProject(title: String, areaId: String? = nil) -> ProjectModel {
        let project = ProjectModel.create(title: title, areaId: areaId)
        persist(project)
        projects.append(project)
        return project
    }

    func deleteProject(_ id: String) {
        // Remove tasks from project first
        for i in tasks.indices where tasks[i].projectId == id {
            tasks[i].projectId = nil
            tasks[i].sectionId = nil
            tasks[i].touch()
            persist(tasks[i])
        }
        // Remove sections
        sections.removeAll { $0.projectId == id }
        // Remove project
        projects.removeAll { $0.id == id }
        do {
            try db.dbQueue.write { db in
                _ = try ProjectModel.deleteOne(db, id: id)
            }
        } catch {
            print("[TaskStore] Delete project failed: \(error)")
        }
        if case .project(let pid) = activeView, pid == id {
            activeView = .inbox
            sidebarSelection = .inbox
        }
    }

    func moveProjectToArea(_ projectId: String, _ areaId: String?) {
        guard let idx = projects.firstIndex(where: { $0.id == projectId }) else { return }
        projects[idx].areaId = areaId
        projects[idx].updatedAt = ISO8601DateFormatter().string(from: Date())
        persist(projects[idx])
    }

    // MARK: - Area Mutations

    @discardableResult
    func createArea(title: String) -> AreaModel {
        let area = AreaModel.create(title: title)
        persist(area)
        areas.append(area)
        return area
    }

    // MARK: - Tag Mutations

    @discardableResult
    func createTag(title: String) -> TagModel? {
        let tag = TagModel.create(title: title)
        do {
            try db.dbQueue.write { db in try tag.insert(db) }
            tags.append(tag)
            return tag
        } catch {
            print("[TaskStore] Create tag failed (duplicate?): \(error)")
            return nil
        }
    }

    // MARK: - Task-Tag Associations

    func tagsForTask(_ taskId: String) -> [TagModel] {
        do {
            return try db.dbQueue.read { db in
                let sql = """
                    SELECT tags.* FROM tags
                    JOIN taskTags ON taskTags.tagId = tags.id
                    WHERE taskTags.taskId = ?
                    ORDER BY tags.title
                """
                return try TagModel.fetchAll(db, sql: sql, arguments: [taskId])
            }
        } catch { return [] }
    }

    func addTagToTask(_ taskId: String, tagId: String) {
        do {
            try db.dbQueue.write { db in
                try db.execute(
                    sql: "INSERT OR IGNORE INTO taskTags (taskId, tagId) VALUES (?, ?)",
                    arguments: [taskId, tagId]
                )
            }
        } catch {
            print("[TaskStore] Add tag to task failed: \(error)")
        }
    }

    func removeTagFromTask(_ taskId: String, tagId: String) {
        do {
            try db.dbQueue.write { db in
                try db.execute(
                    sql: "DELETE FROM taskTags WHERE taskId = ? AND tagId = ?",
                    arguments: [taskId, tagId]
                )
            }
        } catch {
            print("[TaskStore] Remove tag from task failed: \(error)")
        }
    }

    // MARK: - Section Mutations

    @discardableResult
    func createSection(title: String, projectId: String) -> SectionModel {
        let section = SectionModel.create(title: title, projectId: projectId)
        persist(section)
        sections.append(section)
        return section
    }

    // MARK: - Persistence Helpers

    // MARK: - Checklist

    func checklistFor(_ taskId: String) -> [ChecklistItemModel] {
        do {
            return try db.dbQueue.read { db in
                try ChecklistItemModel
                    .filter(Column("taskId") == taskId)
                    .order(Column("index"))
                    .fetchAll(db)
            }
        } catch { return [] }
    }

    @discardableResult
    func addChecklistItem(title: String, taskId: String) -> ChecklistItemModel {
        let item = ChecklistItemModel.create(title: title, taskId: taskId)
        persist(item)
        return item
    }

    func toggleChecklistItem(_ id: String) {
        do {
            try db.dbQueue.write { db in
                if var item = try ChecklistItemModel.fetchOne(db, id: id) {
                    item.status = item.isCompleted ? 0 : 1
                    item.updatedAt = ISO8601DateFormatter().string(from: Date())
                    try item.update(db)
                }
            }
        } catch { print("[TaskStore] Toggle checklist failed: \(error)") }
    }

    func deleteChecklistItem(_ id: String) {
        do {
            try db.dbQueue.write { db in
                _ = try ChecklistItemModel.deleteOne(db, id: id)
            }
        } catch { print("[TaskStore] Delete checklist failed: \(error)") }
    }

    /// Look up project for a task
    func projectFor(_ task: TaskModel) -> ProjectModel? {
        guard let pid = task.projectId else { return nil }
        return projects.first { $0.id == pid }
    }

    /// Persist a new task directly (used by project view for section-specific creation)
    func persist(task: inout TaskModel) {
        do {
            try db.dbQueue.write { db in try task.save(db) }
            tasks.append(task)
        } catch {
            print("[TaskStore] Persist task failed: \(error)")
        }
    }

    private func persist(_ record: some PersistableRecord) {
        do {
            try db.dbQueue.write { db in
                try record.save(db)
            }
        } catch {
            print("[TaskStore] Persist failed: \(error)")
        }
    }
}

// MARK: - Active View

enum ActiveView: Equatable, Hashable {
    case inbox, today, upcoming, someday, logbook
    case project(String)
}
