import SwiftUI

struct AppSettingsView: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Form {
                Section("Appearance") {
                    Text("Theme: Dark")
                        .foregroundStyle(.retroText)
                }

                Section("About") {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text("1.0.0")
                            .foregroundStyle(.retroMuted)
                    }
                }
            }
            .scrollContentBackground(.hidden)
            .background(Color.retroDark)
            .navigationTitle("App Settings")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
    }
}
