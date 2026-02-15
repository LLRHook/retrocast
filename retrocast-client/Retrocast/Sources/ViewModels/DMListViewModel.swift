import Foundation

@Observable @MainActor
final class DMListViewModel {
    var isLoading = false
    var errorMessage: String?
    var showNewDM = false

    private let api: APIClient
    private let appState: AppState

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    func loadDMs() async {
        isLoading = true
        do {
            let channels: [DMChannel] = try await api.request(.listDMs())
            appState.dms = channels.sorted { $0.id > $1.id }
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func createDM(recipientID: Snowflake) async {
        isLoading = true
        do {
            let dm: DMChannel = try await api.request(.createDM(recipientID: recipientID))
            // Add to list if not already present
            if !appState.dms.contains(where: { $0.id == dm.id }) {
                appState.dms.insert(dm, at: 0)
            }
            appState.selectDMChannel(dm.id)
            showNewDM = false
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }
}
