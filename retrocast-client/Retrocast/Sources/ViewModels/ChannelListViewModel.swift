import Foundation

@Observable @MainActor
final class ChannelListViewModel {
    var isLoading = false
    var errorMessage: String?

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
