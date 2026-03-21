import SwiftUI

struct ContentView: View {
    @Bindable var store: TaskStore

    var body: some View {
        NavigationSplitView {
            SidebarView(store: store)
                .frame(minWidth: 200)
        } content: {
            VStack(alignment: .leading, spacing: 0) {
                // Toolbar
                toolbarView
                    .frame(height: 52)
                    .padding(.horizontal)

                Divider()

                // Content
                ScrollView {
                    viewContent
                        .padding()
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

    // MARK: - Toolbar

    @ViewBuilder
    private var toolbarView: some View {
        HStack {
            viewIcon
            Text(viewTitle)
                .font(.system(size: 20, weight: .bold))
                .foregroundStyle(Theme.inkPrimary)

            if store.activeView == .today {
                Text(todayDateString)
                    .font(.system(size: 13))
                    .foregroundStyle(Theme.inkTertiary)
            }

            Spacer()
        }
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
        if tasks.isEmpty {
            VStack {
                Spacer(minLength: 100)
                Text(emptyMessage)
                    .foregroundStyle(emptyColor)
                    .font(.system(size: 15))
                Spacer()
            }
            .frame(maxWidth: .infinity)
        } else {
            VStack(alignment: .leading, spacing: 0) {
                ForEach(tasks) { task in
                    taskRow(task)
                }
            }
        }
    }

    // Temporary task row — replaced in Task 5 with full TaskRow component
    private func taskRow(_ task: TaskModel) -> some View {
        HStack(spacing: 12) {
            // Checkbox
            Button {
                if task.isCompleted {
                    store.reopenTask(task.id)
                } else {
                    store.completeTask(task.id)
                }
            } label: {
                Circle()
                    .strokeBorder(
                        store.activeView == .today ? Theme.todayStar : Theme.inkQuaternary,
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
            }
            .buttonStyle(.plain)

            // Title
            Text(task.title)
                .lineLimit(1)
                .foregroundStyle(task.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                .strikethrough(task.isCompleted)

            Spacer()

            // Deadline
            if let deadline = task.deadline {
                let (label, variant) = DateFormatting.formatDeadline(deadline)
                Text(label)
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(
                        variant == .overdue ? Theme.deadlineRed :
                        variant == .today ? Theme.todayStar :
                        Theme.inkTertiary
                    )
            }
        }
        .padding(.vertical, 6)
        .padding(.horizontal, 16)
        .frame(height: 32)
        .background(
            store.selectedTaskId == task.id
                ? Theme.accentSubtle
                : Color.clear
        )
        .cornerRadius(8)
        .contentShape(Rectangle())
        .onTapGesture {
            store.selectedTaskId = task.id
        }
        .contextMenu {
            Button("Complete") { store.completeTask(task.id) }
            Button("Schedule for Today") { store.setSchedule(task.id, 1) }
            Button("Defer to Someday") { store.setSchedule(task.id, 2) }
            Divider()
            Button("Delete", role: .destructive) { store.deleteTask(task.id) }
        }
    }
}
