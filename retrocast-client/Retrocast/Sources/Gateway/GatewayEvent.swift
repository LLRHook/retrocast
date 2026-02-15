import Foundation

/// All dispatch event names from the server gateway.
enum GatewayEventType: String, Sendable {
    case ready = "READY"
    case messageCreate = "MESSAGE_CREATE"
    case messageUpdate = "MESSAGE_UPDATE"
    case messageDelete = "MESSAGE_DELETE"
    case guildCreate = "GUILD_CREATE"
    case guildUpdate = "GUILD_UPDATE"
    case guildDelete = "GUILD_DELETE"
    case channelCreate = "CHANNEL_CREATE"
    case channelUpdate = "CHANNEL_UPDATE"
    case channelDelete = "CHANNEL_DELETE"
    case guildMemberAdd = "GUILD_MEMBER_ADD"
    case guildMemberRemove = "GUILD_MEMBER_REMOVE"
    case guildMemberUpdate = "GUILD_MEMBER_UPDATE"
    case guildRoleCreate = "GUILD_ROLE_CREATE"
    case guildRoleUpdate = "GUILD_ROLE_UPDATE"
    case guildRoleDelete = "GUILD_ROLE_DELETE"
    case typingStart = "TYPING_START"
    case presenceUpdate = "PRESENCE_UPDATE"
    case voiceStateUpdate = "VOICE_STATE_UPDATE"
    case guildBanAdd = "GUILD_BAN_ADD"
    case guildBanRemove = "GUILD_BAN_REMOVE"
}

/// Message delete event (only contains IDs, not full message).
struct MessageDeleteData: Codable, Sendable {
    let id: Snowflake
    let channelID: Snowflake

    enum CodingKeys: String, CodingKey {
        case id
        case channelID = "channel_id"
    }
}
