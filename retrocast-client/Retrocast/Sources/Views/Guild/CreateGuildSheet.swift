import SwiftUI

struct CreateGuildSheet: View {
    @Bindable var viewModel: ServerListViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                VStack(spacing: 8) {
                    Image(systemName: "plus.circle.fill")
                        .font(.system(size: 48))
                        .foregroundStyle(.retroAccent)
                    Text("Create a Server")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroText)
                    Text("Give your new server a name")
                        .font(.subheadline)
                        .foregroundStyle(.retroMuted)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("SERVER NAME")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroMuted)
                    TextField("My Server", text: $viewModel.newGuildName)
                        .textFieldStyle(.roundedBorder)
                }

                if let error = viewModel.errorMessage {
                    ErrorBanner(message: error) {
                        viewModel.errorMessage = nil
                    }
                }

                Button {
                    Task {
                        await viewModel.createGuild()
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
                .disabled(viewModel.isLoading || viewModel.newGuildName.isEmpty)

                Divider()

                // Join server option
                Button {
                    viewModel.showCreateGuild = false
                    viewModel.showJoinGuild = true
                } label: {
                    Text("Or join an existing server")
                        .font(.subheadline)
                        .foregroundStyle(.retroAccent)
                }

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
