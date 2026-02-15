import Foundation

struct DMChannel: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    let type: Int
    let recipients: [User]
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case type
        case recipients
        case createdAt = "created_at"
    }

    /// The other user in a 1-on-1 DM (first recipient).
    var recipient: User? {
        recipients.first
    }

    /// Display name for this DM conversation.
    var displayName: String {
        recipient?.displayName ?? "Unknown User"
    }
}
