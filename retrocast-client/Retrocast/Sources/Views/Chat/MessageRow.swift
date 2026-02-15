import SwiftUI

struct MessageRow: View {
    let message: Message
    let isGrouped: Bool
    let currentUserID: Snowflake?
    var onEdit: ((String) -> Void)?
    var onDelete: (() -> Void)?

    @State private var showActions = false
    @State private var showEditAlert = false
    @State private var editText = ""

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
                if !message.content.isEmpty {
                    Text(message.content)
                        .font(.body)
                        .foregroundStyle(.retroText)
                        .textSelection(.enabled)
                }

                // Attachments
                if let attachments = message.attachments, !attachments.isEmpty {
                    ForEach(attachments) { attachment in
                        if attachment.contentType.hasPrefix("image/") {
                            AsyncImage(url: URL(string: attachment.url)) { phase in
                                switch phase {
                                case .success(let image):
                                    image
                                        .resizable()
                                        .aspectRatio(contentMode: .fit)
                                        .frame(maxWidth: 300, maxHeight: 300)
                                        .clipShape(RoundedRectangle(cornerRadius: 4))
                                case .failure:
                                    Label(attachment.filename, systemImage: "photo")
                                        .font(.caption)
                                        .foregroundStyle(.retroMuted)
                                default:
                                    ProgressView()
                                        .frame(width: 100, height: 100)
                                }
                            }
                        } else {
                            HStack(spacing: 8) {
                                Image(systemName: "doc")
                                    .foregroundStyle(.retroMuted)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(attachment.filename)
                                        .font(.subheadline)
                                        .foregroundStyle(.retroText)
                                    Text(formatFileSize(attachment.size))
                                        .font(.caption)
                                        .foregroundStyle(.retroMuted)
                                }
                            }
                            .padding(8)
                            .background(Color.retroDark.opacity(0.5))
                            .clipShape(RoundedRectangle(cornerRadius: 4))
                        }
                    }
                }
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
                        editText = message.content
                        showEditAlert = true
                    }
                    Button("Delete Message", role: .destructive) {
                        onDelete?()
                    }
                }
                Button("Copy Text") {
                    UIPasteboard.general.string = message.content
                }
            }
        }
        .alert("Edit Message", isPresented: $showEditAlert) {
            TextField("Message", text: $editText)
            Button("Save") {
                let trimmed = editText.trimmingCharacters(in: .whitespacesAndNewlines)
                if !trimmed.isEmpty, trimmed != message.content {
                    onEdit?(trimmed)
                }
            }
            Button("Cancel", role: .cancel) {}
        }
    }

    private func formatFileSize(_ bytes: Int64) -> String {
        let formatter = ByteCountFormatter()
        formatter.countStyle = .file
        return formatter.string(fromByteCount: bytes)
    }
}
