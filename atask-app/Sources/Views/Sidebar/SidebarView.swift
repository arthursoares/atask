import SwiftUI

struct SidebarView: View {
    @Bindable var store: TaskStore
    @State private var addingProject = false
    @State private var projectTitle = ""
    @State private var addingArea = false
    @State private var areaTitle = ""

    var body: some View {
        List {
            // Nav items
            Section {
                sidebarButton(.inbox) {
                    Label("Inbox", systemImage: "tray")
                        .badge(store.inbox.count)
                }

                sidebarButton(.today) {
                    Label {
                        Text("Today")
                    } icon: {
                        Image(systemName: "star.fill")
                            .foregroundStyle(Theme.todayStar)
                    }
                    .badge(store.today.count)
                }

                sidebarButton(.upcoming) {
                    Label("Upcoming", systemImage: "calendar")
                }

                sidebarButton(.someday) {
                    Label {
                        Text("Someday")
                    } icon: {
                        Image(systemName: "clock")
                            .foregroundStyle(Theme.somedayTint)
                    }
                }

                sidebarButton(.logbook) {
                    Label("Logbook", systemImage: "archivebox")
                }
            }

            // Areas with nested projects
            ForEach(store.areas) { area in
                Section(area.title) {
                    ForEach(projectsForArea(area.id)) { project in
                        projectButton(project)
                    }
                }
            }

            // Orphan projects (no area)
            let orphans = store.projects.filter { $0.areaId == nil && !$0.isCompleted }
            if !orphans.isEmpty {
                Section {
                    ForEach(orphans) { project in
                        projectButton(project)
                    }
                }
            }

            // Add project / area
            Section {
                if addingProject {
                    TextField("Project name", text: $projectTitle)
                        .onSubmit {
                            if !projectTitle.isEmpty {
                                store.createProject(title: projectTitle)
                            }
                            projectTitle = ""
                            addingProject = false
                        }
                        .onExitCommand {
                            projectTitle = ""
                            addingProject = false
                        }
                } else {
                    Button {
                        addingProject = true
                    } label: {
                        Label("Project", systemImage: "plus")
                            .foregroundStyle(Theme.inkTertiary)
                    }
                    .buttonStyle(.plain)
                }

                if addingArea {
                    TextField("Area name", text: $areaTitle)
                        .onSubmit {
                            if !areaTitle.isEmpty {
                                store.createArea(title: areaTitle)
                            }
                            areaTitle = ""
                            addingArea = false
                        }
                        .onExitCommand {
                            areaTitle = ""
                            addingArea = false
                        }
                } else {
                    Button {
                        addingArea = true
                    } label: {
                        Label("Area", systemImage: "plus")
                            .foregroundStyle(Theme.inkTertiary)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .listStyle(.sidebar)
    }

    // MARK: - Sidebar Button

    private func sidebarButton<Content: View>(_ item: SidebarItem, @ViewBuilder content: () -> Content) -> some View {
        Button {
            store.activeView = item.toActiveView()
            store.sidebarSelection = item
        } label: {
            content()
        }
        .buttonStyle(.plain)
        .padding(.vertical, 2)
        .background(
            store.sidebarSelection == item
                ? Theme.accentSubtle.cornerRadius(6)
                : Color.clear.cornerRadius(6)
        )
    }

    private func projectButton(_ project: ProjectModel) -> some View {
        let item = SidebarItem.project(project.id)
        return Button {
            store.activeView = .project(project.id)
            store.sidebarSelection = item
        } label: {
            Label {
                Text(project.title)
            } icon: {
                Circle()
                    .fill(Color(hex: project.color.isEmpty ? "#4670a0" : project.color))
                    .frame(width: 8, height: 8)
            }
        }
        .buttonStyle(.plain)
        .padding(.vertical, 2)
        .background(
            store.sidebarSelection == item
                ? Theme.accentSubtle.cornerRadius(6)
                : Color.clear.cornerRadius(6)
        )
        .contextMenu {
            Button("Rename...") { /* TODO */ }
            Button("Set Color...") { /* TODO */ }
            Divider()
            Button("Delete", role: .destructive) { /* TODO */ }
        }
    }

    // MARK: - Helpers

    private func projectsForArea(_ areaId: String) -> [ProjectModel] {
        store.projects.filter { $0.areaId == areaId && !$0.isCompleted }
    }
}

// MARK: - Sidebar Selection

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
