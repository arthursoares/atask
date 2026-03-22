import SwiftUI

/// Command palette — ⇧⌘O
/// MEASUREMENTS.md: 560px, top 18%, radius-xl, border-strong, shadow-popover
/// Input: 17px (text-lg), items: 14px (text-base), 6px 16px padding
struct CommandPaletteView: View {
    @Bindable var store: TaskStore
    @Binding var isOpen: Bool
    @State private var query = ""
    @State private var selectedIndex = 0
    @FocusState private var focused: Bool

    private var filteredCommands: [Command] {
        let hasTask = store.selectedTaskId != nil
        let cmds = Command.all.filter { !$0.requiresTask || hasTask }
        if query.isEmpty { return cmds }
        return cmds.filter { $0.label.localizedCaseInsensitiveContains(query) }
    }

    var body: some View {
        ZStack {
            // Backdrop
            Color.black.opacity(0.15)
                .ignoresSafeArea()
                .onTapGesture { close() }

            // Palette
            VStack(spacing: 0) {
                // Input
                HStack(spacing: Spacing.sp3) {
                    Image(systemName: "magnifyingglass")
                        .foregroundStyle(Theme.inkTertiary)
                        .frame(width: 20)
                    TextField("Type a command or search...", text: $query)
                        .font(.sectionHeader) // 17px
                        .textFieldStyle(.plain)
                        .focused($focused)
                        .onSubmit { executeSelected() }
                }
                .padding(.horizontal, Spacing.sp4)
                .padding(.vertical, Spacing.sp3)

                Rectangle().fill(Theme.separator).frame(height: 1)

                // Results
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        let grouped = Dictionary(grouping: filteredCommands) { $0.category }
                        let categories = ["Navigation", "Edit", "Schedule", "Create"]

                        ForEach(categories, id: \.self) { category in
                            if let cmds = grouped[category], !cmds.isEmpty {
                                Text(category.uppercased())
                                    .font(.groupLabel)
                                    .foregroundStyle(Theme.inkTertiary)
                                    .tracking(0.8)
                                    .padding(.horizontal, Spacing.sp4)
                                    .padding(.top, Spacing.sp2)
                                    .padding(.bottom, Spacing.sp1)

                                ForEach(Array(cmds.enumerated()), id: \.element.id) { idx, cmd in
                                    let globalIdx = globalIndex(for: cmd)
                                    HStack(spacing: Spacing.sp3) {
                                        Text(cmd.icon)
                                            .frame(width: 20)
                                            .font(.taskTitle)
                                        Text(cmd.label)
                                            .font(.taskTitle)
                                        Spacer()
                                        if !cmd.shortcut.isEmpty {
                                            Text(cmd.shortcut)
                                                .font(.system(size: 11, design: .monospaced))
                                                .foregroundStyle(Theme.inkTertiary)
                                        }
                                    }
                                    .padding(.horizontal, Spacing.sp4)
                                    .padding(.vertical, 6)
                                    .foregroundStyle(globalIdx == selectedIndex ? Theme.accent : Theme.inkSecondary)
                                    .background(
                                        globalIdx == selectedIndex
                                            ? Theme.accentSubtle
                                            : Color.clear
                                    )
                                    .contentShape(Rectangle())
                                    .onTapGesture { execute(cmd) }
                                }
                            }
                        }
                    }
                    .padding(.vertical, Spacing.sp2)
                }
                .frame(maxHeight: 400)
            }
            .frame(width: Spacing.cmdPaletteWidth)
            .background(Theme.canvasElevated)
            .clipShape(RoundedRectangle(cornerRadius: Radius.xl))
            .overlay(
                RoundedRectangle(cornerRadius: Radius.xl)
                    .strokeBorder(Theme.borderStrong, lineWidth: 1)
            )
            .shadow(color: .black.opacity(0.12), radius: 20, y: 6)
            .shadow(color: .black.opacity(0.06), radius: 6, y: 2)
            .padding(.top, 80) // ~18% from top
            .frame(maxHeight: .infinity, alignment: .top)
        }
        .onAppear {
            focused = true
            selectedIndex = 0
            query = ""
        }
        .onChange(of: query) { _, _ in
            selectedIndex = 0
        }
        .onKeyPress(.upArrow) {
            selectedIndex = max(0, selectedIndex - 1)
            return .handled
        }
        .onKeyPress(.downArrow) {
            selectedIndex = min(filteredCommands.count - 1, selectedIndex + 1)
            return .handled
        }
        .onKeyPress(.escape) {
            close()
            return .handled
        }
    }

    private func globalIndex(for cmd: Command) -> Int {
        filteredCommands.firstIndex(where: { $0.id == cmd.id }) ?? 0
    }

    private func executeSelected() {
        guard selectedIndex < filteredCommands.count else { return }
        execute(filteredCommands[selectedIndex])
    }

    private func execute(_ cmd: Command) {
        cmd.action(store)
        close()
    }

    private func close() {
        isOpen = false
        query = ""
    }
}

