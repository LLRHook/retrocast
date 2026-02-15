import Foundation

enum ChannelType: Int, Codable, Sendable {
    case text = 0
    case voice = 2
    case category = 4
}

struct Channel: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    let guildID: Snowflake
    var name: String
    let type: ChannelType
    var position: Int
    var topic: String?
    var parentID: Snowflake?

    enum CodingKeys: String, CodingKey {
        case id
        case guildID = "guild_id"
        case name
        case type
        case position
        case topic
        case parentID = "parent_id"
    }
}
