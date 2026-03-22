import SwiftUI

/// Tag picker popover — searchable list of tags + create new.
struct TagPicker: View {
    @Bindable var store: TaskStore
    let taskId: String
    @Binding var isPresented: Bool
    @State private var search = ""
    @FocusState private var focused: Bool

    private var filteredTags: [TagModel] {
        if search.isEmpty { return store.tags }
        return store.tags.filter { $0.title.localizedCaseInsensitiveContains(search) }
    }

    private var taskTagIds: Set<String> {
        Set(store.tagsForTask(taskId).map(\.id))
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Search
            HStack(spacing: 6) {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: 11))
                    .foregroundStyle(Theme.inkTertiary)
                TextField("Search or create tag...", text: $search)
                    .font(.metadataRegular)
                    .textFieldStyle(.plain)
                    .focused($focused)
                    .onSubmit { createOrSelect() }
            }
            .padding(8)

            Divider()

            // Tag list
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    ForEach(filteredTags) { tag in
                        Button {
                            toggleTag(tag)
                        } label: {
                            HStack(spacing: 8) {
                                Image(systemName: taskTagIds.contains(tag.id) ? "checkmark.circle.fill" : "circle")
                                    .font(.system(size: 12))
                                    .foregroundStyle(taskTagIds.contains(tag.id) ? Theme.accent : Theme.inkTertiary)
                                Text(tag.title)
                                    .font(.metadataRegular)
                                    .foregroundStyle(Theme.inkPrimary)
                                Spacer()
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                    }

                    // Create new tag option
                    if !search.isEmpty && !store.tags.contains(where: { $0.title.lowercased() == search.lowercased() }) {
                        Button { createAndAssign() } label: {
                            HStack(spacing: 8) {
                                Image(systemName: "plus.circle")
                                    .font(.system(size: 12))
                                    .foregroundStyle(Theme.accent)
                                Text("Create \"\(search)\"")
                                    .font(.metadataRegular)
                                    .foregroundStyle(Theme.accent)
                                Spacer()
                            }
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.vertical, 4)
            }
            .frame(maxHeight: 200)
        }
        .frame(width: 220)
        .onAppear { focused = true }
    }

    private func toggleTag(_ tag: TagModel) {
        if taskTagIds.contains(tag.id) {
            store.removeTagFromTask(taskId, tagId: tag.id)
        } else {
            store.addTagToTask(taskId, tagId: tag.id)
        }
    }

    private func createOrSelect() {
        guard !search.isEmpty else { return }
        if let existing = store.tags.first(where: { $0.title.lowercased() == search.lowercased() }) {
            toggleTag(existing)
        } else {
            createAndAssign()
        }
    }

    private func createAndAssign() {
        guard !search.isEmpty else { return }
        if let tag = store.createTag(title: search) {
            toggleTag(tag)
        }
    }
}
