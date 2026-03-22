import SwiftUI

/// Project picker popover — list of projects with colored dots.
struct ProjectPicker: View {
    @Bindable var store: TaskStore
    let taskId: String
    @Binding var isPresented: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // No project option
            Button {
                store.moveToProject(taskId, nil)
                isPresented = false
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "minus.circle")
                        .font(.system(size: 12))
                        .foregroundStyle(Theme.inkTertiary)
                    Text("No Project")
                        .font(.metadataRegular)
                        .foregroundStyle(Theme.inkSecondary)
                    Spacer()
                    if store.tasks.first(where: { $0.id == taskId })?.projectId == nil {
                        Image(systemName: "checkmark")
                            .font(.system(size: 10, weight: .bold))
                            .foregroundStyle(Theme.accent)
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            Divider().padding(.vertical, 4)

            // Project list
            ForEach(store.projects.filter { !$0.isCompleted }) { project in
                Button {
                    store.moveToProject(taskId, project.id)
                    isPresented = false
                } label: {
                    HStack(spacing: 8) {
                        Circle()
                            .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                            .frame(width: 8, height: 8)
                        Text(project.title)
                            .font(.metadataRegular)
                            .foregroundStyle(Theme.inkPrimary)
                        Spacer()
                        if store.tasks.first(where: { $0.id == taskId })?.projectId == project.id {
                            Image(systemName: "checkmark")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(Theme.accent)
                        }
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
            }
        }
        .padding(8)
        .frame(width: 200)
    }
}
