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
            try await pullDeltas()
        } catch {
            lastSyncError = error.localizedDescription
            print("[SyncEngine] Sync failed: \(error)")
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

    private func pullDeltas() async throws {
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
