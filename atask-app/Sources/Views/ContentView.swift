import SwiftUI

struct ContentView: View {
    @Bindable var store: TaskStore

    var body: some View {
        NavigationSplitView {
            Text("Sidebar")
                .frame(minWidth: 200)
        } content: {
            VStack {
                Text("atask v3 — SwiftUI")
                    .font(.title)
                Text("Local-first task manager")
                    .foregroundColor(.secondary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } detail: {
            Text("Select a task")
                .foregroundColor(.secondary)
        }
        .navigationSplitViewStyle(.balanced)
    }
}
