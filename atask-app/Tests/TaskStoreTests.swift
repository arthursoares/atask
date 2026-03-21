import Testing
@testable import atask

@Test func createTask() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Test task")
    #expect(store.inbox.count == 1)
    #expect(store.inbox.first?.title == "Test task")
    #expect(task.status == 0) // pending
    #expect(task.schedule == 0) // inbox
}

@Test func completeTask() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Complete me")
    store.completeTask(task.id)
    #expect(store.inbox.count == 0)
    #expect(store.logbook.count == 1)
    #expect(store.logbook.first?.isCompleted == true)
}

@Test func reopenTask() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Reopen me")
    store.completeTask(task.id)
    #expect(store.logbook.count == 1)

    store.reopenTask(task.id)
    #expect(store.inbox.count == 1)
    #expect(store.logbook.count == 0)
}

@Test func scheduleToday() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Today task")
    store.setSchedule(task.id, 1) // anytime
    #expect(store.inbox.count == 0)
    #expect(store.today.count == 1)
}

@Test func scheduleSomeday() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Someday task")
    store.setSchedule(task.id, 2) // someday
    #expect(store.inbox.count == 0)
    #expect(store.someday.count == 1)
}

@Test func inboxExcludesDated() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Dated task")
    store.setStartDate(task.id, "2026-04-01")
    #expect(store.inbox.count == 0) // dated tasks not in inbox
}

@Test func somedayExcludesDated() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Dated someday")
    store.setSchedule(task.id, 2)
    store.setStartDate(task.id, "2026-04-01")
    #expect(store.someday.count == 0) // dated tasks not in someday
}

@Test func todayEvening() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Evening task")
    store.setSchedule(task.id, 1)
    store.setTimeSlot(task.id, "evening")
    #expect(store.todayEvening.count == 1)
    #expect(store.todayMorning.count == 0)
}

@Test func deleteTask() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let task = store.createTask(title: "Delete me")
    #expect(store.inbox.count == 1)
    store.deleteTask(task.id)
    #expect(store.inbox.count == 0)
}

@Test func createProject() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let project = store.createProject(title: "My Project")
    #expect(store.projects.count == 1)
    #expect(project.title == "My Project")
}

@Test func createArea() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let area = store.createArea(title: "Work")
    #expect(store.areas.count == 1)
    #expect(area.title == "Work")
}

@Test func uniqueTags() throws {
    let db = try LocalDatabase(inMemory: true)
    let store = TaskStore(db: db)

    let tag1 = store.createTag(title: "important")
    #expect(tag1 != nil)

    let tag2 = store.createTag(title: "important") // duplicate
    #expect(tag2 == nil) // should fail
    #expect(store.tags.count == 1)
}
