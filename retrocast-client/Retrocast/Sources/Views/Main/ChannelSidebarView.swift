import SwiftUI

struct ChannelSidebarView: View {
    @Bindable var viewModel: ChannelListViewModel
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
                    if let guildID = appState.selectedGuildID {
                        let groups = viewModel.groupedChannels(for: guildID)
                        ForEach(Array(groups.enumerated()), id: \.offset) { _, group in
                            if let category = group.category {
                                categoryHeader(category)
                            }
                            ForEach(group.channels) { channel in
                                channelRow(channel)
                                    .contextMenu {
                                        Button {
                                            viewModel.editChannelName = channel.name
                                            viewModel.editingChannel = channel
                                        } label: {
                                            Label("Rename", systemImage: "pencil")
                                        }
                                        Button(role: .destructive) {
                                            Task {
                                                await viewModel.deleteChannel(channel)
                                            }
                                        } label: {
                                            Label("Delete", systemImage: "trash")
                                        }
                                    }
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
                await viewModel.loadChannels(guildID: guildID)
            }
        }
        .sheet(isPresented: $viewModel.showCreateChannel) {
            if let guildID = appState.selectedGuildID {
                let categories = (appState.channels[guildID] ?? []).filter { $0.type == .category }
                CreateChannelSheet(viewModel: viewModel, guildID: guildID, categories: categories)
            }
        }
        .alert("Rename Channel", isPresented: Binding(
            get: { viewModel.editingChannel != nil },
            set: { if !$0 { viewModel.editingChannel = nil } }
        )) {
            TextField("Channel name", text: $viewModel.editChannelName)
            Button("Cancel", role: .cancel) {
                viewModel.editingChannel = nil
                viewModel.editChannelName = ""
            }
            Button("Save") {
                if let channel = viewModel.editingChannel {
                    Task {
                        await viewModel.renameChannel(channel, newName: viewModel.editChannelName)
                    }
                }
            }
        } message: {
            Text("Enter a new name for this channel.")
        }
    }

    private func guildHeader(_ guild: Guild) -> some View {
        HStack {
            Text(guild.name)
                .font(.headline)
                .foregroundStyle(.retroText)
                .lineLimit(1)
            Spacer()
            Button {
                viewModel.showCreateChannel = true
            } label: {
                Image(systemName: "plus")
                    .font(.subheadline)
                    .foregroundStyle(.retroMuted)
            }
            .buttonStyle(.plain)
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
        .contextMenu {
            Button {
                viewModel.editChannelName = channel.name
                viewModel.editingChannel = channel
            } label: {
                Label("Rename", systemImage: "pencil")
            }
            Button(role: .destructive) {
                Task {
                    await viewModel.deleteChannel(channel)
                }
            } label: {
                Label("Delete", systemImage: "trash")
            }
        }
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
