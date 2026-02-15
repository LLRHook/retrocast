import SwiftUI

struct ContentView: View {
    @Environment(AppState.self) private var appState
    @Environment(APIClient.self) private var api
    @Environment(GatewayClient.self) private var gateway
    @Environment(TokenManager.self) private var tokenManager

    @State private var authVM: AuthViewModel?

    var body: some View {
        Group {
            if appState.currentUser != nil {
                MainView()
            } else {
                authFlow
            }
        }
        .preferredColorScheme(.dark)
        .task {
            let vm = AuthViewModel(api: api, tokenManager: tokenManager, appState: appState, gateway: gateway)
            authVM = vm
            await vm.attemptAutoLogin()
        }
    }

    @ViewBuilder
    private var authFlow: some View {
        if let vm = authVM {
            if vm.isServerConnected {
                if vm.showRegister {
                    RegisterView(viewModel: vm)
                } else {
                    LoginView(viewModel: vm)
                }
            } else {
                ServerAddressView(viewModel: vm)
            }
        } else {
            LoadingView(message: "Loading...")
        }
    }
}
