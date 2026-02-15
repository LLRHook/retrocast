import SwiftUI

struct UserSettingsView: View {
    @Environment(APIClient.self) private var api
    @Environment(AppState.self) private var appState
    @Environment(TokenManager.self) private var tokenManager
    @Environment(\.dismiss) private var dismiss

    @State private var viewModel: SettingsViewModel?
    @State private var showAppSettings = false

    var body: some View {
        NavigationStack {
            Form {
                if let user = appState.currentUser {
                    Section("Profile") {
                        HStack {
                            AvatarView(name: user.displayName, avatarHash: user.avatarHash, size: 48)
                            VStack(alignment: .leading) {
                                Text(user.displayName)
                                    .font(.headline)
                                    .foregroundStyle(.retroText)
                                Text("@\(user.username)")
                                    .font(.subheadline)
                                    .foregroundStyle(.retroMuted)
                            }
                        }
                    }

                    Section("Edit Profile") {
                        TextField("Display Name", text: Binding(
                            get: { viewModel?.displayName ?? "" },
                            set: { viewModel?.displayName = $0 }
                        ))
                    }

                    if let error = viewModel?.errorMessage {
                        Section {
                            Text(error).foregroundStyle(.red)
                        }
                    }

                    if let success = viewModel?.successMessage {
                        Section {
                            Text(success).foregroundStyle(.green)
                        }
                    }
                }

                Section {
                    Button("Save Changes") {
                        Task { await viewModel?.updateProfile() }
                    }
                    .disabled(viewModel?.isLoading ?? false)
                }

                Section {
                    Button("App Settings") {
                        showAppSettings = true
                    }
                }

                Section {
                    Button("Log Out", role: .destructive) {
                        Task {
                            await viewModel?.logout()
                            dismiss()
                        }
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Color.retroDark)
            .navigationTitle("User Settings")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") { dismiss() }
                }
            }
            .sheet(isPresented: $showAppSettings) {
                AppSettingsView()
            }
        }
        .task {
            viewModel = SettingsViewModel(api: api, appState: appState, tokenManager: tokenManager)
        }
    }
}
