import SwiftUI

struct NewTaskRow: View {
    let onCreate: (String) -> Void

    @State private var editing = false
    @State private var title = ""
    @FocusState private var focused: Bool

    var body: some View {
        if editing {
            HStack(spacing: 12) {
                // Dashed circle placeholder
                Circle()
                    .strokeBorder(style: StrokeStyle(lineWidth: 1.5, dash: [3, 3]))
                    .foregroundStyle(Theme.inkQuaternary)
                    .frame(width: 20, height: 20)

                TextField("New Task", text: $title)
                    .textFieldStyle(.plain)
                    .font(.system(size: 14))
                    .focused($focused)
                    .onSubmit {
                        if !title.isEmpty {
                            onCreate(title)
                        }
                        title = ""
                        // Stay in editing mode for rapid entry
                    }
                    .onExitCommand {
                        title = ""
                        editing = false
                    }
            }
            .padding(.vertical, 6)
            .padding(.horizontal, 16)
            .frame(height: 32)
            .onAppear { focused = true }
        } else {
            Button {
                editing = true
            } label: {
                HStack(spacing: 12) {
                    Circle()
                        .strokeBorder(style: StrokeStyle(lineWidth: 1.5, dash: [3, 3]))
                        .foregroundStyle(Theme.inkTertiary)
                        .frame(width: 20, height: 20)

                    Text("New Task")
                        .foregroundStyle(Theme.inkTertiary)
                        .font(.system(size: 14))

                    Spacer()
                }
                .padding(.vertical, 6)
                .padding(.horizontal, 16)
                .frame(height: 32)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        }
    }
}
