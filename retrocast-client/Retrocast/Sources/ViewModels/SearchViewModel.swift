import Foundation

@Observable @MainActor
final class SearchViewModel {
    var query = ""
    var results: [Message] = []
    var isSearching = false
    var hasSearched = false
    var errorMessage: String?

    private let api: APIClient
    private let appState: AppState
    private var debounceTask: Task<Void, Never>?

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    // MARK: - Search

    func searchDebounced(guildID: Snowflake) {
        debounceTask?.cancel()
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            clear()
            return
        }
        debounceTask = Task {
            try? await Task.sleep(for: .milliseconds(300))
            guard !Task.isCancelled else { return }
            await search(guildID: guildID)
        }
    }

    func search(guildID: Snowflake) async {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }
        guard !isSearching else { return }

        isSearching = true
        errorMessage = nil

        do {
            let messages: [Message] = try await api.request(
                .searchMessages(guildID: guildID, query: trimmed)
            )
            results = messages
            hasSearched = true
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isSearching = false
    }

    func clear() {
        debounceTask?.cancel()
        query = ""
        results = []
        hasSearched = false
        errorMessage = nil
    }

    // MARK: - Navigation

    func selectResult(_ message: Message) {
        appState.selectChannel(message.channelID)
    }

    /// Look up the channel name for a message from the current guild's channels.
    func channelName(for message: Message) -> String? {
        guard let guildID = appState.selectedGuildID,
              let channels = appState.channels[guildID] else { return nil }
        return channels.first { $0.id == message.channelID }?.name
    }
}
