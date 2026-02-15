import SwiftUI

struct DMListView: View {
    let viewModel: DMListViewModel?
    @Environment(AppState.self) private var appState

    var body: some View {
        VStack(spacing: 0) {
            dmHeader

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 2) {
                    // New Message button
                    Button {
                        viewModel?.showNewDM = true
                    } label: {
                        HStack(spacing: 8) {
                            Image(systemName: "plus.circle.fill")
                                .foregroundStyle(.retroGreen)
                            Text("New Message")
                                .font(.body)
                                .foregroundStyle(.retroText)
                            Spacer()
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 8)
                        .background(Color.retroHover.opacity(0.5))
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                    }
                    .buttonStyle(.plain)

                    Divider()
                        .padding(.vertical, 4)

                    // DM conversations
                    ForEach(appState.dms) { dm in
                        dmRow(dm)
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
            }
        }
        .background(Color.retroSidebar)
        .task {
            await viewModel?.loadDMs()
        }
        .sheet(isPresented: Binding(
            get: { viewModel?.showNewDM ?? false },
            set: { viewModel?.showNewDM = $0 }
        )) {
            if let vm = viewModel {
                NewDMSheet(viewModel: vm)
            }
        }
    }

    private var dmHeader: some View {
        HStack {
            Text("Direct Messages")
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

    private func dmRow(_ dm: DMChannel) -> some View {
        let isSelected = appState.selectedChannelID == dm.id

        return Button {
            appState.selectDMChannel(dm.id)
        } label: {
            HStack(spacing: 8) {
                AvatarView(
                    name: dm.displayName,
                    avatarHash: dm.recipient?.avatarHash,
                    size: 32
                )

                Text(dm.displayName)
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

// MARK: - New DM Sheet

struct NewDMSheet: View {
    let viewModel: DMListViewModel
    @Environment(\.dismiss) private var dismiss
    @Environment(APIClient.self) private var api

    @State private var recipientIDText = ""
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Recipient User ID", text: $recipientIDText)
                        #if os(iOS)
                        .keyboardType(.numberPad)
                        #endif
                } header: {
                    Text("Start a conversation")
                } footer: {
                    Text("Enter the user ID of the person you want to message.")
                }

                if let error = errorMessage ?? viewModel.errorMessage {
                    Section {
                        Text(error)
                            .foregroundStyle(.retroRed)
                    }
                }
            }
            .navigationTitle("New Message")
            #if os(iOS)
            .navigationBarTitleDisplayMode(.inline)
            #endif
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Start") {
                        Task { await startDM() }
                    }
                    .disabled(recipientIDText.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
        }
    }

    private func startDM() async {
        let text = recipientIDText.trimmingCharacters(in: .whitespaces)
        guard let rawValue = Int64(text) else {
            errorMessage = "Please enter a valid user ID."
            return
        }
        let recipientID = Snowflake(rawValue)
        errorMessage = nil
        await viewModel.createDM(recipientID: recipientID)
    }
}
