import SwiftUI

struct GuildSettingsView: View {
    let guild: Guild
    @Environment(APIClient.self) private var api
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss

    @State private var name: String = ""
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showInvites = false

    var body: some View {
        NavigationStack {
            Form {
                Section("Server Info") {
                    TextField("Server Name", text: $name)
                }

                Section("Invites") {
                    Button("Manage Invites") {
                        showInvites = true
                    }
                }

                Section("Roles") {
                    if let roles = appState.roles[guild.id] {
                        ForEach(roles) { role in
                            HStack {
                                RoleTag(name: role.name, color: role.color)
                                Spacer()
                                Text("Position \(role.position)")
                                    .font(.caption)
                                    .foregroundStyle(.retroMuted)
                            }
                        }
                    }
                }

                if let error = errorMessage {
                    Section {
                        Text(error)
                            .foregroundStyle(.red)
                    }
                }

                Section {
                    if guild.ownerID == appState.currentUser?.id {
                        Button("Delete Server", role: .destructive) {
                            Task { await deleteGuild() }
                        }
                    } else {
                        Button("Leave Server", role: .destructive) {
                            Task { await leaveGuild() }
                        }
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Color.retroDark)
            .navigationTitle("Server Settings")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await save() }
                    }
                    .disabled(isLoading || name == guild.name)
                }
            }
        }
        .onAppear {
            name = guild.name
        }
        .sheet(isPresented: $showInvites) {
            InviteSheet(guildID: guild.id)
        }
    }

    private func save() async {
        isLoading = true
        do {
            let updated: Guild = try await api.request(.updateGuild(id: guild.id, name: name))
            appState.guilds[guild.id] = updated
            dismiss()
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    private func deleteGuild() async {
        do {
            try await api.requestVoid(.deleteGuild(id: guild.id))
            appState.guilds.removeValue(forKey: guild.id)
            appState.selectGuild(nil)
            dismiss()
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    private func leaveGuild() async {
        do {
            try await api.requestVoid(.leaveGuild(guildID: guild.id))
            appState.guilds.removeValue(forKey: guild.id)
            appState.selectGuild(nil)
            dismiss()
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }
}
