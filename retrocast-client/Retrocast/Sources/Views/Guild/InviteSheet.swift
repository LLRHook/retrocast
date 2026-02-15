import SwiftUI

struct InviteSheet: View {
    let guildID: Snowflake
    @Environment(APIClient.self) private var api
    @Environment(\.dismiss) private var dismiss
    @State private var viewModel: InviteViewModel?

    var body: some View {
        NavigationStack {
            VStack(spacing: 16) {
                if let code = viewModel?.generatedCode {
                    // Show generated invite
                    VStack(spacing: 12) {
                        Text("Invite Link")
                            .font(.headline)
                            .foregroundStyle(.retroText)

                        HStack {
                            Text(code)
                                .font(.system(.title3, design: .monospaced))
                                .foregroundStyle(.retroText)
                                .padding()
                                .frame(maxWidth: .infinity)
                                .background(Color.retroInput)
                                .clipShape(RoundedRectangle(cornerRadius: 8))

                            Button {
                                UIPasteboard.general.string = code
                            } label: {
                                Image(systemName: "doc.on.doc")
                                    .font(.title3)
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.retroAccent)
                        }
                    }
                } else {
                    // Generate invite
                    Button {
                        Task { await viewModel?.createInvite(guildID: guildID) }
                    } label: {
                        Label("Generate Invite Code", systemImage: "link.badge.plus")
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 12)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.retroAccent)
                }

                // Existing invites
                if let invites = viewModel?.invites, !invites.isEmpty {
                    Divider()
                    Text("Active Invites")
                        .font(.headline)
                        .foregroundStyle(.retroText)
                        .frame(maxWidth: .infinity, alignment: .leading)

                    ForEach(invites) { invite in
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(invite.code)
                                    .font(.subheadline)
                                    .fontWeight(.medium)
                                    .foregroundStyle(.retroText)
                                Text("Uses: \(invite.uses)/\(invite.maxUses == 0 ? "unlimited" : String(invite.maxUses))")
                                    .font(.caption)
                                    .foregroundStyle(.retroMuted)
                            }
                            Spacer()
                            Button(role: .destructive) {
                                Task { await viewModel?.revokeInvite(code: invite.code) }
                            } label: {
                                Image(systemName: "trash")
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }

                Spacer()
            }
            .padding(24)
            .background(Color.retroDark)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
        .task {
            let vm = InviteViewModel(api: api)
            viewModel = vm
            await vm.loadInvites(guildID: guildID)
        }
        .presentationDetents([.medium, .large])
    }
}
