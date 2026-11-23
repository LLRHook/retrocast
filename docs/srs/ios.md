# SRS component — iOS client

Part of [docs/SRS.md](../SRS.md). Prefix `IOS-`. Component: CMP-009 (retrocast-client,
SwiftUI app; iOS 17 deployment target, Swift 6.0, XcodeGen `project.yml`). Server behaviour it
relies on is in [server.md](server.md).

Conventions for this file:
- **Evidence** paths are relative to `retrocast-client/Retrocast/Sources/`.
- The iOS client has no tests and no CI job, and it cannot be built on the Windows review host
  (Stack Profile flag 4). Every requirement is therefore `Partial`: the Swift source is
  present, but it was not compiled or run in this review.
- iOS requirements are `Should` until the owner sets client priority (TBD-004); the web
  client's MVP requirements are `Must`.
- Defect ids (D-nn) are listed in
  [ARCHITECTURE_SPEC.md §9](../ARCHITECTURE_SPEC.md#9-risks-and-technical-debt).

## 3. Requirements

### 3.1 Functions

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| IOS-FR-001 | Should | The iOS client shall ask for the server address, then let the user register or sign in. | Given a fresh install, when the user enters a server address and signs in as alice, then the server list opens. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — ServerAddressView; LoginView; RegisterView | API-001, API-002 | CAP-001 | Views/Auth/ServerAddressView.swift; Views/Auth/LoginView.swift; Views/Auth/RegisterView.swift; ViewModels/AuthViewModel.swift |
| IOS-FR-002 | Should | The iOS client shall keep its tokens in the Keychain (server address in user defaults) and refresh the access token once on a 401. | Given an access token older than 15 minutes, when the user opens a channel, then the request succeeds after one refresh and the rotated refresh token is saved in the Keychain. | Demonstration | Partial (Swift source present; not compiled or run in this review; the refresh path decodes only a `data`-wrapped body, which API-003 does not send, D-14) | none (system-only) | API-003 | CAP-002 | Networking/TokenManager.swift:10–24; Utilities/KeychainHelper.swift; Networking/APIClient.swift:160–215 |
| IOS-FR-003 | Should | The iOS client shall keep a gateway connection with heartbeats and reconnect with backoff after a drop. | Given a signed-in user, when the network drops and returns, then the client reconnects and receives new messages without a restart. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — ErrorBanner | EVT-001–EVT-007 | CAP-009 | Gateway/GatewayClient.swift; Gateway/ReconnectionStrategy.swift; Gateway/GatewayPayload.swift |
| IOS-FR-004 | Should | The iOS client shall show the user's servers with their channels (create a server, join one by invite code, create a channel). | Given an invite code, when the user joins with it, then the server appears in the server list. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — ServerListView; ChannelSidebarView; CreateGuildSheet; JoinGuildSheet; CreateChannelSheet | API-007, API-013, API-017, API-018, API-050 | CAP-004, CAP-005, CAP-006 | Views/Main/ServerListView.swift; Views/Main/ChannelSidebarView.swift; Views/Guild/*.swift; ViewModels/ServerListViewModel.swift; ViewModels/ChannelListViewModel.swift |
| IOS-FR-005 | Should | The iOS client shall show a channel's messages live (history paging, sending, typing display). | Given an open channel, when another member sends "hi", then it appears at the bottom without a refresh. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — ChatAreaView; MessageListView; MessageInput; TypingIndicator | API-022, API-023, API-028, EVT-009, EVT-024 | CAP-009, CAP-010, CAP-011 | Views/Main/ChatAreaView.swift; Views/Chat/*.swift; ViewModels/ChatViewModel.swift |
| IOS-FR-006 | Should | The iOS client shall list a server's members with their presence and roles. | Given a member going idle, when EVT-025 arrives, then their presence dot changes. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — MemberListView; MemberRow; PresenceDot; UserProfilePopover | API-033, EVT-025 | CAP-012 | Views/Members/*.swift; Components/PresenceDot.swift; State/PresenceState.swift; ViewModels/MemberListViewModel.swift |
| IOS-FR-007 | Should | The iOS client shall let a member with `CREATE_INVITE` create and share an invite code. | Given an admin, when they create an invite, then the code is shown with a share action. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — InviteSheet | API-047 | CAP-008 | Views/Guild/InviteSheet.swift; ViewModels/InviteViewModel.swift |
| IOS-FR-008 | Should | The iOS client shall let members who may manage roles edit a server's roles and settings. | Given the owner, when they rename the server in settings, then the new name shows in the server list. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — GuildSettingsView; RoleEditorView | API-015, API-039–API-042 | CAP-006, CAP-014 | Views/Guild/GuildSettingsView.swift; Views/Guild/RoleEditorView.swift |
| IOS-FR-009 | Could | The iOS client shall list the user's direct messages and open a conversation. | Given a DM with bob, when the user opens the DM list, then the conversation with bob is listed. | Demonstration | Partial (Swift source present; not compiled or run in this review; DM messages fail on the server, D-08) | ios — DMListView | API-008, API-009 | CAP-020 | Views/Main/DMListView.swift; ViewModels/DMListViewModel.swift |
| IOS-FR-010 | Could | The iOS client shall search a server's messages. | Given messages containing "pizza", when the user searches "pizza", then those messages are listed. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — SearchView | API-055 | CAP-028 | Views/Chat/SearchView.swift; ViewModels/SearchViewModel.swift |
| IOS-FR-011 | Could | The iOS client shall join a voice channel and carry the user's audio through LiveKit. | Given two users in one voice channel, when one speaks, then the other hears them. | Demonstration | Partial (the client requests a LiveKit token and shows voice state; no LiveKit package is declared in project.yml, so no audio is connected) | ios — VoiceChannelView; VoiceControlBar | API-056, API-057, EVT-026 | CAP-017 | ViewModels/VoiceViewModel.swift:37; Views/Main/VoiceChannelView.swift; Views/Main/VoiceControlBar.swift; project.yml |
| IOS-FR-012 | Should | The iOS client shall render message Markdown without running content as code. | Given a message with `**bold**`, when it is shown, then the text is bold. | Demonstration | Partial (Swift source present; not compiled or run in this review) | ios — MessageRow | EVT-009 | CAP-009, RISK-011 | Utilities/MarkdownParser.swift; Views/Chat/MessageRow.swift |

### 3.2 Performance requirements
No iOS-specific targets are stated; system targets are SYS-NFR-001 and SYS-NFR-002 (TBD-031).

### 3.3 Usability requirements
No usability or accessibility targets are stated (TBD-032). The iOS 7-era look (CAP-026) is a
Later capability with no requirement yet.

### 3.4 Interface requirements
The client uses SV-IR-001–SV-IR-004. `Networking/APIClient.swift:47–61` accepts both wrapped
and bare JSON bodies for ordinary requests; the refresh path accepts only the wrapped form.

### 3.5 Logical database requirements
The client stores tokens in the Keychain and the server address in user defaults
(IOS-FR-002); it has no local message store.

### 3.6 Design constraints
Bound by SYS-CON-003 (SwiftUI, iOS 17 deployment target, Swift 6.0, XcodeGen). Push
notifications (CAP-023) would need the Apple Push Notification service, which conflicts with
SYS-CON-002 (TBD-025).

### 3.7 Software system attributes
- Security: tokens in the Keychain (IOS-FR-002).
- Other attributes have no iOS-specific requirement.

### 3.8 Supporting information
- `retrocast-client/system_design.md` describes the intended client architecture.
- Distribution through the App Store (STK-005) is not decided (TBD-004).

## 5.1 Traceability matrix

| Req | Priority | Delivery | Entities | APIs / Events | Components | Verification |
|---|---|---|---|---|---|---|
| IOS-FR-001 | Should | Partial | ENT-001 | API-001, API-002 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-002 | Should | Partial | ENT-019 | API-003 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-003 | Should | Partial | — | EVT-001–EVT-007 | CMP-009, CMP-002 | Demonstration |
| IOS-FR-004 | Should | Partial | ENT-002, ENT-003 | API-007, API-013, API-017, API-018, API-050 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-005 | Should | Partial | ENT-008 | API-022, API-023, API-028, EVT-009, EVT-024 | CMP-009, CMP-001, CMP-002 | Demonstration |
| IOS-FR-006 | Should | Partial | ENT-004 | API-033, EVT-025 | CMP-009, CMP-002 | Demonstration |
| IOS-FR-007 | Should | Partial | ENT-010 | API-047 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-008 | Should | Partial | ENT-002, ENT-006 | API-015, API-039–API-042 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-009 | Could | Partial | ENT-012 | API-008, API-009 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-010 | Could | Partial | ENT-008 | API-055 | CMP-009, CMP-001 | Demonstration |
| IOS-FR-011 | Could | Partial | ENT-016 | API-056, API-057, EVT-026 | CMP-009, CMP-007 | Demonstration |
| IOS-FR-012 | Should | Partial | ENT-008 | EVT-009 | CMP-009 | Demonstration |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial iOS client requirements IOS-FR-001–IOS-FR-012 |
