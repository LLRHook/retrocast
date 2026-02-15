import SwiftUI

struct CreateChannelSheet: View {
    @Bindable var viewModel: ChannelListViewModel
    let guildID: Snowflake
    let categories: [Channel]
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                VStack(spacing: 8) {
                    Image(systemName: "number")
                        .font(.system(size: 48))
                        .foregroundStyle(.retroAccent)
                    Text("Create a Channel")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroText)
                    Text("Give your new channel a name and type")
                        .font(.subheadline)
                        .foregroundStyle(.retroMuted)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("CHANNEL NAME")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroMuted)
                    TextField("new-channel", text: $viewModel.newChannelName)
                        .textFieldStyle(.roundedBorder)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("CHANNEL TYPE")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroMuted)
                    Picker("Type", selection: $viewModel.newChannelType) {
                        Label("Text", systemImage: "number")
                            .tag(ChannelType.text)
                        Label("Voice", systemImage: "speaker.wave.2")
                            .tag(ChannelType.voice)
                        Label("Category", systemImage: "folder")
                            .tag(ChannelType.category)
                    }
                    .pickerStyle(.segmented)
                }

                if viewModel.newChannelType != .category && !categories.isEmpty {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("CATEGORY")
                            .font(.caption)
                            .fontWeight(.bold)
                            .foregroundStyle(.retroMuted)
                        Picker("Category", selection: $viewModel.newChannelParentID) {
                            Text("None").tag(Snowflake?.none)
                            ForEach(categories) { category in
                                Text(category.name).tag(Optional(category.id))
                            }
                        }
                    }
                }

                if let error = viewModel.errorMessage {
                    ErrorBanner(message: error) {
                        viewModel.errorMessage = nil
                    }
                }

                Button {
                    Task {
                        await viewModel.createChannel(guildID: guildID)
                        if viewModel.errorMessage == nil {
                            dismiss()
                        }
                    }
                } label: {
                    Group {
                        if viewModel.isLoading {
                            ProgressView().tint(.white)
                        } else {
                            Text("Create")
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .tint(.retroAccent)
                .disabled(viewModel.isLoading || viewModel.newChannelName.isEmpty)

                Spacer()
            }
            .padding(24)
            .background(Color.retroDark)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium])
    }
}
