import GRDB
import Foundation

final class LocalDatabase: Sendable {
    let dbQueue: DatabaseQueue

    /// Production: persists to ~/Library/Application Support/atask/atask.sqlite
    init() throws {
        let appSupport = FileManager.default.urls(
            for: .applicationSupportDirectory, in: .userDomainMask
        ).first!
        let dbDir = appSupport.appendingPathComponent("atask", isDirectory: true)
        try FileManager.default.createDirectory(at: dbDir, withIntermediateDirectories: true)
        let dbPath = dbDir.appendingPathComponent("atask.sqlite").path
        dbQueue = try DatabaseQueue(path: dbPath)
        try migrate()
    }

    /// In-memory for testing
    init(inMemory: Bool) throws {
        dbQueue = try DatabaseQueue()
        try migrate()
    }

    private func migrate() throws {
        var migrator = DatabaseMigrator()

        migrator.registerMigration("v1_schema") { db in
            // Areas
            try db.create(table: "areas") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("index", .integer).defaults(to: 0)
                t.column("archived", .boolean).defaults(to: false)
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
            }

            // Projects
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

            // Sections
            try db.create(table: "sections") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("projectId", .text).notNull().references("projects")
                t.column("index", .integer).defaults(to: 0)
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
            }

            // Tasks
            try db.create(table: "tasks") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("notes", .text).defaults(to: "")
                t.column("status", .integer).defaults(to: 0)   // 0=pending, 1=completed, 2=cancelled
                t.column("schedule", .integer).defaults(to: 0) // 0=inbox, 1=anytime, 2=someday
                t.column("startDate", .text)
                t.column("deadline", .text)
                t.column("completedAt", .text)
                t.column("index", .integer).defaults(to: 0)
                t.column("todayIndex", .integer)
                t.column("timeSlot", .text) // nil, "morning", "evening"
                t.column("projectId", .text).references("projects")
                t.column("sectionId", .text).references("sections")
                t.column("areaId", .text).references("areas")
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
                t.column("syncStatus", .integer).defaults(to: 0) // 0=local, 1=synced
            }

            // Tags (unique title)
            try db.create(table: "tags") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).notNull().unique()
                t.column("index", .integer).defaults(to: 0)
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
            }

            // Task-Tag join
            try db.create(table: "taskTags") { t in
                t.column("taskId", .text).notNull().references("tasks")
                t.column("tagId", .text).notNull().references("tags")
                t.primaryKey(["taskId", "tagId"])
            }

            // Checklist items
            try db.create(table: "checklistItems") { t in
                t.primaryKey("id", .text)
                t.column("title", .text).defaults(to: "")
                t.column("status", .integer).defaults(to: 0) // 0=pending, 1=completed
                t.column("taskId", .text).notNull().references("tasks")
                t.column("index", .integer).defaults(to: 0)
                t.column("createdAt", .text).notNull()
                t.column("updatedAt", .text).notNull()
            }

            // Pending sync operations (outbound queue)
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
