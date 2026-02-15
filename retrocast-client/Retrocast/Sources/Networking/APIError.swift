import Foundation

/// Error response from the server: `{"error": {"code": "...", "message": "..."}}`
struct APIErrorResponse: Codable, Sendable {
    let error: APIErrorBody
}

struct APIErrorBody: Codable, Sendable {
    let code: String
    let message: String
}

/// Client-side API error types.
enum APIError: LocalizedError {
    case serverError(code: String, message: String, status: Int)
    case networkError(Error)
    case decodingError(Error)
    case unauthorized
    case noData
    case invalidURL

    var errorDescription: String? {
        switch self {
        case .serverError(_, let message, _):
            return message
        case .networkError(let error):
            return "Network error: \(error.localizedDescription)"
        case .decodingError(let error):
            return "Decoding error: \(error.localizedDescription)"
        case .unauthorized:
            return "Session expired. Please log in again."
        case .noData:
            return "No data received."
        case .invalidURL:
            return "Invalid server URL."
        }
    }

    var isUnauthorized: Bool {
        if case .serverError(_, _, let status) = self { return status == 401 }
        if case .unauthorized = self { return true }
        return false
    }
}
