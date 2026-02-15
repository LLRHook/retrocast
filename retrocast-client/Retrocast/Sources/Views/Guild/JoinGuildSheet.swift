import SwiftUI

struct JoinGuildSheet: View {
    @Bindable var viewModel: ServerListViewModel
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                VStack(spacing: 8) {
                    Image(systemName: "link.circle.fill")
                        .font(.system(size: 48))
                        .foregroundStyle(.retroAccent)
                    Text("Join a Server")
                        .font(.title2)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroText)
                    Text("Enter an invite code to join")
                        .font(.subheadline)
                        .foregroundStyle(.retroMuted)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text("INVITE CODE")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundStyle(.retroMuted)
                    TextField("e.g. ab12cd34", text: $viewModel.inviteCode)
                        .textFieldStyle(.roundedBorder)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                }

                if let error = viewModel.errorMessage {
                    ErrorBanner(message: error) {
                        viewModel.errorMessage = nil
                    }
                }

                Button {
                    Task {
                        await viewModel.joinGuild()
                        if viewModel.errorMessage == nil {
                            dismiss()
                        }
                    }
                } label: {
                    Group {
                        if viewModel.isLoading {
                            ProgressView().tint(.white)
                        } else {
                            Text("Join Server")
                        }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 12)
                }
                .buttonStyle(.borderedProminent)
                .tint(.retroAccent)
                .disabled(viewModel.isLoading || viewModel.inviteCode.isEmpty)

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
