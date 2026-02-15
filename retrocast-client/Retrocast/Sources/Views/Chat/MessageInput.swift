import SwiftUI

struct MessageInput: View {
    @Bindable var viewModel: ChatViewModel
    let channelID: Snowflake
    @Environment(AppState.self) private var appState

    @FocusState private var isFocused: Bool
    @State private var typingThrottle: Task<Void, Never>?
    @State private var lastTypingSent: Date?

    var body: some View {
        HStack(spacing: 8) {
            TextField(placeholder, text: $viewModel.messageText, axis: .vertical)
                .textFieldStyle(.plain)
                .lineLimit(1...6)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(Color.retroInput)
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .foregroundStyle(.retroText)
                .focused($isFocused)
                .onSubmit {
                    Task { await viewModel.sendMessage(channelID: channelID) }
                }
                .onChange(of: viewModel.messageText) { _, newValue in
                    if !newValue.isEmpty {
                        throttledTyping()
                    }
                }

            if !viewModel.messageText.trimmingCharacters(in: .whitespaces).isEmpty {
                Button {
                    Task { await viewModel.sendMessage(channelID: channelID) }
                } label: {
                    Image(systemName: "arrow.up.circle.fill")
                        .font(.title2)
                        .foregroundStyle(.retroAccent)
                }
                .disabled(viewModel.isSending)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
        .background(Color.retroChat)
    }

    private var placeholder: String {
        if let guildID = appState.selectedGuildID,
           let channel = appState.channels[guildID]?.first(where: { $0.id == channelID }) {
            return "Message #\(channel.name)"
        }
        return "Send a message"
    }

    /// Send typing indicator at most once per 8 seconds.
    private func throttledTyping() {
        let now = Date()
        if let last = lastTypingSent, now.timeIntervalSince(last) < 8 {
            return
        }
        lastTypingSent = now
        Task {
            await viewModel.sendTyping(channelID: channelID)
        }
    }
}
