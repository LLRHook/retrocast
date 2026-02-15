import SwiftUI

struct PresenceDot: View {
    let status: String
    var size: CGFloat = 10

    var body: some View {
        Circle()
            .fill(statusColor)
            .frame(width: size, height: size)
            .overlay {
                Circle()
                    .strokeBorder(Color.retroDark, lineWidth: 2)
            }
    }

    private var statusColor: Color {
        switch status {
        case "online": return .retroGreen
        case "idle": return .retroYellow
        case "dnd": return .retroRed
        default: return .retroGray
        }
    }
}
