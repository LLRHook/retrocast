import Foundation

struct Message: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    let channelID: Snowflake
    let authorID: Snowflake
    var content: String
    let createdAt: Date
    var editedAt: Date?

    // Joined author fields from server
    let authorUsername: String?
    let authorDisplayName: String?
    let authorAvatarHash: String?

    enum CodingKeys: String, CodingKey {
        case id
        case channelID = "channel_id"
        case authorID = "author_id"
        case content
        case createdAt = "created_at"
        case editedAt = "edited_at"
        case authorUsername = "author_username"
        case authorDisplayName = "author_display_name"
        case authorAvatarHash = "author_avatar_hash"
    }

    /// Display name: prefer display_name, fall back to username.
    var displayName: String {
        authorDisplayName ?? authorUsername ?? "Unknown"
    }
}
