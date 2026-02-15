import SwiftUI

struct AvatarView: View {
    let name: String
    let avatarHash: String?
    var size: CGFloat = 32

    var body: some View {
        if let hash = avatarHash, !hash.isEmpty {
            // TODO: Load avatar image from server
            initialsView
        } else {
            initialsView
        }
    }

    private var initialsView: some View {
        ZStack {
            Circle()
                .fill(avatarColor)
            Text(initials)
                .font(.system(size: size * 0.4, weight: .semibold))
                .foregroundStyle(.white)
        }
        .frame(width: size, height: size)
    }

    private var initials: String {
        let parts = name.split(separator: " ")
        if parts.count >= 2 {
            return String(parts[0].prefix(1) + parts[1].prefix(1)).uppercased()
        }
        return String(name.prefix(2)).uppercased()
    }

    /// Deterministic color based on name.
    private var avatarColor: Color {
        let colors: [Color] = [.retroAccent, .retroGreen, .retroYellow, .retroRed, .purple, .orange, .teal, .pink]
        let hash = abs(name.hashValue)
        return colors[hash % colors.count]
    }
}
