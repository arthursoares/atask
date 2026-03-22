import Foundation
import GRDB

/// Local-first sync engine.
///
/// Outbound: mutations are queued as pendingOps, flushed to server periodically.
/// Inbound: polls /sync/deltas to apply remote changes to local DB.
///
/// The local SQLite DB is always the source of truth. Sync is optional —
/// the app works fully offline.
@Observable
@MainActor
class SyncEngine {
    let api: APIClient
    let db: LocalDatabase

    private(set) var isSyncing = false
    private(set) var lastSyncError: String?
    private(set) var lastDeltaId: Int = 0

    private var syncTask: Task<Void, Never>?
    private var sseClient: SSEClient?

    /// Called when SSE receives a remote change — store should reload.
    var onRemoteChange: (@Sendable @MainActor () -> Void)?

    init(api: APIClient, db: LocalDatabase) {
        self.api = api
        self.db = db
        loadLastDeltaId()
    }

    // MARK: - Start/Stop

    func startPeriodicSync(interval: TimeInterval = 30) {
        guard syncTask == nil else { return }
        syncTask = Task { [weak self] in
            while !Task.isCancelled {
                await self?.sync()
                try? await Task.sleep(for: .seconds(interval))
            }
        }
    }

    func stopSync() {
        syncTask?.cancel()
        syncTask = nil
        Task { await sseClient?.disconnect() }
    }

    /// Start SSE for real-time server push.
    func startSSE(baseURL: String, token: String) {
        sseClient = SSEClient { [weak self] eventType, _ in
            print("[SSE] Event: \(eventType)")
            Task { @MainActor [weak self] in
                guard let self else { return }
                await self.pullDeltas()
                self.onRemoteChange?()
            }
        }
        Task { await sseClient?.connect(baseURL: baseURL, token: token) }
    }

    // MARK: - Full Sync

    func sync() async {
        guard await api.isConfigured else { return }
        guard !isSyncing else { return }

        isSyncing = true
        lastSyncError = nil

        do {
            // 1. Flush pending outbound ops
            try await flushPendingOps()

            // 2. Pull inbound deltas
            try await pullDeltasInternal()
        } catch {
            lastSyncError = error.localizedDescription
            print("[SyncEngine] Sync failed: \(error)")
        }

        isSyncing = false
    }

    // MARK: - Initial Full Sync

    /// Pull all entities from server on first connect.
    /// Merges by ID — server data overwrites local for matching IDs,
    /// local-only records are preserved.
    func initialSync(reloadStore: @escaping () -> Void) async {
        guard await api.isConfigured else { return }
        isSyncing = true
        lastSyncError = nil

        do {
            // Pull tasks
            let remoteTasks = try await api.listTasks()
            try await db.dbQueue.write { db in
                for rt in remoteTasks {
                    let status: Int = rt.status == "completed" ? 1 : rt.status == "cancelled" ? 2 : 0
                    let schedule: Int = rt.schedule == "anytime" ? 1 : rt.schedule == "someday" ? 2 : 0
                    try db.execute(sql: """
                        INSERT INTO tasks (id, title, notes, status, schedule, startDate, deadline, completedAt,
                            \("index"), todayIndex, timeSlot, projectId, sectionId, areaId, createdAt, updatedAt, syncStatus)
                        VALUES (?, ?, '', ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, ?, 1)
                        ON CONFLICT(id) DO UPDATE SET
                            title=excluded.title, status=excluded.status, schedule=excluded.schedule,
                            startDate=excluded.startDate, deadline=excluded.deadline, completedAt=excluded.completedAt,
                            todayIndex=excluded.todayIndex, timeSlot=excluded.timeSlot,
                            projectId=excluded.projectId, sectionId=excluded.sectionId, areaId=excluded.areaId,
                            updatedAt=excluded.updatedAt, syncStatus=1
                    """, arguments: [
                        rt.id, rt.title, status, schedule,
                        rt.startDate, rt.deadline, rt.completedAt,
                        rt.todayIndex, rt.timeSlot,
                        rt.projectId, rt.sectionId, rt.areaId,
                        rt.createdAt, rt.updatedAt
                    ])
                }
            }

            // Pull projects
            let remoteProjects = try await api.listProjects()
            try await db.dbQueue.write { db in
                for rp in remoteProjects {
                    try db.execute(sql: """
                        INSERT INTO projects (id, title, notes, status, color, areaId, \("index"), completedAt, createdAt, updatedAt)
                        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                        ON CONFLICT(id) DO UPDATE SET
                            title=excluded.title, notes=excluded.notes, status=excluded.status,
                            color=excluded.color, areaId=excluded.areaId, completedAt=excluded.completedAt,
                            updatedAt=excluded.updatedAt
                    """, arguments: [
                        rp.id, rp.title, rp.notes, rp.status, rp.color,
                        rp.areaId, rp.index, rp.completedAt, rp.createdAt, rp.updatedAt
                    ])
                }
            }

            // Pull areas
            let remoteAreas = try await api.listAreas()
            try await db.dbQueue.write { db in
                for ra in remoteAreas {
                    try db.execute(sql: """
                        INSERT INTO areas (id, title, \("index"), archived, createdAt, updatedAt)
                        VALUES (?, ?, ?, 0, ?, ?)
                        ON CONFLICT(id) DO UPDATE SET title=excluded.title, updatedAt=excluded.updatedAt
                    """, arguments: [ra.id, ra.title, ra.index, ra.createdAt, ra.updatedAt])
                }
            }

            // Pull tags
            let remoteTags = try await api.listTags()
            try await db.dbQueue.write { db in
                for rt in remoteTags {
                    try db.execute(sql: """
                        INSERT INTO tags (id, title, \("index"), createdAt, updatedAt)
                        VALUES (?, ?, ?, ?, ?)
                        ON CONFLICT(id) DO UPDATE SET title=excluded.title, updatedAt=excluded.updatedAt
                    """, arguments: [rt.id, rt.title, rt.index, rt.createdAt, rt.updatedAt])
                }
            }

            reloadStore()
            UserDefaults.standard.set(true, forKey: "hasCompletedInitialSync")
        } catch {
            lastSyncError = error.localizedDescription
            print("[SyncEngine] Initial sync failed: \(error)")
        }

        isSyncing = false
    }

