import Foundation

/// Exponential backoff with jitter for WebSocket reconnection.
struct ReconnectionStrategy: Sendable {
    var attempt = 0
    let maxAttempts = 10
    let baseDelay: TimeInterval = 1.0
    let maxDelay: TimeInterval = 60.0

    mutating func nextDelay() -> TimeInterval {
        let delay = min(baseDelay * pow(2.0, Double(attempt)), maxDelay)
        let jitter = Double.random(in: 0...delay * 0.1)
        attempt += 1
        return delay + jitter
    }

    mutating func reset() {
        attempt = 0
    }

    var canRetry: Bool {
        attempt < maxAttempts
    }
}
