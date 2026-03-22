import SwiftUI

/// When picker popover — schedule + start date + deadline.
/// Quick options (Today, This Evening, Someday) plus date pickers.
struct WhenPicker: View {
    @Bindable var store: TaskStore
    let taskId: String
    @Binding var isPresented: Bool

    @State private var startDate: Date?
    @State private var deadline: Date?
    @State private var showStartPicker = false
    @State private var showDeadlinePicker = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Quick schedule options
            quickOption("Today", icon: "star.fill", color: Theme.todayStar) {
                store.setSchedule(taskId, 1)
                isPresented = false
            }
            quickOption("This Evening", icon: "moon.fill", color: Theme.accent) {
                store.setSchedule(taskId, 1)
                store.setTimeSlot(taskId, "evening")
                isPresented = false
            }
            quickOption("Someday", icon: "clock", color: Theme.somedayTint) {
                store.setSchedule(taskId, 2)
                isPresented = false
            }

            Divider().padding(.vertical, 4)

            // Start Date
            VStack(alignment: .leading, spacing: 4) {
                Text("START DATE")
                    .font(.groupLabel)
                    .foregroundStyle(Theme.inkTertiary)
                    .tracking(0.5)

                if showStartPicker {
                    DatePicker("", selection: Binding(
                        get: { startDate ?? Date() },
                        set: { date in
                            startDate = date
                            let fmt = DateFormatter()
                            fmt.dateFormat = "yyyy-MM-dd"
                            store.setStartDate(taskId, fmt.string(from: date))
                        }
                    ), displayedComponents: .date)
                    .datePickerStyle(.graphical)
                    .frame(maxWidth: 260)
                } else {
                    Button {
                        showStartPicker = true
                    } label: {
                        HStack {
                            let task = store.tasks.first { $0.id == taskId }
                            Text(task?.startDate ?? "None")
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkSecondary)
                            Spacer()
                            if task?.startDate != nil {
                                Button {
                                    store.setStartDate(taskId, nil)
                                    startDate = nil
                                } label: {
                                    Image(systemName: "xmark.circle.fill")
                                        .font(.system(size: 12))
                                        .foregroundStyle(Theme.inkTertiary)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.vertical, 4)

            Divider().padding(.vertical, 4)

            // Deadline
            VStack(alignment: .leading, spacing: 4) {
                Text("DEADLINE")
                    .font(.groupLabel)
                    .foregroundStyle(Theme.inkTertiary)
                    .tracking(0.5)

                if showDeadlinePicker {
                    DatePicker("", selection: Binding(
                        get: { deadline ?? Date() },
                        set: { date in
                            deadline = date
                            let fmt = DateFormatter()
                            fmt.dateFormat = "yyyy-MM-dd"
                            store.setDeadline(taskId, fmt.string(from: date))
                        }
                    ), displayedComponents: .date)
                    .datePickerStyle(.graphical)
                    .frame(maxWidth: 260)
                } else {
                    Button {
                        showDeadlinePicker = true
                    } label: {
                        HStack {
                            let task = store.tasks.first { $0.id == taskId }
                            Text(task?.deadline ?? "None")
                                .font(.metadataRegular)
                                .foregroundStyle(Theme.inkSecondary)
                            Spacer()
                            if task?.deadline != nil {
                                Button {
                                    store.setDeadline(taskId, nil)
                                    deadline = nil
                                } label: {
                                    Image(systemName: "xmark.circle.fill")
                                        .font(.system(size: 12))
                                        .foregroundStyle(Theme.inkTertiary)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    .buttonStyle(.plain)
                }
            }
            .padding(.vertical, 4)

            Divider().padding(.vertical, 4)

            // Clear all
            Button {
                store.setSchedule(taskId, 0)
                store.setStartDate(taskId, nil)
                store.setDeadline(taskId, nil)
                store.setTimeSlot(taskId, nil)
                isPresented = false
            } label: {
                HStack {
                    Image(systemName: "xmark")
                        .font(.system(size: 11))
                    Text("Clear")
                        .font(.metadataRegular)
                }
                .foregroundStyle(Theme.inkTertiary)
            }
            .buttonStyle(.plain)
        }
        .padding(12)
        .frame(width: 280)
    }

    private func quickOption(_ label: String, icon: String, color: Color, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Image(systemName: icon)
                    .foregroundStyle(color)
                    .frame(width: 16)
                Text(label)
                    .font(.metadataRegular)
                    .foregroundStyle(Theme.inkPrimary)
                Spacer()
            }
            .padding(.vertical, 4)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
