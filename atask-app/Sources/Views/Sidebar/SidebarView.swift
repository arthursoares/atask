import SwiftUI

/// Sidebar per MEASUREMENTS.md:
/// - Item: 5px 12px padding, 12px gap, 14px font, radius-sm (6px)
/// - Area labels: 11px bold uppercase, letter-spacing 0.8px — NON-SELECTABLE
/// - Badge: 11px, ink-tertiary, right-aligned
/// - Dot: 8px
struct SidebarView: View {
    @Bindable var store: TaskStore
    @State private var addingProject = false
    @State private var projectTitle = ""
    @State private var addingArea = false
    @State private var areaTitle = ""
    @State private var renamingProjectId: String?
    @State private var renamingAreaId: String?
    @State private var renameDraft = ""

    var body: some View {
        List(selection: $store.sidebarSelection) {
            // Nav items
            Section {
                navRow("Inbox", icon: "tray", color: nil, tag: .inbox, count: store.inbox.count)
                navRow("Today", icon: "star.fill", color: Theme.todayStar, tag: .today, count: store.today.count)
                navRow("Upcoming", icon: "calendar", color: nil, tag: .upcoming, count: store.upcoming.count)
                navRow("Someday", icon: "clock", color: Theme.somedayTint, tag: .someday, count: store.someday.count)
                navRow("Logbook", icon: "archivebox", color: nil, tag: .logbook, count: 0)
            }

            // Areas with nested projects (areas are NON-SELECTABLE headers)
            ForEach(store.areas) { area in
                Section {
                    ForEach(projectsForArea(area.id)) { project in
                        projectRow(project)
                    }
                } header: {
                    Group {
                        if renamingAreaId == area.id {
                            TextField("Area name", text: $renameDraft)
                                .font(.groupLabel)
                                .textFieldStyle(.plain)
                                .onSubmit {
                                    if !renameDraft.isEmpty { store.renameArea(area.id, renameDraft) }
                                    renamingAreaId = nil
                                }
                                .onExitCommand { renamingAreaId = nil }
                        } else {
                            Text(area.title)
                                .font(.groupLabel)
                                .foregroundStyle(Theme.inkTertiary)
                                .textCase(.uppercase)
                                .tracking(0.8)
                        }
                    }
                    .contextMenu {
                        Button("Rename") {
                            renameDraft = area.title
                            renamingAreaId = area.id
                        }
                        Divider()
                        Button("Delete", role: .destructive) { store.deleteArea(area.id) }
                    }
                    .dropDestination(for: String.self) { projectIds, _ in
                            guard let projectId = projectIds.first else { return false }
                            // Only accept if it's a project ID (not a task)
                            if store.projects.contains(where: { $0.id == projectId }) {
                                store.moveProjectToArea(projectId, area.id)
                                return true
                            }
                            return false
                        }
                }
            }

            // Orphan projects (no area)
            let orphans = store.projects.filter { $0.areaId == nil && !$0.isCompleted }
            if !orphans.isEmpty {
                Section {
                    ForEach(orphans) { project in
                        projectRow(project)
                    }
                }
            }

            // Add buttons
            Section {
                if addingProject {
                    TextField("Project name", text: $projectTitle)
                        .font(.taskTitle)
                        .onSubmit { submitProject() }
                        .onExitCommand { projectTitle = ""; addingProject = false }
                } else {
                    Button { addingProject = true } label: {
                        Label("Project", systemImage: "plus")
                            .font(.taskTitle)
                            .foregroundStyle(Theme.inkTertiary)
                    }
                    .buttonStyle(.plain)
                }

                if addingArea {
                    TextField("Area name", text: $areaTitle)
                        .font(.taskTitle)
                        .onSubmit { submitArea() }
                        .onExitCommand { areaTitle = ""; addingArea = false }
                } else {
                    Button { addingArea = true } label: {
                        Label("Area", systemImage: "plus")
                            .font(.taskTitle)
                            .foregroundStyle(Theme.inkTertiary)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .listStyle(.sidebar)
        .onChange(of: store.sidebarSelection) { _, newValue in
            if let item = newValue {
                store.activeView = item.toActiveView()
            }
        }
    }

    // ── Nav row with manual badge (not .badge which broke selection) ──
    private func navRow(_ title: String, icon: String, color: Color?, tag: SidebarItem, count: Int) -> some View {
        HStack {
            Label {
                Text(title).font(.taskTitle)
            } icon: {
                Image(systemName: icon)
                    .foregroundStyle(color ?? Theme.inkSecondary)
            }
            Spacer()
            if count > 0 {
                Text("\(count)")
                    .font(.groupLabel)
                    .foregroundStyle(Theme.inkTertiary)
            }
        }
        .tag(tag)
        .dropDestination(for: String.self) { taskIds, _ in
            guard let taskId = taskIds.first else { return false }
            switch tag {
            case .inbox:
                store.setSchedule(taskId, 0)
                store.moveToProject(taskId, nil)
            case .today:
                store.setSchedule(taskId, 1)
            case .someday:
                store.setSchedule(taskId, 2)
            default:
                return false
            }
            return true
        }
    }

    // ── Project row ──
    private func projectRow(_ project: ProjectModel) -> some View {
        HStack {
            Circle()
                .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                .frame(width: Spacing.sidebarDot, height: Spacing.sidebarDot)
            if renamingProjectId == project.id {
                TextField("Project name", text: $renameDraft)
                    .font(.taskTitle)
                    .textFieldStyle(.plain)
                    .onSubmit {
                        if !renameDraft.isEmpty { store.renameProject(project.id, renameDraft) }
                        renamingProjectId = nil
                    }
                    .onExitCommand { renamingProjectId = nil }
            } else {
                Text(project.title).font(.taskTitle)
            }
            Spacer()
        }
        .tag(SidebarItem.project(project.id))
        .draggable(project.id) {
            HStack(spacing: Spacing.sp3) {
                Circle()
                    .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                    .frame(width: Spacing.sidebarDot, height: Spacing.sidebarDot)
                Text(project.title)
                    .font(.taskTitle)
                    .foregroundStyle(Theme.inkPrimary)
            }
            .padding(.horizontal, Spacing.sp3)
            .padding(.vertical, 5)
            .background(
                RoundedRectangle(cornerRadius: Radius.sm)
                    .fill(Theme.sidebarSelected)
            )
        }
        .dropDestination(for: String.self) { taskIds, _ in
            guard let taskId = taskIds.first else { return false }
            store.moveToProject(taskId, project.id)
            return true
        }
        .contextMenu {
            Button("Rename") {
                renameDraft = project.title
                renamingProjectId = project.id
            }

            if !store.areas.isEmpty {
                Menu("Move to Area") {
                    Button("No Area") { store.moveProjectToArea(project.id, nil) }
                    Divider()
                    ForEach(store.areas) { area in
                        Button(area.title) { store.moveProjectToArea(project.id, area.id) }
                    }
                }
            }

            Divider()
            Button("Delete", role: .destructive) { store.deleteProject(project.id) }
        }
    }

    private func projectsForArea(_ areaId: String) -> [ProjectModel] {
        store.projects.filter { $0.areaId == areaId && !$0.isCompleted }
    }

    private func submitProject() {
        if !projectTitle.isEmpty { store.createProject(title: projectTitle) }
        projectTitle = ""; addingProject = false
    }

    private func submitArea() {
        if !areaTitle.isEmpty { store.createArea(title: areaTitle) }
        areaTitle = ""; addingArea = false
    }
}

enum SidebarItem: Hashable {
    case inbox, today, upcoming, someday, logbook
    case project(String)

    func toActiveView() -> ActiveView {
        switch self {
        case .inbox: .inbox
        case .today: .today
        case .upcoming: .upcoming
        case .someday: .someday
        case .logbook: .logbook
        case .project(let id): .project(id)
        }
    }
}
