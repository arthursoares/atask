import GRDB
import Foundation

struct AreaModel: Codable, FetchableRecord, PersistableRecord, Identifiable, Equatable, Hashable {
    static let databaseTableName = "areas"

    var id: String
    var title: String
    var index: Int
    var archived: Bool
    var createdAt: String
    var updatedAt: String

    static func create(title: String) -> AreaModel {
        let now = ISO8601DateFormatter().string(from: Date())
        return AreaModel(
            id: UUID().uuidString,
            title: title, index: 0, archived: false,
            createdAt: now, updatedAt: now
        )
    }
}
