import GRDB
import Foundation

struct TaskModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable, Hashable {
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

    // MARK: - Computed

    var isPending: Bool { status == 0 }
    var isCompleted: Bool { status == 1 }
    var isCancelled: Bool { status == 2 }

    // MARK: - Factory

    static func create(title: String, schedule: Int = 0, projectId: String? = nil) -> TaskModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return TaskModel(
            id: UUID().uuidString,
            title: title, notes: "", status: 0, schedule: schedule,
            startDate: nil, deadline: nil, completedAt: nil,
            index: 0, todayIndex: nil, timeSlot: nil,
            projectId: projectId, sectionId: nil, areaId: nil,
            createdAt: now, updatedAt: now, syncStatus: 0
        )
    }

    mutating func touch() {
        updatedAt = ISO8601DateFormatter().string(from: Date())
    }
}
