import SwiftUI

struct MemberRow: View {
    let member: Member
    @Environment(AppState.self) private var appState

    var body: some View {
        HStack(spacing: 8) {
            ZStack(alignment: .bottomTrailing) {
                AvatarView(
                    name: displayName,
                    avatarHash: nil,
                    size: 28
                )
                if let status = appState.presence[member.userID] {
                    PresenceDot(status: status, size: 8)
                        .offset(x: 2, y: 2)
                }
            }

            Text(displayName)
                .font(.subheadline)
                .foregroundStyle(isOnline ? .retroText : .retroMuted)
                .lineLimit(1)

            Spacer()
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .clipShape(RoundedRectangle(cornerRadius: 4))
    }

    private var displayName: String {
        member.nickname ?? "User \(member.userID)"
    }

    private var isOnline: Bool {
        let status = appState.presence[member.userID] ?? "offline"
        return status != "offline"
    }
}
