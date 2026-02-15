import Foundation

enum PresenceStatus: String, Sendable {
    case online
    case idle
    case dnd
    case offline

    var label: String {
        switch self {
        case .online: return "Online"
        case .idle: return "Idle"
        case .dnd: return "Do Not Disturb"
        case .offline: return "Offline"
        }
    }
}
