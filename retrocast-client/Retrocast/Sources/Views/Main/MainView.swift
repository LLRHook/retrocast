import SwiftUI

struct MainView: View {
    @Environment(AppState.self) private var appState
    @Environment(APIClient.self) private var api
    @Environment(GatewayClient.self) private var gateway

    @State private var serverListVM: ServerListViewModel?
    @State private var channelListVM: ChannelListViewModel?
    @State private var chatVM: ChatViewModel?

    var body: some View {
        NavigationSplitView {
            ServerListView(viewModel: serverListVM)
                .navigationSplitViewColumnWidth(min: 72, ideal: 72, max: 72)
        } content: {
            if appState.selectedGuildID != nil {
                ChannelSidebarView(viewModel: channelListVM)
            } else {
                emptyGuildState
            }
        } detail: {
            if appState.selectedChannelID != nil {
                ChatAreaView(viewModel: chatVM)
            } else {
                emptyChannelState
            }
        }
        .navigationSplitViewStyle(.balanced)
        .task {
            let slVM = ServerListViewModel(api: api, appState: appState)
            let clVM = ChannelListViewModel(api: api, appState: appState)
            let cVM = ChatViewModel(api: api, appState: appState)
            serverListVM = slVM
            channelListVM = clVM
            chatVM = cVM

            setupGatewayEventHandler()
            await slVM.loadGuilds()
        }
    }

    private var emptyGuildState: some View {
        VStack(spacing: 12) {
            Image(systemName: "bubble.left.and.bubble.right")
                .font(.system(size: 40))
                .foregroundStyle(.retroMuted)
            Text("Select a server")
                .font(.headline)
                .foregroundStyle(.retroMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.retroSidebar)
    }

    private var emptyChannelState: some View {
        VStack(spacing: 12) {
            Image(systemName: "number")
                .font(.system(size: 40))
                .foregroundStyle(.retroMuted)
            Text("Select a channel")
                .font(.headline)
                .foregroundStyle(.retroMuted)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.retroChat)
    }

    // MARK: - Gateway event handling

    private func setupGatewayEventHandler() {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let str = try container.decode(String.self)
            if let date = ISO8601DateFormatter.withFractionalSeconds.date(from: str) { return date }
            if let date = ISO8601DateFormatter.standard.date(from: str) { return date }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Bad date: \(str)")
        }

        gateway.onEvent = { eventType, data in
            Task { @MainActor in
                switch eventType {
                case .messageCreate:
                    if let msg = try? decoder.decode(Message.self, from: data) {
                        appState.addMessage(msg, to: msg.channelID)
                        appState.removeTypingUser(msg.authorID, from: msg.channelID)
                    }
                case .messageUpdate:
                    if let msg = try? decoder.decode(Message.self, from: data) {
                        appState.updateMessage(msg, in: msg.channelID)
                    }
                case .messageDelete:
                    if let del = try? decoder.decode(MessageDeleteData.self, from: data) {
                        appState.deleteMessage(id: del.id, from: del.channelID)
                    }
                case .guildCreate:
                    if let guild = try? decoder.decode(Guild.self, from: data) {
                        appState.guilds[guild.id] = guild
                    }
                case .guildUpdate:
                    if let guild = try? decoder.decode(Guild.self, from: data) {
                        appState.guilds[guild.id] = guild
                    }
                case .guildDelete:
                    if let guild = try? decoder.decode(Guild.self, from: data) {
                        appState.guilds.removeValue(forKey: guild.id)
                        if appState.selectedGuildID == guild.id {
                            appState.selectGuild(nil)
                        }
                    }
                case .channelCreate:
                    if let ch = try? decoder.decode(Channel.self, from: data) {
                        if appState.channels[ch.guildID] == nil {
                            appState.channels[ch.guildID] = []
                        }
                        appState.channels[ch.guildID]?.append(ch)
                        appState.channels[ch.guildID]?.sort { $0.position < $1.position }
                    }
                case .channelUpdate:
                    if let ch = try? decoder.decode(Channel.self, from: data) {
                        if let idx = appState.channels[ch.guildID]?.firstIndex(where: { $0.id == ch.id }) {
                            appState.channels[ch.guildID]?[idx] = ch
                        }
                    }
                case .channelDelete:
                    if let ch = try? decoder.decode(Channel.self, from: data) {
                        appState.channels[ch.guildID]?.removeAll { $0.id == ch.id }
                        if appState.selectedChannelID == ch.id {
                            appState.selectChannel(nil)
                        }
                    }
                case .typingStart:
                    if let typing = try? decoder.decode(TypingStartData.self, from: data) {
                        if typing.userID != appState.currentUser?.id {
                            appState.addTypingUser(typing.userID, to: typing.channelID)
                        }
                    }
                case .presenceUpdate:
                    if let p = try? decoder.decode(PresenceUpdateData.self, from: data) {
                        appState.updatePresence(userID: p.userID, status: p.status)
                    }
                case .guildMemberAdd:
                    if let member = try? decoder.decode(Member.self, from: data) {
                        if appState.members[member.guildID] == nil {
                            appState.members[member.guildID] = []
                        }
                        appState.members[member.guildID]?.append(member)
                    }
                case .guildMemberRemove:
                    if let member = try? decoder.decode(Member.self, from: data) {
                        appState.members[member.guildID]?.removeAll { $0.userID == member.userID }
                    }
                default:
                    break
                }
            }
        }
    }
}
