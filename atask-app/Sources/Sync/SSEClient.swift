import Foundation

/// Server-Sent Events client for real-time updates.
/// Connects to GET /events/stream?topics=task.*,project.*,area.*
/// and notifies the sync engine when changes arrive.
actor SSEClient {
    private var task: Task<Void, Never>?
    private let onEvent: @Sendable (String, String) -> Void // (event type, data)

    init(onEvent: @escaping @Sendable (String, String) -> Void) {
        self.onEvent = onEvent
    }

    func connect(baseURL: String, token: String) {
        disconnect()

        let topics = "task.*,project.*,area.*,section.*,tag.*"
        guard let url = URL(string: "\(baseURL)/events/stream?topics=\(topics)") else { return }

        var request = URLRequest(url: url)
        request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
        request.timeoutInterval = .infinity

        task = Task {
            while !Task.isCancelled {
                do {
                    let (bytes, response) = try await URLSession.shared.bytes(for: request)
                    guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
                        print("[SSE] Bad response, retrying in 5s")
                        try await Task.sleep(for: .seconds(5))
                        continue
                    }

                    var eventType = ""
                    var data = ""

                    for try await line in bytes.lines {
                        if line.hasPrefix("event:") {
                            eventType = String(line.dropFirst(6)).trimmingCharacters(in: .whitespaces)
                        } else if line.hasPrefix("data:") {
                            data = String(line.dropFirst(5)).trimmingCharacters(in: .whitespaces)
                        } else if line.isEmpty && !eventType.isEmpty {
                            // End of event
                            onEvent(eventType, data)
                            eventType = ""
                            data = ""
                        }
                    }
                } catch {
                    if Task.isCancelled { break }
                    print("[SSE] Connection lost: \(error), reconnecting in 5s")
                    try? await Task.sleep(for: .seconds(5))
                }
            }
        }
    }

    func disconnect() {
        task?.cancel()
        task = nil
    }
}
