import SwiftUI

struct VoiceChannelView: View {
    let channel: Channel
    @Bindable var voiceVM: VoiceViewModel
    @Environment(AppState.self) private var appState

    private var voiceStates: [VoiceState] {
        appState.voiceStates[channel.id] ?? []
    }

    private var isCurrentChannel: Bool {
        voiceVM.currentChannelID == channel.id
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Channel row header
            channelHeader

            // Connected users list
            if !voiceStates.isEmpty {
                VStack(alignment: .leading, spacing: 2) {
                    ForEach(voiceStates) { state in
                        voiceUserRow(state)
                    }
                }
                .padding(.leading, 28)
            }
        }
    }

    private var channelHeader: some View {
        Button {
            Task {
                if isCurrentChannel {
                    await voiceVM.leaveChannel()
                } else {
                    await voiceVM.joinChannel(channel.id, channelName: channel.name)
                }
            }
        } label: {
            HStack(spacing: 6) {
                Image(systemName: "speaker.wave.2")
                    .font(.subheadline)
                    .foregroundStyle(isCurrentChannel ? .retroGreen : .retroMuted)
                    .frame(width: 20)
                Text(channel.name)
                    .font(.body)
                    .foregroundStyle(isCurrentChannel ? .retroText : .retroMuted)
                    .lineLimit(1)
                Spacer()
                if !voiceStates.isEmpty {
                    Text("\(voiceStates.count)")
                        .font(.caption2)
                        .foregroundStyle(.retroMuted)
                }
            }
            .padding(.horizontal, 8)
            .padding(.vertical, 6)
            .background(isCurrentChannel ? Color.retroHover : Color.clear)
            .clipShape(RoundedRectangle(cornerRadius: 4))
        }
        .buttonStyle(.plain)
    }

    private func voiceUserRow(_ state: VoiceState) -> some View {
        let member = appState.members[channel.guildID]?.first { $0.userID == state.userID }
        let displayName = member?.nickname ?? state.userID.description

        return HStack(spacing: 6) {
            AvatarView(name: displayName, avatarHash: nil, size: 20)

            Text(displayName)
                .font(.caption)
                .foregroundStyle(.retroMuted)
                .lineLimit(1)

            Spacer()

            if state.selfMute {
                Image(systemName: "mic.slash.fill")
                    .font(.caption2)
                    .foregroundStyle(.retroRed)
            }
            if state.selfDeaf {
                Image(systemName: "speaker.slash.fill")
                    .font(.caption2)
                    .foregroundStyle(.retroRed)
            }
        }
        .padding(.vertical, 2)
        .padding(.trailing, 8)
    }
}
