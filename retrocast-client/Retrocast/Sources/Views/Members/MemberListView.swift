import SwiftUI

struct MemberListView: View {
    @Environment(AppState.self) private var appState
    @Environment(APIClient.self) private var api

    @State private var viewModel: MemberListViewModel?

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 0) {
                if let vm = viewModel, let guildID = appState.selectedGuildID {
                    let groups = vm.groupedMembers(for: guildID)
                    ForEach(Array(groups.enumerated()), id: \.offset) { _, group in
                        roleHeader(group.role)
                        ForEach(group.members) { member in
                            MemberRow(member: member)
                        }
                    }
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 8)
        }
        .frame(width: 200)
        .background(Color.retroSidebar)
        .task(id: appState.selectedGuildID) {
            let vm = MemberListViewModel(api: api, appState: appState)
            viewModel = vm
            if let guildID = appState.selectedGuildID {
                async let _ = vm.loadMembers(guildID: guildID)
                async let _ = vm.loadRoles(guildID: guildID)
            }
        }
    }

    private func roleHeader(_ role: Role?) -> some View {
        Text((role?.name ?? "Online").uppercased())
            .font(.caption)
            .fontWeight(.bold)
            .foregroundStyle(.retroMuted)
            .padding(.top, 16)
            .padding(.bottom, 4)
            .padding(.leading, 8)
    }
}
