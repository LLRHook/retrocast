import SwiftUI

struct ChatAreaView: View {
    let viewModel: ChatViewModel?
    @Environment(AppState.self) private var appState

    var body: some View {
        VStack(spacing: 0) {
            // Channel header
            channelHeader

            Divider()

            // Messages
            if let vm = viewModel {
                MessageListView(viewModel: vm)
            }

            // Typing indicator
            typingIndicator

            // Message input
            if let vm = viewModel, let channelID = appState.selectedChannelID {
                MessageInput(viewModel: vm, channelID: channelID)
            }
        }
        .background(Color.retroChat)
        .task(id: appState.selectedChannelID) {
            if let channelID = appState.selectedChannelID {
                await viewModel?.loadMessages(channelID: channelID)
            }
        }
    }

    private var channelHeader: some View {
        HStack(spacing: 8) {
            if let dm = appState.selectedDM {
                Image(systemName: "at")
                    .foregroundStyle(.retroMuted)
                Text(dm.displayName)
                    .font(.headline)
                    .foregroundStyle(.retroText)
            } else {
                Image(systemName: "number")
                    .foregroundStyle(.retroMuted)
                if let channelID = appState.selectedChannelID,
                   let guildID = appState.selectedGuildID,
                   let channel = appState.channels[guildID]?.first(where: { $0.id == channelID }) {
                    Text(channel.name)
                        .font(.headline)
                        .foregroundStyle(.retroText)

                    if let topic = channel.topic, !topic.isEmpty {
                        Divider()
                            .frame(height: 20)
                        Text(topic)
                            .font(.subheadline)
                            .foregroundStyle(.retroMuted)
                            .lineLimit(1)
                    }
                }
            }
            Spacer()
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    @ViewBuilder
    private var typingIndicator: some View {
        if let channelID = appState.selectedChannelID,
           let typing = appState.typingUsers[channelID],
           !typing.isEmpty {
            HStack(spacing: 4) {
                TypingDotsView()
                Text(typingText(for: typing))
                    .font(.caption)
                    .foregroundStyle(.retroMuted)
                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 4)
            .transition(.opacity)
        }
    }

    private func typingText(for userIDs: Set<Snowflake>) -> String {
        let count = userIDs.count
        if count == 1 {
            return "Someone is typing..."
        } else if count <= 3 {
            return "\(count) people are typing..."
        } else {
            return "Several people are typing..."
        }
    }
}

// MARK: - Typing dots animation

struct TypingDotsView: View {
    @State private var phase = 0

    var body: some View {
        HStack(spacing: 2) {
            ForEach(0..<3) { index in
                Circle()
                    .fill(Color.retroMuted)
                    .frame(width: 4, height: 4)
                    .offset(y: phase == index ? -3 : 0)
            }
        }
        .onAppear {
            withAnimation(.easeInOut(duration: 0.4).repeatForever(autoreverses: true)) {
                phase = (phase + 1) % 3
            }
        }
    }
}
