import SwiftUI

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let r, g, b: Double
        switch hex.count {
        case 6:
            r = Double((int >> 16) & 0xFF) / 255.0
            g = Double((int >> 8) & 0xFF) / 255.0
            b = Double(int & 0xFF) / 255.0
        default:
            r = 0; g = 0; b = 0
        }
        self.init(red: r, green: g, blue: b)
    }

    // Discord-inspired dark theme colors
    static let retroDark    = Color(hex: "1E1F22")   // Background
    static let retroSidebar = Color(hex: "2B2D31")   // Sidebar background
    static let retroChat    = Color(hex: "313338")   // Chat area background
    static let retroAccent  = Color(hex: "5865F2")   // Blurple
    static let retroGreen   = Color(hex: "23A559")   // Online
    static let retroYellow  = Color(hex: "F0B232")   // Idle
    static let retroRed     = Color(hex: "F23F43")   // DND / errors
    static let retroGray    = Color(hex: "80848E")   // Offline / muted text
    static let retroText    = Color(hex: "DBDEE1")   // Primary text
    static let retroMuted   = Color(hex: "949BA4")   // Secondary text
    static let retroInput   = Color(hex: "383A40")   // Input field background
    static let retroHover   = Color(hex: "35373C")   // Hover state
}

// Allow `.retroText` etc. in `.foregroundStyle()` and other ShapeStyle contexts
extension ShapeStyle where Self == Color {
    static var retroDark: Color    { .init(hex: "1E1F22") }
    static var retroSidebar: Color { .init(hex: "2B2D31") }
    static var retroChat: Color    { .init(hex: "313338") }
    static var retroAccent: Color  { .init(hex: "5865F2") }
    static var retroGreen: Color   { .init(hex: "23A559") }
    static var retroYellow: Color  { .init(hex: "F0B232") }
    static var retroRed: Color     { .init(hex: "F23F43") }
    static var retroGray: Color    { .init(hex: "80848E") }
    static var retroText: Color    { .init(hex: "DBDEE1") }
    static var retroMuted: Color   { .init(hex: "949BA4") }
    static var retroInput: Color   { .init(hex: "383A40") }
    static var retroHover: Color   { .init(hex: "35373C") }
}
