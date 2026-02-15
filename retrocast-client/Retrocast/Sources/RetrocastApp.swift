import SwiftUI

@main
struct RetrocastApp: App {
    @State private var appState = AppState()
    @State private var tokenManager = TokenManager()
    @State private var apiClient: APIClient
    @State private var gatewayClient = GatewayClient()

    init() {
        let tm = TokenManager()
        _tokenManager = State(initialValue: tm)
        _apiClient = State(initialValue: APIClient(tokenManager: tm))
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(appState)
                .environment(apiClient)
                .environment(gatewayClient)
                .environment(tokenManager)
        }
    }
}
