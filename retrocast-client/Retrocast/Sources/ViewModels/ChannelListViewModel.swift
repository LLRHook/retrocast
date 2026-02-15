import Foundation

@Observable @MainActor
final class ChannelListViewModel {
    var isLoading = false
    var errorMessage: String?

    // Create channel sheet state
    var showCreateChannel = false
    var newChannelName = ""
    var newChannelType: ChannelType = .text
    var newChannelParentID: Snowflake?

    // Edit channel state
    var editingChannel: Channel?
    var editChannelName = ""

    private let api: APIClient
    private let appState: AppState

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    func loadChannels(guildID: Snowflake) async {
        isLoading = true
        do {
            let channels: [Channel] = try await api.request(.getChannels(guildID: guildID))
            appState.channels[guildID] = channels.sorted { $0.position < $1.position }

            // Auto-select first text channel if none selected
            if appState.selectedChannelID == nil {
                let firstText = channels.first { $0.type == .text }
                if let ch = firstText {
                    appState.selectChannel(ch.id)
                }
            }
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func createChannel(guildID: Snowflake) async {
        let name = newChannelName.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else {
            errorMessage = "Channel name is required."
            return
        }

        isLoading = true
        do {
            let channel: Channel = try await api.request(
                .createChannel(guildID: guildID, name: name, type: newChannelType.rawValue, parentID: newChannelParentID)
            )
            // Append to local channel list and re-sort
            var current = appState.channels[guildID] ?? []
            current.append(channel)
            appState.channels[guildID] = current.sorted { $0.position < $1.position }

            // Reset form state
            newChannelName = ""
            newChannelType = .text
            newChannelParentID = nil
            showCreateChannel = false
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func renameChannel(_ channel: Channel, newName: String) async {
        let name = newName.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else {
            errorMessage = "Channel name is required."
            return
        }

        isLoading = true
        do {
            let updated: Channel = try await api.request(.updateChannel(id: channel.id, name: name))
            // Replace in local list
            if let guildID = appState.selectedGuildID,
               var current = appState.channels[guildID],
               let index = current.firstIndex(where: { $0.id == channel.id }) {
                current[index] = updated
                appState.channels[guildID] = current.sorted { $0.position < $1.position }
            }
            editingChannel = nil
            editChannelName = ""
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func deleteChannel(_ channel: Channel) async {
        isLoading = true
        do {
            try await api.requestVoid(.deleteChannel(id: channel.id))
            // Remove from local list
            if let guildID = appState.selectedGuildID {
                appState.channels[guildID]?.removeAll { $0.id == channel.id }
                // If the deleted channel was selected, clear selection
                if appState.selectedChannelID == channel.id {
                    let firstText = appState.channels[guildID]?.first { $0.type == .text }
                    appState.selectChannel(firstText?.id)
                }
            }
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    /// Group channels by category for display.
    func groupedChannels(for guildID: Snowflake) -> [(category: Channel?, channels: [Channel])] {
        let allChannels = appState.channels[guildID] ?? []
        let categories = allChannels.filter { $0.type == .category }.sorted { $0.position < $1.position }
        let nonCategoryChannels = allChannels.filter { $0.type != .category }

        var groups: [(category: Channel?, channels: [Channel])] = []

        // Channels without a parent category
        let uncategorized = nonCategoryChannels.filter { $0.parentID == nil }.sorted { $0.position < $1.position }
        if !uncategorized.isEmpty {
            groups.append((category: nil, channels: uncategorized))
        }

        // Channels grouped by category
        for category in categories {
            let children = nonCategoryChannels
                .filter { $0.parentID == category.id }
                .sorted { $0.position < $1.position }
            groups.append((category: category, channels: children))
        }

        return groups
    }
}