// MARK: - Command Registry

struct Command: Identifiable, Sendable {
    let id: String
    let label: String
    let icon: String
    let shortcut: String
    let category: String
    let requiresTask: Bool
    let action: @Sendable @MainActor (TaskStore) -> Void

    @MainActor static let all: [Command] = [
        // Navigation
        Command(id: "nav-inbox", label: "Go to Inbox", icon: "📥", shortcut: "⌘1", category: "Navigation", requiresTask: false) { $0.activeView = .inbox; $0.sidebarSelection = .inbox },
        Command(id: "nav-today", label: "Go to Today", icon: "⭐", shortcut: "⌘2", category: "Navigation", requiresTask: false) { $0.activeView = .today; $0.sidebarSelection = .today },
        Command(id: "nav-upcoming", label: "Go to Upcoming", icon: "📅", shortcut: "⌘3", category: "Navigation", requiresTask: false) { $0.activeView = .upcoming; $0.sidebarSelection = .upcoming },
        Command(id: "nav-someday", label: "Go to Someday", icon: "🕐", shortcut: "⌘5", category: "Navigation", requiresTask: false) { $0.activeView = .someday; $0.sidebarSelection = .someday },
        Command(id: "nav-logbook", label: "Go to Logbook", icon: "📦", shortcut: "⌘6", category: "Navigation", requiresTask: false) { $0.activeView = .logbook; $0.sidebarSelection = .logbook },

        // Edit
        Command(id: "complete", label: "Complete Task", icon: "✓", shortcut: "⌘K", category: "Edit", requiresTask: true) { if let id = $0.selectedTaskId { $0.completeTask(id) } },
        Command(id: "cancel", label: "Cancel Task", icon: "✕", shortcut: "⌥⌘K", category: "Edit", requiresTask: true) { if let id = $0.selectedTaskId { $0.cancelTask(id) } },
        Command(id: "delete", label: "Delete Task", icon: "🗑", shortcut: "⌫", category: "Edit", requiresTask: true) { if let id = $0.selectedTaskId { $0.deleteTask(id); $0.selectedTaskId = nil } },

        // Schedule
        Command(id: "today", label: "Schedule for Today", icon: "⭐", shortcut: "⌘T", category: "Schedule", requiresTask: true) { if let id = $0.selectedTaskId { $0.setSchedule(id, 1) } },
        Command(id: "evening", label: "This Evening", icon: "🌙", shortcut: "⌘E", category: "Schedule", requiresTask: true) { if let id = $0.selectedTaskId { $0.setTimeSlot(id, "evening") } },
        Command(id: "someday", label: "Defer to Someday", icon: "📦", shortcut: "⌘O", category: "Schedule", requiresTask: true) { if let id = $0.selectedTaskId { $0.setSchedule(id, 2) } },
        Command(id: "inbox", label: "Move to Inbox", icon: "📥", shortcut: "", category: "Schedule", requiresTask: true) { if let id = $0.selectedTaskId { $0.setSchedule(id, 0) } },

        // Create
        Command(id: "new-task", label: "New Task", icon: "+", shortcut: "⌘N", category: "Create", requiresTask: false) { let t = $0.createTask(title: ""); $0.expandedTaskId = t.id },
    ]
}
