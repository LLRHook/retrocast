import Foundation

@Observable
final class TokenManager: Sendable {
    private static let accessTokenKey = "access_token"
    private static let refreshTokenKey = "refresh_token"
    private static let serverURLKey = "server_url"

    nonisolated var accessToken: String? {
        KeychainHelper.loadString(key: TokenManager.accessTokenKey)
    }

    nonisolated var refreshToken: String? {
        KeychainHelper.loadString(key: TokenManager.refreshTokenKey)
    }

    var isLoggedIn: Bool {
        accessToken != nil
    }

    // Server URL stored in UserDefaults (not sensitive)
    var serverURL: String? {
        get { UserDefaults.standard.string(forKey: TokenManager.serverURLKey) }
        set { UserDefaults.standard.set(newValue, forKey: TokenManager.serverURLKey) }
    }

    func storeTokens(accessToken: String, refreshToken: String) {
        try? KeychainHelper.save(accessToken, for: TokenManager.accessTokenKey)
        try? KeychainHelper.save(refreshToken, for: TokenManager.refreshTokenKey)
    }

    func clearTokens() {
        KeychainHelper.delete(key: TokenManager.accessTokenKey)
        KeychainHelper.delete(key: TokenManager.refreshTokenKey)
    }
}
