import GRDB
import Foundation

struct TagModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable, Hashable {
    static let databaseTableName = "tags"

    var id: String
    var title: String
    var index: Int
    var createdAt: String
    var updatedAt: String

    static func create(title: String) -> TagModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return TagModel(
            id: UUID().uuidString,
            title: title, index: 0,
            createdAt: now, updatedAt: now
        )
    }
}
