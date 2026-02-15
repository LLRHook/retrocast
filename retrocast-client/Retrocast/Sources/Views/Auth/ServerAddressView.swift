import SwiftUI

struct ServerAddressView: View {
    @Bindable var viewModel: AuthViewModel

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 24) {
                // Logo / Title
                VStack(spacing: 8) {
                    Image(systemName: "antenna.radiowaves.left.and.right")
                        .font(.system(size: 48))
                        .foregroundStyle(.retroAccent)
                    Text("Retrocast")
                        .font(.largeTitle)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroText)
                    Text("Connect to your server")
                        .font(.subheadline)
                        .foregroundStyle(.retroMuted)
                }

                // Server address input
                VStack(spacing: 16) {
                    TextField("Server address (e.g. 192.168.1.100:8080)", text: $viewModel.serverAddress)
                        .textFieldStyle(.roundedBorder)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .keyboardType(.URL)
                        .onSubmit { Task { await viewModel.connectToServer() } }

                    if let error = viewModel.errorMessage {
                        ErrorBanner(message: error) {
                            viewModel.errorMessage = nil
                        }
                    }

                    Button {
                        Task { await viewModel.connectToServer() }
                    } label: {
                        Group {
                            if viewModel.isLoading {
                                ProgressView()
                                    .tint(.white)
                            } else {
                                Text("Connect")
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.retroAccent)
                    .disabled(viewModel.isLoading || viewModel.serverAddress.isEmpty)
                }
            }
            .padding(32)

            Spacer()
        }
        .background(Color.retroDark)
    }
}
