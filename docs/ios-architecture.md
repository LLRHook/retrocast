# iOS Client Architecture

The iOS/macOS client is a native SwiftUI app using MVVM architecture. Source: `retrocast-client/Retrocast/Sources/`.

## Project Structure

```
Sources/
  RetrocastApp.swift              # App entry point
  Models/                          # Data models (Codable structs)
  ViewModels/                      # @Observable view models
  Views/                           # SwiftUI views organized by feature
    Auth/                          # Login, Register, ServerAddress
    Chat/                          # MessageList, MessageInput, MessageRow, TypingIndicator
    Guild/                         # Create, Join, Settings, Invite, RoleEditor
    Main/                          # ServerList, ChannelSidebar, ChatArea, MainView
    Members/                       # MemberList, MemberRow, UserProfilePopover
    Settings/                      # AppSettings, UserSettings
  Networking/                      # APIClient, TokenManager, Endpoints
  Gateway/                         # GatewayClient, GatewayPayload, GatewayEvent
  State/                           # AppState, PresenceState
  Components/                      # Reusable UI: AvatarView, ErrorBanner, GuildIcon, etc.
  Utilities/                       # Colors, DateFormatters, KeychainHelper, MarkdownParser
```

## Models

All models conform to `Codable` and `Sendable`. IDs use the `Snowflake` type alias (Int64, decoded from JSON strings).

| Model | File | Key Fields |
|-------|------|------------|
| `User` | `Models/User.swift` | id, username, displayName, avatarHash |
| `Guild` | `Models/Guild.swift` | id, name, iconHash, ownerID |
| `Channel` | `Models/Channel.swift` | id, guildID, name, type, position, topic |
| `Message` | `Models/Message.swift` | id, channelID, authorID, content, timestamps, author info |
| `Member` | `Models/Member.swift` | guildID, userID, nickname, roles |
| `Role` | `Models/Role.swift` | id, guildID, name, color, permissions, position |
| `Invite` | `Models/Invite.swift` | code, guildID, maxUses, uses, expiresAt |
| `Attachment` | `Models/Attachment.swift` | id, messageID, filename, contentType, size, url |
| `Snowflake` | `Models/Snowflake.swift` | Type alias for Int64, custom Codable (string <-> int64) |

## State Management

### AppState

`State/AppState.swift` -- Single `@Observable @MainActor` object holding all application state:

```swift
@Observable @MainActor
final class AppState {
    var currentUser: User?
    var guilds: [Snowflake: Guild] = [:]
    var channels: [Snowflake: [Channel]] = [:]       // guildID -> channels
    var members: [Snowflake: [Member]] = [:]          // guildID -> members
    var roles: [Snowflake: [Role]] = [:]              // guildID -> roles
    var presence: [Snowflake: String] = [:]           // userID -> status
    var typingUsers: [Snowflake: Set<Snowflake>] = [:] // channelID -> typing users
    var selectedGuildID: Snowflake?
    var selectedChannelID: Snowflake?
    var messageCache: [Snowflake: [Message]] = [:]    // channelID -> messages
    var hasMoreMessages: [Snowflake: Bool] = [:]
}
```

Key behaviors:
- Messages stored newest-first in arrays
- Typing users auto-expire after 8 seconds
- `hasMoreMessages` tracks whether more history can be loaded (based on page size of 50)
- `reset()` clears all state on logout

### PresenceState

`State/PresenceState.swift` -- Separate presence tracking, updated by gateway events.

## Networking

### APIClient

`Networking/APIClient.swift` -- `@Observable @MainActor` class wrapping `URLSession`.

Key features:
- Unwraps `{"data": <payload>}` envelope automatically
- Custom ISO 8601 date decoding (handles both with and without fractional seconds)
- Automatic 401 retry: on expired access token, refreshes and retries once
- Concurrent refresh coalescing: multiple simultaneous 401s share one refresh task
- `uploadFile()` for multipart form data uploads
- `checkHealth()` to verify server connectivity

```swift
let user: User = try await apiClient.request(.getMe())
try await apiClient.requestVoid(.logout())
```

### TokenManager

`Networking/TokenManager.swift` -- Stores tokens in iOS Keychain via `KeychainHelper`.

- `accessToken` / `refreshToken`: read from Keychain
- `serverURL`: stored in `UserDefaults` (not sensitive)
- `storeTokens()` / `clearTokens()` for auth state management
- `isLoggedIn` computed property

### Endpoints

`Networking/Endpoints.swift` -- Type-safe endpoint definitions as static factory methods on `Endpoint`.

```swift
struct Endpoint: Sendable {
    let method: HTTPMethod
    let path: String
    let body: (any Encodable & Sendable)?
    let queryItems: [URLQueryItem]?
    let requiresAuth: Bool
}

// Usage
Endpoint.login(username: "victor", password: "s3cret")
Endpoint.getMessages(channelID: id, before: cursor, limit: 50)
Endpoint.sendMessage(channelID: id, content: "Hello!")
```

All endpoint bodies use private inline Codable structs for type safety.

## Gateway Client

`Gateway/GatewayClient.swift` -- `@Observable @MainActor` class managing the WebSocket connection.

### States

```swift
enum State: Sendable {
    case disconnected
    case connecting
    case connected
    case resuming
}
```

### Connection Flow

