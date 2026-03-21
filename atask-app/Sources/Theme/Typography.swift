import SwiftUI

extension Font {
    /// Atkinson Hyperlegible with size and weight
    static func atkinson(_ size: CGFloat, weight: Font.Weight = .regular) -> Font {
        weight == .bold
            ? .custom("AtkinsonHyperlegible-Bold", size: size)
            : .custom("AtkinsonHyperlegible-Regular", size: size)
    }

    // Named presets from MEASUREMENTS.md
    static let viewTitle       = atkinson(20, weight: .bold)   // "Today", "Inbox"
    static let sectionHeader   = atkinson(17, weight: .bold)   // section dividers
    static let inlineTitle     = atkinson(16, weight: .bold)   // inline editor title
    static let taskTitle       = atkinson(14)                  // row text
    static let detailBody      = atkinson(15)                  // detail panel notes
    static let inlineNotes     = atkinson(13)                  // inline editor notes
    static let metadata        = atkinson(12, weight: .bold)   // badges, timestamps, pills
    static let metadataRegular = atkinson(12)                  // field values
    static let groupLabel      = atkinson(11, weight: .bold)   // sidebar area headers
    static let tagPill         = atkinson(11, weight: .bold)   // tag pills, attr pills
}
