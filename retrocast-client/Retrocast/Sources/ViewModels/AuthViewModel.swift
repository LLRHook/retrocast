import Foundation

@Observable @MainActor
final class AuthViewModel {
    var serverAddress: String = ""
    var username: String = ""
    var password: String = ""
    var displayName: String = ""
    var isLoading = false
    var errorMessage: String?
    var showRegister = false
    var isServerConnected = false

    private let api: APIClient
    private let tokenManager: TokenManager
    private let appState: AppState
    private let gateway: GatewayClient

    init(api: APIClient, tokenManager: TokenManager, appState: AppState, gateway: GatewayClient) {
        self.api = api
        self.tokenManager = tokenManager
        self.appState = appState
        self.gateway = gateway

        // Restore saved server address
        if let saved = tokenManager.serverURL {
            self.serverAddress = saved.replacingOccurrences(of: "http://", with: "")
                                      .replacingOccurrences(of: "https://", with: "")
        }
    }

    // MARK: - Server connection

    func connectToServer() async {
        isLoading = true
        errorMessage = nil

        let address = serverAddress.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !address.isEmpty else {
            errorMessage = "Please enter a server address."
            isLoading = false
            return
        }

        let urlString = address.hasPrefix("http") ? address : "http://\(address)"
        guard let url = URL(string: urlString) else {
            errorMessage = "Invalid server address."
            isLoading = false
            return
        }

        api.baseURL = url
        tokenManager.serverURL = urlString

        let healthy = await api.checkHealth()
        isLoading = false

        if healthy {
            isServerConnected = true
        } else {
            errorMessage = "Cannot connect to server. Check the address and try again."
        }
    }

    // MARK: - Login

    func login() async {
        guard validateLoginFields() else { return }
        isLoading = true
        errorMessage = nil

        do {
            struct AuthResponse: Decodable {
                let access_token: String
                let refresh_token: String
                let user: User
            }

            let response: AuthResponse = try await api.request(
                .login(username: username, password: password)
            )

            tokenManager.storeTokens(
                accessToken: response.access_token,
                refreshToken: response.refresh_token
            )
            appState.currentUser = response.user

            connectGateway()
            isLoading = false
        } catch {
            isLoading = false
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    // MARK: - Register

    func register() async {
        guard validateRegisterFields() else { return }
        isLoading = true
        errorMessage = nil

        do {
            struct AuthResponse: Decodable {
                let access_token: String
                let refresh_token: String
                let user: User
            }

            let response: AuthResponse = try await api.request(
                .register(username: username, password: password, displayName: displayName)
            )

            tokenManager.storeTokens(
                accessToken: response.access_token,
                refreshToken: response.refresh_token
            )
            appState.currentUser = response.user

            connectGateway()
            isLoading = false
        } catch {
            isLoading = false
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    // MARK: - Auto-login

    func attemptAutoLogin() async {
        guard tokenManager.isLoggedIn, api.baseURL != nil else { return }
        isLoading = true

        do {
            let user: User = try await api.request(.getMe())
            appState.currentUser = user
            isServerConnected = true
            connectGateway()
        } catch {
            // Token expired or invalid — show login
            tokenManager.clearTokens()
        }

        isLoading = false
    }

    // MARK: - Logout

    func logout() async {
        try? await api.requestVoid(.logout())
        tokenManager.clearTokens()
        gateway.disconnect()
        appState.reset()
        isServerConnected = false
    }

    // MARK: - Private

    private func connectGateway() {
        guard let baseURL = api.baseURL,
              let token = tokenManager.accessToken else { return }
        let wsScheme = baseURL.scheme == "https" ? "wss" : "ws"
        guard let host = baseURL.host, let port = baseURL.port else { return }
        guard let wsURL = URL(string: "\(wsScheme)://\(host):\(port)/gateway") else { return }
        gateway.connect(to: wsURL, token: token)
    }

    private func validateLoginFields() -> Bool {
        if username.trimmingCharacters(in: .whitespaces).isEmpty {
            errorMessage = "Username is required."
            return false
        }
        if password.isEmpty {
            errorMessage = "Password is required."
            return false
        }
        return true
    }

    private func validateRegisterFields() -> Bool {
        if username.trimmingCharacters(in: .whitespaces).isEmpty {
            errorMessage = "Username is required."
            return false
        }
        if password.count < 6 {
            errorMessage = "Password must be at least 6 characters."
            return false
        }
        if displayName.trimmingCharacters(in: .whitespaces).isEmpty {
            errorMessage = "Display name is required."
            return false
        }
        return true
    }
}
