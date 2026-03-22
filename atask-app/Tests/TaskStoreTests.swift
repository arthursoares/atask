import Testing
@testable import atask

private func makeInboxStore() throws -> TaskStore {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)
    store.activeView = .inbox
    return store
}

@Test func createTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Test task")
    #expect(store.inbox.count == 1)
    #expect(store.inbox.first?.title == "Test task")
    #expect(task.status == 0)
    #expect(task.schedule == 0)
}

@Test func completeTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Complete me")
    store.completeTask(task.id)
    #expect(store.inbox.count == 1) // still visible with strikethrough (completed today)
    #expect(store.inbox.first?.isCompleted == true)
    #expect(store.logbook.count == 1)
}

@Test func reopenTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Reopen me")
    store.completeTask(task.id)
    #expect(store.logbook.count == 1)
    store.reopenTask(task.id)
    #expect(store.inbox.count == 1)
    #expect(store.logbook.count == 0)
}

@Test func scheduleToday() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Today task")
    store.setSchedule(task.id, 1)
    #expect(store.inbox.count == 0)
    #expect(store.today.count == 1)
}

@Test func scheduleSomeday() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Someday task")
    store.setSchedule(task.id, 2)
    #expect(store.inbox.count == 0)
    #expect(store.someday.count == 1)
}

@Test func inboxExcludesDated() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Dated task")
    store.setStartDate(task.id, "2026-04-01")
    #expect(store.inbox.count == 0)
}

@Test func somedayExcludesDated() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Dated someday")
    store.setSchedule(task.id, 2)
    store.setStartDate(task.id, "2026-04-01")
    #expect(store.someday.count == 0)
}

@Test func todayEvening() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Evening task")
    store.setSchedule(task.id, 1)
    store.setTimeSlot(task.id, "evening")
    #expect(store.todayEvening.count == 1)
    #expect(store.todayMorning.count == 0)
}

@Test func deleteTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Delete me")
    #expect(store.inbox.count == 1)
    store.deleteTask(task.id)
    #expect(store.inbox.count == 0)
}

@Test func createProject() throws {
    let store = try makeInboxStore()
    let project = store.createProject(title: "My Project")
    #expect(store.projects.count == 1)
    #expect(project.title == "My Project")
}

@Test func createArea() throws {
    let store = try makeInboxStore()
    let area = store.createArea(title: "Work")
    #expect(store.areas.count == 1)
    #expect(area.title == "Work")
}

@Test func uniqueTags() throws {
    let store = try makeInboxStore()
    let tag1 = store.createTag(title: "important")
    #expect(tag1 != nil)
    let tag2 = store.createTag(title: "important")
    #expect(tag2 == nil)
    #expect(store.tags.count == 1)
}

@Test func contextAwareCreation() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    // Today view creates anytime tasks
    store.activeView = .today
    let todayTask = store.createTask(title: "Today task")
    #expect(todayTask.schedule == 1)

    // Someday view creates someday tasks
    store.activeView = .someday
    let somedayTask = store.createTask(title: "Someday task")
    #expect(somedayTask.schedule == 2)

    // Inbox creates inbox tasks
    store.activeView = .inbox
    let inboxTask = store.createTask(title: "Inbox task")
    #expect(inboxTask.schedule == 0)
}

// MARK: - Tag Associations

@Test func addTagToTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Tagged task")
    let tag = store.createTag(title: "urgent")!

    store.addTagToTask(task.id, tagId: tag.id)
    let tags = store.tagsForTask(task.id)
    #expect(tags.count == 1)
    #expect(tags.first?.title == "urgent")
}

@Test func removeTagFromTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Tagged task")
    let tag = store.createTag(title: "urgent")!

    store.addTagToTask(task.id, tagId: tag.id)
    #expect(store.tagsForTask(task.id).count == 1)

    store.removeTagFromTask(task.id, tagId: tag.id)
    #expect(store.tagsForTask(task.id).count == 0)
}

@Test func multipleTagsOnTask() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "Multi-tagged")
    let t1 = store.createTag(title: "alpha")!
    let t2 = store.createTag(title: "beta")!

    store.addTagToTask(task.id, tagId: t1.id)
    store.addTagToTask(task.id, tagId: t2.id)
    #expect(store.tagsForTask(task.id).count == 2)

    // Duplicate add is ignored (INSERT OR IGNORE)
    store.addTagToTask(task.id, tagId: t1.id)
    #expect(store.tagsForTask(task.id).count == 2)
}

// MARK: - Reorder

@Test func reorderTask() throws {
    let store = try makeInboxStore()
    let t1 = store.createTask(title: "First")
    let t2 = store.createTask(title: "Second")
    let t3 = store.createTask(title: "Third")

    // Move third to first position
    store.reorderTask(t3.id, toIndex: 0)
    let inbox = store.inbox
    #expect(inbox[0].id == t3.id)
}

@Test func moveTaskUpDown() throws {
    let store = try makeInboxStore()
    let t1 = store.createTask(title: "First")
    let t2 = store.createTask(title: "Second")

    store.moveTaskDown(t1.id)
    // After moving down, t1 should be at index 1
    let inbox = store.inbox
    #expect(inbox.last?.id == t1.id)
}

// MARK: - Sync Callback

@Test func onMutationFires() throws {
    let store = try makeInboxStore()
    var mutations: [(String, String)] = []
    store.onMutation = { method, path, _ in
        mutations.append((method, path))
    }

    let task = store.createTask(title: "Synced")
    #expect(mutations.count == 1)
    #expect(mutations[0].0 == "POST")

    store.completeTask(task.id)
    #expect(mutations.count == 2)
    #expect(mutations[1].1.contains("/complete"))

    store.deleteTask(task.id)
    #expect(mutations.count == 3)
    #expect(mutations[2].0 == "DELETE")
}

// MARK: - Checklist

@Test func checklistCRUD() throws {
    let store = try makeInboxStore()
    let task = store.createTask(title: "With checklist")

    let item = store.addChecklistItem(title: "Step 1", taskId: task.id)
    #expect(store.checklistFor(task.id).count == 1)

    store.toggleChecklistItem(item.id)
    let updated = store.checklistFor(task.id)
    #expect(updated.first?.isCompleted == true)

    store.deleteChecklistItem(item.id)
    #expect(store.checklistFor(task.id).count == 0)
}

// MARK: - Project Operations

@Test func deleteProjectCleansUpTasks() throws {
    let store = try makeInboxStore()
    let project = store.createProject(title: "My Project")
    store.activeView = .project(project.id)
    let task = store.createTask(title: "Project task")
    #expect(task.projectId == project.id)

    store.deleteProject(project.id)
    #expect(store.projects.isEmpty)
    // Task should have nil projectId after project deletion
    let updated = store.tasks.first { $0.id == task.id }
    #expect(updated?.projectId == nil)
}

@Test func moveProjectToArea() throws {
    let store = try makeInboxStore()
    let area = store.createArea(title: "Work")
    let project = store.createProject(title: "Project")

    store.moveProjectToArea(project.id, area.id)
    #expect(store.projects.first?.areaId == area.id)

    store.moveProjectToArea(project.id, nil)
    #expect(store.projects.first?.areaId == nil)
}
