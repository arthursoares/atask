import SwiftUI

/// Detail panel — 340px right side.
/// Measurements from MEASUREMENTS.md:
/// - Width: 340px, bg: canvasElevated, border-left: 1px border
/// - Header: padding 20px 20px 12px, border-bottom separator
/// - Title: 17px (text-lg) bold
/// - Meta row: gap 8px, margin-top 8px
/// - Body: padding 16px 20px, overflow-y auto
/// - Field label: 11px bold uppercase, letter-spacing 0.5px, ink-tertiary, margin-bottom 4px
/// - Field value: 12px, ink-secondary
/// - Field margin-bottom: 16px
struct TaskDetailView: View {
    @Bindable var store: TaskStore

    @State private var titleDraft = ""
    @State private var notesDraft = ""
    @State private var initialized = false

    var body: some View {
        if let taskId = store.selectedTaskId,
           let task = store.tasks.first(where: { $0.id == taskId }) {
            VStack(spacing: 0) {
                // ── Header: title + meta ──
                VStack(alignment: .leading, spacing: 0) {
                    // Title
                    TextField("Task title", text: $titleDraft)
                        .font(.sectionHeader) // 17px bold
                        .textFieldStyle(.plain)
                        .foregroundStyle(Theme.inkPrimary)
                        .onSubmit {
                            store.updateTitle(taskId, titleDraft)
                        }

                    // Meta row: tags
                    HStack(spacing: Spacing.sp2) {
                        if task.schedule == 1 {
                            tagPill("★ Today", bg: Theme.todayBg, fg: Theme.todayStar)
                        }
                        if let project = store.projectFor(task) {
                            tagPill(project.title, bg: Theme.canvasSunken, fg: Theme.inkSecondary)
                        }
                    }
                    .padding(.top, Spacing.sp2)
                }
                .padding(.horizontal, Spacing.sp5)
                .padding(.top, Spacing.sp5)
                .padding(.bottom, Spacing.sp3)

                Rectangle().fill(Theme.separator).frame(height: 1)

                // ── Body: scrollable fields ──
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        // PROJECT
                        fieldRow("PROJECT") {
                            Text(store.projectFor(task)?.title ?? "None")
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkSecondary)
                        }

                        // SCHEDULE
                        fieldRow("SCHEDULE") {
                            Text(scheduleName(task.schedule))
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkSecondary)
                        }

                        // START DATE
                        fieldRow("START DATE") {
                            if let d = task.startDate {
                                Text(DateFormatting.formatRelative(d))
                                    .font(.metadataRegular)
                                    .foregroundStyle(Theme.inkSecondary)
                            } else {
                                Text("None")
                                    .font(.metadataRegular)
                                    .foregroundStyle(Theme.inkTertiary)
                            }
                        }

                        // DEADLINE
                        fieldRow("DEADLINE") {
                            if let d = task.deadline {
                                let (label, variant) = DateFormatting.formatDeadline(d)
                                Text(label)
                                    .font(.metadataRegular)
                                    .foregroundStyle(
                                        variant == .overdue ? Theme.deadlineRed :
                                        variant == .today ? Theme.todayStar :
                                        Theme.inkSecondary
                                    )
                            } else {
                                Text("None")
                                    .font(.metadataRegular)
                                    .foregroundStyle(Theme.inkTertiary)
                            }
                        }

                        // TAGS
                        fieldRow("TAGS") {
                            Text("None")
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkTertiary)
                            // TODO: show tag pills + add button
                        }

                        // NOTES
                        VStack(alignment: .leading, spacing: Spacing.sp1) {
                            Text("NOTES")
                                .font(.groupLabel)
                                .foregroundStyle(Theme.inkTertiary)
                                .textCase(.uppercase)
                                .tracking(0.5)

                            TextField("Add notes...", text: $notesDraft, axis: .vertical)
                                .font(.detailBody) // 15px
                                .foregroundStyle(Theme.inkPrimary)
                                .textFieldStyle(.plain)
                                .lineLimit(3...20)
                        }
                        .padding(.bottom, Spacing.sp4)

                        // CHECKLIST
                        VStack(alignment: .leading, spacing: Spacing.sp1) {
                            Text("CHECKLIST")
                                .font(.groupLabel)
                                .foregroundStyle(Theme.inkTertiary)
                                .textCase(.uppercase)
                                .tracking(0.5)

                            // TODO: checklist items from store
                            Text("No checklist items")
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkTertiary)
                        }
                        .padding(.bottom, Spacing.sp4)

                        // ACTIVITY
                        VStack(alignment: .leading, spacing: Spacing.sp1) {
                            Text("ACTIVITY")
                                .font(.groupLabel)
                                .foregroundStyle(Theme.inkTertiary)
                                .textCase(.uppercase)
                                .tracking(0.5)

                            // TODO: activity entries
                            Text("No activity yet")
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkTertiary)
                        }
                    }
                    .padding(.horizontal, Spacing.sp5)
                    .padding(.top, Spacing.sp4)
                }
            }
            .frame(width: Spacing.detailWidth)
            .background(Theme.canvasElevated)
            .onAppear { initDrafts(task) }
            .onChange(of: store.selectedTaskId) { _, _ in
                if let newTask = store.tasks.first(where: { $0.id == store.selectedTaskId }) {
                    initDrafts(newTask)
                }
            }
            .onChange(of: titleDraft) { _, new in store.updateTitle(taskId, new) }
            .onChange(of: notesDraft) { _, new in store.updateNotes(taskId, new) }
        }
    }

    // ── Helpers ──

    private func initDrafts(_ task: TaskModel) {
        titleDraft = task.title
        notesDraft = task.notes
        initialized = true
    }

    private func fieldRow<Content: View>(_ label: String, @ViewBuilder content: () -> Content) -> some View {
        VStack(alignment: .leading, spacing: Spacing.sp1) {
            Text(label)
                .font(.groupLabel)
                .foregroundStyle(Theme.inkTertiary)
                .textCase(.uppercase)
                .tracking(0.5)
            content()
        }
        .padding(.bottom, Spacing.sp4)
    }

    private func tagPill(_ label: String, bg: Color, fg: Color) -> some View {
        Text(label)
            .font(.tagPill)
            .foregroundStyle(fg)
            .padding(.horizontal, 8)
            .padding(.vertical, 2)
            .background(bg)
            .clipShape(Capsule())
    }

    private func scheduleName(_ schedule: Int) -> String {
        switch schedule {
        case 0: "Inbox"
        case 1: "Today (Anytime)"
        case 2: "Someday"
        default: "Unknown"
        }
    }
}
