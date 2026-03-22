import SwiftUI

/// Settings — ⌘,
/// Server URL for sync, auto-archive preferences.
struct SettingsView: View {
    @AppStorage("serverURL") private var serverURL = ""
    @AppStorage("autoArchiveDays") private var autoArchiveDays = 7

    var body: some View {
        Form {
            Section("Sync") {
                TextField("Server URL", text: $serverURL, prompt: Text("https://api.atask.app"))
                    .textFieldStyle(.roundedBorder)
                Text("Leave empty for local-only mode.")
                    .font(.metadataRegular)
                    .foregroundStyle(Theme.inkTertiary)
            }

            Section("Behavior") {
                Picker("Auto-archive completed tasks after", selection: $autoArchiveDays) {
                    Text("Never").tag(0)
                    Text("1 day").tag(1)
                    Text("3 days").tag(3)
                    Text("7 days").tag(7)
                    Text("14 days").tag(14)
                    Text("30 days").tag(30)
                }
            }
        }
        .formStyle(.grouped)
        .frame(width: 400, height: 200)
    }
}
