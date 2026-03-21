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