    // MARK: - Outbound: Pending Ops

    /// Queue a mutation for later sync to server.
    func enqueue(method: String, path: String, body: String? = nil) {
        do {
            let now = ISO8601DateFormatter().string(from: Date())
            try db.dbQueue.write { db in
                try db.execute(
                    sql: "INSERT INTO pendingOps (method, path, body, createdAt, synced) VALUES (?, ?, ?, ?, 0)",
                    arguments: [method, path, body, now]
                )
            }
        } catch {
            print("[SyncEngine] Enqueue failed: \(error)")
        }
    }

    private struct PendingOp: FetchableRecord, Codable {
        let id: Int
        let method: String
        let path: String
        let body: String?
    }

    private func flushPendingOps() async throws {
        let ops: [PendingOp] = try await db.dbQueue.read { db in
            try PendingOp.fetchAll(db, sql: "SELECT id, method, path, body FROM pendingOps WHERE synced = 0 ORDER BY id")
        }

        for op in ops {
            do {
                try await executeOp(op)
                try await db.dbQueue.write { db in
                    try db.execute(sql: "UPDATE pendingOps SET synced = 1 WHERE id = ?", arguments: [op.id])
                }
            } catch let error as APIError where error == .notFound {
                // Entity deleted on server — mark synced, skip
                try await db.dbQueue.write { db in
                    try db.execute(sql: "UPDATE pendingOps SET synced = 1 WHERE id = ?", arguments: [op.id])
                }
            } catch {
                // Stop flushing on network/auth errors — retry next cycle
                print("[SyncEngine] Flush op \(op.id) failed: \(error)")
                break
            }
        }

        try await db.dbQueue.write { db in
            try db.execute(sql: "DELETE FROM pendingOps WHERE synced = 1")
        }
    }

    private func executeOp(_ op: PendingOp) async throws {
        try await api.execute(method: op.method, path: op.path, body: op.body)
    }

    // MARK: - Inbound: Delta Events

    func pullDeltas() async {
        do {
            try await pullDeltasInternal()
        } catch {
            print("[SyncEngine] Pull deltas failed: \(error)")
        }
    }

    private func pullDeltasInternal() async throws {
        let deltas = try await api.fetchDeltas(since: lastDeltaId)
        guard !deltas.isEmpty else { return }

        for delta in deltas {
            await applyDelta(delta)
            if delta.id > lastDeltaId {
                lastDeltaId = delta.id
            }
        }

        saveLastDeltaId()
    }

    private func applyDelta(_ delta: APIClient.DeltaEvent) async {
        do {
            try await db.dbQueue.write { db in
                // Allowlisted tables to prevent SQL injection via entityType
                let allowedTables = ["task", "project", "area", "section", "tag"]
                guard allowedTables.contains(delta.entityType) else { return }
                let table = delta.entityType + "s"
                let sql = "UPDATE \(table) SET \"\(delta.field)\" = ? WHERE id = ?"
                try db.execute(sql: sql, arguments: [delta.newValue, delta.entityId])
            }
        } catch {
            print("[SyncEngine] Apply delta failed: \(error)")
        }
    }

    // MARK: - Persistence

    private func loadLastDeltaId() {
        lastDeltaId = UserDefaults.standard.integer(forKey: "lastDeltaId")
    }

    private func saveLastDeltaId() {
        UserDefaults.standard.set(lastDeltaId, forKey: "lastDeltaId")
    }
}
