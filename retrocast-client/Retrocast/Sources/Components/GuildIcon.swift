import SwiftUI

struct GuildIcon: View {
    let guild: Guild
    let isSelected: Bool
    var size: CGFloat = 48

    var body: some View {
        ZStack {
            if let icon = guild.iconHash, !icon.isEmpty {
                // TODO: Load icon image from server
                abbreviationView
            } else {
                abbreviationView
            }
        }
        .frame(width: size, height: size)
        .clipShape(isSelected ? AnyShape(RoundedRectangle(cornerRadius: 16)) : AnyShape(Circle()))
        .animation(.easeInOut(duration: 0.15), value: isSelected)
    }

    private var abbreviationView: some View {
        ZStack {
            Color.retroSidebar
            Text(abbreviation)
                .font(.system(size: size * 0.35, weight: .semibold))
                .foregroundStyle(.retroText)
        }
    }

    private var abbreviation: String {
        let words = guild.name.split(separator: " ")
        if words.count >= 2 {
            return String(words.prefix(2).map { $0.prefix(1) }.joined()).uppercased()
        }
        return String(guild.name.prefix(2)).uppercased()
    }
}
