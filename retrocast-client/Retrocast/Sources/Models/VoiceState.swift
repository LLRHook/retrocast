import Foundation

struct VoiceState: Codable, Identifiable, Hashable, Sendable {
    let guildID: Snowflake
    let channelID: Snowflake
    let userID: Snowflake
    let sessionID: String
    var selfMute: Bool
    var selfDeaf: Bool
    let joinedAt: Date

    var id: Snowflake { userID }

    enum CodingKeys: String, CodingKey {
        case guildID = "guild_id"
        case channelID = "channel_id"
        case userID = "user_id"
        case sessionID = "session_id"
        case selfMute = "self_mute"
        case selfDeaf = "self_deaf"
        case joinedAt = "joined_at"
    }
}

struct JoinVoiceResponse: Codable, Sendable {
    let token: String
    let voiceStates: [VoiceState]

    enum CodingKeys: String, CodingKey {
        case token
        case voiceStates = "voice_states"
    }
}
