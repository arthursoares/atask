import SwiftUI

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet(charactersIn: "#"))
        let scanner = Scanner(string: hex)
        var rgbValue: UInt64 = 0
        scanner.scanHexInt64(&rgbValue)
        self.init(
            red: Double((rgbValue & 0xFF0000) >> 16) / 255.0,
            green: Double((rgbValue & 0x00FF00) >> 8) / 255.0,
            blue: Double(rgbValue & 0x0000FF) / 255.0
        )
    }
}

// Bone theme — warm neutral palette
enum Theme {
    // Canvas
    static let canvas = Color(hex: "#f6f5f2")
    static let canvasElevated = Color(hex: "#fefefe")
    static let canvasSunken = Color(hex: "#eceae7")

    // Sidebar
    static let sidebarBg = Color(hex: "#eeece7").opacity(0.72)

    // Ink
    static let inkPrimary = Color(hex: "#222120")
    static let inkSecondary = Color(hex: "#686664")
    static let inkTertiary = Color(hex: "#a09e9a")
    static let inkQuaternary = Color(hex: "#c8c6c2")

    // Accent
    static let accent = Color(hex: "#4670a0")
    static let accentHover = Color(hex: "#3a5f8a")
    static let accentSubtle = Color(hex: "#4670a0").opacity(0.10)

    // Semantic
    static let todayStar = Color(hex: "#c88c30")
    static let todayBg = Color(hex: "#c88c30").opacity(0.08)
    static let somedayTint = Color(hex: "#8878a0")
    static let deadlineRed = Color(hex: "#c04848")
    static let deadlineBg = Color(hex: "#c04848").opacity(0.08)
    static let success = Color(hex: "#4a8860")
    static let successBg = Color(hex: "#4a8860").opacity(0.08)
    static let agentTint = Color(hex: "#7868a8")
    static let agentBg = Color(hex: "#7868a8").opacity(0.07)
}
