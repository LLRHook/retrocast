import Foundation

/// Lightweight payload for sending messages to the gateway.
struct GatewaySendPayload<D: Encodable>: Encodable {
    let op: Int
    let d: D?
}

/// Op codes matching the server gateway protocol.
enum GatewayOpCode: Int, Codable, Sendable {
    case dispatch = 0
    case heartbeat = 1
    case identify = 2
    case presenceUpdate = 3
    case voiceStateUpdate = 4
    case resume = 6
    case reconnect = 7
    case hello = 10
    case heartbeatAck = 11
}

/// The envelope for all gateway messages.
struct GatewayPayload: Codable, Sendable {
    let op: Int
    let d: AnyCodable?
    let s: Int64?
    let t: String?

    init(op: GatewayOpCode, data: AnyCodable? = nil) {
        self.op = op.rawValue
        self.d = data
        self.s = nil
        self.t = nil
    }
}

/// HELLO payload from server.
struct HelloData: Codable, Sendable {
    let heartbeatInterval: Int

    enum CodingKeys: String, CodingKey {
        case heartbeatInterval = "heartbeat_interval"
    }
}

/// IDENTIFY payload sent to server.
struct IdentifyData: Codable, Sendable {
    let token: String
}

/// RESUME payload sent to server.
struct ResumeData: Codable, Sendable {
    let token: String
    let sessionID: String
    let seq: Int64

    enum CodingKeys: String, CodingKey {
        case token
        case sessionID = "session_id"
        case seq
    }
}

/// READY payload from server.
struct ReadyData: Codable, Sendable {
    let sessionID: String
    let userID: Snowflake
    let guilds: [Snowflake]

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case userID = "user_id"
        case guilds
    }
}

/// Typing start event data.
struct TypingStartData: Codable, Sendable {
    let channelID: Snowflake
    let guildID: Snowflake
    let userID: Snowflake
    let timestamp: Int64

    enum CodingKeys: String, CodingKey {
        case channelID = "channel_id"
        case guildID = "guild_id"
        case userID = "user_id"
        case timestamp
    }
}

/// Presence update event data.
struct PresenceUpdateData: Codable, Sendable {
    let userID: Snowflake
    let status: String

    enum CodingKeys: String, CodingKey {
        case userID = "user_id"
        case status
    }
}

// MARK: - AnyCodable (lightweight type-erased wrapper)

struct AnyCodable: Codable, Sendable {
    let value: any Sendable

    init(_ value: any Sendable) {
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let int = try? container.decode(Int64.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else {
            value = NSNull()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch value {
        case let v as String: try container.encode(v)
        case let v as Int: try container.encode(v)
        case let v as Int64: try container.encode(v)
        case let v as Double: try container.encode(v)
        case let v as Bool: try container.encode(v)
        case let v as [String: AnyCodable]: try container.encode(v)
        case let v as [AnyCodable]: try container.encode(v)
        case let v as any Encodable: try v.encode(to: encoder)
        default: try container.encodeNil()
        }
    }
}
