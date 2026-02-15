import Foundation

enum DateFormatting {
    /// "2:42 PM" — time only
    private static let timeFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateStyle = .none
        f.timeStyle = .short
        return f
    }()

    /// "January 15, 2025"
    private static let fullDateFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "MMMM d, yyyy"
        return f
    }()

    /// "01/15/2025 2:42 PM"
    private static let dateTimeFormatter: DateFormatter = {
        let f = DateFormatter()
        f.dateStyle = .short
        f.timeStyle = .short
        return f
    }()

    /// Relative timestamp for messages: "Today at 2:42 PM", "Yesterday at 10:15 AM", "01/13/2025 2:42 PM"
    static func messageTimestamp(_ date: Date) -> String {
        let calendar = Calendar.current
        if calendar.isDateInToday(date) {
            return "Today at \(timeFormatter.string(from: date))"
        } else if calendar.isDateInYesterday(date) {
            return "Yesterday at \(timeFormatter.string(from: date))"
        } else {
            return dateTimeFormatter.string(from: date)
        }
    }

    /// Short time: "2:42 PM"
    static func shortTime(_ date: Date) -> String {
        timeFormatter.string(from: date)
    }

    /// Date separator: "January 15, 2025"
    static func dateSeparator(_ date: Date) -> String {
        fullDateFormatter.string(from: date)
    }

    /// Check if two dates are on the same calendar day.
    static func isSameDay(_ a: Date, _ b: Date) -> Bool {
        Calendar.current.isDate(a, inSameDayAs: b)
    }
}
