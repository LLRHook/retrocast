import SwiftUI

struct RoleTag: View {
    let name: String
    let color: Int

    var body: some View {
        Text(name)
            .font(.caption2)
            .fontWeight(.medium)
            .foregroundStyle(roleColor)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(roleColor.opacity(0.15))
            .clipShape(Capsule())
    }

    private var roleColor: Color {
        if color == 0 { return .retroMuted }
        let r = Double((color >> 16) & 0xFF) / 255.0
        let g = Double((color >> 8) & 0xFF) / 255.0
        let b = Double(color & 0xFF) / 255.0
        return Color(red: r, green: g, blue: b)
    }
}
