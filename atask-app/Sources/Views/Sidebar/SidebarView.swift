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
                Label("Inbox", systemImage: "tray")
                    .tag(SidebarItem.inbox)
                    .badge(store.inbox.count)

                Label {
                    Text("Today")
                } icon: {
                    Image(systemName: "star.fill")
                        .foregroundStyle(Theme.todayStar)
                }
                .tag(SidebarItem.today)
                .badge(store.today.count)

                Label("Upcoming", systemImage: "calendar")
                    .tag(SidebarItem.upcoming)

                Label {
                    Text("Someday")
                } icon: {
                    Image(systemName: "clock")
                        .foregroundStyle(Theme.somedayTint)
                }
                .tag(SidebarItem.someday)

                Label("Logbook", systemImage: "archivebox")
                    .tag(SidebarItem.logbook)
            }

            // Areas with nested projects
            ForEach(store.areas) { area in
                Section(area.title) {
                    ForEach(projectsForArea(area.id)) { project in
                        Label {
                            Text(project.title)
                        } icon: {
                            Circle()
                                .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                                .frame(width: 8, height: 8)
                        }
                        .tag(SidebarItem.project(project.id))
                        .contextMenu {
                            Button("Rename...") { /* TODO */ }
                            Button("Set Color...") { /* TODO */ }
                            Divider()
                            Button("Delete", role: .destructive) { /* TODO */ }
                        }
                    }
                }
            }

            // Orphan projects
            let orphans = store.projects.filter { $0.areaId == nil && !$0.isCompleted }
            if !orphans.isEmpty {
                Section {
                    ForEach(orphans) { project in
                        Label {
                            Text(project.title)
                        } icon: {
                            Circle()
                                .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                                .frame(width: 8, height: 8)
                        }
                        .tag(SidebarItem.project(project.id))
                    }
                }
            }

            // Add buttons
            Section {
                if addingProject {
                    TextField("Project name", text: $projectTitle)
                        .onSubmit {
                            if !projectTitle.isEmpty { store.createProject(title: projectTitle) }
                            projectTitle = ""
                            addingProject = false
                        }
                        .onExitCommand { projectTitle = ""; addingProject = false }
                } else {
                    Button { addingProject = true } label: {
                        Label("Project", systemImage: "plus")
                            .foregroundStyle(Theme.inkTertiary)
                    }
                    .buttonStyle(.plain)
                }

                if addingArea {
                    TextField("Area name", text: $areaTitle)
                        .onSubmit {
                            if !areaTitle.isEmpty { store.createArea(title: areaTitle) }
                            areaTitle = ""
                            addingArea = false
                        }
                        .onExitCommand { areaTitle = ""; addingArea = false }
                } else {
                    Button { addingArea = true } label: {
                        Label("Area", systemImage: "plus")
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

    private func projectsForArea(_ areaId: String) -> [ProjectModel] {
        store.projects.filter { $0.areaId == areaId && !$0.isCompleted }
    }
}

enum SidebarItem: Hashable {
    case inbox, today, upcoming, someday, logbook
    case project(String)

    func toActiveView() -> ActiveView {
        switch self {
        case .inbox: return .inbox
        case .today: return .today
        case .upcoming: return .upcoming
        case .someday: return .someday
        case .logbook: return .logbook
        case .project(let id): return .project(id)
        }
    }
}
