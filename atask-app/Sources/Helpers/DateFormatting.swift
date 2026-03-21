import Foundation

enum DateFormatting {
    private static let dayFormatter: DateFormatter = {
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd"
        return fmt
    }()

    static func todayString() -> String {
        dayFormatter.string(from: Date())
    }

    static func formatRelative(_ dateStr: String) -> String {
        guard let date = parseDate(dateStr) else { return dateStr }
        let cal = Calendar.current
        let today = cal.startOfDay(for: Date())
        let target = cal.startOfDay(for: date)
        let days = cal.dateComponents([.day], from: today, to: target).day ?? 0

        switch days {
        case 0: return "Today"
        case 1: return "Tomorrow"
        case -1: return "Yesterday"
        case 2...6:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE"
            return fmt.string(from: date)
        case -6...(-2):
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE"
            return "Last \(fmt.string(from: date))"
        default:
            let fmt = DateFormatter()
            if cal.component(.year, from: date) == cal.component(.year, from: Date()) {
                fmt.dateFormat = "MMM d"
            } else {
                fmt.dateFormat = "MMM d, yyyy"
            }
            return fmt.string(from: date)
        }
    }

    static func formatDeadline(_ dateStr: String) -> (String, DeadlineVariant) {
        guard let date = parseDate(dateStr) else { return (dateStr, .normal) }
        let cal = Calendar.current
        let today = cal.startOfDay(for: Date())
        let target = cal.startOfDay(for: date)
        let days = cal.dateComponents([.day], from: today, to: target).day ?? 0

        switch days {
        case ..<0:
            let fmt = DateFormatter()
            fmt.dateFormat = "MMM d"
            return ("Overdue · \(fmt.string(from: date))", .overdue)
        case 0:
            return ("Due Today", .today)
        case 1:
            return ("Due Tomorrow", .normal)
        case 2...6:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE"
            return ("Due \(fmt.string(from: date))", .normal)
        default:
            let fmt = DateFormatter()
            if cal.component(.year, from: date) == cal.component(.year, from: Date()) {
                fmt.dateFormat = "MMM d"
            } else {
                fmt.dateFormat = "MMM d, yyyy"
            }
            return ("Due \(fmt.string(from: date))", .normal)
        }
    }

    enum DeadlineVariant {
        case normal, today, overdue
    }

    static func formatSectionDate(_ dateStr: String) -> String {
        guard let date = parseDate(dateStr) else { return dateStr }
        let cal = Calendar.current
        let today = cal.startOfDay(for: Date())
        let target = cal.startOfDay(for: date)
        let days = cal.dateComponents([.day], from: today, to: target).day ?? 0

        switch days {
        case 0:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEE, MMM d"
            return "Today — \(fmt.string(from: date))"
        case 1:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEE, MMM d"
            return "Tomorrow — \(fmt.string(from: date))"
        case -1:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEE, MMM d"
            return "Yesterday — \(fmt.string(from: date))"
        case 2...6:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEEE, MMM d"
            return fmt.string(from: date)
        case 7...13:
            let fmt = DateFormatter()
            fmt.dateFormat = "EEE, MMM d"
            return "Next Week — \(fmt.string(from: date))"
        default:
            let fmt = DateFormatter()
            if cal.component(.year, from: date) == cal.component(.year, from: Date()) {
                fmt.dateFormat = "MMM d"
            } else {
                fmt.dateFormat = "MMM d, yyyy"
            }
            return fmt.string(from: date)
        }
    }

    // MARK: - Parsing

    private static func parseDate(_ str: String) -> Date? {
        // YYYY-MM-DD
        if let d = dayFormatter.date(from: str) { return d }
        // ISO8601 with fractional seconds (Go API format)
        let iso1 = ISO8601DateFormatter()
        iso1.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let d = iso1.date(from: str) { return d }
        // ISO8601 without fractional
        let iso2 = ISO8601DateFormatter()
        iso2.formatOptions = [.withInternetDateTime]
        if let d = iso2.date(from: str) { return d }
        return nil
    }
}
