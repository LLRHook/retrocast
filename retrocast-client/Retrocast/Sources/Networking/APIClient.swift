import Foundation

/// Wraps all server API responses: `{"data": <payload>}`.
private struct APIResponse<T: Decodable>: Decodable {
    let data: T
}

@Observable @MainActor
final class APIClient {
    var baseURL: URL?
    private let session: URLSession
    private let tokenManager: TokenManager
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    /// Set when a token refresh is already in progress to avoid concurrent refreshes.
    private var refreshTask: Task<String, Error>?

    init(tokenManager: TokenManager) {
        self.tokenManager = tokenManager
        self.session = .shared

        self.decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let str = try container.decode(String.self)
            // Try ISO 8601 with fractional seconds first
            if let date = ISO8601DateFormatter.withFractionalSeconds.date(from: str) {
                return date
            }
            if let date = ISO8601DateFormatter.standard.date(from: str) {
                return date
            }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Cannot decode date: \(str)")
        }

        self.encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601

        if let saved = tokenManager.serverURL, let url = URL(string: saved) {
            self.baseURL = url
        }
    }

    // MARK: - Public

    /// Make an API request and decode the response from `{"data": ...}` envelope.
    func request<T: Decodable>(_ endpoint: Endpoint) async throws -> T {
        let (data, _) = try await performRequest(endpoint)
        do {
            let wrapped = try decoder.decode(APIResponse<T>.self, from: data)
            return wrapped.data
        } catch {
            // Some endpoints return bare JSON (not wrapped in data)
            do {
                return try decoder.decode(T.self, from: data)
            } catch {
                throw APIError.decodingError(error)
            }
        }
    }

    /// Make an API request that returns no meaningful body (e.g., DELETE → 204).
    func requestVoid(_ endpoint: Endpoint) async throws {
        let _ = try await performRequest(endpoint)
    }

    /// Make a request and return raw response data (for non-standard responses).
    func requestRaw(_ endpoint: Endpoint) async throws -> Data {
        let (data, _) = try await performRequest(endpoint)
        return data
    }

    // MARK: - Health check

    func checkHealth() async -> Bool {
        guard let baseURL else { return false }
        let url = baseURL.appendingPathComponent("/health")
        do {
            let (_, response) = try await session.data(from: url)
            return (response as? HTTPURLResponse)?.statusCode == 200
        } catch {
            return false
        }
    }

    // MARK: - Internal

    private func performRequest(_ endpoint: Endpoint, isRetry: Bool = false) async throws -> (Data, HTTPURLResponse) {
        guard let baseURL else { throw APIError.invalidURL }

        var components = URLComponents(url: baseURL.appendingPathComponent(endpoint.path), resolvingAgainstBaseURL: false)!
        components.queryItems = endpoint.queryItems

        guard let url = components.url else { throw APIError.invalidURL }

        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue

        if endpoint.requiresAuth, let token = tokenManager.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        if let body = endpoint.body {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try encoder.encode(AnyEncodable(body))
        }

        let (data, response) = try await session.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw APIError.noData
        }

        // Handle 401 — attempt token refresh once
        if httpResponse.statusCode == 401 && endpoint.requiresAuth && !isRetry {
            let newToken = try await refreshAccessToken()
            // Update the request with new token
            var retryEndpoint = endpoint
            _ = newToken // token is stored in tokenManager, next performRequest picks it up
            return try await performRequest(retryEndpoint, isRetry: true)
        }

        // Handle error responses
        if httpResponse.statusCode >= 400 {
            if let errorResponse = try? decoder.decode(APIErrorResponse.self, from: data) {
                throw APIError.serverError(
                    code: errorResponse.error.code,
                    message: errorResponse.error.message,
                    status: httpResponse.statusCode
                )
            }
            throw APIError.serverError(code: "UNKNOWN", message: "Request failed", status: httpResponse.statusCode)
        }

        return (data, httpResponse)
    }

    private func refreshAccessToken() async throws -> String {
        // Coalesce concurrent refresh attempts
        if let existing = refreshTask {
            return try await existing.value
        }

        let task = Task<String, Error> {
            defer { refreshTask = nil }

            guard let refreshToken = tokenManager.refreshToken else {
                tokenManager.clearTokens()
                throw APIError.unauthorized
            }

            let endpoint = Endpoint.refresh(refreshToken: refreshToken)
            let (data, response) = try await performRequest(endpoint, isRetry: true)

            guard response.statusCode == 200 else {
                tokenManager.clearTokens()
                throw APIError.unauthorized
            }

            struct TokenResponse: Decodable {
                let access_token: String
                let refresh_token: String
            }

            let wrapped = try decoder.decode(APIResponse<TokenResponse>.self, from: data)
            tokenManager.storeTokens(
                accessToken: wrapped.data.access_token,
                refreshToken: wrapped.data.refresh_token
            )
            return wrapped.data.access_token
        }

        refreshTask = task
        return try await task.value
    }
}

// MARK: - Helpers

/// Type-erased Encodable wrapper.
private struct AnyEncodable: Encodable {
    private let encode: (Encoder) throws -> Void

    init(_ value: any Encodable) {
        self.encode = { encoder in try value.encode(to: encoder) }
    }

    func encode(to encoder: Encoder) throws {
        try encode(encoder)
    }
}

extension ISO8601DateFormatter {
    nonisolated(unsafe) static let withFractionalSeconds: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()

    nonisolated(unsafe) static let standard: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()
}
