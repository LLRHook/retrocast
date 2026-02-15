import Foundation

struct Member: Codable, Identifiable, Hashable, Sendable {
    let guildID: Snowflake
    let userID: Snowflake
    var nickname: String?
    let joinedAt: Date
    var roles: [Snowflake]

    var id: Snowflake { userID }

    enum CodingKeys: String, CodingKey {
        case guildID = "guild_id"
        case userID = "user_id"
        case nickname
        case joinedAt = "joined_at"
        case roles
    }
}
