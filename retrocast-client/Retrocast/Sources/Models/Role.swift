import Foundation

struct Role: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    let guildID: Snowflake
    var name: String
    var color: Int
    var permissions: Int64
    var position: Int
    var isDefault: Bool

    enum CodingKeys: String, CodingKey {
        case id
        case guildID = "guild_id"
        case name
        case color
        case permissions
        case position
        case isDefault = "is_default"
    }
}
