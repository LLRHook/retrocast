import Foundation

@Observable @MainActor
final class SettingsViewModel {
    var displayName: String = ""
    var isLoading = false
    var errorMessage: String?
    var successMessage: String?

    private let api: APIClient
    private let appState: AppState
    private let tokenManager: TokenManager

    init(api: APIClient, appState: AppState, tokenManager: TokenManager) {
        self.api = api
        self.appState = appState
        self.tokenManager = tokenManager

        self.displayName = appState.currentUser?.displayName ?? ""
    }

    func updateProfile() async {
        let name = displayName.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else {
            errorMessage = "Display name cannot be empty."
            return
        }

        isLoading = true
        errorMessage = nil
        successMessage = nil

        do {
            let user: User = try await api.request(.updateMe(displayName: name))
            appState.currentUser = user
            successMessage = "Profile updated."
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func logout() async {
        try? await api.requestVoid(.logout())
        tokenManager.clearTokens()
        appState.reset()
    }
}
