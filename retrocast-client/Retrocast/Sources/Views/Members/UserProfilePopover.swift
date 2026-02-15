import SwiftUI

struct UserProfilePopover: View {
    let member: Member
    let roles: [Role]
    @Environment(AppState.self) private var appState

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Avatar + name
            HStack(spacing: 12) {
                AvatarView(
                    name: displayName,
                    avatarHash: nil,
                    size: 48
                )
                VStack(alignment: .leading, spacing: 2) {
                    Text(displayName)
                        .font(.headline)
                        .foregroundStyle(.retroText)
                    if let status = appState.presence[member.userID] {
                        Text(status.capitalized)
                            .font(.caption)
                            .foregroundStyle(.retroMuted)
                    }
                }
            }

            Divider()

            // Roles
            if !memberRoles.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    Text("ROLES")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroMuted)
                    FlowLayout(spacing: 4) {
                        ForEach(memberRoles) { role in
                            RoleTag(name: role.name, color: role.color)
                        }
                    }
                }
            }

            // Member since
            VStack(alignment: .leading, spacing: 2) {
                Text("MEMBER SINCE")
                    .font(.caption)
                    .fontWeight(.bold)
                    .foregroundStyle(.retroMuted)
                Text(DateFormatting.dateSeparator(member.joinedAt))
                    .font(.subheadline)
                    .foregroundStyle(.retroText)
            }
        }
        .padding(16)
        .frame(width: 280)
        .background(Color.retroSidebar)
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    private var displayName: String {
        member.nickname ?? "User \(member.userID)"
    }

    private var memberRoles: [Role] {
        roles.filter { member.roles.contains($0.id) && !$0.isDefault }
             .sorted { $0.position > $1.position }
    }
}

// Simple flow layout for role tags
struct FlowLayout: Layout {
    var spacing: CGFloat = 4

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = arrange(proposal: proposal, subviews: subviews)
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = arrange(proposal: proposal, subviews: subviews)
        for (index, offset) in result.offsets.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + offset.x, y: bounds.minY + offset.y), proposal: .unspecified)
        }
    }

    private func arrange(proposal: ProposedViewSize, subviews: Subviews) -> (size: CGSize, offsets: [CGPoint]) {
        let maxWidth = proposal.width ?? .infinity
        var offsets: [CGPoint] = []
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowHeight: CGFloat = 0
        var maxX: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > maxWidth && x > 0 {
                x = 0
                y += rowHeight + spacing
                rowHeight = 0
            }
            offsets.append(CGPoint(x: x, y: y))
            rowHeight = max(rowHeight, size.height)
            x += size.width + spacing
            maxX = max(maxX, x)
        }

        return (CGSize(width: maxX, height: y + rowHeight), offsets)
    }
}
