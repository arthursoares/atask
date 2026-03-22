import Foundation

/// HTTP client for the atask Go API.
/// Handles JWT auth, JSON encoding/decoding, and error mapping.
actor APIClient {
    private let session: URLSession
    private var baseURL: URL?
    private var token: String?

    init() {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        self.session = URLSession(configuration: config)
    }

    func configure(baseURL: String, token: String?) {
        self.baseURL = URL(string: baseURL)
        self.token = token
    }

    var isConfigured: Bool { baseURL != nil }

    // MARK: - Auth

    struct AuthResponse: Codable {
        let token: String
        let user: UserResponse
    }

    struct UserResponse: Codable {
        let id: String
        let email: String
        let name: String
    }

    func login(email: String, password: String) async throws -> AuthResponse {
        let body: [String: String] = ["email": email, "password": password]
        return try await post("/auth/login", body: body)
    }

    func register(email: String, password: String, name: String) async throws -> AuthResponse {
        let body: [String: String] = ["email": email, "password": password, "name": name]
        return try await post("/auth/register", body: body)
    }

    // MARK: - Tasks

    struct TaskResponse: Codable {
        let id: String
        let title: String
        let notes: String
        let status: String
        let schedule: String
        let startDate: String?
        let deadline: String?
        let completedAt: String?
        let index: Int
        let todayIndex: Int?
        let timeSlot: String?
        let projectId: String?
        let sectionId: String?
        let areaId: String?
        let createdAt: String
        let updatedAt: String

        enum CodingKeys: String, CodingKey {
            case id, title, notes, status, schedule
            case startDate = "start_date"
            case deadline
            case completedAt = "completed_at"
            case index
            case todayIndex = "today_index"
            case timeSlot = "time_slot"
            case projectId = "project_id"
            case sectionId = "section_id"
            case areaId = "area_id"
            case createdAt = "created_at"
            case updatedAt = "updated_at"
        }
    }

    func listTasks() async throws -> [TaskResponse] {
        try await get("/tasks")
    }

    func createTask(title: String, schedule: String = "inbox") async throws -> TaskResponse {
        try await post("/tasks", body: ["title": title, "schedule": schedule])
    }

    func completeTask(_ id: String) async throws {
        let _: EmptyBody = try await post("/tasks/\(id)/complete", body: EmptyBody())
    }

    func cancelTask(_ id: String) async throws {
        let _: EmptyBody = try await post("/tasks/\(id)/cancel", body: EmptyBody())
    }

    func reopenTask(_ id: String) async throws {
        let _: EmptyBody = try await post("/tasks/\(id)/reopen", body: EmptyBody())
    }

    func updateTaskTitle(_ id: String, _ title: String) async throws {
        let _: EmptyBody = try await put("/tasks/\(id)/title", body: ["title": title])
    }

    func updateTaskNotes(_ id: String, _ notes: String) async throws {
        let _: EmptyBody = try await put("/tasks/\(id)/notes", body: ["notes": notes])
    }

    func updateTaskSchedule(_ id: String, _ schedule: String) async throws {
        let _: EmptyBody = try await put("/tasks/\(id)/schedule", body: ["schedule": schedule])
    }

    func deleteTask(_ id: String) async throws {
        try await delete("/tasks/\(id)")
    }

    // MARK: - Projects

    struct ProjectResponse: Codable {
        let id: String
        let title: String
        let notes: String
        let status: Int
        let color: String
        let areaId: String?
        let index: Int
        let completedAt: String?
        let createdAt: String
        let updatedAt: String

        enum CodingKeys: String, CodingKey {
            case id, title, notes, status, color, index
            case areaId = "area_id"
            case completedAt = "completed_at"
            case createdAt = "created_at"
            case updatedAt = "updated_at"
        }
    }

    func listProjects() async throws -> [ProjectResponse] {
        try await get("/projects")
    }

    // MARK: - Areas

    struct AreaResponse: Codable {
        let id: String
        let title: String
        let index: Int
        let createdAt: String
        let updatedAt: String

        enum CodingKeys: String, CodingKey {
            case id, title, index
            case createdAt = "created_at"
            case updatedAt = "updated_at"
        }
    }

    func listAreas() async throws -> [AreaResponse] {
        try await get("/areas")
    }

    // MARK: - Tags

    struct TagResponse: Codable {
        let id: String
        let title: String
        let index: Int
        let createdAt: String
        let updatedAt: String

        enum CodingKeys: String, CodingKey {
            case id, title, index
            case createdAt = "created_at"
            case updatedAt = "updated_at"
        }
    }

    func listTags() async throws -> [TagResponse] {
        try await get("/tags")
    }

    // MARK: - Sync Deltas

    struct DeltaEvent: Codable {
        let id: Int
        let entityType: String
        let entityId: String
        let field: String
        let oldValue: String?
        let newValue: String?
        let actorId: String
        let createdAt: String

        enum CodingKeys: String, CodingKey {
            case id
            case entityType = "entity_type"
            case entityId = "entity_id"
            case field
            case oldValue = "old_value"
            case newValue = "new_value"
            case actorId = "actor_id"
            case createdAt = "created_at"
        }
    }

    func fetchDeltas(since: Int) async throws -> [DeltaEvent] {
        try await get("/sync/deltas?since=\(since)")
    }

    // MARK: - Generic Request (for sync engine)

    /// Execute a raw HTTP request — used by SyncEngine to replay pending ops.
    func execute(method: String, path: String, body: String?) async throws {
        var request = try makeRequest(path, method: method)
        if let body, let data = body.data(using: .utf8) {
            request.httpBody = data
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        }
        let (data, response) = try await session.data(for: request)
        try checkResponse(response, data: data)
    }

    // MARK: - Generic HTTP

    private struct EmptyBody: Codable {}

    private func get<T: Decodable>(_ path: String) async throws -> T {
        let request = try makeRequest(path, method: "GET")
        let (data, response) = try await session.data(for: request)
        try checkResponse(response, data: data)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func post<B: Encodable, T: Decodable>(_ path: String, body: B) async throws -> T {
        var request = try makeRequest(path, method: "POST")
        request.httpBody = try JSONEncoder().encode(body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let (data, response) = try await session.data(for: request)
        try checkResponse(response, data: data)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func put<B: Encodable>(_ path: String, body: B) async throws -> EmptyBody {
        var request = try makeRequest(path, method: "PUT")
        request.httpBody = try JSONEncoder().encode(body)
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        let (data, response) = try await session.data(for: request)
        try checkResponse(response, data: data)
        // Some PUT endpoints return the updated entity, some return empty
        return EmptyBody()
    }

    private func delete(_ path: String) async throws {
        let request = try makeRequest(path, method: "DELETE")
        let (data, response) = try await session.data(for: request)
        try checkResponse(response, data: data)
    }

    private func makeRequest(_ path: String, method: String) throws -> URLRequest {
        guard let base = baseURL else { throw APIError.notConfigured }
        guard let url = URL(string: path, relativeTo: base) else { throw APIError.invalidResponse }
        var request = URLRequest(url: url)
        request.httpMethod = method
        if let token {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return request
    }

    private func checkResponse(_ response: URLResponse, data: Data) throws {
        guard let http = response as? HTTPURLResponse else {
            throw APIError.invalidResponse
        }
        switch http.statusCode {
        case 200...299: return
        case 401: throw APIError.unauthorized
        case 404: throw APIError.notFound
        default:
            let body = String(data: data, encoding: .utf8) ?? ""
            throw APIError.serverError(http.statusCode, body)
        }
    }
}

enum APIError: Error, LocalizedError, Equatable {
    case notConfigured
    case invalidResponse
    case unauthorized
    case notFound
    case serverError(Int, String)

    var errorDescription: String? {
        switch self {
        case .notConfigured: "API not configured. Set server URL in Settings."
        case .invalidResponse: "Invalid server response."
        case .unauthorized: "Authentication failed. Please log in again."
        case .notFound: "Resource not found."
        case .serverError(let code, let body): "Server error \(code): \(body)"
        }
    }
}
