import SwiftUI

struct LoadingView: View {
    var message: String?

    var body: some View {
        VStack(spacing: 12) {
            ProgressView()
                .tint(.retroAccent)
            if let message {
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(.retroMuted)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color.retroDark)
    }
}
