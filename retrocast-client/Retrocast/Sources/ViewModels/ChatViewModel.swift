import Foundation

@Observable @MainActor
final class ChatViewModel {
    var messageText = ""
    var isLoading = false
    var isSending = false
    var isLoadingMore = false
    var isUploading = false
    var errorMessage: String?

    private let api: APIClient
    private let appState: AppState

    init(api: APIClient, appState: AppState) {
        self.api = api
        self.appState = appState
    }

    // MARK: - Load messages

    func loadMessages(channelID: Snowflake) async {
        guard !isLoading else { return }
        isLoading = true
        errorMessage = nil

        do {
            let messages: [Message] = try await api.request(.getMessages(channelID: channelID))
            appState.setMessages(messages, for: channelID)
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoading = false
    }

    /// Load older messages (cursor pagination).
    func loadMoreMessages(channelID: Snowflake) async {
        guard !isLoadingMore else { return }
        guard appState.hasMoreMessages[channelID] ?? true else { return }

        let messages = appState.messageCache[channelID] ?? []
        guard let oldest = messages.last else { return }

        isLoadingMore = true
        do {
            let older: [Message] = try await api.request(
                .getMessages(channelID: channelID, before: oldest.id)
            )
            appState.setMessages(older, for: channelID, append: true)
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isLoadingMore = false
    }

    // MARK: - Send message

    func sendMessage(channelID: Snowflake) async {
        let content = messageText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !content.isEmpty else { return }
        guard !isSending else { return }

        isSending = true
        messageText = ""

        do {
            let _: Message = try await api.request(
                .sendMessage(channelID: channelID, content: content)
            )
            // Message will arrive via gateway — no need to manually add
        } catch {
            // Restore the message text on failure
            messageText = content
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isSending = false
    }

    // MARK: - Upload attachment

    func uploadAttachment(channelID: Snowflake, data: Data, filename: String, contentType: String) async {
        guard !isUploading else { return }
        isUploading = true
        errorMessage = nil

        do {
            let attachment = try await api.uploadFile(
                channelID: channelID,
                data: data,
                filename: filename,
                contentType: contentType
            )
            // Send a message with the attachment URL so it appears in chat
            let _: Message = try await api.request(
                .sendMessage(channelID: channelID, content: attachment.url)
            )
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
        isUploading = false
    }

    // MARK: - Edit / Delete

    func editMessage(channelID: Snowflake, messageID: Snowflake, newContent: String) async {
        do {
            let updated: Message = try await api.request(
                .editMessage(channelID: channelID, messageID: messageID, content: newContent)
            )
            appState.updateMessage(updated, in: channelID)
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    func deleteMessage(channelID: Snowflake, messageID: Snowflake) async {
        do {
            try await api.requestVoid(.deleteMessage(channelID: channelID, messageID: messageID))
            appState.deleteMessage(id: messageID, from: channelID)
        } catch {
            errorMessage = (error as? APIError)?.errorDescription ?? error.localizedDescription
        }
    }

    // MARK: - Typing

    func sendTyping(channelID: Snowflake) async {
        try? await api.requestVoid(.sendTyping(channelID: channelID))
    }
}
