import SwiftUI

/// Things-style inline editor — expanded card.
/// Measurements from MEASUREMENTS.md:
/// - Background: sidebarSelected (accent 10%)
/// - Border: 1.5px accent, radius-md (8px)
/// - Padding: 6px 16px 8px
/// - Title: 14px (text-base) — matches task row
/// - Notes: 13px inline notes
/// - Attr bar: left padding 27px (checkbox 20 + gap 7)
struct TaskInlineEditor: View {
    @Bindable var store: TaskStore
    let taskId: String

    @State private var titleDraft = ""
    @State private var notesDraft = ""
    @State private var initialized = false
    @FocusState private var titleFocused: Bool

    var body: some View {
        if let task = store.tasks.first(where: { $0.id == taskId }) {
            VStack(alignment: .leading, spacing: 0) {
                // ── Top row: checkbox + title (same height as task row: 32px) ──
                HStack(spacing: Spacing.sp3) {
                    CheckboxView(
                        isChecked: task.isCompleted,
                        isToday: store.activeView == .today,
                        onToggle: { store.completeTask(taskId) }
                    )

                    TextField("What needs to happen?", text: $titleDraft)
                        .font(.taskTitle)
                        .textFieldStyle(.plain)
                        .focused($titleFocused)
                        .onSubmit {
                            if titleDraft != task.title { store.updateTitle(taskId, titleDraft) }
                        }
                }
                .frame(height: Spacing.taskRowHeight)

                // ── Attribute bar ──
                HStack(spacing: 6) {
                    // Schedule badge
                    if task.schedule == 1 {
                        attrPill("★ Today", variant: .today, removable: true) {
                            store.setSchedule(taskId, 0)
                        }
                    }

                    // Project
                    if let project = store.projectFor(task) {
                        attrPill("● \(project.title)", variant: .project)
                    }

                    // Action buttons
                    attrPill("📅 When", variant: .add)
                    attrPill("🏷 +Tag", variant: .add)
                }
                .padding(.top, Spacing.sp1)
                .padding(.leading, Spacing.attrBarLeftPad)
            }
            .padding(.horizontal, Spacing.sp4)
            .padding(.vertical, 6)
            .padding(.bottom, 2)
            .background(
                RoundedRectangle(cornerRadius: Radius.md)
                    .fill(Theme.sidebarSelected)
                    .overlay(
                        RoundedRectangle(cornerRadius: Radius.md)
                            .strokeBorder(Theme.accent, lineWidth: 1.5)
                    )
            )
            .onAppear {
                if !initialized {
                    titleDraft = task.title
                    notesDraft = task.notes
                    initialized = true
                    if task.title.isEmpty { titleFocused = true }
                }
            }
            .onChange(of: titleDraft) { _, new in store.updateTitle(taskId, new) }
            .onChange(of: notesDraft) { _, new in store.updateNotes(taskId, new) }
            .onExitCommand { store.expandedTaskId = nil }
        }
    }

    // ── Attribute pill ──
    enum AttrVariant { case today, project, tag, add }

    private func attrPill(_ label: String, variant: AttrVariant, removable: Bool = false, onRemove: (() -> Void)? = nil) -> some View {
        HStack(spacing: 4) {
            Text(label)
                .font(.tagPill)
            if removable {
                Button { onRemove?() } label: {
                    Text("×")
                        .font(.system(size: 10, weight: .bold))
                        .foregroundStyle(Theme.inkTertiary)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 2)
        .background(pillBackground(variant))
        .foregroundStyle(pillForeground(variant))
        .clipShape(Capsule())
        .overlay(
            variant == .add
                ? Capsule().strokeBorder(Theme.border, style: StrokeStyle(lineWidth: 1, dash: [3, 3]))
                : nil
        )
    }

    private func pillBackground(_ v: AttrVariant) -> Color {
        switch v {
        case .today: Theme.todayBg
        case .project: Theme.canvasSunken
        case .tag: Theme.accentSubtle
        case .add: .clear
        }
    }

    private func pillForeground(_ v: AttrVariant) -> Color {
        switch v {
        case .today: Theme.todayStar
        case .project: Theme.inkSecondary
        case .tag: Theme.accent
        case .add: Theme.inkTertiary
        }
    }
}

// ── Checkbox (reusable) ──
struct CheckboxView: View {
    let isChecked: Bool
    let isToday: Bool
    let onToggle: () -> Void

    var body: some View {
        Button(action: onToggle) {
            Circle()
                .strokeBorder(borderColor, lineWidth: Spacing.checkboxBorder)
                .background(Circle().fill(isChecked ? Theme.accent : .clear))
                .frame(width: Spacing.checkboxSize, height: Spacing.checkboxSize)
                .overlay {
                    if isChecked {
                        Image(systemName: "checkmark")
                            .font(.system(size: 9, weight: .bold))
                            .foregroundStyle(.white)
                    }
                }
                .contentShape(Circle().inset(by: -6))
        }
        .buttonStyle(.plain)
    }

    private var borderColor: Color {
        if isChecked { return Theme.accent }
        if isToday { return Theme.todayStar }
        return Theme.inkQuaternary
    }
}
