import SwiftUI

struct TypingIndicatorView: View {
    let usernames: [String]

    var body: some View {
        if !usernames.isEmpty {
            HStack(spacing: 4) {
                BouncingDotsView()
                Text(typingText)
                    .font(.caption)
                    .foregroundStyle(.retroMuted)
                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 4)
            .transition(.opacity.combined(with: .move(edge: .bottom)))
        }
    }

    private var typingText: String {
        switch usernames.count {
        case 1: return "\(usernames[0]) is typing..."
        case 2: return "\(usernames[0]) and \(usernames[1]) are typing..."
        case 3: return "\(usernames[0]), \(usernames[1]), and \(usernames[2]) are typing..."
        default: return "Several people are typing..."
        }
    }
}

struct BouncingDotsView: View {
    @State private var animate = false

    var body: some View {
        HStack(spacing: 2) {
            ForEach(0..<3) { index in
                Circle()
                    .fill(Color.retroMuted)
                    .frame(width: 4, height: 4)
                    .offset(y: animate ? -3 : 0)
                    .animation(
                        .easeInOut(duration: 0.4)
                        .repeatForever(autoreverses: true)
                        .delay(Double(index) * 0.15),
                        value: animate
                    )
            }
        }
        .onAppear { animate = true }
    }
}
