import Foundation

@Observable @MainActor
final class MemberListViewModel {
    var isLoading = false
    var errorMessage: String?

    private let api: APIClient
    private let appState: AppState

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    func loadMembers(guildID: Snowflake) async {
        isLoading = true
        do {
            let members: [Member] = try await api.request(.getMembers(guildID: guildID))
            appState.members[guildID] = members
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    func loadRoles(guildID: Snowflake) async {
        do {
            let roles: [Role] = try await api.request(.getRoles(guildID: guildID))
            appState.roles[guildID] = roles.sorted { $0.position > $1.position }
        } catch {
            // Roles are non-critical for member display
        }
    }

    /// Group members by their highest role for display.
    func groupedMembers(for guildID: Snowflake) -> [(role: Role?, members: [Member])] {
        let members = appState.members[guildID] ?? []
        let roles = appState.roles[guildID] ?? []

        // Build role lookup
        let roleMap = Dictionary(uniqueKeysWithValues: roles.map { ($0.id, $0) })

        // Sort members by highest role position
        var grouped: [Snowflake?: [Member]] = [:]
        for member in members {
            let highestRole = member.roles
                .compactMap { roleMap[$0] }
                .filter { !$0.isDefault }
                .sorted { $0.position > $1.position }
                .first
            grouped[highestRole?.id, default: []].append(member)
        }

        // Convert to sorted array
        var result: [(role: Role?, members: [Member])] = []
        for role in roles where !role.isDefault {
            if let group = grouped[role.id], !group.isEmpty {
                result.append((role: role, members: group))
            }
        }
        // Members with no special role
        if let ungrouped = grouped[nil], !ungrouped.isEmpty {
            result.append((role: nil, members: ungrouped))
        }
        return result
    }
}
