import SwiftUI

struct RegisterView: View {
    @Bindable var viewModel: AuthViewModel

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 24) {
                VStack(spacing: 8) {
                    Text("Create an account")
                        .font(.title)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroText)
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
                        Text("DISPLAY NAME")
                            .font(.caption)
                            .fontWeight(.bold)
                            .foregroundStyle(.retroMuted)
                        TextField("", text: $viewModel.displayName)
                            .textFieldStyle(.roundedBorder)
                    }

                    VStack(alignment: .leading, spacing: 4) {
                        Text("PASSWORD")
                            .font(.caption)
                            .fontWeight(.bold)
                            .foregroundStyle(.retroMuted)
                        SecureField("", text: $viewModel.password)
                            .textFieldStyle(.roundedBorder)
                            .onSubmit { Task { await viewModel.register() } }
                    }
                }

                if let error = viewModel.errorMessage {
                    ErrorBanner(message: error) {
                        viewModel.errorMessage = nil
                    }
                }

                Button {
                    Task { await viewModel.register() }
                } label: {
                    Group {
                        if viewModel.isLoading {
                            ProgressView().tint(.white)
                        } else {
                            Text("Register")
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .tint(.retroAccent)
                .disabled(viewModel.isLoading)

                HStack(spacing: 4) {
                    Text("Already have an account?")
                        .foregroundStyle(.retroMuted)
                    Button("Log In") {
                        viewModel.showRegister = false
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
