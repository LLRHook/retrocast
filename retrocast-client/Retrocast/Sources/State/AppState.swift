import Foundation

@Observable @MainActor
final class AppState {
    // MARK: - Current user

    var currentUser: User?

    // MARK: - Guild data

    var guilds: [Snowflake: Guild] = [:]
    var channels: [Snowflake: [Channel]] = [:]       // guildID -> sorted channels
    var members: [Snowflake: [Member]] = [:]          // guildID -> members
    var roles: [Snowflake: [Role]] = [:]              // guildID -> sorted roles

    // MARK: - DM data

    var dms: [DMChannel] = []
    var showDMList = false                             // true when DM list is active instead of guild channels

    // MARK: - Presence & typing

    var presence: [Snowflake: String] = [:]           // userID -> status ("online", "idle", "dnd", "offline")
    var typingUsers: [Snowflake: Set<Snowflake>] = [:] // channelID -> set of typing userIDs

    // MARK: - Selection

    var selectedGuildID: Snowflake?
    var selectedChannelID: Snowflake?

    // MARK: - Message cache

    var messageCache: [Snowflake: [Message]] = [:]    // channelID -> messages (newest first)
    var hasMoreMessages: [Snowflake: Bool] = [:]      // channelID -> whether more history exists

    // MARK: - Computed

    var selectedGuild: Guild? {
        guard let id = selectedGuildID else { return nil }
        return guilds[id]
    }

    var selectedChannels: [Channel] {
        guard let id = selectedGuildID else { return [] }
        return channels[id] ?? []
    }

    var selectedChannelMessages: [Message] {
        guard let id = selectedChannelID else { return [] }
        return messageCache[id] ?? []
    }

    var sortedGuilds: [Guild] {
        guilds.values.sorted { $0.id < $1.id }
    }

    var selectedDM: DMChannel? {
        guard showDMList, let channelID = selectedChannelID else { return nil }
        return dms.first { $0.id == channelID }
    }

    // MARK: - Mutations

    func selectGuild(_ id: Snowflake?) {
        showDMList = false
        selectedGuildID = id
        selectedChannelID = nil
    }

    func selectDMList() {
        showDMList = true
        selectedGuildID = nil
        selectedChannelID = nil
    }

    func selectDMChannel(_ id: Snowflake) {
        selectedChannelID = id
    }

    func selectChannel(_ id: Snowflake?) {
        selectedChannelID = id
    }

    func addMessage(_ message: Message, to channelID: Snowflake) {
        if messageCache[channelID] == nil {
            messageCache[channelID] = []
        }
        // Insert at beginning (newest first)
        messageCache[channelID]?.insert(message, at: 0)
    }

    func updateMessage(_ message: Message, in channelID: Snowflake) {
        guard let index = messageCache[channelID]?.firstIndex(where: { $0.id == message.id }) else { return }
        messageCache[channelID]?[index] = message
    }

    func deleteMessage(id: Snowflake, from channelID: Snowflake) {
        messageCache[channelID]?.removeAll { $0.id == id }
    }

    func setMessages(_ messages: [Message], for channelID: Snowflake, append: Bool = false) {
        if append {
            messageCache[channelID]?.append(contentsOf: messages)
        } else {
            messageCache[channelID] = messages
        }
        hasMoreMessages[channelID] = messages.count >= 50
    }

    func addTypingUser(_ userID: Snowflake, to channelID: Snowflake) {
        if typingUsers[channelID] == nil {
            typingUsers[channelID] = []
        }
        typingUsers[channelID]?.insert(userID)

        // Auto-remove after 8 seconds
        Task { @MainActor in
            try? await Task.sleep(for: .seconds(8))
            self.typingUsers[channelID]?.remove(userID)
        }
    }

    func removeTypingUser(_ userID: Snowflake, from channelID: Snowflake) {
        typingUsers[channelID]?.remove(userID)
    }

    func updatePresence(userID: Snowflake, status: String) {
        presence[userID] = status
    }

    func reset() {
        currentUser = nil
        guilds = [:]
        channels = [:]
        members = [:]
        roles = [:]
        dms = []
        showDMList = false
        presence = [:]
        typingUsers = [:]
        selectedGuildID = nil
        selectedChannelID = nil
        messageCache = [:]
        hasMoreMessages = [:]
    }
}
