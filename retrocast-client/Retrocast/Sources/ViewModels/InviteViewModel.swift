import Foundation

@Observable @MainActor
final class InviteViewModel {
    var invites: [Invite] = []
    var generatedCode: String?
    var isLoading = false
    var errorMessage: String?

    private let api: APIClient

    init(api: APIClient) {
        self.api = api
    }

    func loadInvites(guildID: Snowflake) async {
        isLoading = true
        do {
            invites = try await api.request(.listInvites(guildID: guildID))
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func createInvite(guildID: Snowflake, maxUses: Int = 0, maxAgeSeconds: Int = 86400) async {
        isLoading = true
        do {
            let invite: Invite = try await api.request(
                .createInvite(guildID: guildID, maxUses: maxUses, maxAgeSeconds: maxAgeSeconds)
            )
            generatedCode = invite.code
            invites.insert(invite, at: 0)
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func revokeInvite(code: String) async {
        do {
            try await api.requestVoid(.revokeInvite(code: code))
            invites.removeAll { $0.code == code }
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }
}
