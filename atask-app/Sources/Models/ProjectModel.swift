import GRDB
import Foundation

struct ProjectModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable, Hashable {
    static let databaseTableName = "projects"

    var id: String
    var title: String
    var notes: String
    var status: Int
    var color: String
    var areaId: String?
    var index: Int
    var completedAt: String?
    var createdAt: String
    var updatedAt: String

    var isCompleted: Bool { completedAt != nil }

    static func create(title: String, areaId: String? = nil) -> ProjectModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return ProjectModel(
            id: UUID().uuidString,
            title: title, notes: "", status: 0, color: "",
            areaId: areaId, index: 0, completedAt: nil,
            createdAt: now, updatedAt: now
        )
    }
}
