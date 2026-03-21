import SwiftUI

/// Things-style inline task editor — expands when a task is clicked.
/// Shows: checkbox + editable title, notes area, schedule badge, action buttons.
struct TaskInlineEditor: View {
    @Bindable var store: TaskStore
    let taskId: String

    @State private var titleDraft = ""
    @State private var notesDraft = ""
    @State private var initialized = false
    @FocusState private var titleFocused: Bool

    var body: some View {
        if let task = store.tasks.first(where: { $0.id == taskId }) {
            VStack(alignment: .leading, spacing: Spacing.sp2) {
                // ── Row 1: Checkbox + Title ──
                HStack(spacing: Spacing.sp3) {
                    // Checkbox
                    Button {
                        store.completeTask(taskId)
                    } label: {
                        Circle()
                            .strokeBorder(
                                task.isCompleted ? Theme.accent :
                                store.activeView == .today ? Theme.todayStar :
                                Theme.inkQuaternary,
                                lineWidth: Size.checkboxBorder
                            )
                            .background(Circle().fill(task.isCompleted ? Theme.accent : .clear))
                            .frame(width: Size.checkboxSize, height: Size.checkboxSize)
                            .overlay {
                                if task.isCompleted {
                                    Image(systemName: "checkmark")
                                        .font(.system(size: 10, weight: .bold))
                                        .foregroundStyle(.white)
                                }
                            }
                    }
                    .buttonStyle(.plain)

                    // Editable title
                    TextField("Task title", text: $titleDraft)
                        .textFieldStyle(.plain)
                        .font(.system(size: FontSize.base))
                        .foregroundStyle(task.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                        .focused($titleFocused)
                        .onSubmit {
                            saveTitleIfChanged(task)
                        }
                }

                // ── Row 2: Notes ──
                TextField("Notes", text: $notesDraft, axis: .vertical)
                    .textFieldStyle(.plain)
                    .font(.system(size: FontSize.sm))
                    .foregroundStyle(Theme.inkSecondary)
                    .lineLimit(1...5)
                    .padding(.leading, Size.checkboxSize + Spacing.sp3) // align with title

                // ── Row 3: Schedule badge + action buttons ──
                HStack {
                    // Schedule badge
                    if task.schedule == 1 {
                        HStack(spacing: 4) {
                            Image(systemName: "star.fill")
                                .font(.system(size: 10))
                                .foregroundStyle(Theme.todayStar)
                            Text("Today")
                                .font(.system(size: FontSize.xs, weight: .semibold))
                                .foregroundStyle(Theme.todayStar)
                            Button {
                                store.setSchedule(taskId, 0) // back to inbox
                            } label: {
                                Image(systemName: "xmark")
                                    .font(.system(size: 8, weight: .bold))
                                    .foregroundStyle(Theme.inkTertiary)
                            }
                            .buttonStyle(.plain)
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 3)
                        .background(Theme.todayBg)
                        .clipShape(Capsule())
                    }

                    Spacer()

                    // Action buttons (right side)
                    HStack(spacing: Spacing.sp2) {
                        // Tags
                        Button { /* TODO: tag picker */ } label: {
                            Image(systemName: "tag")
                                .font(.system(size: FontSize.sm))
                                .foregroundStyle(Theme.inkTertiary)
                        }
                        .buttonStyle(.plain)
                        .help("Tags")

                        // Checklist
                        Button { /* TODO: toggle checklist */ } label: {
                            Image(systemName: "checklist")
                                .font(.system(size: FontSize.sm))
                                .foregroundStyle(Theme.inkTertiary)
                        }
                        .buttonStyle(.plain)
                        .help("Checklist")

                        // When / Deadline
                        Button { /* TODO: when picker */ } label: {
                            Image(systemName: "calendar")
                                .font(.system(size: FontSize.sm))
                                .foregroundStyle(Theme.inkTertiary)
                        }
                        .buttonStyle(.plain)
                        .help("When")
                    }
                }
                .padding(.leading, Size.checkboxSize + Spacing.sp3)

                // ── Row 4: Project selector ──
                if let pid = task.projectId,
                   let project = store.projects.first(where: { $0.id == pid }) {
                    HStack(spacing: 4) {
                        Circle()
                            .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                            .frame(width: Size.metaDot, height: Size.metaDot)
                        Text(project.title)
                            .font(.system(size: FontSize.xs))
                            .foregroundStyle(Theme.inkTertiary)
                        Image(systemName: "chevron.right")
                            .font(.system(size: 8))
                            .foregroundStyle(Theme.inkQuaternary)
                    }
                    .padding(.leading, Size.checkboxSize + Spacing.sp3)
                }
            }
            .padding(Spacing.sp4)
            .background(
                RoundedRectangle(cornerRadius: 10)
                    .fill(Theme.canvasElevated)
                    .shadow(color: .black.opacity(0.06), radius: 8, y: 2)
                    .shadow(color: .black.opacity(0.03), radius: 2, y: 1)
            )
            .padding(.horizontal, Spacing.sp3)
            .padding(.vertical, Spacing.sp1)
            .onAppear {
                if !initialized {
                    titleDraft = task.title
                    notesDraft = task.notes
                    initialized = true
                    // Focus title for new tasks (empty title)
                    if task.title.isEmpty {
                        titleFocused = true
                    }
                }
            }
            .onChange(of: titleDraft) { _, newValue in
                // Auto-save title on change (debounced by SwiftUI)
                store.updateTitle(taskId, newValue)
            }
            .onChange(of: notesDraft) { _, newValue in
                store.updateNotes(taskId, newValue)
            }
        }
    }

    private func saveTitleIfChanged(_ task: TaskModel) {
        if titleDraft != task.title {
            store.updateTitle(taskId, titleDraft)
        }
    }
}
