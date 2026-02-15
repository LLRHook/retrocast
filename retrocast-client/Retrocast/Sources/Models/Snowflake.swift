import Foundation

/// A Snowflake ID — Int64 that encodes as a JSON string to avoid JS precision loss.
struct Snowflake: Hashable, Comparable, Sendable {
    let rawValue: Int64

    init(_ value: Int64) {
        self.rawValue = value
    }

    /// Extract the Unix timestamp from this snowflake.
    /// Custom epoch: Jan 1 2025 00:00:00 UTC.
    var timestamp: Date {
        let customEpoch: Int64 = 1_735_689_600_000 // 2025-01-01 in ms
        let ms = (rawValue >> 22) + customEpoch
        return Date(timeIntervalSince1970: TimeInterval(ms) / 1000.0)
    }

    static func < (lhs: Snowflake, rhs: Snowflake) -> Bool {
        lhs.rawValue < rhs.rawValue
    }
}

extension Snowflake: Codable {
    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let str = try? container.decode(String.self),
           let value = Int64(str) {
            self.rawValue = value
        } else {
            self.rawValue = try container.decode(Int64.self)
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        try container.encode(String(rawValue))
    }
}

extension Snowflake: CustomStringConvertible {
    var description: String { String(rawValue) }
}
