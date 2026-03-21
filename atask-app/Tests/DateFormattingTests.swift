import Testing
import Foundation
@testable import atask

@Test func formatToday() {
    #expect(DateFormatting.formatRelative(DateFormatting.todayString()) == "Today")
}

@Test func formatTomorrow() {
    let tomorrow = Calendar.current.date(byAdding: .day, value: 1, to: Date())!
    let fmt = DateFormatter()
    fmt.dateFormat = "yyyy-MM-dd"
    #expect(DateFormatting.formatRelative(fmt.string(from: tomorrow)) == "Tomorrow")
}

@Test func formatYesterday() {
    let yesterday = Calendar.current.date(byAdding: .day, value: -1, to: Date())!
    let fmt = DateFormatter()
    fmt.dateFormat = "yyyy-MM-dd"
    #expect(DateFormatting.formatRelative(fmt.string(from: yesterday)) == "Yesterday")
}

@Test func deadlineToday() {
    let (label, variant) = DateFormatting.formatDeadline(DateFormatting.todayString())
    #expect(label == "Due Today")
    #expect(variant == .today)
}

@Test func deadlineOverdue() {
    let past = Calendar.current.date(byAdding: .day, value: -3, to: Date())!
    let fmt = DateFormatter()
    fmt.dateFormat = "yyyy-MM-dd"
    let (label, variant) = DateFormatting.formatDeadline(fmt.string(from: past))
    #expect(label.starts(with: "Overdue"))
    #expect(variant == .overdue)
}

@Test func isoDateTimeParsing() {
    let iso = "\(DateFormatting.todayString())T00:00:00Z"
    #expect(DateFormatting.formatRelative(iso) == "Today")
}

@Test func isoWithFractionalSeconds() {
    let iso = "\(DateFormatting.todayString())T19:31:29.028844Z"
    #expect(DateFormatting.formatRelative(iso) == "Today")
}

@Test func invalidDate() {
    #expect(DateFormatting.formatRelative("not-a-date") == "not-a-date")
}
