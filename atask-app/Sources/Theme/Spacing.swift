import Foundation

/// Spacing tokens from MEASUREMENTS.md (4px base)
enum Spacing {
    static let sp1: CGFloat = 4
    static let sp2: CGFloat = 8
    static let sp3: CGFloat = 12
    static let sp4: CGFloat = 16
    static let sp5: CGFloat = 20
    static let sp6: CGFloat = 24
    static let sp8: CGFloat = 32
    static let sp10: CGFloat = 40
    static let sp12: CGFloat = 48
    static let sp20: CGFloat = 80

    // Named layout constants
    static let sidebarWidth: CGFloat = 240
    static let detailWidth: CGFloat = 340
    static let toolbarHeight: CGFloat = 52
    static let taskRowHeight: CGFloat = 32
    static let checkboxSize: CGFloat = 20
    static let checkboxBorder: CGFloat = 1.5
    static let checklistCheckSize: CGFloat = 16
    static let sidebarDot: CGFloat = 8
    static let metaDot: CGFloat = 6
    static let toolbarBtnSize: CGFloat = 30
    static let activityAvatar: CGFloat = 24
    static let progressBarWidth: CGFloat = 80
    static let progressBarHeight: CGFloat = 4
    static let whenPopoverWidth: CGFloat = 260
    static let cmdPaletteWidth: CGFloat = 560
    static let inboxActionBtn: CGFloat = 26
    static let emptyStateIcon: CGFloat = 48
    static let attrBarLeftPad: CGFloat = 27  // checkbox 20 + ~7 gap
}

/// Border radius tokens from MEASUREMENTS.md
enum Radius {
    static let xs: CGFloat = 4
    static let sm: CGFloat = 6
    static let md: CGFloat = 8
    static let lg: CGFloat = 12
    static let xl: CGFloat = 16
    static let full: CGFloat = 9999
}
