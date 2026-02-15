import SwiftUI

struct ServerListView: View {
    let viewModel: ServerListViewModel?
    @Environment(AppState.self) private var appState

    @State private var showSettings = false
    @State private var guildForSettings: Guild? = nil

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 8) {
                // DM button
                dmButton

                Divider()
                    .frame(width: 32)
                    .padding(.vertical, 4)

                ForEach(appState.sortedGuilds) { guild in
                    guildButton(guild)
                }

                Divider()
                    .frame(width: 32)
                    .padding(.vertical, 4)

                // Add server button
                Button {
                    viewModel?.showCreateGuild = true
                } label: {
                    ZStack {
                        Circle()
                            .fill(Color.retroSidebar)
                            .frame(width: 48, height: 48)
                        Image(systemName: "plus")
                            .font(.title2)
                            .foregroundStyle(.retroGreen)
                    }
                }

                Divider()
                    .frame(width: 32)
                    .padding(.vertical, 4)

                // Settings button
                Button {
                    showSettings = true
                } label: {
                    ZStack {
                        Circle()
                            .fill(Color.retroSidebar)
                            .frame(width: 48, height: 48)
                        Image(systemName: "gearshape.fill")
                            .font(.title2)
                            .foregroundStyle(.retroMuted)
                    }
                }
            }
            .padding(.vertical, 12)
        }
        .frame(maxWidth: .infinity)
        .background(Color.retroDark)
        .sheet(isPresented: Binding(
            get: { viewModel?.showCreateGuild ?? false },
            set: { viewModel?.showCreateGuild = $0 }
        )) {
            if let vm = viewModel {
                CreateGuildSheet(viewModel: vm)
            }
        }
        .sheet(isPresented: Binding(
            get: { viewModel?.showJoinGuild ?? false },
            set: { viewModel?.showJoinGuild = $0 }
        )) {
            if let vm = viewModel {
                JoinGuildSheet(viewModel: vm)
            }
        }
        .sheet(isPresented: $showSettings) {
            UserSettingsView()
        }
        .sheet(item: $guildForSettings) { guild in
            GuildSettingsView(guild: guild)
        }
    }

    private var dmButton: some View {
        let isSelected = appState.showDMList

        return Button {
            appState.selectDMList()
        } label: {
            HStack(spacing: 0) {
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color.white)
                    .frame(width: 4, height: isSelected ? 40 : 0)
                    .opacity(isSelected ? 1 : 0)
                    .padding(.trailing, 8)

                ZStack {
                    Circle()
                        .fill(isSelected ? Color.retroAccent : Color.retroSidebar)
                        .frame(width: 48, height: 48)
                    Image(systemName: "envelope.fill")
                        .font(.title2)
                        .foregroundStyle(isSelected ? .white : .retroMuted)
                }
            }
        }
        .buttonStyle(.plain)
    }

    private func guildButton(_ guild: Guild) -> some View {
        let isSelected = appState.selectedGuildID == guild.id

        return Button {
            appState.selectGuild(guild.id)
        } label: {
            HStack(spacing: 0) {
                // Selection indicator pill
                RoundedRectangle(cornerRadius: 4)
                    .fill(Color.white)
                    .frame(width: 4, height: isSelected ? 40 : 0)
                    .opacity(isSelected ? 1 : 0)
                    .padding(.trailing, 8)

                GuildIcon(guild: guild, isSelected: isSelected)
            }
        }
        .buttonStyle(.plain)
        .contextMenu {
            Button {
                guildForSettings = guild
            } label: {
                Label("Server Settings", systemImage: "gearshape")
            }
        }
    }
}
