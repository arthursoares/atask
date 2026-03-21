import GRDB
import Foundation

struct ChecklistItemModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable, Hashable {
    static let databaseTableName = "checklistItems"

    var id: String
    var title: String
    var status: Int  // 0=pending, 1=completed
    var taskId: String
    var index: Int
    var createdAt: String
    var updatedAt: String

    var isCompleted: Bool { status == 1 }

    static func create(title: String, taskId: String) -> ChecklistItemModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return ChecklistItemModel(
            id: UUID().uuidString,
            title: title, status: 0, taskId: taskId, index: 0,
            createdAt: now, updatedAt: now
        )
    }
}
