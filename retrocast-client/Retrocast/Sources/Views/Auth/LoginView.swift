import SwiftUI

struct LoginView: View {
    @Bindable var viewModel: AuthViewModel

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 24) {
                VStack(spacing: 8) {
                    Text("Welcome back!")
                        .font(.title)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroText)
                    Text("Log in to continue")
                        .font(.subheadline)
                        .foregroundStyle(.retroMuted)
                }

                VStack(spacing: 12) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("USERNAME")
                            .font(.caption)
                            .fontWeight(.bold)
                            .foregroundStyle(.retroMuted)
                        TextField("", text: $viewModel.username)
                            .textFieldStyle(.roundedBorder)
                            .textInputAutocapitalization(.never)
                            .autocorrectionDisabled()
                    }

                    VStack(alignment: .leading, spacing: 4) {
                        Text("PASSWORD")
                            .font(.caption)
                            .fontWeight(.bold)
                            .foregroundStyle(.retroMuted)
                        SecureField("", text: $viewModel.password)
                            .textFieldStyle(.roundedBorder)
                            .onSubmit { Task { await viewModel.login() } }
                    }
                }

                if let error = viewModel.errorMessage {
                    ErrorBanner(message: error) {
                        viewModel.errorMessage = nil
                    }
                }

                Button {
                    Task { await viewModel.login() }
                } label: {
                    Group {
                        if viewModel.isLoading {
                            ProgressView().tint(.white)
                        } else {
                            Text("Log In")
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .tint(.retroAccent)
                .disabled(viewModel.isLoading)

                HStack(spacing: 4) {
                    Text("Don't have an account?")
                        .foregroundStyle(.retroMuted)
                    Button("Register") {
                        viewModel.showRegister = true
                        viewModel.errorMessage = nil
                    }
                    .foregroundStyle(.retroAccent)
                }
                .font(.subheadline)
            }
            .padding(32)

            Spacer()
        }
        .background(Color.retroDark)
    }
}
