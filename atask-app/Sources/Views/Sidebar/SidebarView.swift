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

    var body: some View {
        List(selection: $store.sidebarSelection) {
            // Nav items
            Section {
                navRow("Inbox", icon: "tray", color: nil, tag: .inbox, count: store.inbox.count)
                navRow("Today", icon: "star.fill", color: Theme.todayStar, tag: .today, count: store.today.count)
                navRow("Upcoming", icon: "calendar", color: nil, tag: .upcoming, count: 0)
                navRow("Someday", icon: "clock", color: Theme.somedayTint, tag: .someday, count: 0)
                navRow("Logbook", icon: "archivebox", color: nil, tag: .logbook, count: 0)
            }

            // Areas with nested projects (areas are NON-SELECTABLE headers)
            ForEach(store.areas) { area in
                Section {
                    ForEach(projectsForArea(area.id)) { project in
                        projectRow(project)
                    }
                } header: {
                    Text(area.title)
                        .font(.groupLabel)
                        .foregroundStyle(Theme.inkTertiary)
                        .textCase(.uppercase)
                        .tracking(0.8)
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
    }

    // ── Project row ──
    private func projectRow(_ project: ProjectModel) -> some View {
        HStack {
            Label {
                Text(project.title).font(.taskTitle)
            } icon: {
                Circle()
                    .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                    .frame(width: Spacing.sidebarDot, height: Spacing.sidebarDot)
            }
            Spacer()
        }
        .tag(SidebarItem.project(project.id))
        .contextMenu {
            Button("Rename...") { }
            Button("Set Color...") { }
            Divider()
            Button("Delete", role: .destructive) { }
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
