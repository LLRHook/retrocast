import SwiftUI

struct SearchView: View {
    @Bindable var viewModel: SearchViewModel
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            searchBar
            Divider()
            resultsList
        }
        .background(Color.retroChat)
    }

    // MARK: - Search bar

    private var searchBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(.retroMuted)
            TextField("Search messages...", text: $viewModel.query)
                .textFieldStyle(.plain)
                .foregroundStyle(.retroText)
                .onSubmit {
                    if let guildID = appState.selectedGuildID {
                        Task { await viewModel.search(guildID: guildID) }
                    }
                }
                .onChange(of: viewModel.query) {
                    if let guildID = appState.selectedGuildID {
                        viewModel.searchDebounced(guildID: guildID)
                    }
                }
            if !viewModel.query.isEmpty {
                Button {
                    viewModel.clear()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.retroMuted)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
    }

    // MARK: - Results

    @ViewBuilder
    private var resultsList: some View {
        if viewModel.isSearching {
            Spacer()
            ProgressView()
                .frame(maxWidth: .infinity)
            Spacer()
        } else if let error = viewModel.errorMessage {
            Spacer()
            VStack(spacing: 8) {
                Image(systemName: "exclamationmark.triangle")
                    .font(.title2)
                    .foregroundStyle(.retroMuted)
                Text(error)
                    .font(.subheadline)
                    .foregroundStyle(.retroMuted)
                    .multilineTextAlignment(.center)
            }
            .padding()
            Spacer()
        } else if viewModel.hasSearched && viewModel.results.isEmpty {
            Spacer()
            VStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .font(.title2)
                    .foregroundStyle(.retroMuted)
                Text("No results found")
                    .font(.subheadline)
                    .foregroundStyle(.retroMuted)
            }
            Spacer()
        } else if viewModel.results.isEmpty {
            Spacer()
            VStack(spacing: 8) {
                Image(systemName: "text.magnifyingglass")
                    .font(.title2)
                    .foregroundStyle(.retroMuted)
                Text("Search messages in this server")
                    .font(.subheadline)
                    .foregroundStyle(.retroMuted)
            }
            Spacer()
        } else {
            ScrollView {
                LazyVStack(spacing: 0) {
                    ForEach(viewModel.results) { message in
                        searchResultRow(message)
                            .contentShape(Rectangle())
                            .onTapGesture {
                                viewModel.selectResult(message)
                                dismiss()
                            }
                        Divider()
                            .padding(.leading, 60)
                    }
                }
            }
        }
    }

    // MARK: - Result row

    private func searchResultRow(_ message: Message) -> some View {
        HStack(alignment: .top, spacing: 12) {
            AvatarView(
                name: message.displayName,
                avatarHash: message.authorAvatarHash,
                size: 32
            )
            .padding(.top, 2)

            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 8) {
                    Text(message.displayName)
                        .font(.subheadline)
                        .fontWeight(.semibold)
                        .foregroundStyle(.retroText)

                    if let channelName = viewModel.channelName(for: message) {
                        Text("#\(channelName)")
                            .font(.caption)
                            .foregroundStyle(.retroMuted)
                    }

                    Spacer()

                    Text(DateFormatting.messageTimestamp(message.createdAt))
                        .font(.caption)
                        .foregroundStyle(.retroMuted)
                }

                Text(message.content)
                    .font(.body)
                    .foregroundStyle(.retroText)
                    .lineLimit(3)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 8)
    }
}
