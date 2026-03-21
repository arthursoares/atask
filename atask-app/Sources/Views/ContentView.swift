import SwiftUI

struct ContentView: View {
    @Bindable var store: TaskStore

    var body: some View {
        NavigationSplitView {
            SidebarView(store: store)
                .frame(minWidth: 200)
        } content: {
            VStack(spacing: 0) {
                // ── Toolbar: 52px, padding 0 24px ──
                HStack {
                    HStack(spacing: Spacing.sp3) {
                        viewIcon
                        Text(viewTitle)
                            .font(.system(size: FontSize.xl, weight: .bold))
                            .foregroundStyle(Theme.inkPrimary)
                        if store.activeView == .today {
                            Text(todayDateString)
                                .font(.system(size: FontSize.sm))
                                .foregroundStyle(Theme.inkTertiary)
                        }
                    }

                    Spacer()

                    HStack(spacing: Spacing.sp2) {
                        toolbarButton(icon: "magnifyingglass") { /* search */ }
                        toolbarButton(icon: "plus") {
                            let task = store.createTaskInView(title: "")
                            store.expandedTaskId = task.id
                        }
                    }
                }
                .padding(.horizontal, Spacing.sp4) // 16px — same as task row padding
                .frame(height: Size.toolbarHeight)

                // ── Separator ──
                Rectangle()
                    .fill(Color.black.opacity(0.05))
                    .frame(height: 1)

                // ── Task list ──
                ScrollView {
                    viewContent
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(Theme.canvas)
        } detail: {
            if store.selectedTaskId != nil {
                Text("Detail panel")
                    .foregroundColor(Theme.inkTertiary)
                    .frame(minWidth: 300)
            }
        }
        .navigationSplitViewStyle(.balanced)
    }

    // ── Toolbar button: 30×30, radius-sm ──
    private func toolbarButton(icon: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: FontSize.base))
                .foregroundStyle(Theme.inkTertiary)
                .frame(width: Size.toolbarButton, height: Size.toolbarButton)
        }
        .buttonStyle(.plain)
    }

    // ── View icon ──
    @ViewBuilder
    private var viewIcon: some View {
        switch store.activeView {
        case .inbox:
            Image(systemName: "tray")
                .font(.system(size: FontSize.base))
                .foregroundStyle(Theme.accent)
        case .today:
            Image(systemName: "star.fill")
                .font(.system(size: FontSize.base))
                .foregroundStyle(Theme.todayStar)
        case .upcoming:
            Image(systemName: "calendar")
                .font(.system(size: FontSize.base))
                .foregroundStyle(Theme.accent)
        case .someday:
            Image(systemName: "clock")
                .font(.system(size: FontSize.base))
                .foregroundStyle(Theme.somedayTint)
        case .logbook:
            Image(systemName: "archivebox")
                .font(.system(size: FontSize.base))
                .foregroundStyle(Theme.accent)
        case .project(let id):
            let color = store.projects.first { $0.id == id }?.color ?? ""
            Circle()
                .fill(Color(hex: color.isEmpty ? "#4670a0" : color))
                .frame(width: 10, height: 10)
        }
    }

    private var viewTitle: String {
        switch store.activeView {
        case .inbox: "Inbox"
        case .today: "Today"
        case .upcoming: "Upcoming"
        case .someday: "Someday"
        case .logbook: "Logbook"
        case .project(let id):
            store.projects.first { $0.id == id }?.title ?? "Project"
        }
    }

    private var todayDateString: String {
        let fmt = DateFormatter()
        fmt.dateFormat = "EEEE, MMM d"
        return fmt.string(from: Date())
    }

    // ── View content ──
    @ViewBuilder
    private var viewContent: some View {
        switch store.activeView {
        case .inbox:
            taskList(store.inbox, empty: "Inbox Zero ✓", emptyColor: Theme.success)
        case .today:
            taskList(store.today, empty: "Your day is clear.")
        case .upcoming:
            taskList(store.upcoming, empty: "Nothing scheduled ahead.")
        case .someday:
            taskList(store.someday, empty: "No someday tasks. Everything is decided.")
        case .logbook:
            taskList(store.logbook, empty: "Nothing completed yet. Get started!")
        case .project(let id):
            taskList(store.tasksForProject(id), empty: "No tasks in this project yet.")
        }
    }

    // ── Task list with empty state + new task row ──
    @ViewBuilder
    private func taskList(_ tasks: [TaskModel], empty: String, emptyColor: Color = Theme.inkTertiary) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            if tasks.isEmpty {
                VStack {
                    Spacer(minLength: 80)
                    Text(empty)
                        .foregroundStyle(emptyColor)
                        .font(.system(size: FontSize.md))
                    Spacer(minLength: 40)
                }
                .frame(maxWidth: .infinity)
            } else {
                ForEach(tasks) { task in
                    taskRow(task)
                }
            }

            if store.activeView != .logbook && store.activeView != .upcoming {
                NewTaskRow { title in
                    store.createTaskInView(title: title)
                }
            }
        }
    }

    // ── Task row: 32px height, 6px 16px padding, 12px gap ──
    private func taskRow(_ task: TaskModel) -> some View {
        let isToday = store.activeView == .today
        let isSelected = store.selectedTaskId == task.id

        return HStack(spacing: Spacing.sp3) {
            // Checkbox: 20×20, with padding for larger hit area
            Button {
                if task.isCompleted {
                    store.reopenTask(task.id)
                } else {
                    store.completeTask(task.id)
                }
            } label: {
                Circle()
                    .strokeBorder(
                        task.isCompleted ? Theme.accent :
                        isToday ? Theme.todayStar :
                        Theme.inkQuaternary,
                        lineWidth: Size.checkboxBorder
                    )
                    .background(
                        Circle().fill(task.isCompleted ? Theme.accent : .clear)
                    )
                    .frame(width: Size.checkboxSize, height: Size.checkboxSize)
                    .overlay {
                        if task.isCompleted {
                            Image(systemName: "checkmark")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(.white)
                        }
                    }
                    .contentShape(Circle().inset(by: -6)) // expand hit area without affecting layout
            }
            .buttonStyle(.plain)

            // Title: 14px, truncates
            Text(task.title)
                .font(.system(size: FontSize.base))
                .lineLimit(1)
                .foregroundStyle(task.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                .strikethrough(task.isCompleted, color: Theme.inkQuaternary)

            Spacer(minLength: Spacing.sp2)

            // Meta: 11px, right-aligned
            HStack(spacing: Spacing.sp2) {
                if let pid = task.projectId,
                   let project = store.projects.first(where: { $0.id == pid }) {
                    HStack(spacing: 3) {
                        Circle()
                            .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                            .frame(width: Size.metaDot, height: Size.metaDot)
                        Text(project.title)
                            .font(.system(size: FontSize.xs, weight: .medium))
                            .foregroundStyle(Theme.inkTertiary)
                    }
                    .padding(.horizontal, 7)
                    .padding(.vertical, 1)
                    .background(Theme.canvasSunken)
                    .clipShape(Capsule())
                }

                if let deadline = task.deadline {
                    let (label, variant) = DateFormatting.formatDeadline(deadline)
                    if task.projectId != nil {
                        Text("·").font(.system(size: FontSize.xs)).foregroundStyle(Theme.inkQuaternary)
                    }
                    Text(label)
                        .font(.system(size: FontSize.xs, weight: .bold))
                        .foregroundStyle(
                            variant == .overdue ? Theme.deadlineRed :
                            variant == .today ? Theme.todayStar :
                            Theme.inkTertiary
                        )
                }
            }
        }
        .padding(.vertical, 2)
        .padding(.horizontal, Spacing.sp4)
        .frame(minHeight: Size.taskRowHeight)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(isSelected ? Theme.accentSubtle : Color.clear)
        )
        .contentShape(Rectangle())
        .onTapGesture {
            store.selectedTaskId = task.id
        }
        .contextMenu {
            Button { store.completeTask(task.id) } label: { Label("Complete", systemImage: "checkmark") }
            Button { store.setSchedule(task.id, 1) } label: { Label("Schedule for Today", systemImage: "star") }
            Button { store.setSchedule(task.id, 2) } label: { Label("Defer to Someday", systemImage: "clock") }
            Divider()
            Button { store.setSchedule(task.id, 0) } label: { Label("Move to Inbox", systemImage: "tray") }
            Divider()
            Button(role: .destructive) { store.deleteTask(task.id) } label: { Label("Delete", systemImage: "trash") }
        }
    }
}
