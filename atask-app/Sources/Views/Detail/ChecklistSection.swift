import SwiftUI

/// Checklist in detail panel.
/// MEASUREMENTS.md: item gap 8px, padding 3px 0, font 12px, checkbox 16×16 radius-xs
struct ChecklistSection: View {
    @Bindable var store: TaskStore
    let taskId: String
    @State private var newItemTitle = ""
    @State private var items: [ChecklistItemModel] = []

    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.sp1) {
            Text("CHECKLIST")
                .font(.groupLabel)
                .foregroundStyle(Theme.inkTertiary)
                .textCase(.uppercase)
                .tracking(0.5)

            ForEach(items) { item in
                HStack(spacing: Spacing.sp2) {
                    // Square checkbox: 16×16, radius-xs (4px)
                    Button {
                        store.toggleChecklistItem(item.id)
                        reload()
                    } label: {
                        RoundedRectangle(cornerRadius: Radius.xs)
                            .strokeBorder(
                                item.isCompleted ? Theme.accent : Theme.inkQuaternary,
                                lineWidth: 1.5
                            )
                            .background(
                                RoundedRectangle(cornerRadius: Radius.xs)
                                    .fill(item.isCompleted ? Theme.accent : .clear)
                            )
                            .frame(width: Spacing.checklistCheckSize, height: Spacing.checklistCheckSize)
                            .overlay {
                                if item.isCompleted {
                                    Image(systemName: "checkmark")
                                        .font(.system(size: 8, weight: .bold))
                                        .foregroundStyle(.white)
                                }
                            }
                    }
                    .buttonStyle(.plain)

                    Text(item.title)
                        .font(.metadataRegular) // 12px
                        .foregroundStyle(item.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                        .strikethrough(item.isCompleted, color: Theme.inkQuaternary)
                }
                .padding(.vertical, 3)
            }

            // Add item input
            HStack(spacing: Spacing.sp2) {
                RoundedRectangle(cornerRadius: Radius.xs)
                    .strokeBorder(style: StrokeStyle(lineWidth: 1, dash: [2, 2]))
                    .foregroundStyle(Theme.inkQuaternary)
                    .frame(width: Spacing.checklistCheckSize, height: Spacing.checklistCheckSize)

                TextField("Add item...", text: $newItemTitle)
                    .font(.metadataRegular)
                    .textFieldStyle(.plain)
                    .onSubmit {
                        if !newItemTitle.isEmpty {
                            store.addChecklistItem(title: newItemTitle, taskId: taskId)
                            newItemTitle = ""
                            reload()
                        }
                    }
            }
            .padding(.vertical, 3)
        }
        .padding(.bottom, Spacing.sp4)
        .onAppear { reload() }
        .onChange(of: taskId) { _, _ in reload() }
    }

    private func reload() {
        items = store.checklistFor(taskId)
    }
}
