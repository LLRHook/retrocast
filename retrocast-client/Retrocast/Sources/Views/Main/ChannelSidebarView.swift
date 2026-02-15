import SwiftUI

struct ChannelSidebarView: View {
    let viewModel: ChannelListViewModel?
    @Environment(AppState.self) private var appState

    var body: some View {
        VStack(spacing: 0) {
            // Guild header
            if let guild = appState.selectedGuild {
                guildHeader(guild)
            }

            // Channel list
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 2) {
                    if let guildID = appState.selectedGuildID, let vm = viewModel {
                        let groups = vm.groupedChannels(for: guildID)
                        ForEach(Array(groups.enumerated()), id: \.offset) { _, group in
                            if let category = group.category {
                                categoryHeader(category)
                            }
                            ForEach(group.channels) { channel in
                                channelRow(channel)
                            }
                        }
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
            }
        }
        .background(Color.retroSidebar)
        .task(id: appState.selectedGuildID) {
            if let guildID = appState.selectedGuildID {
                await viewModel?.loadChannels(guildID: guildID)
            }
        }
    }

    private func guildHeader(_ guild: Guild) -> some View {
        HStack {
            Text(guild.name)
                .font(.headline)
                .foregroundStyle(.retroText)
                .lineLimit(1)
            Spacer()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(Color.retroSidebar)
        .overlay(alignment: .bottom) {
            Divider()
        }
    }

    private func categoryHeader(_ channel: Channel) -> some View {
        HStack(spacing: 4) {
            Image(systemName: "chevron.down")
                .font(.caption2)
            Text(channel.name.uppercased())
                .font(.caption)
                .fontWeight(.bold)
        }
        .foregroundStyle(.retroMuted)
        .padding(.top, 16)
        .padding(.bottom, 4)
        .padding(.leading, 4)
    }

    private func channelRow(_ channel: Channel) -> some View {
        let isSelected = appState.selectedChannelID == channel.id
        let icon = channel.type == .voice ? "speaker.wave.2" : "number"

        return Button {
            appState.selectChannel(channel.id)
        } label: {
            HStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.subheadline)
                    .foregroundStyle(isSelected ? .retroText : .retroMuted)
                    .frame(width: 20)
                Text(channel.name)
                    .font(.body)
                    .foregroundStyle(isSelected ? .retroText : .retroMuted)
                    .lineLimit(1)
                Spacer()
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 6)
            .background(isSelected ? Color.retroHover : Color.clear)
            .clipShape(RoundedRectangle(cornerRadius: 4))
        }
        .buttonStyle(.plain)
    }
}
