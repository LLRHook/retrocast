# Retrocast Client — SwiftUI iOS/macOS App

> Native Apple client for Retrocast, the self-hosted Discord clone. Targets iOS 17+ and macOS 14+ with a shared SwiftUI codebase. Aesthetic inspired by Discord circa 2015 meets iOS 7 flat design.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Tech Stack](#tech-stack)
3. [App Structure](#app-structure)
4. [Networking Layer](#networking-layer)
5. [WebSocket Gateway Client](#websocket-gateway-client)
6. [Data Layer](#data-layer)
7. [Screen Map](#screen-map)
8. [UI Component Inventory](#ui-component-inventory)
9. [Navigation Architecture](#navigation-architecture)
10. [Build Phases](#build-phases)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                    SwiftUI Views                     │
│  ServerList │ ChannelSidebar │ ChatView │ MemberList │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                   ViewModels                         │
│  @Observable classes managing UI state per screen    │
└──────────┬───────────────────────┬──────────────────┘
           │                       │
┌──────────▼──────────┐ ┌─────────▼──────────────────┐
│    API Client        │ │   Gateway Client            │
│  URLSession-based    │ │  URLSessionWebSocketTask    │
│  REST calls          │ │  Real-time events           │
└──────────┬──────────┘ └─────────┬──────────────────┘
           │                       │
┌──────────▼───────────────────────▼──────────────────┐
│              App State (AppState singleton)           │
│  Current user, guilds, channels, members, presence   │
│  Source of truth — views observe via @Observable      │
└─────────────────────────────────────────────────────┘
```

---

## Tech Stack

| Layer | Technology | Why |
|-------|-----------|-----|
| **UI** | SwiftUI | Native Apple, declarative, shared iOS/macOS |
| **State** | @Observable (Observation framework) | iOS 17+, no Combine boilerplate, direct property observation |
| **Networking (REST)** | URLSession + async/await | Built-in, no dependencies needed |
| **Networking (WS)** | URLSessionWebSocketTask | Native WebSocket, automatic reconnection support |
| **Persistence** | SwiftData | Local cache for offline message history, user prefs |
| **Keychain** | Security framework | Token storage (access + refresh) |
| **Voice** | LiveKit Swift SDK | WebRTC voice/video (Phase 2) |
| **Images** | AsyncImage + URLCache | Built-in image loading with caching |
| **Navigation** | NavigationSplitView | Three-column layout (servers / channels / chat) |
| **Dependencies** | Swift Package Manager | Zero external deps for Phase 1 (LiveKit SDK added in Phase 2) |

**Target:** iOS 17.0+ / macOS 14.0+ (Sonoma)
**Architecture:** MVVM with @Observable
**Zero external dependencies for Phase 1** — everything uses Foundation/SwiftUI built-ins.

---

## App Structure

```
RetrocastApp/
├── RetrocastApp.swift              # @main entry point
├── Info.plist
│
├── Models/                         # Domain models (mirror server)
│   ├── User.swift                  # id (Int64), username, displayName, avatarHash
│   ├── Guild.swift                 # id, name, iconHash, ownerID
│   ├── Channel.swift               # id, guildID, name, type, position, topic
│   ├── Message.swift               # id, channelID, authorID, content, createdAt, editedAt
│   ├── Member.swift                # guildID, userID, nickname, joinedAt, roles
│   ├── Role.swift                  # id, guildID, name, color, permissions, position
│   ├── Invite.swift                # code, guildID, creatorID, maxUses, uses, expiresAt
│   └── Snowflake.swift             # Int64 wrapper with string JSON coding, timestamp extraction
│
├── Networking/
│   ├── APIClient.swift             # URLSession wrapper, auth header injection, token refresh
│   ├── APIError.swift              # Error types matching server error codes
│   ├── Endpoints.swift             # All endpoint definitions as static functions
│   └── TokenManager.swift          # Keychain storage for access/refresh tokens, auto-refresh
│
├── Gateway/
│   ├── GatewayClient.swift         # WebSocket connection, heartbeat, identify, resume
│   ├── GatewayEvent.swift          # Event types (MESSAGE_CREATE, PRESENCE_UPDATE, etc.)
│   ├── GatewayPayload.swift        # Op codes, payload encoding/decoding
│   └── ReconnectionStrategy.swift  # Exponential backoff with jitter
│
├── State/
│   ├── AppState.swift              # @Observable — current user, guilds, channels, etc.
│   ├── GuildState.swift            # Per-guild state: channels, members, roles
│   ├── ChannelState.swift          # Per-channel state: messages, typing users
│   └── PresenceState.swift         # User presence map (userID → status)
│
├── ViewModels/
│   ├── AuthViewModel.swift         # Login/register flow
│   ├── ServerListViewModel.swift   # Guild list, create/join guild
│   ├── ChannelListViewModel.swift  # Channel sidebar for selected guild
│   ├── ChatViewModel.swift         # Message list, send message, load history
│   ├── MemberListViewModel.swift   # Member sidebar with roles and presence
│   ├── SettingsViewModel.swift     # User settings, server settings
│   └── InviteViewModel.swift       # Create/accept invites
│
├── Views/
│   ├── ContentView.swift           # Root — auth gate + main navigation
│   │
│   ├── Auth/
│   │   ├── LoginView.swift         # Username + password form
│   │   ├── RegisterView.swift      # Registration form
│   │   └── ServerAddressView.swift # Enter server URL (or discover via Bonjour)
│   │
│   ├── Main/
│   │   ├── MainView.swift          # NavigationSplitView (3-column)
│   │   ├── ServerListView.swift    # Left column — guild icons
│   │   ├── ChannelSidebarView.swift # Middle column — channel list + header
│   │   └── ChatAreaView.swift      # Right column — messages + input
│   │
│   ├── Chat/
│   │   ├── MessageListView.swift   # ScrollView with lazy loading, infinite scroll up
│   │   ├── MessageRow.swift        # Single message: avatar, name, content, timestamp
│   │   ├── MessageInput.swift      # Text field + send button + attachment button
│   │   ├── TypingIndicator.swift   # "User is typing..." bar
│   │   └── DateSeparator.swift     # "January 15, 2025" between message groups
│   │
│   ├── Guild/
│   │   ├── CreateGuildSheet.swift  # Create new server
│   │   ├── JoinGuildSheet.swift    # Enter invite code
│   │   ├── GuildSettingsView.swift # Server settings (name, icon, roles)
│   │   └── InviteSheet.swift       # Generate/share invite link
│   │
│   ├── Members/
│   │   ├── MemberListView.swift    # Right sidebar — members grouped by role
│   │   ├── MemberRow.swift         # Avatar + name + presence dot
│   │   └── UserProfilePopover.swift # Tap on member → profile card
│   │
│   └── Settings/
│       ├── UserSettingsView.swift  # Display name, avatar, password
│       └── AppSettingsView.swift   # Theme, notifications
│
├── Components/                     # Reusable UI components
│   ├── AvatarView.swift            # Circle image with fallback initials
│   ├── PresenceDot.swift           # Green/yellow/red/gray status indicator
│   ├── GuildIcon.swift             # Server icon (image or initials)
│   ├── RoleTag.swift               # Colored role pill
│   ├── LoadingView.swift           # Spinner/skeleton
│   └── ErrorBanner.swift           # Toast-style error messages
│
├── Utilities/
│   ├── KeychainHelper.swift        # Keychain read/write/delete
│   ├── DateFormatters.swift        # Relative timestamps ("2 min ago", "Yesterday at 3:42 PM")
│   ├── MarkdownParser.swift        # Basic markdown → AttributedString (bold, italic, code)
│   └── HapticFeedback.swift        # Haptic feedback helpers
│
└── Resources/
    ├── Assets.xcassets             # App icon, color palette
    └── Localizable.strings         # String localization
```

---

## Networking Layer

### APIClient

```swift
@Observable
final class APIClient {
    var baseURL: URL
    private let session = URLSession.shared
    private let tokenManager: TokenManager

    // Generic request with automatic auth header injection + token refresh
    func request<T: Decodable>(_ endpoint: Endpoint) async throws -> T

    // On 401: attempt token refresh, retry once. If refresh fails → logout.
}
```

### Endpoint Pattern

```swift
struct Endpoint {
    let method: HTTPMethod
    let path: String
    let body: Encodable?
    let queryItems: [URLQueryItem]?
    let requiresAuth: Bool
}

// Usage:
extension Endpoint {
    static func login(username: String, password: String) -> Endpoint
    static func getMyGuilds() -> Endpoint
    static func getChannels(guildID: Int64) -> Endpoint
    static func getMessages(channelID: Int64, before: Int64?, limit: Int) -> Endpoint
    static func sendMessage(channelID: Int64, content: String) -> Endpoint
    static func createGuild(name: String) -> Endpoint
    static func createInvite(guildID: Int64, maxUses: Int, maxAge: Int) -> Endpoint
    static func acceptInvite(code: String) -> Endpoint
    // ... all ~30 endpoints
}
```

### Token Management

```swift
final class TokenManager {
    // Store in Keychain (not UserDefaults — tokens are sensitive)
    var accessToken: String? { get/set via Keychain }
    var refreshToken: String? { get/set via Keychain }

    var isLoggedIn: Bool { accessToken != nil }

    func refreshAccessToken() async throws -> String
    func clearTokens() // logout
}
```

---

## WebSocket Gateway Client

### Connection Lifecycle

```
App Launch
    │
    ▼
[Check saved tokens] ──no──▶ [Login Screen]
    │ yes                          │
    ▼                              ▼
[Connect to Gateway]          [POST /auth/login]
    │                              │
    ▼                              ▼
[Send IDENTIFY]          [Store tokens, connect]
    │
    ▼
[Receive READY]
    │
    ▼
[Populate AppState with guilds, channels, members, presence]
    │
    ▼
[Listen for events] ◀──────────────────────────────────┐
    │                                                    │
    ▼                                                    │
[MESSAGE_CREATE → update ChannelState]                  │
[PRESENCE_UPDATE → update PresenceState]                │
[TYPING_START → update typing indicators]               │
[GUILD_CREATE/UPDATE/DELETE → update GuildState]        │
[...etc]                                                │
    │                                                    │
    ▼ (on disconnect)                                    │
[Reconnect with exponential backoff] ───────────────────┘
    │
    ▼ (on reconnect)
[Send RESUME with session_id + last sequence]
    │
    ▼
[Receive missed events, continue listening]
```

### GatewayClient

```swift
@Observable
final class GatewayClient {
    enum State { case disconnected, connecting, connected, resuming }

    var state: State = .disconnected
    private var webSocket: URLSessionWebSocketTask?
    private var sessionID: String?
    private var lastSequence: Int64 = 0
    private var heartbeatTask: Task<Void, Never>?

    func connect(to url: URL, token: String) async
    func disconnect()

    // Heartbeat: send Op 1 every 41.25s, expect Op 11 ACK
    private func startHeartbeat(intervalMs: Int)

    // Read loop: continuously receive messages, dispatch to handlers
    private func readLoop() async

    // Event handler — called for every Op 0 (DISPATCH) event
    var onEvent: ((String, Data) -> Void)?  // (eventName, jsonData)
}
```

### Reconnection Strategy

```swift
struct ReconnectionStrategy {
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

    mutating func reset() { attempt = 0 }
}
```

---

## Data Layer

### AppState (Source of Truth)

```swift
@Observable
final class AppState {
    var currentUser: User?
    var guilds: [Int64: Guild] = [:]          // guildID → Guild
    var channels: [Int64: [Channel]] = [:]     // guildID → sorted channels
    var members: [Int64: [Member]] = [:]       // guildID → members
    var roles: [Int64: [Role]] = [:]           // guildID → sorted roles
    var presence: [Int64: PresenceStatus] = [:] // userID → status
    var typingUsers: [Int64: Set<Int64>] = [:]  // channelID → set of typing userIDs

    // Selected state
    var selectedGuildID: Int64?
    var selectedChannelID: Int64?

    // Computed
    var selectedGuild: Guild? { guilds[selectedGuildID ?? 0] }
    var selectedChannelMessages: [Message] { ... }

    // Message cache: channelID → [Message] (ordered by ID desc)
    var messageCache: [Int64: [Message]] = [:]
    var hasMoreMessages: [Int64: Bool] = [:]  // for pagination
}
```

### Message Loading Pattern

```swift
// Initial load: fetch last 50 messages for selected channel
// Scroll up: fetch 50 more before the oldest loaded message ID
// New message via gateway: prepend to cache
// Never re-fetch messages already in cache

func loadMessages(channelID: Int64, before: Int64? = nil) async throws {
    let messages = try await api.request(.getMessages(
        channelID: channelID,
        before: before,
        limit: 50
    ))

    if before == nil {
        // Initial load — replace
        messageCache[channelID] = messages
    } else {
        // Pagination — append older messages
        messageCache[channelID]?.append(contentsOf: messages)
    }

    hasMoreMessages[channelID] = messages.count == 50
}
```

---

## Screen Map

### Auth Flow
```
ServerAddressView → LoginView ↔ RegisterView → MainView
```

### Main Layout (iPad / macOS = 3-column, iPhone = push navigation)
```
┌──────────┬──────────────────┬────────────────────────────┬──────────┐
│          │                  │                            │          │
│  Server  │  Channel         │  Chat Area                 │  Member  │
│  List    │  Sidebar         │                            │  List    │
│          │                  │  ┌──────────────────────┐  │          │
│  [icon]  │  # general       │  │ MessageRow            │  │  Online │
│  [icon]  │  # random        │  │ MessageRow            │  │  ──────│
│  [icon]  │  # dev           │  │ MessageRow            │  │  @user │
│          │                  │  │ ...                   │  │  @user │
│  [+]     │  🔊 Voice        │  │                       │  │          │
│          │  🔊 Gaming       │  ├──────────────────────┤  │  Offline │
│          │                  │  │ MessageInput          │  │  ──────│
│          │                  │  │ [Type a message...]   │  │  @user │
│          │                  │  └──────────────────────┘  │          │
└──────────┴──────────────────┴────────────────────────────┴──────────┘
     72pt       220pt              flexible                   200pt
```

### iPhone Layout (Compact)
```
ServerList → push → ChannelList → push → ChatView (swipe for members)
```

---

## UI Component Inventory

### Colors (iOS 7-inspired flat palette)

```swift
extension Color {
    static let retroDark    = Color(hex: "#1E1F22")  // Background
    static let retroSidebar = Color(hex: "#2B2D31")  // Sidebar background
    static let retroChat    = Color(hex: "#313338")  // Chat area background
    static let retroAccent  = Color(hex: "#5865F2")  // Discord blurple
    static let retroGreen   = Color(hex: "#23A559")  // Online
    static let retroYellow  = Color(hex: "#F0B232")  // Idle
    static let retroRed     = Color(hex: "#F23F43")  // DND / errors
    static let retroGray    = Color(hex: "#80848E")  // Offline / muted text
    static let retroText    = Color(hex: "#DBDEE1")  // Primary text
    static let retroMuted   = Color(hex: "#949BA4")  // Secondary text
}
```

### Typography

```swift
// Clean, modern, iOS 7-inspired
// System font (SF Pro) throughout — no custom fonts needed
// Sizes:
// - Server name: .title3, .bold
// - Channel name: .body
// - Message author: .subheadline, .semibold
// - Message content: .body
// - Timestamp: .caption, .foregroundStyle(.secondary)
// - Typing indicator: .caption, .foregroundStyle(.secondary)
```

### Key Components

**AvatarView**: Circle with image or initials fallback. Sizes: 24pt (message), 32pt (member list), 40pt (profile), 80pt (settings). Presence dot overlaid at bottom-right.

**GuildIcon**: 48pt circle. Server icon image or 2-letter abbreviation from server name. Selected state: rounded rectangle instead of circle (Discord-style morph).

**MessageRow**: Left-aligned. Avatar (24pt) | name + timestamp on first line | content below. Grouped messages from same author within 5 minutes share the avatar (subsequent messages show only content, indented).

**MessageInput**: Text field with rounded rect border. Send button appears when text is non-empty. Placeholder: "Message #channel-name".

---

## Navigation Architecture

```swift
@main
struct RetrocastApp: App {
    @State private var appState = AppState()
    @State private var apiClient: APIClient
    @State private var gatewayClient: GatewayClient

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environment(appState)
                .environment(apiClient)
                .environment(gatewayClient)
        }
    }
}

struct ContentView: View {
    @Environment(AppState.self) var state

    var body: some View {
        if state.currentUser != nil {
            MainView()
        } else {
            AuthFlowView()
        }
    }
}

struct MainView: View {
    var body: some View {
        NavigationSplitView {
            ServerListView()
        } content: {
            ChannelSidebarView()
        } detail: {
            ChatAreaView()
        }
    }
}
```

---

## Build Phases

### Phase 1: Auth + Server/Channel Navigation (MVP-critical)

- [ ] **Models**: All domain model structs with Codable conformance
- [ ] **APIClient**: URLSession wrapper, endpoint definitions, token injection
- [ ] **TokenManager**: Keychain storage, auto-refresh
- [ ] **GatewayClient**: WebSocket connection, heartbeat, identify, reconnection
- [ ] **AppState**: Core state container
- [ ] **Auth screens**: Server address entry, login, register
- [ ] **Server list**: Display guild icons, selection, create/join guild
- [ ] **Channel sidebar**: List channels for selected guild, selection
- [ ] **Basic chat**: Display messages, send messages, load history on scroll

**This phase gets you to: "I can log in, see my servers, and send/receive messages in real time."**

### Phase 2: Rich Chat + Members

- [ ] **Message grouping**: Consecutive messages from same author collapse
- [ ] **Typing indicators**: Show "User is typing..." bar
- [ ] **Presence**: Online/idle/DND dots on avatars
- [ ] **Member list**: Right sidebar with role grouping and presence
- [ ] **Message editing/deletion**: Long-press context menu
- [ ] **Infinite scroll**: Load older messages on scroll to top
- [ ] **Date separators**: Between message groups on different days
- [ ] **Basic markdown**: Bold, italic, code in messages

### Phase 3: Guild Management

- [ ] **Invite system**: Create invite link, share sheet, accept invite
- [ ] **Role management**: View roles, assign/remove (admin only)
- [ ] **Channel management**: Create/edit/delete channels (admin only)
- [ ] **Server settings**: Edit name, icon
- [ ] **User settings**: Edit display name, avatar
- [ ] **Kick/ban**: Admin moderation actions

### Phase 4: Voice + Media

- [ ] **LiveKit SDK integration**: Swift Package Manager
- [ ] **Voice channel join/leave**: Tap to join, UI for connected state
- [ ] **Voice controls**: Mute, deafen, volume per user
- [ ] **Speaking indicators**: Green ring on avatar when speaking
- [ ] **File uploads**: Image picker → pre-signed URL → MinIO upload
- [ ] **Image previews**: Inline image display in messages

### Phase 5: Polish

- [ ] **Notifications**: Local + APNs push for mentions and DMs
- [ ] **Bonjour discovery**: NWBrowser to find servers on LAN
- [ ] **Offline support**: SwiftData cache, queue messages for send
- [ ] **Haptics**: Feedback on send, receive, navigation
- [ ] **macOS adaptation**: Keyboard shortcuts, menu bar, window management
- [ ] **Dark/light theme**: System-following with manual override

---

## Server Connection Flow

Since this is self-hosted, the client needs to know where the server is:

1. **First launch**: User enters server URL (e.g., `192.168.1.100:8080` or `retrocast.local`)
2. **Save to UserDefaults**: Remember last server
3. **Bonjour discovery** (Phase 5): Auto-discover `_retrocast._tcp` services on LAN
4. **Health check**: `GET /health` to verify server is reachable before showing login

```swift
struct ServerAddressView: View {
    @State private var serverAddress = ""
    @State private var isChecking = false
    @State private var error: String?

    var body: some View {
        VStack {
            TextField("Server address", text: $serverAddress)
                .textInputAutocapitalization(.never)

            Button("Connect") {
                // GET http://{address}/health
                // On success → show LoginView
                // On failure → show error
            }
        }
    }
}
```

---

## What "Done" Looks Like (Client Phase 1)

A user can:
1. Open the app, enter the server address
2. Register a new account or log in
3. See their server list (guild icons on the left)
4. Create a new server or join via invite code
5. Browse channels in a server
6. Tap a text channel and see message history
7. Type and send a message — see it appear in real time
8. See other users' messages appear in real time via WebSocket
9. Close and reopen the app — still logged in (Keychain tokens)
10. Backgrounding and returning reconnects the gateway automatically
