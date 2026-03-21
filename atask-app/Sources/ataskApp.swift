import SwiftUI

@main
struct ataskApp: App {
    @State private var store = TaskStore(db: try! LocalDatabase())

    var body: some Scene {
        WindowGroup {
            ContentView(store: store)
                .frame(minWidth: 640, minHeight: 480)
        }
        .windowStyle(.hiddenTitleBar)
        .defaultSize(width: 1080, height: 720)
        .commands {
            // ⌘N — New Task (replaces default New Window)
            CommandGroup(replacing: .newItem) {
                Button("New Task") {
                    let task = store.createTask(title: "")
                    store.expandedTaskId = task.id
                }
                .keyboardShortcut("n", modifiers: .command)
            }

            // ⌘K — Complete Task (Things-compatible)
            CommandMenu("Edit Tasks") {
                Button("Complete Task") {
                    if let id = store.selectedTaskId { store.completeTask(id) }
                }
                .keyboardShortcut("k", modifiers: .command)

                Button("Cancel Task") {
                    if let id = store.selectedTaskId { store.cancelTask(id) }
                }
                .keyboardShortcut("k", modifiers: [.command, .option])

                Divider()

                Button("Schedule for Today") {
                    if let id = store.selectedTaskId { store.setSchedule(id, 1) }
                }
                .keyboardShortcut("t", modifiers: .command)

                Button("Start This Evening") {
                    if let id = store.selectedTaskId { store.setTimeSlot(id, "evening") }
                }
                .keyboardShortcut("e", modifiers: .command)

                Button("Start Someday") {
                    if let id = store.selectedTaskId { store.setSchedule(id, 2) }
                }
                .keyboardShortcut("o", modifiers: .command)

                Divider()

                Button("Delete Task") {
                    if let id = store.selectedTaskId { store.deleteTask(id) }
                }
                .keyboardShortcut(.delete)
            }

            // ⌘1-6 — View Navigation
            CommandMenu("Navigate") {
                Button("Inbox") { navigate(.inbox) }
                    .keyboardShortcut("1", modifiers: .command)
                Button("Today") { navigate(.today) }
                    .keyboardShortcut("2", modifiers: .command)
                Button("Upcoming") { navigate(.upcoming) }
                    .keyboardShortcut("3", modifiers: .command)
                Button("Someday") { navigate(.someday) }
                    .keyboardShortcut("5", modifiers: .command)
                Button("Logbook") { navigate(.logbook) }
                    .keyboardShortcut("6", modifiers: .command)
            }
        }
    }

    private func navigate(_ view: ActiveView) {
        store.activeView = view
        store.expandedTaskId = nil
        store.selectedTaskId = nil
        switch view {
        case .inbox: store.sidebarSelection = .inbox
        case .today: store.sidebarSelection = .today
        case .upcoming: store.sidebarSelection = .upcoming
        case .someday: store.sidebarSelection = .someday
        case .logbook: store.sidebarSelection = .logbook
        default: break
        }
    }
}
