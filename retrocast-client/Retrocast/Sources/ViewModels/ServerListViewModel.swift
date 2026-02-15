import Foundation

@Observable @MainActor
final class ServerListViewModel {
    var isLoading = false
    var errorMessage: String?
    var showCreateGuild = false
    var showJoinGuild = false
    var newGuildName = ""
    var inviteCode = ""

    private let api: APIClient
    private let appState: AppState

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    func loadGuilds() async {
        isLoading = true
        do {
            let guilds: [Guild] = try await api.request(.getMyGuilds())
            for guild in guilds {
                appState.guilds[guild.id] = guild
            }
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func createGuild() async {
        let name = newGuildName.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else {
            errorMessage = "Guild name is required."
            return
        }

        isLoading = true
        do {
            let guild: Guild = try await api.request(.createGuild(name: name))
            appState.guilds[guild.id] = guild
            appState.selectGuild(guild.id)
            newGuildName = ""
            showCreateGuild = false
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func joinGuild() async {
        let code = inviteCode.trimmingCharacters(in: .whitespaces)
        guard !code.isEmpty else {
            errorMessage = "Invite code is required."
            return
        }

        isLoading = true
        do {
            let guild: Guild = try await api.request(.acceptInvite(code: code))
            appState.guilds[guild.id] = guild
            appState.selectGuild(guild.id)
            inviteCode = ""
            showJoinGuild = false
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }
}
