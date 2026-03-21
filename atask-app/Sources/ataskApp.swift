import SwiftUI

@main
struct ataskApp: App {
    @State private var store = TaskStore(db: try! LocalDatabase())

    var body: some Scene {
        WindowGroup {
            ContentView(store: store)
                .frame(minWidth: 640, minHeight: 480)
        }
        .windowStyle(.titleBar)
        .defaultSize(width: 1080, height: 720)
    }
}
