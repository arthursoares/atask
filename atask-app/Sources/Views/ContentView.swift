import SwiftUI

struct ContentView: View {
    @Bindable var store: TaskStore

    var body: some View {
        NavigationSplitView {
            SidebarView(store: store)
                .frame(minWidth: 200)
        } content: {
            VStack(spacing: 0) {
                // ── Toolbar: 52px, padding 0 24px, border-bottom separator ──
                HStack {
                    HStack(spacing: Spacing.sp2) {
                        viewIcon
                        Text(viewTitle)
                            .font(.viewTitle)
                            .foregroundStyle(Theme.inkPrimary)
                        if store.activeView == .today {
                            Text(todayDateString)
                                .font(.metadata)
                                .foregroundStyle(Theme.inkTertiary)
                        }
                    }

                    Spacer()

                    HStack(spacing: Spacing.sp2) {
                        toolbarButton(icon: "magnifyingglass") { }
                        toolbarButton(icon: "plus") {
                            let task = store.createTask(title: "")
                            store.expandedTaskId = task.id
                        }
                    }
                }
                .padding(.horizontal, Spacing.sp6)
                .frame(height: Spacing.toolbarHeight)

                // ── Separator: 1px ──
                Rectangle()
                    .fill(Theme.separator)
                    .frame(height: 1)

                // ── Content: padding 24px (sp-6) ──
                ScrollView {
                    viewContent
                        .padding(Spacing.sp6)
                }
                .focusable()
                .onKeyPress(.upArrow) { navigateTask(direction: -1); return .handled }
                .onKeyPress(.downArrow) { navigateTask(direction: 1); return .handled }
                .onKeyPress(.return) {
                    if let id = store.selectedTaskId, store.expandedTaskId == nil {
                        store.expandedTaskId = id
                    }
                    return .handled
                }
                .onKeyPress(.escape) {
                    if store.expandedTaskId != nil {
                        store.expandedTaskId = nil
                    } else if store.selectedTaskId != nil {
                        store.selectedTaskId = nil
                    }
                    return .handled
                }
                .onTapGesture {
                    if store.expandedTaskId != nil {
                        store.expandedTaskId = nil
                    }
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(Theme.canvas)
        } detail: {
            if store.selectedTaskId != nil {
                TaskDetailView(store: store)
            }
        }
        .navigationSplitViewStyle(.balanced)
    }

    // ── Arrow key navigation ──
    private func navigateTask(direction: Int) {
        guard store.expandedTaskId == nil else { return } // don't navigate while editing
        let tasks = currentViewTasks()
        guard !tasks.isEmpty else { return }

        if let currentId = store.selectedTaskId,
           let currentIdx = tasks.firstIndex(where: { $0.id == currentId }) {
            let newIdx = max(0, min(tasks.count - 1, currentIdx + direction))
            store.selectedTaskId = tasks[newIdx].id
        } else {
            // Nothing selected — select first or last
            store.selectedTaskId = direction > 0 ? tasks.first?.id : tasks.last?.id
        }
    }

    private func currentViewTasks() -> [TaskModel] {
        switch store.activeView {
        case .inbox: store.inbox
        case .today: store.today
        case .upcoming: store.upcoming
        case .someday: store.someday
        case .logbook: store.logbook
        case .project(let id): store.tasksForProject(id)
        }
    }

    // ── Toolbar button: 30×30, radius-sm ──
    private func toolbarButton(icon: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(Theme.inkTertiary)
                .frame(width: Spacing.toolbarBtnSize, height: Spacing.toolbarBtnSize)
        }
        .buttonStyle(.plain)
    }

    // ── View icon ──
    @ViewBuilder
    private var viewIcon: some View {
        Group {
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
        .font(.system(size: 14))
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
            todayView
        case .upcoming:
            upcomingView
        case .someday:
            taskList(store.someday, empty: "No someday tasks. Everything is decided.")
        case .logbook:
            taskList(store.logbook, empty: "Nothing completed yet. Get started!")
        case .project(let id):
            projectView(id)
        }
    }

    // ── Today: morning + evening sections ──
    @ViewBuilder
    private var todayView: some View {
        let morning = store.todayMorning
        let evening = store.todayEvening

        if morning.isEmpty && evening.isEmpty {
            emptyState("Your day is clear.")
        } else {
            VStack(alignment: .leading, spacing: 0) {
                ForEach(morning) { task in taskOrEditor(task) }
                if !evening.isEmpty {
                    sectionHeader("This Evening")
                    ForEach(evening) { task in taskOrEditor(task) }
                }
            }
        }
        NewTaskRow { title in store.createTask(title: title) }
    }

    // ── Upcoming: grouped by startDate ──
    @ViewBuilder
    private var upcomingView: some View {
        let tasks = store.upcoming
        if tasks.isEmpty {
            emptyState("Nothing scheduled ahead.")
        } else {
            let grouped = Dictionary(grouping: tasks) { String(($0.startDate ?? "").prefix(10)) }
            let sortedKeys = grouped.keys.sorted()
            VStack(alignment: .leading, spacing: 0) {
                ForEach(sortedKeys, id: \.self) { dateKey in
                    dateGroupHeader(DateFormatting.formatSectionDate(dateKey))
                    ForEach(grouped[dateKey] ?? []) { task in taskOrEditor(task) }
                }
            }
        }
    }

    // ── Project: sections + tasks ──
    @ViewBuilder
    private func projectView(_ projectId: String) -> some View {
        let allTasks = store.tasksForProject(projectId)
        let projectSections = store.sections.filter { $0.projectId == projectId }
        let sectionlessTasks = allTasks.filter { $0.sectionId == nil }

        VStack(alignment: .leading, spacing: 0) {
            ForEach(sectionlessTasks) { task in taskOrEditor(task) }
            NewTaskRow { title in
                var task = TaskModel.create(title: title, projectId: projectId)
                store.persist(task: &task)
            }
            ForEach(projectSections) { section in
                let sectionTasks = allTasks.filter { $0.sectionId == section.id }
                sectionHeaderWithCount(section.title, count: sectionTasks.count)
                ForEach(sectionTasks) { task in taskOrEditor(task) }
                NewTaskRow { title in
                    var task = TaskModel.create(title: title, projectId: projectId)
                    task.sectionId = section.id
                    store.persist(task: &task)
                }
            }
            if allTasks.isEmpty && projectSections.isEmpty {
                emptyState("No tasks in this project yet.")
            }
        }
    }

    // ── Generic task list ──
    @ViewBuilder
    private func taskList(_ tasks: [TaskModel], empty: String, emptyColor: Color = Theme.inkTertiary) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            if tasks.isEmpty {
                emptyState(empty, color: emptyColor)
            } else {
                ForEach(tasks) { task in taskOrEditor(task) }
            }
            if store.activeView != .logbook && store.activeView != .upcoming {
                NewTaskRow { title in store.createTask(title: title) }
            }
        }
    }

    // ── Task or inline editor ──
    @ViewBuilder
    private func taskOrEditor(_ task: TaskModel) -> some View {
        if store.expandedTaskId == task.id {
            TaskInlineEditor(store: store, taskId: task.id)
        } else {
            taskRow(task)
        }
    }

    // ── Task row: 32px, 12px gap, 6px 16px padding, radius-md ──
    private func taskRow(_ task: TaskModel) -> some View {
        let isToday = store.activeView == .today
        let isSelected = store.selectedTaskId == task.id

        return HStack(spacing: Spacing.sp3) {
            // Checkbox: 20×20
            Button {
                if task.isCompleted { store.reopenTask(task.id) }
                else { store.completeTask(task.id) }
            } label: {
                Circle()
                    .strokeBorder(
                        task.isCompleted ? Theme.accent :
                        isToday ? Theme.todayStar :
                        Theme.inkQuaternary,
                        lineWidth: Spacing.checkboxBorder
                    )
                    .background(Circle().fill(task.isCompleted ? Theme.accent : .clear))
                    .frame(width: Spacing.checkboxSize, height: Spacing.checkboxSize)
                    .overlay {
                        if task.isCompleted {
                            Image(systemName: "checkmark")
                                .font(.system(size: 9, weight: .bold))
                                .foregroundStyle(.white)
                        }
                    }
                    .contentShape(Circle().inset(by: -6))
            }
            .buttonStyle(.plain)

            // Title: 14px Atkinson
            Text(task.title.isEmpty ? "Untitled" : task.title)
                .font(.taskTitle)
                .lineLimit(1)
                .foregroundStyle(task.isCompleted ? Theme.inkTertiary : Theme.inkPrimary)
                .strikethrough(task.isCompleted, color: Theme.inkQuaternary)

            Spacer(minLength: Spacing.sp2)

            // Meta: 11px, right-aligned
            taskMeta(task)
        }
        .frame(height: Spacing.taskRowHeight)
        .padding(.vertical, 6)
        .padding(.horizontal, Spacing.sp4)
        .background(
            RoundedRectangle(cornerRadius: Radius.md)
                .fill(isSelected ? Theme.sidebarSelected : Color.clear)
        )
        .contentShape(Rectangle())
        .onTapGesture {
            store.selectedTaskId = task.id
            store.expandedTaskId = task.id
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

    // ── Task metadata: project pill + deadline ──
    @ViewBuilder
    private func taskMeta(_ task: TaskModel) -> some View {
        HStack(spacing: Spacing.sp2) {
            if let project = store.projectFor(task) {
                HStack(spacing: 3) {
                    Circle()
                        .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                        .frame(width: Spacing.metaDot, height: Spacing.metaDot)
                    Text(project.title)
                        .font(.tagPill)
                        .foregroundStyle(Theme.inkSecondary)
                }
                .padding(.horizontal, 7)
                .padding(.vertical, 1)
                .background(Theme.canvasSunken)
                .clipShape(Capsule())
            }

            if let deadline = task.deadline {
                if task.projectId != nil {
                    Text("·").font(.tagPill).foregroundStyle(Theme.inkQuaternary)
                }
                let (label, variant) = DateFormatting.formatDeadline(deadline)
                Text(label)
                    .font(.metadata)
                    .foregroundStyle(
                        variant == .overdue ? Theme.deadlineRed :
                        variant == .today ? Theme.todayStar :
                        Theme.inkTertiary
                    )
            }
        }
    }

    // ── Section header: title + line ──
    private func sectionHeader(_ title: String) -> some View {
        HStack(spacing: Spacing.sp2) {
            Text(title)
                .font(.atkinson(14, weight: .bold))
                .foregroundStyle(Theme.inkPrimary)
            Rectangle()
                .fill(Theme.separator)
                .frame(height: 1)
        }
        .padding(.top, Spacing.sp3)
        .padding(.bottom, Spacing.sp1)
    }

    // ── Section header with count ──
    private func sectionHeaderWithCount(_ title: String, count: Int) -> some View {
        HStack(spacing: Spacing.sp2) {
            Text(title)
                .font(.atkinson(14, weight: .bold))
                .foregroundStyle(Theme.inkPrimary)
            if count > 0 {
                Text("\(count)")
                    .font(.groupLabel)
                    .foregroundStyle(Theme.inkTertiary)
            }
            Rectangle()
                .fill(Theme.separator)
                .frame(height: 1)
        }
        .padding(.top, Spacing.sp3)
        .padding(.bottom, Spacing.sp1)
    }

    // ── Date group header (Upcoming/Logbook) ──
    private func dateGroupHeader(_ title: String) -> some View {
        Text(title)
            .font(.metadata)
            .foregroundStyle(Theme.inkPrimary)
            .padding(.vertical, Spacing.sp1)
    }

    // ── Empty state ──
    private func emptyState(_ message: String, color: Color = Theme.inkTertiary) -> some View {
        VStack {
            Spacer(minLength: Spacing.sp20)
            Text(message)
                .font(.detailBody)
                .foregroundStyle(color)
            Spacer(minLength: Spacing.sp10)
        }
        .frame(maxWidth: .infinity)
    }
}
