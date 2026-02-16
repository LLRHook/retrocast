import SwiftUI

struct VoiceControlBar: View {
    @Bindable var voiceVM: VoiceViewModel

    var body: some View {
        VStack(spacing: 0) {
            Divider()
            HStack(spacing: 12) {
                // Connection info
                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 4) {
                        Circle()
                            .fill(Color.retroGreen)
                            .frame(width: 8, height: 8)
                        Text("Voice Connected")
                            .font(.caption)
                            .fontWeight(.semibold)
                            .foregroundStyle(.retroGreen)
                    }
                    if let name = voiceVM.currentChannelName {
                        Text(name)
                            .font(.caption2)
                            .foregroundStyle(.retroMuted)
                            .lineLimit(1)
                    }
                }

                Spacer()

                // Mute button
                Button {
                    voiceVM.toggleMute()
                } label: {
                    Image(systemName: voiceVM.isMuted ? "mic.slash.fill" : "mic.fill")
                        .font(.subheadline)
                        .foregroundStyle(voiceVM.isMuted ? .retroRed : .retroMuted)
                        .frame(width: 28, height: 28)
                        .background(Color.retroInput)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }
                .buttonStyle(.plain)

                // Deafen button
                Button {
                    voiceVM.toggleDeafen()
                } label: {
                    Image(systemName: voiceVM.isDeafened ? "speaker.slash.fill" : "speaker.wave.2.fill")
                        .font(.subheadline)
                        .foregroundStyle(voiceVM.isDeafened ? .retroRed : .retroMuted)
                        .frame(width: 28, height: 28)
                        .background(Color.retroInput)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }
                .buttonStyle(.plain)

                // Disconnect button
                Button {
                    Task {
                        await voiceVM.leaveChannel()
                    }
                } label: {
                    Image(systemName: "phone.down.fill")
                        .font(.subheadline)
                        .foregroundStyle(.retroRed)
                        .frame(width: 28, height: 28)
                        .background(Color.retroInput)
                        .clipShape(RoundedRectangle(cornerRadius: 4))
                }
                .buttonStyle(.plain)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(Color.retroDark)
        }
    }
}
