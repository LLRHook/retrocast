import Foundation

struct Attachment: Codable, Identifiable, Hashable, Sendable {
    let id: Snowflake
    let messageID: Snowflake
    let filename: String
    let contentType: String
    let size: Int64
    let url: String

    enum CodingKeys: String, CodingKey {
        case id
        case messageID = "message_id"
        case filename
        case contentType = "content_type"
        case size
        case url
    }
}
