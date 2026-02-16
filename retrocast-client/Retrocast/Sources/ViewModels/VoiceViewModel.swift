import Foundation

@Observable @MainActor
final class VoiceViewModel {
    var currentChannelID: Snowflake?
    var currentChannelName: String?
    var isMuted = false
    var isDeafened = false
    var isConnecting = false
    var errorMessage: String?
    var livekitToken: String?

    private let api: APIClient
    private let appState: AppState

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    var isConnected: Bool {
        currentChannelID != nil
    }

    func joinChannel(_ channelID: Snowflake, channelName: String) async {
        guard currentChannelID != channelID else { return }

        // Leave current channel first if connected
        if currentChannelID != nil {
            await leaveChannel()
        }

        isConnecting = true
        errorMessage = nil
        do {
            let response: JoinVoiceResponse = try await api.request(.joinVoice(channelID: channelID))
            livekitToken = response.token
            currentChannelID = channelID
            currentChannelName = channelName
            appState.setVoiceStates(response.voiceStates, for: channelID)
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isConnecting = false
    }

    func leaveChannel() async {
        guard let channelID = currentChannelID else { return }

        do {
            try await api.requestVoid(.leaveVoice(channelID: channelID))
        } catch {
            // Best-effort: clear local state even if the request fails
        }

        livekitToken = nil
        currentChannelID = nil
        currentChannelName = nil
        isMuted = false
        isDeafened = false
    }

    func toggleMute() {
        isMuted.toggle()
        if !isMuted {
            isDeafened = false
        }
    }

    func toggleDeafen() {
        isDeafened.toggle()
        if isDeafened {
            isMuted = true
        }
    }

    func fetchVoiceStates(_ channelID: Snowflake) async {
        do {
            let states: [VoiceState] = try await api.request(.voiceStates(channelID: channelID))
            appState.setVoiceStates(states, for: channelID)
        } catch {
            // Non-critical: voice states will update via gateway events
        }
    }

    /// Handle a VOICE_STATE_UPDATE gateway event.
    func handleVoiceStateUpdate(_ state: VoiceState) {
        // A channelID of 0 means the user left voice
        if state.channelID.rawValue == 0 {
            appState.removeVoiceState(userID: state.userID)
            // If it's us, clear local state
            if state.userID == appState.currentUser?.id {
                livekitToken = nil
                currentChannelID = nil
                currentChannelName = nil
                isMuted = false
                isDeafened = false
            }
        } else {
            appState.updateVoiceState(state)
        }
    }
}
