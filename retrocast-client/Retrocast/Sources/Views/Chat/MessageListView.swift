import SwiftUI

struct MessageListView: View {
    @Bindable var viewModel: ChatViewModel
    @Environment(AppState.self) private var appState

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    // Load more indicator
                    if viewModel.isLoadingMore {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                            .padding()
                    }

                    // Messages (reversed — newest at bottom)
                    let messages = appState.selectedChannelMessages.reversed()
                    ForEach(Array(messages.enumerated()), id: \.element.id) { index, message in
                        let previous: Message? = index > 0 ? Array(messages)[index - 1] : nil

                        // Date separator
                        if let prev = previous, !DateFormatting.isSameDay(prev.createdAt, message.createdAt) {
                            dateSeparator(for: message.createdAt)
                        } else if previous == nil {
                            dateSeparator(for: message.createdAt)
                        }

                        // Message row
                        let isGrouped = shouldGroup(message: message, previous: previous)
                        MessageRow(
                            message: message,
                            isGrouped: isGrouped,
                            currentUserID: appState.currentUser?.id
                        )
                        .id(message.id)
                    }
                }
                .padding(.bottom, 8)
            }
            .defaultScrollAnchor(.bottom)
            .onChange(of: appState.selectedChannelMessages.first?.id) { _, newID in
                if let id = newID {
                    withAnimation(.easeOut(duration: 0.2)) {
                        proxy.scrollTo(id, anchor: .bottom)
                    }
                }
            }
        }
        .background(Color.retroChat)
    }

    private func dateSeparator(for date: Date) -> some View {
        HStack {
            VStack { Divider() }
            Text(DateFormatting.dateSeparator(date))
                .font(.caption)
                .fontWeight(.semibold)
                .foregroundStyle(.retroMuted)
                .fixedSize()
            VStack { Divider() }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }

    /// Group consecutive messages from the same author within 5 minutes.
    private func shouldGroup(message: Message, previous: Message?) -> Bool {
        guard let prev = previous else { return false }
        guard prev.authorID == message.authorID else { return false }
        let interval = message.createdAt.timeIntervalSince(prev.createdAt)
        return abs(interval) < 300 // 5 minutes
    }
}
