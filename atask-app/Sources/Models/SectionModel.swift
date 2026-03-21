import GRDB
import Foundation

struct SectionModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable, Hashable {
    static let databaseTableName = "sections"

    var id: String
    var title: String
    var projectId: String
    var index: Int
    var createdAt: String
    var updatedAt: String

    static func create(title: String, projectId: String) -> SectionModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return SectionModel(
            id: UUID().uuidString,
            title: title, projectId: projectId, index: 0,
            createdAt: now, updatedAt: now
        )
    }
}
