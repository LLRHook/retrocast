import SwiftUI

// MARK: - Permission Definition

struct Permission: Identifiable, Sendable {
    let name: String
    let bit: Int64
    var id: Int64 { bit }
}

enum PermissionGroup: String, CaseIterable {
    case general = "General"
    case membership = "Membership"
    case text = "Text"
    case voice = "Voice"

    var permissions: [Permission] {
        switch self {
        case .general:
            return [
                Permission(name: "Administrator", bit: 1 << 31),
                Permission(name: "Manage Guild", bit: 1 << 7),
                Permission(name: "Manage Channels", bit: 1 << 3),
                Permission(name: "Manage Roles", bit: 1 << 4),
                Permission(name: "Manage Nicknames", bit: 1 << 18),
            ]
        case .membership:
            return [
                Permission(name: "Create Invite", bit: 1 << 16),
                Permission(name: "Kick Members", bit: 1 << 5),
                Permission(name: "Ban Members", bit: 1 << 6),
            ]
        case .text:
            return [
                Permission(name: "View Channel", bit: 1 << 0),
                Permission(name: "Send Messages", bit: 1 << 1),
                Permission(name: "Manage Messages", bit: 1 << 2),
                Permission(name: "Attach Files", bit: 1 << 14),
                Permission(name: "Read Message History", bit: 1 << 15),
                Permission(name: "Mention Everyone", bit: 1 << 13),
            ]
        case .voice:
            return [
                Permission(name: "Connect", bit: 1 << 8),
                Permission(name: "Speak", bit: 1 << 9),
                Permission(name: "Mute Members", bit: 1 << 10),
                Permission(name: "Deafen Members", bit: 1 << 11),
                Permission(name: "Move Members", bit: 1 << 12),
            ]
        }
    }
}

// MARK: - Role Editor View

struct RoleEditorView: View {
    let guildID: Snowflake
    let role: Role?

    @Environment(APIClient.self) private var api
    @Environment(AppState.self) private var appState
    @Environment(\.dismiss) private var dismiss

    @State private var name: String = ""
    @State private var roleColor: Color = .retroMuted
    @State private var permissions: Int64 = 0
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var showDeleteConfirmation = false

    private var isEditing: Bool { role != nil }

    var body: some View {
        NavigationStack {
            Form {
                Section("Role Name") {
                    TextField("Role name", text: $name)
                }

                Section("Color") {
                    ColorPicker("Role Color", selection: $roleColor, supportsOpacity: false)
                }

                ForEach(PermissionGroup.allCases, id: \.rawValue) { group in
                    Section(group.rawValue) {
                        ForEach(group.permissions) { perm in
                            Toggle(perm.name, isOn: permissionBinding(for: perm.bit))
                        }
                    }
                }

                if let error = errorMessage {
                    Section {
                        Text(error)
                            .foregroundStyle(.red)
                    }
                }

                if isEditing, let role, !role.isDefault {
                    Section {
                        Button("Delete Role", role: .destructive) {
                            showDeleteConfirmation = true
                        }
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Color.retroDark)
            .navigationTitle(isEditing ? "Edit Role" : "Create Role")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task { await save() }
                    }
                    .disabled(isLoading || name.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .confirmationDialog("Delete Role", isPresented: $showDeleteConfirmation, titleVisibility: .visible) {
                Button("Delete", role: .destructive) {
                    Task { await deleteRole() }
                }
            } message: {
                Text("Are you sure you want to delete this role? This action cannot be undone.")
            }
        }
        .onAppear {
            if let role {
                name = role.name
                permissions = role.permissions
                roleColor = colorFromInt(role.color)
            }
        }
    }

    // MARK: - Permission Binding

    private func permissionBinding(for bit: Int64) -> Binding<Bool> {
        Binding(
            get: { permissions & bit == bit },
            set: { enabled in
                if enabled {
                    permissions |= bit
                } else {
                    permissions &= ~bit
                }
            }
        )
    }

    // MARK: - Color Conversion

    private func colorFromInt(_ value: Int) -> Color {
        if value == 0 { return .retroMuted }
        let r = Double((value >> 16) & 0xFF) / 255.0
        let g = Double((value >> 8) & 0xFF) / 255.0
        let b = Double(value & 0xFF) / 255.0
        return Color(red: r, green: g, blue: b)
    }

    private func colorToInt(_ color: Color) -> Int {
        let resolved = color.resolve(in: EnvironmentValues())
        let r = Int(min(max(resolved.red, 0), 1) * 255)
        let g = Int(min(max(resolved.green, 0), 1) * 255)
        let b = Int(min(max(resolved.blue, 0), 1) * 255)
        return (r << 16) | (g << 8) | b
    }

    // MARK: - Actions

    private func save() async {
        isLoading = true
        errorMessage = nil

        let trimmedName = name.trimmingCharacters(in: .whitespaces)
        let colorInt = colorToInt(roleColor)

        do {
            if let existingRole = role {
                let updated: Role = try await api.request(
                    .updateRole(
                        guildID: guildID,
                        roleID: existingRole.id,
                        name: trimmedName,
                        color: colorInt,
                        permissions: permissions
                    )
                )
                updateRoleInState(updated)
            } else {
                let created: Role = try await api.request(
                    .createRole(
                        guildID: guildID,
                        name: trimmedName,
                        permissions: permissions,
                        color: colorInt
                    )
                )
                appendRoleToState(created)
            }
            dismiss()
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }

        isLoading = false
    }

    private func deleteRole() async {
        guard let role else { return }
        isLoading = true
        errorMessage = nil

        do {
            try await api.requestVoid(.deleteRole(guildID: guildID, roleID: role.id))
            removeRoleFromState(role.id)
            dismiss()
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }

        isLoading = false
    }

    // MARK: - State Helpers

    private func updateRoleInState(_ updated: Role) {
        guard var roles = appState.roles[guildID] else { return }
        if let index = roles.firstIndex(where: { $0.id == updated.id }) {
            roles[index] = updated
        }
        appState.roles[guildID] = roles
    }

    private func appendRoleToState(_ created: Role) {
        if appState.roles[guildID] == nil {
            appState.roles[guildID] = []
        }
        appState.roles[guildID]?.append(created)
    }

    private func removeRoleFromState(_ roleID: Snowflake) {
        appState.roles[guildID]?.removeAll { $0.id == roleID }
    }
}
