import Foundation

struct User: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    let username: String
    var displayName: String
    var avatarHash: String?
    let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case id
        case username
        case displayName = "display_name"
        case avatarHash = "avatar_hash"
        case createdAt = "created_at"
    }
}
