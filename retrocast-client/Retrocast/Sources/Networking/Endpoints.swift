import Foundation

enum HTTPMethod: String, Sendable {
    case GET, POST, PATCH, PUT, DELETE
}

struct Endpoint: Sendable {
    let method: HTTPMethod
    let path: String
    let body: (any Encodable & Sendable)?
    let queryItems: [URLQueryItem]?
    let requiresAuth: Bool

    init(method: HTTPMethod, path: String, body: (any Encodable & Sendable)? = nil, queryItems: [URLQueryItem]? = nil, requiresAuth: Bool = true) {
        self.method = method
        self.path = path
        self.body = body
        self.queryItems = queryItems
        self.requiresAuth = requiresAuth
    }
}

// MARK: - Auth

extension Endpoint {
    static func register(username: String, password: String, displayName: String) -> Endpoint {
        struct Body: Encodable, Sendable { let username, password, display_name: String }
        return Endpoint(method: .POST, path: "/api/v1/auth/register",
                        body: Body(username: username, password: password, display_name: displayName),
                        requiresAuth: false)
    }

    static func login(username: String, password: String) -> Endpoint {
        struct Body: Encodable, Sendable { let username, password: String }
        return Endpoint(method: .POST, path: "/api/v1/auth/login",
                        body: Body(username: username, password: password),
                        requiresAuth: false)
    }

    static func refresh(refreshToken: String) -> Endpoint {
        struct Body: Encodable, Sendable { let refresh_token: String }
        return Endpoint(method: .POST, path: "/api/v1/auth/refresh",
                        body: Body(refresh_token: refreshToken),
                        requiresAuth: false)
    }

    static func logout() -> Endpoint {
        Endpoint(method: .POST, path: "/api/v1/auth/logout")
    }
}

// MARK: - Users

extension Endpoint {
    static func getMe() -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/users/@me")
    }

    static func updateMe(displayName: String? = nil, avatarHash: String? = nil) -> Endpoint {
        struct Body: Encodable, Sendable { let display_name: String?; let avatar_hash: String? }
        return Endpoint(method: .PATCH, path: "/api/v1/users/@me",
                        body: Body(display_name: displayName, avatar_hash: avatarHash))
    }

    static func getMyGuilds() -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/users/@me/guilds")
    }
}

// MARK: - Guilds

extension Endpoint {
    static func createGuild(name: String) -> Endpoint {
        struct Body: Encodable, Sendable { let name: String }
        return Endpoint(method: .POST, path: "/api/v1/guilds", body: Body(name: name))
    }

    static func getGuild(id: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/guilds/\(id)")
    }

    static func updateGuild(id: Snowflake, name: String? = nil) -> Endpoint {
        struct Body: Encodable, Sendable { let name: String? }
        return Endpoint(method: .PATCH, path: "/api/v1/guilds/\(id)", body: Body(name: name))
    }

    static func deleteGuild(id: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/guilds/\(id)")
    }
}

// MARK: - Channels

extension Endpoint {
    static func getChannels(guildID: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/guilds/\(guildID)/channels")
    }

    static func createChannel(guildID: Snowflake, name: String, type: Int = 0, parentID: Snowflake? = nil) -> Endpoint {
        struct Body: Encodable, Sendable { let name: String; let type: Int; let parent_id: String? }
        return Endpoint(method: .POST, path: "/api/v1/guilds/\(guildID)/channels",
                        body: Body(name: name, type: type, parent_id: parentID?.description))
    }

    static func getChannel(id: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/channels/\(id)")
    }

    static func updateChannel(id: Snowflake, name: String? = nil, topic: String? = nil) -> Endpoint {
        struct Body: Encodable, Sendable { let name: String?; let topic: String? }
        return Endpoint(method: .PATCH, path: "/api/v1/channels/\(id)", body: Body(name: name, topic: topic))
    }

    static func deleteChannel(id: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/channels/\(id)")
    }
}

// MARK: - Messages

extension Endpoint {
    static func getMessages(channelID: Snowflake, before: Snowflake? = nil, limit: Int = 50) -> Endpoint {
        var items = [URLQueryItem(name: "limit", value: String(limit))]
        if let before { items.append(URLQueryItem(name: "before", value: before.description)) }
        return Endpoint(method: .GET, path: "/api/v1/channels/\(channelID)/messages", queryItems: items)
    }

    static func sendMessage(channelID: Snowflake, content: String) -> Endpoint {
        struct Body: Encodable, Sendable { let content: String }
        return Endpoint(method: .POST, path: "/api/v1/channels/\(channelID)/messages",
                        body: Body(content: content))
    }

