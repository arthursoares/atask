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
        return completedAt.hasPrefix(DateFormatting.todayString())
    }

    // MARK: - Task Mutations (local-first)

    /// Create task in inbox by default. Use `createTaskInView` to create in the active view's context.
    @discardableResult
    func createTask(title: String) -> TaskModel {
        let task = TaskModel.create(title: title)
        persist(task)
        tasks.append(task)
        return task
    }

    /// Create task with schedule/project matching the active view.
    @discardableResult
    func createTaskInView(title: String) -> TaskModel {
        var task = TaskModel.create(title: title)
        switch activeView {
        case .today:
            task.schedule = 1
        case .someday:
            task.schedule = 2
        case .project(let pid):
            task.projectId = pid
        default:
            break // inbox
        }
        persist(task)
        tasks.append(task)
        return task
    }

    func completeTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 1
        tasks[idx].completedAt = ISO8601DateFormatter().string(from: Date())
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func cancelTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 2
        tasks[idx].completedAt = ISO8601DateFormatter().string(from: Date())
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func reopenTask(_ id: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].status = 0
        tasks[idx].completedAt = nil
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func updateTitle(_ id: String, _ title: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].title = title
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func updateNotes(_ id: String, _ notes: String) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].notes = notes
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func setSchedule(_ id: String, _ schedule: Int) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].schedule = schedule
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func setTimeSlot(_ id: String, _ slot: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].timeSlot = slot
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func setStartDate(_ id: String, _ date: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].startDate = date
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func setDeadline(_ id: String, _ date: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].deadline = date
        tasks[idx].touch()
        persist(tasks[idx])
    }

    func moveToProject(_ id: String, _ projectId: String?) {
        guard let idx = tasks.firstIndex(where: { $0.id == id }) else { return }
        tasks[idx].projectId = projectId
        if projectId == nil { tasks[idx].sectionId = nil }
        tasks[idx].touch()
        persist(tasks[idx])
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
    }

    // MARK: - Project Mutations

    @discardableResult
    func createProject(title: String, areaId: String? = nil) -> ProjectModel {
        let project = ProjectModel.create(title: title, areaId: areaId)
        persist(project)
        projects.append(project)
        return project
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

    // MARK: - Section Mutations

    @discardableResult
    func createSection(title: String, projectId: String) -> SectionModel {
        let section = SectionModel.create(title: title, projectId: projectId)
        persist(section)
        sections.append(section)
        return section
    }

    // MARK: - Persistence Helpers

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
