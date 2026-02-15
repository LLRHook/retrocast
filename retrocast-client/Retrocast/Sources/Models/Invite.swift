import Foundation

struct Invite: Codable, Identifiable, Hashable, Sendable {
    let code: String
    let guildID: Snowflake
    let creatorID: Snowflake
    var maxUses: Int
    var uses: Int
    var expiresAt: Date?
    let createdAt: Date?

    var id: String { code }

    enum CodingKeys: String, CodingKey {
        case code
        case guildID = "guild_id"
        case creatorID = "creator_id"
        case maxUses = "max_uses"
        case uses
        case expiresAt = "expires_at"
        case createdAt = "created_at"
    }
}

/// Public invite info returned by GET /invites/:code (no auth).
struct InviteInfo: Codable, Sendable {
    let code: String
    let guildName: String
    let memberCount: Int
    let creatorID: Snowflake

    enum CodingKeys: String, CodingKey {
        case code
        case guildName = "guild_name"
        case memberCount = "member_count"
        case creatorID = "creator_id"
    }
}