    static func getMessage(channelID: Snowflake, messageID: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/channels/\(channelID)/messages/\(messageID)")
    }

    static func editMessage(channelID: Snowflake, messageID: Snowflake, content: String) -> Endpoint {
        struct Body: Encodable, Sendable { let content: String }
        return Endpoint(method: .PATCH, path: "/api/v1/channels/\(channelID)/messages/\(messageID)",
                        body: Body(content: content))
    }

    static func deleteMessage(channelID: Snowflake, messageID: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/channels/\(channelID)/messages/\(messageID)")
    }
}

// MARK: - Members

extension Endpoint {
    static func getMembers(guildID: Snowflake, limit: Int = 100, offset: Int = 0) -> Endpoint {
        let items = [URLQueryItem(name: "limit", value: String(limit)),
                     URLQueryItem(name: "offset", value: String(offset))]
        return Endpoint(method: .GET, path: "/api/v1/guilds/\(guildID)/members", queryItems: items)
    }

    static func getMember(guildID: Snowflake, userID: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/guilds/\(guildID)/members/\(userID)")
    }

    static func updateMember(guildID: Snowflake, userID: Snowflake, nickname: String?) -> Endpoint {
        struct Body: Encodable, Sendable { let nickname: String? }
        return Endpoint(method: .PATCH, path: "/api/v1/guilds/\(guildID)/members/\(userID)",
                        body: Body(nickname: nickname))
    }

    static func updateSelf(guildID: Snowflake, nickname: String?) -> Endpoint {
        struct Body: Encodable, Sendable { let nickname: String? }
        return Endpoint(method: .PATCH, path: "/api/v1/guilds/\(guildID)/members/@me",
                        body: Body(nickname: nickname))
    }

    static func kickMember(guildID: Snowflake, userID: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/guilds/\(guildID)/members/\(userID)")
    }

    static func leaveGuild(guildID: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/guilds/\(guildID)/members/@me")
    }
}

// MARK: - Roles

extension Endpoint {
    static func getRoles(guildID: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/guilds/\(guildID)/roles")
    }

    static func createRole(guildID: Snowflake, name: String, permissions: Int64 = 0, color: Int = 0) -> Endpoint {
        struct Body: Encodable, Sendable { let name: String; let permissions: Int64; let color: Int }
        return Endpoint(method: .POST, path: "/api/v1/guilds/\(guildID)/roles",
                        body: Body(name: name, permissions: permissions, color: color))
    }

    static func updateRole(guildID: Snowflake, roleID: Snowflake, name: String? = nil, color: Int? = nil, permissions: Int64? = nil) -> Endpoint {
        struct Body: Encodable, Sendable { let name: String?; let color: Int?; let permissions: Int64? }
        return Endpoint(method: .PATCH, path: "/api/v1/guilds/\(guildID)/roles/\(roleID)",
                        body: Body(name: name, color: color, permissions: permissions))
    }

    static func deleteRole(guildID: Snowflake, roleID: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/guilds/\(guildID)/roles/\(roleID)")
    }

    static func assignRole(guildID: Snowflake, userID: Snowflake, roleID: Snowflake) -> Endpoint {
        Endpoint(method: .PUT, path: "/api/v1/guilds/\(guildID)/members/\(userID)/roles/\(roleID)")
    }

    static func removeRole(guildID: Snowflake, userID: Snowflake, roleID: Snowflake) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/guilds/\(guildID)/members/\(userID)/roles/\(roleID)")
    }
}

// MARK: - Invites

extension Endpoint {
    static func createInvite(guildID: Snowflake, maxUses: Int = 0, maxAgeSeconds: Int = 86400) -> Endpoint {
        struct Body: Encodable, Sendable { let max_uses: Int; let max_age_seconds: Int }
        return Endpoint(method: .POST, path: "/api/v1/guilds/\(guildID)/invites",
                        body: Body(max_uses: maxUses, max_age_seconds: maxAgeSeconds))
    }

    static func getInvite(code: String) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/invites/\(code)", requiresAuth: false)
    }

    static func listInvites(guildID: Snowflake) -> Endpoint {
        Endpoint(method: .GET, path: "/api/v1/guilds/\(guildID)/invites")
    }

    static func acceptInvite(code: String) -> Endpoint {
        Endpoint(method: .POST, path: "/api/v1/invites/\(code)")
    }

    static func revokeInvite(code: String) -> Endpoint {
        Endpoint(method: .DELETE, path: "/api/v1/invites/\(code)")
    }
}

// MARK: - Typing

extension Endpoint {
    static func sendTyping(channelID: Snowflake) -> Endpoint {
        Endpoint(method: .POST, path: "/api/v1/channels/\(channelID)/typing")
    }
}
