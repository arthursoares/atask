import SwiftUI

struct SidebarView: View {
    @Bindable var store: TaskStore
    @State private var addingProject = false
    @State private var projectTitle = ""
    @State private var addingArea = false
    @State private var areaTitle = ""

    var body: some View {
        List(selection: $store.sidebarSelection) {
            Section {
                navRow("Inbox", icon: "tray", color: nil, tag: .inbox, count: store.inbox.count)
                navRow("Today", icon: "star.fill", color: Theme.todayStar, tag: .today, count: store.today.count)
                navRow("Upcoming", icon: "calendar", color: nil, tag: .upcoming, count: 0)
                navRow("Someday", icon: "clock", color: Theme.somedayTint, tag: .someday, count: 0)
                navRow("Logbook", icon: "archivebox", color: nil, tag: .logbook, count: 0)
            }

            ForEach(store.areas) { area in
                Section(area.title) {
                    ForEach(projectsForArea(area.id)) { project in
                        projectRow(project)
                    }
                }
            }

            let orphans = store.projects.filter { $0.areaId == nil && !$0.isCompleted }
            if !orphans.isEmpty {
                Section {
                    ForEach(orphans) { project in
                        projectRow(project)
                    }
                }
            }

            Section {
                if addingProject {
                    TextField("Project name", text: $projectTitle)
                        .onSubmit { submitProject() }
                        .onExitCommand { projectTitle = ""; addingProject = false }
                } else {
                    Button { addingProject = true } label: {
                        Label("Project", systemImage: "plus").foregroundStyle(Theme.inkTertiary)
                    }.buttonStyle(.plain)
                }

                if addingArea {
                    TextField("Area name", text: $areaTitle)
                        .onSubmit { submitArea() }
                        .onExitCommand { areaTitle = ""; addingArea = false }
                } else {
                    Button { addingArea = true } label: {
                        Label("Area", systemImage: "plus").foregroundStyle(Theme.inkTertiary)
                    }.buttonStyle(.plain)
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

    // MARK: - Nav Row (no .badge — manual count)

    private func navRow(_ title: String, icon: String, color: Color?, tag: SidebarItem, count: Int) -> some View {
        HStack {
            Label {
                Text(title)
            } icon: {
                Image(systemName: icon)
                    .foregroundStyle(color ?? Theme.inkSecondary)
            }
            Spacer()
            if count > 0 {
                Text("\(count)")
                    .font(.system(size: FontSize.xs))
                    .foregroundStyle(Theme.inkTertiary)
            }
        }
        .tag(tag)
    }

    // MARK: - Project Row

    private func projectRow(_ project: ProjectModel) -> some View {
        Label {
            Text(project.title)
        } icon: {
            Circle()
                .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                .frame(width: Size.sidebarDot, height: Size.sidebarDot)
        }
        .tag(SidebarItem.project(project.id))
        .contextMenu {
            Button("Rename...") { }
            Button("Set Color...") { }
            Divider()
            Button("Delete", role: .destructive) { }
        }
    }

    // MARK: - Helpers

    private func projectsForArea(_ areaId: String) -> [ProjectModel] {
        store.projects.filter { $0.areaId == areaId && !$0.isCompleted }
    }

    private func submitProject() {
        if !projectTitle.isEmpty { store.createProject(title: projectTitle) }
        projectTitle = ""
        addingProject = false
    }

    private func submitArea() {
        if !areaTitle.isEmpty { store.createArea(title: areaTitle) }
        areaTitle = ""
        addingArea = false
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
