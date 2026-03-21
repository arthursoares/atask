import SwiftUI

struct ContentView: View {
    @Bindable var store: TaskStore

    var body: some View {
        NavigationSplitView {
            SidebarView(store: store)
                .frame(minWidth: 200)
        } content: {
            VStack(spacing: 0) {
                // Custom toolbar matching design reference
                HStack {
                    // Left: icon + title + subtitle
                    HStack(spacing: 8) {
                        viewIcon
                            .font(.system(size: 16))
                        Text(viewTitle)
                            .font(.system(size: 20, weight: .bold))
                            .foregroundStyle(Theme.inkPrimary)
                        if store.activeView == .today {
                            Text(todayDateString)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.inkTertiary)
                        }
                    }

                    Spacer()

                    // Right: action buttons
                    HStack(spacing: 4) {
                        toolbarButton(icon: "magnifyingglass", tooltip: "Search (⌘F)") {
                            // TODO: search
                        }
                        toolbarButton(icon: "plus", tooltip: "New Task (⌘N)") {
                            let task = store.createTaskInView(title: "")
                            store.expandedTaskId = task.id
                        }
                    }
                }
                .padding(.horizontal, 20)
                .frame(height: 52)

                // Divider line
                Divider()

                // Content
                ScrollView {
                    viewContent
                        .padding(.horizontal, 20)
                        .padding(.vertical, 8)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(Theme.canvas)
        } detail: {
            if store.selectedTaskId != nil {
                Text("Detail panel — coming in Task 8")
                    .foregroundColor(Theme.inkTertiary)
                    .frame(minWidth: 300)
            }
        }
        .navigationSplitViewStyle(.balanced)
    }

    // MARK: - Toolbar Button

    private func toolbarButton(icon: String, tooltip: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(Theme.inkSecondary)
                .frame(width: 30, height: 30)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .help(tooltip)
    }

    private var viewTitle: String {
        switch store.activeView {
        case .inbox: return "Inbox"
        case .today: return "Today"
        case .upcoming: return "Upcoming"
        case .someday: return "Someday"
        case .logbook: return "Logbook"
        case .project(let id):
            return store.projects.first { $0.id == id }?.title ?? "Project"
        }
    }

    @ViewBuilder
    private var viewIcon: some View {
        switch store.activeView {
        case .inbox:
            Image(systemName: "tray").foregroundStyle(Theme.accent)
        case .today:
            Image(systemName: "star.fill").foregroundStyle(Theme.todayStar)
        case .upcoming:
            Image(systemName: "calendar").foregroundStyle(Theme.accent)
        case .someday:
            Image(systemName: "clock").foregroundStyle(Theme.somedayTint)
        case .logbook:
            Image(systemName: "archivebox").foregroundStyle(Theme.accent)
        case .project(let id):
            let color = store.projects.first { $0.id == id }?.color ?? ""
            Circle()
                .fill(Color(hex: color.isEmpty ? "#4670a0" : color))
                .frame(width: 10, height: 10)
        }
    }

    private var todayDateString: String {
        let fmt = DateFormatter()
        fmt.dateFormat = "EEEE, MMM d"
        return fmt.string(from: Date())
    }

    // MARK: - View Content

    @ViewBuilder
    private var viewContent: some View {
        switch store.activeView {
        case .inbox:
            taskListContent(store.inbox, emptyMessage: "Inbox Zero ✓", emptyColor: Theme.success)
        case .today:
            taskListContent(store.today, emptyMessage: "Your day is clear.")
        case .upcoming:
            taskListContent(store.upcoming, emptyMessage: "Nothing scheduled ahead.")
        case .someday:
            taskListContent(store.someday, emptyMessage: "No someday tasks. Everything is decided.")
        case .logbook:
            taskListContent(store.logbook, emptyMessage: "Nothing completed yet. Get started!")
        case .project(let id):
            taskListContent(store.tasksForProject(id), emptyMessage: "No tasks in this project yet.")
        }
    }

    @ViewBuilder
    private func taskListContent(_ tasks: [TaskModel], emptyMessage: String, emptyColor: Color = Theme.inkTertiary) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            if tasks.isEmpty {
                VStack {
                    Spacer(minLength: 100)
                    Text(emptyMessage)
                        .foregroundStyle(emptyColor)
                        .font(.system(size: 15))
                    Spacer(minLength: 40)
                }
                .frame(maxWidth: .infinity)
            } else {
                ForEach(tasks) { task in
                    taskRow(task)
                }
            }

            // New task row (not in logbook or upcoming)
            if store.activeView != .logbook && store.activeView != .upcoming {
                NewTaskRow { title in
                    store.createTaskInView(title: title)
                }
            }
        }
    }

    private func taskRow(_ task: TaskModel) -> some View {
        let isToday = store.activeView == .today
        let isSelected = store.selectedTaskId == task.id

        return HStack(spacing: 12) {
            // Checkbox — large hit area
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
                        lineWidth: 1.5
                    )
                    .background(
                        Circle().fill(task.isCompleted ? Theme.accent : .clear)
                    )
                    .frame(width: 20, height: 20)
                    .overlay {
                        if task.isCompleted {
                            Image(systemName: "checkmark")
                                .font(.system(size: 10, weight: .bold))
                                .foregroundStyle(.white)
                        }
                    }
                    .padding(6) // expands tap target to 32x32
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            // Title
            Text(task.title)
                .font(.system(size: 14))
                .lineLimit(1)
                .foregroundStyle(task.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                .strikethrough(task.isCompleted, color: Theme.inkTertiary)

            Spacer()

            // Metadata: project pill + deadline
            HStack(spacing: 6) {
                // Project pill
                if let pid = task.projectId,
                   let project = store.projects.first(where: { $0.id == pid }) {
                    HStack(spacing: 4) {
                        Circle()
                            .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                            .frame(width: 6, height: 6)
                        Text(project.title)
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(Theme.inkTertiary)
                    }
                }

                // Deadline
                if let deadline = task.deadline {
                    let (label, variant) = DateFormatting.formatDeadline(deadline)
                    if task.projectId != nil { Text("·").foregroundStyle(Theme.inkQuaternary).font(.system(size: 11)) }
                    Text(label)
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(
                            variant == .overdue ? Theme.deadlineRed :
                            variant == .today ? Theme.todayStar :
                            Theme.inkTertiary
                        )
                }
            }
        }
        .padding(.vertical, 4)
        .padding(.horizontal, 12)
        .frame(minHeight: 32)
        .background(
            RoundedRectangle(cornerRadius: 6)
                .fill(isSelected ? Theme.accentSubtle : Color.clear)
        )
        .contentShape(Rectangle())
        .onTapGesture {
            store.selectedTaskId = task.id
            store.expandedTaskId = task.id
        }
        .contextMenu {
            Button { store.completeTask(task.id) } label: {
                Label("Complete", systemImage: "checkmark")
            }
            Button { store.setSchedule(task.id, 1) } label: {
                Label("Schedule for Today", systemImage: "star")
            }
            Button { store.setSchedule(task.id, 2) } label: {
                Label("Defer to Someday", systemImage: "clock")
            }
            Divider()
            Button { store.setSchedule(task.id, 0) } label: {
                Label("Move to Inbox", systemImage: "tray")
            }
            Divider()
            Button(role: .destructive) { store.deleteTask(task.id) } label: {
                Label("Delete", systemImage: "trash")
            }
        }
    }
}