1. `connect(to:token:)` stores URL and token, initiates WebSocket
2. Read loop receives messages, routes by opcode
3. On HELLO (Op 10): starts heartbeat timer, sends IDENTIFY or RESUME
4. On READY: stores session ID, transitions to `.connected`
5. On DISPATCH: extracts event type and data, calls `onEvent` callback
6. On RECONNECT: triggers reconnection flow

### Event Handling

```swift
var onEvent: ((GatewayEventType, Data) -> Void)?
var onStateChange: ((State) -> Void)?
```

ViewModels subscribe to `onEvent` and decode the raw JSON `Data` based on the event type.

### Gateway Event Types

`Gateway/GatewayEvent.swift` -- Enum of 20 dispatch event types matching the server:

```swift
enum GatewayEventType: String, Sendable {
    case ready = "READY"
    case messageCreate = "MESSAGE_CREATE"
    case messageUpdate = "MESSAGE_UPDATE"
    // ... all 20 events
    case guildBanRemove = "GUILD_BAN_REMOVE"
}
```

### Reconnection

`Gateway/ReconnectionStrategy.swift` -- Exponential backoff with jitter:

- Base delay: 1 second, doubles each attempt
- Max delay: 60 seconds
- Max attempts: 10
- Jitter: 0-10% of current delay
- Resets on successful connection

If session ID exists, reconnects in `.resuming` state (attempts RESUME). Otherwise, starts fresh with IDENTIFY.

### Heartbeat

Client sends heartbeat at the interval specified by the server in HELLO (41,250ms). Uses Swift structured concurrency (`Task.sleep`).

## ViewModels

All ViewModels are `@Observable @MainActor`. They hold references to `APIClient`, `AppState`, and/or `GatewayClient`.

| ViewModel | File | Responsibility |
|-----------|------|---------------|
| `AuthViewModel` | `AuthViewModel.swift` | Login, register, server address, token management |
| `ServerListViewModel` | `ServerListViewModel.swift` | Guild list, create/join guild |
| `ChannelListViewModel` | `ChannelListViewModel.swift` | Channel list for selected guild |
| `ChatViewModel` | `ChatViewModel.swift` | Message send, edit, delete, load history, typing |
| `MemberListViewModel` | `MemberListViewModel.swift` | Member list with roles |
| `InviteViewModel` | `InviteViewModel.swift` | Create/accept invites |
| `SettingsViewModel` | `SettingsViewModel.swift` | User settings, logout |

## Views

Organized by feature area:

### Auth
- `ServerAddressView` -- Enter server URL, checks health
- `LoginView` -- Username/password login
- `RegisterView` -- Account creation

### Main (3-column layout)
- `MainView` -- Root view with navigation
- `ServerListView` -- Left column: guild icons
- `ChannelSidebarView` -- Middle column: channel list for selected guild
- `ChatAreaView` -- Right column: messages for selected channel

### Chat
- `MessageListView` -- Scrollable message list with infinite scroll
- `MessageRow` -- Single message with author info, timestamps, attachments
- `MessageInput` -- Text input with send button
- `TypingIndicator` -- "User is typing..." display
- `DateSeparator` -- Date headers between message groups

### Guild
- `CreateGuildSheet` -- New server form
- `JoinGuildSheet` -- Join via invite code
- `GuildSettingsView` -- Server settings (name, roles, etc.)
- `InviteSheet` -- Create/copy invite links
- `RoleEditorView` -- Role permission editor

### Members
- `MemberListView` -- Right sidebar member list grouped by role
- `MemberRow` -- Member avatar, name, presence dot
- `UserProfilePopover` -- Profile popup on member click

### Settings
- `UserSettingsView` -- Display name, avatar
- `AppSettingsView` -- App-level preferences

## Reusable Components

| Component | File | Purpose |
|-----------|------|---------|
| `AvatarView` | `Components/AvatarView.swift` | User avatar with fallback initials |
| `ErrorBanner` | `Components/ErrorBanner.swift` | Dismissible error alert |
| `GuildIcon` | `Components/GuildIcon.swift` | Server icon with fallback letter |
| `LoadingView` | `Components/LoadingView.swift` | Spinner with optional message |
| `PresenceDot` | `Components/PresenceDot.swift` | Colored status indicator dot |
| `RoleTag` | `Components/RoleTag.swift` | Colored role badge |

## Utilities

| Utility | File | Purpose |
|---------|------|---------|
| `Colors` | `Utilities/Colors.swift` | Color constants and helpers |
| `DateFormatters` | `Utilities/DateFormatters.swift` | Shared date formatting |
| `KeychainHelper` | `Utilities/KeychainHelper.swift` | Keychain read/write/delete |
| `MarkdownParser` | `Utilities/MarkdownParser.swift` | Simple markdown to AttributedString |

## Key Patterns

1. **@Observable over ObservableObject**: Uses Swift 5.9 `@Observable` macro instead of `ObservableObject` + `@Published` for more granular observation
2. **@MainActor everywhere**: All ViewModels, AppState, APIClient, and GatewayClient are `@MainActor` isolated
3. **Sendable conformance**: All models and data types conform to `Sendable`
4. **Keychain for tokens**: Access and refresh tokens stored in Keychain, not UserDefaults
5. **Type-safe endpoints**: Each API call is a static method returning an `Endpoint` struct
6. **Centralized state**: Single `AppState` object holds all cached data
7. **Gateway drives UI**: Real-time events update `AppState` directly, SwiftUI observes and re-renders
