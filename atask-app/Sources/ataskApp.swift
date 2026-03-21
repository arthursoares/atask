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
            // Replace the default ⌘N (New Window) with New Task
            CommandGroup(replacing: .newItem) {
                Button("New Task") {
                    let task = store.createTaskInView(title: "")
                    store.expandedTaskId = task.id
                }
                .keyboardShortcut("n", modifiers: .command)
            }

            // ⌘1-6 — View Navigation
            CommandMenu("Navigate") {
                Button("Inbox") { store.activeView = .inbox; store.sidebarSelection = .inbox }
                    .keyboardShortcut("1", modifiers: .command)
                Button("Today") { store.activeView = .today; store.sidebarSelection = .today }
                    .keyboardShortcut("2", modifiers: .command)
                Button("Upcoming") { store.activeView = .upcoming; store.sidebarSelection = .upcoming }
                    .keyboardShortcut("3", modifiers: .command)
                Button("Someday") { store.activeView = .someday; store.sidebarSelection = .someday }
                    .keyboardShortcut("5", modifiers: .command)
                Button("Logbook") { store.activeView = .logbook; store.sidebarSelection = .logbook }
                    .keyboardShortcut("6", modifiers: .command)
            }
        }
    }
}
