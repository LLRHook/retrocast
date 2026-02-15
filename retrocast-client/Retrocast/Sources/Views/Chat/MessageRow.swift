import SwiftUI

struct MessageRow: View {
    let message: Message
    let isGrouped: Bool
    let currentUserID: Snowflake?

    @State private var showActions = false

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            if isGrouped {
                // Indent to align with content (avatar width + spacing)
                Color.clear
                    .frame(width: 32, height: 1)
            } else {
                AvatarView(
                    name: message.displayName,
                    avatarHash: message.authorAvatarHash,
                    size: 32
                )
                .padding(.top, 2)
            }

            VStack(alignment: .leading, spacing: 2) {
                if !isGrouped {
                    // Author name + timestamp
                    HStack(spacing: 8) {
                        Text(message.displayName)
                            .font(.subheadline)
                            .fontWeight(.semibold)
                            .foregroundStyle(.retroText)
                        Text(DateFormatting.messageTimestamp(message.createdAt))
                            .font(.caption)
                            .foregroundStyle(.retroMuted)
                        if message.editedAt != nil {
                            Text("(edited)")
                                .font(.caption2)
                                .foregroundStyle(.retroMuted)
                        }
                    }
                }

                // Message content
                Text(message.content)
                    .font(.body)
                    .foregroundStyle(.retroText)
                    .textSelection(.enabled)
            }

            Spacer(minLength: 0)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, isGrouped ? 1 : 4)
        .contentShape(Rectangle())
        .contextMenu {
            if let userID = currentUserID {
                if message.authorID == userID {
                    Button("Edit Message") {
                        // TODO: implement inline editing
                    }
                    Button("Delete Message", role: .destructive) {
                        // TODO: implement delete
                    }
                }
                Button("Copy Text") {
                    UIPasteboard.general.string = message.content
                }
            }
        }
    }
}
