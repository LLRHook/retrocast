import Foundation

struct Guild: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    var name: String
    var iconHash: String?
    let ownerID: Snowflake
    let createdAt: Date?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case iconHash = "icon_hash"
        case ownerID = "owner_id"
        case createdAt = "created_at"
    }
}
