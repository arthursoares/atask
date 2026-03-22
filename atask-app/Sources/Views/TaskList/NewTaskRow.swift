import SwiftUI

/// "+ New Task" row — same dimensions as task row (32px, 12px gap, 6px 16px padding)
struct NewTaskRow: View {
    let onCreate: (String) -> Void

    @State private var editing = false
    @State private var title = ""
    @FocusState private var isFocused: Bool

    var body: some View {
        if editing {
            HStack(spacing: Spacing.sp3) {
                Circle()
                    .strokeBorder(style: StrokeStyle(lineWidth: 1.5, dash: [3, 3]))
                    .foregroundStyle(Theme.inkQuaternary)
                    .frame(width: Spacing.checkboxSize, height: Spacing.checkboxSize)

                TextField("New Task", text: $title)
                    .textFieldStyle(.plain)
                    .font(.taskTitle)
                    .focused($isFocused)
                    .onSubmit {
                        if !title.isEmpty { onCreate(title) }
                        title = ""
                        isFocused = true
                    }
                    .onExitCommand {
                        title = ""
                        editing = false
                    }
            }
            .frame(height: Spacing.taskRowHeight)
            .padding(.vertical, 6)
            .padding(.horizontal, Spacing.sp4)
        } else {
            Button { editing = true } label: {
                HStack(spacing: Spacing.sp3) {
                    Circle()
                        .strokeBorder(style: StrokeStyle(lineWidth: 1.5, dash: [3, 3]))
                        .foregroundStyle(Theme.inkTertiary)
                        .frame(width: Spacing.checkboxSize, height: Spacing.checkboxSize)
                    Text("New Task")
                        .font(.taskTitle)
                        .foregroundStyle(Theme.inkTertiary)
                    Spacer()
                }
                .frame(height: Spacing.taskRowHeight)
                .padding(.vertical, 6)
                .padding(.horizontal, Spacing.sp4)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        }
    }
}
