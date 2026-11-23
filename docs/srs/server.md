# SRS component — Server (REST API, gateway, CLI)

Part of [docs/SRS.md](../SRS.md). Prefix `SV-`. Component: CMP-001 (API server), CMP-002
(gateway, same binary), CMP-003 (admin CLI). Contracts: [DATA_CONTRACTS.md](../DATA_CONTRACTS.md).

Conventions for this file:
- **Evidence** paths are relative to `retrocast/`. "test:" names the test file that covers
  the behaviour; handler tests use mock repositories, `internal/database` tests need
  PostgreSQL.
- **Delivery** comes from reading the code and listing the tests at commit `6bc385e`. No
  requirement is marked Verified: the test suite was not run in this review. Two defects were
  confirmed by applying the migrations to a throwaway PostgreSQL 17 database (SV-FR-051,
  SV-FR-055).
- **UI** names the client screens: `web —` (retrocast-web) and `ios —` (retrocast-client).
- Priorities: MVP capabilities are `Must`; Later capabilities are `Could` (TBD-003).

## 3. Requirements

### 3.1 Functions

#### 3.1.1 Accounts and sessions (CAP-001, CAP-002)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-001 | Must | The server shall create an account for an unused username matching `^[a-zA-Z0-9_]{2,32}$` with a password of 6–128 characters. | Given an unused valid username and password, when the client calls API-001, then the server responds 200 with an access token, a refresh token and the user (display name equal to the username). | Test | Implemented | web — Register page; ios — RegisterView | ENT-001, ENT-019, API-001, CMP-001 | CAP-001 | internal/service/auth.go:54; test: internal/api/auth_handler_test.go. docs/api.md says 201; the code returns 200. Registration is open to anyone (TBD-012). |
| SV-FR-002 | Must | The server shall refuse a username that already exists with 409 `USERNAME_TAKEN`. | Given a registered username, when API-001 is called with it, then the server responds 409 `USERNAME_TAKEN` and creates no account. | Test | Implemented | web — Register page; ios — RegisterView | ENT-001, API-001 | CAP-001 | internal/service/auth.go:62–67; unique constraint `users_username_key` (migrations/000001). Uniqueness is case-sensitive. |
| SV-FR-003 | Must | The server shall refuse a username or password outside the SV-FR-001 rules with 400 (`INVALID_USERNAME` or `INVALID_PASSWORD`). | Given a one-character username, when API-001 is called, then the server responds 400 `INVALID_USERNAME`. | Test | Implemented | web — Register page; ios — RegisterView | API-001 | CAP-001 | internal/service/auth.go:55–60 |
| SV-FR-004 | Must | The server shall issue a new token pair for a correct username and password. | Given a registered user, when API-002 is called with the right password, then 200 with tokens; given a wrong password or an unknown username, then 401 `INVALID_CREDENTIALS` with the same message. | Test | Implemented | web — Login page; ios — LoginView | ENT-001, ENT-019, API-002 | CAP-001 | internal/service/auth.go:91–105; test: internal/api/auth_handler_test.go |
| SV-FR-005 | Must | The server shall exchange a valid refresh token for a new token pair and invalidate the old refresh token. | Given a refresh token issued less than 7 days ago, when API-003 is called, then 200 with a new pair; when the old token is used again, then 401 `INVALID_TOKEN`. | Test | Implemented | none (system-only: clients call it on a 401) | ENT-019, API-003 | CAP-002 | internal/service/auth.go:109–141; internal/auth/jwt.go |
| SV-FR-006 | Must | The server shall invalidate the presented refresh token on logout. | Given a signed-in user, when API-004 is called with the refresh token, then 204 and that token no longer refreshes. | Test | Implemented | web — User settings, Log out; ios — UserSettingsView | ENT-019, API-004 | CAP-002 | internal/service/auth.go:144 |

#### 3.1.2 Servers (CAP-004, CAP-006)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-007 | Must | The server shall create a guild owned by the caller when given a name of 2–100 characters. | Given a signed-in user, when API-013 is called with name "Friends", then 201 with the guild, `owner_id` equal to the caller, and EVT-012 is sent to the caller. | Test | Implemented | web — Server list, Create server modal; ios — CreateGuildSheet | ENT-002, ENT-004, API-013, EVT-012, CMP-001 | CAP-004 | internal/service/guild.go:47; test: internal/api/guild_handler_test.go. The steps are not in one transaction (ARCHITECTURE_SPEC D-12). |
| SV-FR-008 | Must | The server shall give each new guild a default role set (an `@everyone` role and an `Admin` role with Administrator held by the owner). | Given a new guild, when its roles are listed, then `@everyone` (is_default, position 0) and `Admin` (Administrator, position 1) exist and the owner holds both. | Test | Implemented | web — Guild settings, Roles; ios — GuildSettingsView | ENT-005, ENT-006 | CAP-004, CAP-014 | internal/service/guild.go:64–104. `@everyone` gets View, Send and Read history only, not `DefaultEveryonePerms` (TBD-014). |
| SV-FR-009 | Must | The server shall give each new guild a `general` text channel and a `General` voice channel. | Given a new guild, when API-018 is called, then exactly those two channels exist. | Test | Implemented | web — Channel sidebar; ios — ChannelSidebarView | ENT-003, API-018 | CAP-004 | internal/service/guild.go:106–125 |
| SV-FR-010 | Must | The server shall return a guild only to its members. | Given a user who is not a member, when API-014 is called, then 404 `NOT_FOUND`. | Test | Implemented | none (system-only) | ENT-002, API-014 | CAP-004 | internal/service/guild.go:130–148 |
| SV-FR-011 | Must | The server shall let a member with `MANAGE_GUILD` change the guild name or icon reference. | Given a member without `MANAGE_GUILD`, when API-015 is called, then 403; given the owner, then 200 and EVT-013 reaches members. | Test | Implemented | web — Guild settings; ios — GuildSettingsView | ENT-002, API-015, EVT-013 | CAP-006 | internal/service/guild.go:151–180 |
| SV-FR-012 | Must | The server shall let only the owner delete a guild. | Given an admin who is not the owner, when API-016 is called, then 403 `FORBIDDEN`; given the owner, then 204, the guild and its channels, roles and members are removed, and EVT-014 is sent. | Test | Implemented | web — Guild settings, Delete; ios — GuildSettingsView | ENT-002, API-016, EVT-014 | CAP-006 | internal/service/guild.go:183–201; ON DELETE CASCADE (migrations/000003–000006) |
| SV-FR-013 | Must | The server shall list the guilds the caller belongs to. | Given a user in two guilds, when API-007 is called, then both guilds are returned. | Test | Implemented | web — Server list; ios — ServerListView | ENT-002, ENT-004, API-007 | CAP-004 | internal/service/guild.go:204 |

#### 3.1.3 Channels (CAP-006, CAP-007)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-014 | Must | The server shall let a member with `MANAGE_CHANNELS` create a text, voice or category channel named 1–100 characters. | Given a member without `MANAGE_CHANNELS`, when API-017 is called, then 403; given an admin with type 0 and name "random", then 201 and EVT-015 reaches members. | Test | Implemented | web — Channel modal; ios — CreateChannelSheet | ENT-003, API-017, EVT-015 | CAP-006 | internal/service/channel.go:40–76; test: internal/api/channel_handler_test.go. Names are unique per guild (migrations/000003). |
| SV-FR-015 | Must | The server shall let a member with `MANAGE_CHANNELS` rename, move or delete a channel. | Given an admin, when API-020 renames a channel, then 200 and EVT-016 is sent; when API-021 deletes it, then 204, its messages are removed and EVT-017 is sent. | Test | Implemented | web — Channel context menu; ios — ChannelSidebarView | ENT-003, API-020, API-021, EVT-016, EVT-017 | CAP-006 | internal/service/channel.go:121–175 |
| SV-FR-016 | Must | The server shall list and return a guild's channels only to its members. | Given a non-member, when API-018 or API-019 is called, then 404. | Test | Implemented | web — Channel sidebar; ios — ChannelSidebarView | ENT-003, API-018, API-019 | CAP-006 | internal/service/channel.go:79–118. Per-channel `VIEW_CHANNEL` overrides do not filter the list. |
| SV-FR-017 | Could | The server shall let a channel name a category channel of the same guild as its parent. | Given a category channel, when API-017 creates a text channel with that `parent_id`, then the channel is returned with `parent_id` set. | Test | Implemented | web — Channel sidebar, grouped by category | ENT-003, API-017 | CAP-007 | internal/models/channel.go (`ParentID`); migrations/000003 (`parent_id` FK) |

#### 3.1.4 Invites and membership (CAP-005, CAP-008)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-018 | Must | The server shall let a member with `CREATE_INVITE` create an invite code with an expiry (default 86 400 s) and an optional use limit. | Given a member with `CREATE_INVITE`, when API-047 is called with no body, then 201 with an 8-character code that expires in 24 hours and `max_uses` 0 (unlimited). | Test | Implemented | web — Invite modal; ios — InviteSheet | ENT-010, API-047 | CAP-008 | internal/service/invite.go:53–89; test: internal/api/invite_handler_test.go. Codes carry 32 random bits. |
| SV-FR-019 | Must | The server shall show an invite's guild name, member count and creator to anyone holding the code. | Given a valid code, when API-049 is called without a token, then 200 with `guild_name`, `member_count` (members at request time) and `creator_id`; given an expired code, then 404 `EXPIRED`. | Test | Implemented | web — Join server modal; ios — JoinGuildSheet | ENT-010, API-049 | CAP-005 | internal/service/invite.go:108–137. No rate limit applies (SV-NFR-003). |
| SV-FR-020 | Must | The server shall add the caller to the guild when they accept a valid invite. | Given a valid code, when API-050 is called, then 200 with the guild, `uses` increases by 1 and EVT-018 reaches members; given an expired or used-up code, then 410 (`EXPIRED`, `MAX_USES`); given a member, then 409 `ALREADY_MEMBER`; given a banned user, then 403 `BANNED`. | Test | Partial (the new member gets no `@everyone` row in member_roles, D-04; the gateway does not subscribe the joiner to the guild until they reconnect, D-05; `uses` is incremented without a row lock, D-20) | web — Join server modal; ios — JoinGuildSheet | ENT-004, ENT-010, API-050, EVT-018 | CAP-005 | internal/service/invite.go:140–195 |
| SV-FR-021 | Must | The server shall let a member with `MANAGE_GUILD` (or the invite's creator, for revoking) list and revoke a guild's invites. | Given a member without `MANAGE_GUILD`, when API-048 is called, then 403; given the creator, when API-051 is called, then 204. | Test | Implemented | web — Invite modal; ios — InviteSheet | ENT-010, API-048, API-051 | CAP-008 | internal/service/invite.go:92–105, :198–215 |
| SV-FR-022 | Must | The server shall let a member other than the owner leave a guild. | Given a member, when API-035 is called, then 204 and EVT-019 is sent; given the owner, then 403 `FORBIDDEN` ("transfer ownership or delete the guild"). | Test | Implemented | web — Guild settings, Leave; ios — GuildSettingsView | ENT-004, API-035, EVT-019 | CAP-005 | internal/service/member.go:221–240. No transfer exists (TBD-026). |
| SV-FR-023 | Must | The server shall list a guild's members to its members in pages of 1–100 (default 50). | Given a guild with 60 members, when API-033 is called with no limit, then 50 members are returned with their role ids. | Test | Implemented | web — Member list; ios — MemberListView | ENT-004, ENT-005, API-033 | CAP-012 | internal/service/member.go:39–63; test: internal/api/member_handler_test.go |
| SV-FR-024 | Must | The server shall return one member's record to members of the same guild. | Given a non-member, when API-036 is called, then 404. | Test | Implemented | web — User profile popover; ios — UserProfilePopover | ENT-004, API-036 | CAP-012 | internal/service/member.go:66–84 |

#### 3.1.5 Messages (CAP-009, CAP-010, CAP-013)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-025 | Must | The server shall store a message of 1–2000 bytes from a member who has `SEND_MESSAGES` in the channel. | Given a member whose role is denied `SEND_MESSAGES` by a channel override, when API-022 is called, then 403; given a member with the permission, then 201 with the message and its author fields. | Test | Implemented | web — Chat area, message input; ios — MessageInput | ENT-008, API-022, CMP-001 | CAP-009 | internal/service/message.go:44–84; test: internal/api/message_handler_test.go |
| SV-FR-026 | Must | The server shall push each new guild message to the guild's connected members as EVT-009. | Given two members connected to the gateway, when one sends a message, then the other receives EVT-009 with the full message. | Test | Partial (members who joined or created the guild after connecting receive nothing until they reconnect, D-05; channel `VIEW_CHANNEL` overrides do not filter recipients, D-06) | web — Chat area; ios — MessageListView | EVT-009, CMP-002 | CAP-009 | internal/service/message.go:77–84; internal/gateway/manager.go (`DispatchToGuild`; `SubscribeToGuild` has no caller) |
| SV-FR-027 | Must | The server shall return a channel's messages newest first in pages of 1–100 (default 50) before an optional message-id cursor. | Given a channel with 120 messages, when API-023 is called with `before` set to the 51st newest id, then the next 50 older messages are returned; given a member without `READ_MESSAGE_HISTORY`, then 403. | Test | Implemented | web — Message list, scroll back; ios — MessageListView | ENT-008, ENT-009, API-023 | CAP-010 | internal/api/message_handler.go:49–82; internal/database/message_repo.go:47; test: internal/database/message_repo_test.go |
| SV-FR-028 | Must | The server shall return a single message to members who can read the channel's history. | Given a message id in another channel, when API-024 is called, then 404. | Test | Implemented | none (system-only) | ENT-008, API-024 | CAP-010 | internal/service/message.go:110–131 |
| SV-FR-029 | Could | The server shall let only the author edit a message, setting `edited_at` and sending EVT-010. | Given another member's message, when API-025 is called, then 403; given the author with valid content, then 200 with `edited_at` set. | Test | Implemented | web — Message context menu, Edit | ENT-008, API-025, EVT-010 | CAP-013 | internal/service/message.go:134–180 |
| SV-FR-030 | Could | The server shall let the author or a member with `MANAGE_MESSAGES` delete a guild message, sending EVT-011. | Given a moderator with `MANAGE_MESSAGES`, when API-026 deletes another member's message, then 204 and EVT-011 is sent; in a DM, only the author may delete. | Test | Implemented | web — Message context menu, Delete | ENT-008, API-026, EVT-011 | CAP-013 | internal/service/message.go:183–222 |

#### 3.1.6 Typing and presence (CAP-011, CAP-012)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-031 | Must | The server shall send EVT-024 to a guild's connected members when a member who can send in the channel signals typing. | Given a member with `SEND_MESSAGES`, when API-028 is called, then other connected members receive EVT-024 with channel, guild and user; given a user who is not a member, then 403. | Test | Partial (the routed handler checks no membership or permission; DM channels return 404; `MessageService.Typing` has no route, D-18) | web — Typing indicator; ios — TypingIndicator | API-028, EVT-024, ENT-021 | CAP-011 | internal/gateway/typing.go:30; internal/api/router.go:194; internal/service/message.go:225 |
| SV-FR-032 | Must | The server shall mark a user online on IDENTIFY and send EVT-025 to members of every guild they share. | Given two members of one guild, when one identifies, then the other receives EVT-025 with status `online`. | Test | Partial (READY carries no presence list, so a new connection does not learn who is already online, D-19) | web — Member list, presence dots; ios — PresenceDot | EVT-003, EVT-025, ENT-020 | CAP-012 | internal/gateway/manager.go:221–280; internal/gateway/events.go:75–80 |
| SV-FR-033 | Must | The server shall accept a client status (`online`, `idle`, `dnd`, `invisible`) and show `invisible` to others as `offline`. | Given a connected user, when they send op 3 with `invisible`, then other members receive EVT-025 with `offline`. | Test | Implemented | web — User settings; ios — AppSettingsView | EVT-008, EVT-025, ENT-020 | CAP-012 | internal/gateway/manager.go:338–370; test: internal/gateway/manager_test.go |
| SV-FR-034 | Must | The server shall send EVT-025 with `offline` 10 seconds after a user's last connection closes unless they reconnect. | Given a connected user, when the connection drops and no new one opens within 10 s, then members receive `offline`. | Test | Implemented | web — Member list, presence dots | EVT-025, ENT-020 | CAP-012 | internal/gateway/manager.go:97–117; test: internal/gateway/manager_test.go |

#### 3.1.7 Gateway session (CAP-009, CAP-002)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-035 | Must | The server shall send EVT-001 with a heartbeat interval of 41 250 ms on every new gateway connection. | Given a WebSocket client, when it connects to `/gateway`, then the first frame is op 10 with `heartbeat_interval` 41250. | Test | Implemented | none (system-only) | EVT-001, CMP-002 | CAP-009 | internal/gateway/server.go; internal/gateway/connection.go |
| SV-FR-036 | Must | The server shall answer a valid IDENTIFY with EVT-003 and close the connection on an invalid token. | Given a valid access token, when op 2 is sent, then EVT-003 arrives with session id, user id, guild ids and read states; given an expired token, then the server closes the socket. | Test | Implemented | none (system-only) | EVT-002, EVT-003, CMP-002 | CAP-009 | internal/gateway/manager.go:221–280. Guild ids are JSON numbers (SV-IR-004). |
| SV-FR-037 | Must | The server shall acknowledge each heartbeat and close a connection that sends none for the interval plus 10 seconds. | Given an identified client, when it sends op 1, then op 11 follows; when it stays silent for 51.25 s, then the server closes the socket. | Test | Implemented | none (system-only) | EVT-004, EVT-005 | CAP-009 | internal/gateway/connection.go:92–155 |
| SV-FR-038 | Must | The server shall replay the events a client missed when it resumes a session within the replay buffer. | Given a client that dropped after event N, when it sends op 6 with its session id and sequence N, then it receives the later events in order. | Test | Partial (the 100-event buffer is per guild with per-guild sequence numbers, while clients track a per-connection sequence; the session id is not checked; no RECONNECT is sent when the buffer has moved past N, D-07) | none (system-only) | EVT-006, CMP-002 | CAP-009 | internal/gateway/manager.go:287–336 |
| SV-FR-039 | Must | The server shall keep one gateway connection per user and send EVT-007 to the older connection when a new one identifies. | Given a user connected on web, when the same user identifies on iOS, then the web connection receives op 7 and is closed. | Test | Implemented | web — reconnect banner; ios — ErrorBanner | EVT-007, CMP-002 | CAP-009 | internal/gateway/manager.go; test: internal/gateway/manager_test.go. Policy is TBD-009. |

#### 3.1.8 Roles and permissions (CAP-014, CAP-015, CAP-016)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-040 | Must | The server shall create a role only for a member who has `MANAGE_ROLES` and only at a position below that member's highest role. | Given a member without `MANAGE_ROLES`, when API-039 is called, then 403; given an admin whose highest role is at position 3, when the new role's position is 3 or more, then 403 `ROLE_HIERARCHY`. | Test | Partial (no `MANAGE_ROLES` check; the hierarchy check accepts any position below the caller's highest, including negative ones, so a member with no roles can create a role with Administrator at position -1 and assign it to themself through SV-FR-043, D-01) | web — Role editor; ios — RoleEditorView | ENT-006, API-039, EVT-021 | CAP-014, CAP-015 | internal/service/role.go:48–82; test: internal/api/role_handler_test.go (no case for a missing permission) |
| SV-FR-041 | Must | The server shall update a role only when it belongs to the guild in the path and the caller has `MANAGE_ROLES` above that role. | Given a role of guild A, when API-041 is called under guild B, then 404; given a member without `MANAGE_ROLES`, then 403. | Test | Partial (no `MANAGE_ROLES` check; the role's guild is not compared with the path; a new position is not checked against the caller's highest role, D-01, D-02) | web — Role editor; ios — RoleEditorView | ENT-006, API-041, EVT-022 | CAP-014, CAP-015 | internal/service/role.go:97–142 |
| SV-FR-042 | Must | The server shall delete a role only under the same rules as SV-FR-041 and never delete `@everyone`. | Given the `@everyone` role, when API-042 is called by the owner, then 403 `CANNOT_DELETE`; given a member without `MANAGE_ROLES`, then 403. | Test | Partial (`@everyone` is protected; the permission and guild checks are missing, D-01, D-02) | web — Role editor | ENT-006, API-042, EVT-023 | CAP-014, CAP-015 | internal/service/role.go:145–177; test: internal/api/role_handler_test.go (`TestDeleteRole_EveryoneRole`) |
| SV-FR-043 | Must | The server shall assign or remove a role only for a caller with `MANAGE_ROLES` whose highest role is above the role. | Given a member without `MANAGE_ROLES`, when API-043 assigns the Admin role to themself, then 403. | Test | Partial (hierarchy and role-guild checks only; no `MANAGE_ROLES` check, D-01) | web — Member list, role assignment; ios — MemberRow | ENT-005, API-043, API-044, API-037, EVT-020 | CAP-014, CAP-015 | internal/service/role.go:180–245; internal/service/member.go:116–190 (`roles` via API-037) |
| SV-FR-044 | Must | The server shall list a guild's roles only to its members. | Given a non-member, when API-040 is called, then 404. | Test | Partial (no membership check, D-03) | web — Guild settings, Roles | ENT-006, API-040 | CAP-014 | internal/service/role.go:85–94 |
| SV-FR-045 | Must | The server shall compute a member's channel permissions from the `@everyone` role, the member's roles and the channel's overrides in the order of docs/permissions.md. | Given `@everyone` allowed `SEND_MESSAGES` and a channel override for `@everyone` that denies it, when a member without other roles sends there, then 403; given a role override that allows it, then 201. | Test | Partial (channel checks follow the documented order; guild-level checks via `RequireGuildPermission` OR only the member's assigned roles and skip `@everyone`, D-04) | none (system-only) | ENT-005, ENT-006, ENT-007, CMP-001 | CAP-015 | internal/service/permission.go; internal/permissions/; test: internal/permissions/resolver_test.go, bitfield_test.go |
| SV-FR-046 | Must | The server shall grant every permission to the guild owner and to holders of a role with Administrator. | Given the owner with no roles, when any guild-scoped action is called, then it is allowed. | Test | Implemented | none (system-only) | ENT-002, ENT-006 | CAP-015 | internal/service/permission.go; test: internal/permissions/resolver_test.go |
| SV-FR-047 | Could | The server shall let only a member with `MANAGE_ROLES` set or delete a channel's override for a role of the same guild. | Given a member without `MANAGE_ROLES`, when API-045 is called, then 403; given a role of another guild, then 404. | Test | Partial (no permission, membership or role-guild check, D-01) | web — Channel settings (none yet); ios — none | ENT-007, API-045, API-046 | CAP-016 | internal/service/role.go:248–277; docs/api.md:284, :292 (documents `MANAGE_ROLES`) |

#### 3.1.9 Voice (CAP-017)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-048 | Could | The server shall give a member with `CONNECT` on a voice channel a LiveKit token for room `voice-{channel id}` and record their voice state. | Given a member with `CONNECT`, when API-056 is called on a voice channel, then 200 with a token and the channel's voice states, and EVT-026 reaches members; given a text channel, then 400. | Test | Implemented | ios — VoiceChannelView; web — none (TBD-024) | ENT-016, API-056, EVT-026, CMP-007 | CAP-017 | internal/service/voice.go:53–112; test: internal/api/voice_handler_test.go. The token lasts 24 h and the response has no LiveKit URL. |
| SV-FR-049 | Could | The server shall remove the caller's voice state when they leave and send EVT-026. | Given a user in a voice channel, when API-057 is called, then 204 and members receive EVT-026 with that user's `channel_id` null. | Test | Implemented | ios — VoiceControlBar | ENT-016, API-057, EVT-026 | CAP-017 | internal/service/voice.go:115–149. A client that disconnects without calling it stays listed: no LiveKit webhook. |
| SV-FR-050 | Could | The server shall list the voice states of a voice channel to guild members. | Given two users in a voice channel, when API-058 is called, then both states are returned. | Test | Implemented | ios — VoiceChannelView | ENT-016, API-058 | CAP-017 | internal/service/voice.go:152 |

#### 3.1.10 Attachments (CAP-019)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-051 | Could | The server shall store a file of at most 10 MB from a member with `ATTACH_FILES` when its type is allowed (TBD-017). | Given a member with `ATTACH_FILES`, when API-027 uploads a 1 MB PNG, then 201 with the attachment id, size, type and a URL; given an 11 MB file, then 400 `FILE_TOO_LARGE`. | Test | Partial (the object reaches MinIO but the attachment row is written with `message_id` 0, which the foreign key to messages rejects, D-09; no API links an attachment to a message) | web — File upload, drag and drop | ENT-009, ENT-022, API-027, CMP-006 | CAP-019 | internal/service/upload.go:61–100 (`MessageID: 0` at :92); migrations/000008; FK violation reproduced on PostgreSQL 17 with all migrations applied |
| SV-FR-052 | Could | The server shall include each message's attachments with a URL the client can fetch. | Given a message with an attachment, when API-023 is called, then the attachment appears with a URL that the client can download through the public entry point. | Test | Partial (URLs have the form `http://{MINIO_ENDPOINT}/retrocast/{key}`, an address inside the compose network, D-10) | web — Message list, image preview | ENT-009, API-023 | CAP-019 | internal/storage/minio.go (`GetURL`); TBD-011 |

#### 3.1.11 Profile (CAP-003)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-053 | Could | The server shall let a user read their profile and change their display name (1–32 characters) or avatar reference. | Given a signed-in user, when API-006 sets display name "Vic", then 200 with the new name; given a 33-character name, then 400. | Test | Partial (the avatar is a free-form hash string; no endpoint uploads an avatar or a server icon) | web — User settings; ios — UserSettingsView | ENT-001, API-005, API-006 | CAP-003 | internal/service/user.go:21–57; test: internal/api/user_handler_test.go |

#### 3.1.12 Direct messages (CAP-020, CAP-021)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-054 | Could | The server shall return the existing one-to-one DM between the caller and a recipient or create it. | Given no DM between A and B, when A calls API-008 with B's id, then 200 with a type-1 channel; when A calls it again, then the same channel is returned. | Test | Implemented | web — DM list, New DM; ios — DMListView | ENT-012, ENT-013, API-008 | CAP-020 | internal/service/dm.go:40–70; test: internal/api/dm_handler_test.go, internal/database/dm_repo_test.go. Any user id is accepted (TBD-013). |
| SV-FR-055 | Could | The server shall store messages in a DM for its recipients and send them only to its recipients. | Given a DM between A and B, when A calls API-022 on it, then 201 and B receives EVT-009; given user C, then 403 or 404. | Test | Partial (the insert fails: `messages.channel_id` references `channels`, but DM channels exist only in `dm_channels`, D-08) | web — DM conversation; ios — DMListView | ENT-008, ENT-012, API-022, EVT-009 | CAP-020 | internal/service/message.go:44–84; internal/database/dm_repo.go:28; migrations/000007; FK violation reproduced on PostgreSQL 17 |
| SV-FR-056 | Could | The server shall create a group DM of up to 10 participants owned by its creator. | Given a creator and 9 recipients, when API-008 is called with `recipient_ids`, then 200 with a type-3 channel; given 10 recipients, then 400. | Test | Implemented | web — DM list; ios — DMListView | ENT-012, ENT-013, API-008 | CAP-021 | internal/service/dm.go:73–139; test: internal/api/group_dm_handler_test.go |
| SV-FR-057 | Could | The server shall let a group DM's owner add or remove participants and let any participant remove themself. | Given a participant who is not the owner, when API-010 adds a user, then 403; when API-011 removes themself, then 204. | Test | Implemented | web — DM conversation header | ENT-013, API-010, API-011 | CAP-021 | internal/service/dm.go:142–258; test: internal/api/group_dm_handler_test.go |
| SV-FR-058 | Could | The server shall list the caller's DMs with their recipients. | Given a user in two DMs, when API-009 is called, then both are returned with recipient profiles. | Test | Implemented | web — DM list; ios — DMListView | ENT-012, ENT-013, API-009 | CAP-020 | internal/service/dm.go:261 |

#### 3.1.13 Search (CAP-028)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-059 | Could | The server shall return a guild's messages matching a text query (optional author and date filters) in pages of 1–100 (default 25) to members who can read the history. | Given messages containing "pizza", when API-055 is called with `q=pizza`, then those messages are returned newest first; given a non-member, then 404. | Test | Partial (permission is checked at guild level only, so messages in channels hidden by overrides are returned, D-06) | ios — SearchView; web — none | ENT-008, API-055 | CAP-028 | internal/service/search.go:33; internal/database/message_repo.go:91; migrations/000015 (English stemming, TBD-023); test: internal/api/search_handler_test.go |

### 3.2 Performance requirements
Server performance targets are system-wide: see SYS-NFR-001 and SYS-NFR-002 in
[system.md](system.md).

### 3.3 Usability requirements
The server has no user interface; client usability is in [web.md](web.md) and [ios.md](ios.md).

### 3.4 Interface requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-IR-001 | Must | The server shall expose the REST operations API-001–API-058 under `/api/v1` with JSON bodies and bearer access tokens. | Given the OpenAPI document, when each operation is called as documented, then the status and body match DATA_CONTRACTS §4. | Test | Implemented | none (system-only) | API-001–API-058 | LIM-003 | internal/api/router.go; docs/openapi.yaml. Success bodies are wrapped in `data` for guild, channel, member and user operations only (TBD-010). |
| SV-IR-002 | Must | The server shall frame every gateway message as one JSON object (`op`, `d`, `s`, `t`) per DATA_CONTRACTS §6. | Given a dispatch, when it is sent, then `op` is 0, `t` names the event and `s` increases per connection. | Test | Implemented | none (system-only) | EVT-001–EVT-030 | LIM-003 | internal/gateway/events.go; test: internal/gateway/manager_test.go |
| SV-IR-003 | Must | The server shall return every error as `{"error": {"code", "message"}}`. | Given a request without a token, when a protected route is called, then 401 with the error body. | Test | Partial (the JWT middleware returns Echo's `{"message"}` body; other handler errors use the envelope) | none (system-only) | DATA_CONTRACTS §5 | LIM-003 | internal/api/response.go; internal/auth/middleware.go |
| SV-IR-004 | Must | The server shall encode every snowflake id in JSON as a decimal string. | Given a member with roles, when API-033 is called, then each id in `roles` is a string; given READY, then each guild id is a string. | Test | Partial (`Member.roles` and READY `guilds` are JSON numbers, D-11) | none (system-only) | ENT-004, EVT-003 | RISK-009 | internal/models/member.go; internal/gateway/events.go:78 |
| SV-IR-005 | Should | The server shall serve its OpenAPI document at `/docs/openapi.yaml` and a reference page at `/docs`. | Given a running server, when `/docs/openapi.yaml` is fetched, then the OpenAPI 3.0.3 document is returned. | Demonstration | Implemented | none (developer docs) | API-001–API-058 | LIM-003 | internal/api/router.go; docs/openapi.yaml. The `/docs` page loads Swagger UI from unpkg.com. |
| SV-IR-006 | Could | The server shall mint LiveKit access tokens signed with the configured API secret that grant joining only the channel's room. | Given a join, when the token is decoded, then `iss` is the API key, `sub` is the user id and the grant is `roomJoin` for `voice-{channel id}`. | Test | Implemented | none (system-only) | DATA_CONTRACTS §7, CMP-007 | CAP-017 | internal/service/voice.go |

### 3.5 Logical database requirements
Entities ENT-001–ENT-022 are defined in [DATA_CONTRACTS.md §3](../DATA_CONTRACTS.md#3-entities).
Integrity rules that the server shall keep are on SV-FR-002 (unique usernames), SV-FR-012
(cascade on guild delete), SV-FR-055 (DM messages need a valid parent) and SV-FR-051
(attachments need a message). Retention is TBD-021.

### 3.6 Design constraints
System-wide constraints SYS-CON-001–SYS-CON-004 are in [system.md](system.md).

### 3.7 Software system attributes

#### 3.7.1 Reliability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-004 | Must | Reliability: the server shall stop accepting requests on SIGINT or SIGTERM and finish in-flight requests before exiting. Measure: in-flight requests finish within the shutdown timeout in cmd/retrocast/main.go. | Given a request in flight, when the process receives SIGTERM, then the request completes and the process exits 0. | Demonstration | Implemented | none (infra) | CMP-001 | LIM-003 | cmd/retrocast/main.go (graceful shutdown) |

#### 3.7.2 Availability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-005 | Must | Availability: the server shall answer `GET /health` with 200 when its stores (PostgreSQL, Redis) respond and with 503 naming the failing store otherwise. Measure: one check per request (container healthcheck every 30 s). | Given Redis stopped, when `/health` is called, then 503 with the failing component named. | Test | Implemented | none (infra) | CMP-001, CMP-004, CMP-005 | LIM-001, LIM-003 | internal/api/router.go (`/health`); Dockerfile `HEALTHCHECK`; docker-compose.yml |

#### 3.7.3 Security

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-001 | Must | Security: the server shall store passwords only as salted Argon2id hashes. Measure: Argon2id parameters (time 1, memory 64 MiB, parallelism 4, salt 16 bytes, key 32 bytes). | Given a new account, when the users row is read, then `password_hash` is an Argon2id encoding with those parameters and no plaintext exists. | Test | Implemented | none (system-only) | ENT-001 | RISK-012 | internal/auth/argon2.go; test: internal/auth/argon2_test.go; SYSTEM_DESIGN.md:683 |
| SV-NFR-002 | Must | Security: the server shall accept an HS256 access token for 15 minutes and a single-use refresh token for 7 days. Measure: token lifetimes (`exp` − `iat` = 900 s; refresh TTL 604 800 s; refresh token of 32 random bytes). | Given a token issued 16 minutes ago, when a protected route is called, then 401. | Test | Implemented | none (system-only) | ENT-019 | RISK-012 | internal/auth/jwt.go; test: internal/auth/jwt_test.go. No minimum length is enforced for `JWT_SECRET`. |
| SV-NFR-003 | Must | Security: the server shall limit each client to 5 auth requests per minute and each user to 50 requests per minute per route, answering 429 with `Retry-After`. Measure: fixed one-minute windows in Redis. | Given 6 login attempts from one client within a minute, when the sixth is sent, then 429 with `Retry-After`. | Test | Partial (the client is identified by `X-Forwarded-For` as sent by the client, so a caller can change buckets, D-13; the public invite lookup has no limit; Redis errors let requests through) | none (system-only) | ENT-021 | RISK-003 | internal/api/ratelimit.go; test: internal/api/ratelimit_test.go; TBD-015 |
| SV-NFR-008 | Must | Security: every client connection shall use HTTPS or WSS through Caddy. Measure: no client-facing plaintext port other than Caddy's redirect on 80. | Given the compose stack, when the published ports are listed, then only Caddy serves clients. | Inspection | Partial (compose also publishes the API on 8080 and MinIO on 9000 over plain HTTP; attachment URLs use `http://`, D-10; CORS and the gateway accept every origin, D-17) | none (infra) | CMP-010 | RISK-012 | docker-compose.yml; Caddyfile; SYSTEM_DESIGN.md:684; TBD-016 |

#### 3.7.4 Maintainability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-007 | Must | Maintainability: the server shall change the database schema only through numbered migrations applied at startup. Measure: every table and column exists in `migrations/`; startup applies pending ones before serving. | Given an empty database, when the server starts, then migrations 000001–000019 are applied and `/health` returns 200. | Test | Implemented | none (infra) | ENT-001–ENT-018 | LIM-003 | cmd/retrocast/main.go (golang-migrate `file://migrations`); CI "Run migrations" step |

#### 3.7.5 Portability
Covered system-wide by SYS-NFR-004.

#### 3.7.6 Privacy and compliance
Covered system-wide by SYS-CON-002 and SYS-NFR-003.

#### 3.7.7 Observability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-006 | Should | Observability: the server shall write one structured JSON log line per HTTP request. Measure: fields on each line (method, path, status, latency). | Given a request, when it completes, then one JSON line with those fields is written to stdout. | Inspection | Implemented | none (infra) | CMP-001 | LIM-003 | cmd/retrocast/main.go (slog JSON handler, Echo RequestLogger) |

Metrics (`/metrics`, Prometheus, Grafana) wait for the release decision on CAP-035 (TBD-039).

### 3.8 Supporting information
- Permission bits: 19 named bits plus Administrator (`1 << 31`)
  (internal/permissions/bitfield.go; DATA_CONTRACTS ENT-006).
- Routes outside `/api/v1`: `/health`, `/metrics`, `/docs`, `/docs/openapi.yaml`, `/gateway`.
- The admin CLI (`cmd/retrocast-cli`: `migrate`, `seed`, `health`, `version`) waits for the
  release decision on CAP-036 (TBD-040).
- `NFR` ids are numbered in the order they were drafted, not by section.

## 5.1 Traceability matrix

| Req | Priority | Delivery | Entities | APIs / Events | Components | Verification |
|---|---|---|---|---|---|---|
| SV-FR-001 | Must | Implemented | ENT-001, ENT-019 | API-001 | CMP-001 | Test |
| SV-FR-002 | Must | Implemented | ENT-001 | API-001 | CMP-001 | Test |
| SV-FR-003 | Must | Implemented | — | API-001 | CMP-001 | Test |
| SV-FR-004 | Must | Implemented | ENT-001, ENT-019 | API-002 | CMP-001 | Test |
| SV-FR-005 | Must | Implemented | ENT-019 | API-003 | CMP-001, CMP-005 | Test |
| SV-FR-006 | Must | Implemented | ENT-019 | API-004 | CMP-001, CMP-005 | Test |
| SV-FR-007 | Must | Implemented | ENT-002, ENT-004 | API-013, EVT-012 | CMP-001 | Test |
| SV-FR-008 | Must | Implemented | ENT-005, ENT-006 | API-013 | CMP-001 | Test |
| SV-FR-009 | Must | Implemented | ENT-003 | API-013, API-018 | CMP-001 | Test |
| SV-FR-010 | Must | Implemented | ENT-002 | API-014 | CMP-001 | Test |
| SV-FR-011 | Must | Implemented | ENT-002 | API-015, EVT-013 | CMP-001 | Test |
| SV-FR-012 | Must | Implemented | ENT-002 | API-016, EVT-014 | CMP-001 | Test |
| SV-FR-013 | Must | Implemented | ENT-002, ENT-004 | API-007 | CMP-001 | Test |
| SV-FR-014 | Must | Implemented | ENT-003 | API-017, EVT-015 | CMP-001 | Test |
| SV-FR-015 | Must | Implemented | ENT-003 | API-020, API-021, EVT-016, EVT-017 | CMP-001 | Test |
| SV-FR-016 | Must | Implemented | ENT-003 | API-018, API-019 | CMP-001 | Test |
| SV-FR-017 | Could | Implemented | ENT-003 | API-017 | CMP-001 | Test |
| SV-FR-018 | Must | Implemented | ENT-010 | API-047 | CMP-001 | Test |
| SV-FR-019 | Must | Implemented | ENT-010 | API-049 | CMP-001 | Test |
| SV-FR-020 | Must | Partial | ENT-004, ENT-010 | API-050, EVT-018 | CMP-001, CMP-002 | Test |
| SV-FR-021 | Must | Implemented | ENT-010 | API-048, API-051 | CMP-001 | Test |
| SV-FR-022 | Must | Implemented | ENT-004 | API-035, EVT-019 | CMP-001 | Test |
| SV-FR-023 | Must | Implemented | ENT-004, ENT-005 | API-033 | CMP-001 | Test |
| SV-FR-024 | Must | Implemented | ENT-004 | API-036 | CMP-001 | Test |
| SV-FR-025 | Must | Implemented | ENT-008 | API-022 | CMP-001 | Test |
| SV-FR-026 | Must | Partial | — | EVT-009 | CMP-002 | Test |
| SV-FR-027 | Must | Implemented | ENT-008, ENT-009 | API-023 | CMP-001 | Test |
| SV-FR-028 | Must | Implemented | ENT-008 | API-024 | CMP-001 | Test |
| SV-FR-029 | Could | Implemented | ENT-008 | API-025, EVT-010 | CMP-001 | Test |
| SV-FR-030 | Could | Implemented | ENT-008 | API-026, EVT-011 | CMP-001 | Test |
| SV-FR-031 | Must | Partial | ENT-021 | API-028, EVT-024 | CMP-002 | Test |
| SV-FR-032 | Must | Partial | ENT-020 | EVT-003, EVT-025 | CMP-002 | Test |
| SV-FR-033 | Must | Implemented | ENT-020 | EVT-008, EVT-025 | CMP-002 | Test |
| SV-FR-034 | Must | Implemented | ENT-020 | EVT-025 | CMP-002 | Test |
| SV-FR-035 | Must | Implemented | — | EVT-001 | CMP-002 | Test |
| SV-FR-036 | Must | Implemented | — | EVT-002, EVT-003 | CMP-002 | Test |
| SV-FR-037 | Must | Implemented | — | EVT-004, EVT-005 | CMP-002 | Test |
| SV-FR-038 | Must | Partial | — | EVT-006 | CMP-002 | Test |
| SV-FR-039 | Must | Implemented | — | EVT-007 | CMP-002 | Test |
| SV-FR-040 | Must | Partial | ENT-006 | API-039, EVT-021 | CMP-001 | Test |
| SV-FR-041 | Must | Partial | ENT-006 | API-041, EVT-022 | CMP-001 | Test |
| SV-FR-042 | Must | Partial | ENT-006 | API-042, EVT-023 | CMP-001 | Test |
| SV-FR-043 | Must | Partial | ENT-005 | API-043, API-044, API-037, EVT-020 | CMP-001 | Test |
| SV-FR-044 | Must | Partial | ENT-006 | API-040 | CMP-001 | Test |
| SV-FR-045 | Must | Partial | ENT-005, ENT-006, ENT-007 | — | CMP-001 | Test |
| SV-FR-046 | Must | Implemented | ENT-002, ENT-006 | — | CMP-001 | Test |
| SV-FR-047 | Could | Partial | ENT-007 | API-045, API-046 | CMP-001 | Test |
| SV-FR-048 | Could | Implemented | ENT-016 | API-056, EVT-026 | CMP-001, CMP-007 | Test |
| SV-FR-049 | Could | Implemented | ENT-016 | API-057, EVT-026 | CMP-001 | Test |
| SV-FR-050 | Could | Implemented | ENT-016 | API-058 | CMP-001 | Test |
| SV-FR-051 | Could | Partial | ENT-009, ENT-022 | API-027 | CMP-001, CMP-006 | Test |
| SV-FR-052 | Could | Partial | ENT-009 | API-023 | CMP-001, CMP-006 | Test |
| SV-FR-053 | Could | Partial | ENT-001 | API-005, API-006 | CMP-001 | Test |
| SV-FR-054 | Could | Implemented | ENT-012, ENT-013 | API-008 | CMP-001 | Test |
| SV-FR-055 | Could | Partial | ENT-008, ENT-012 | API-022, EVT-009 | CMP-001, CMP-002 | Test |
| SV-FR-056 | Could | Implemented | ENT-012, ENT-013 | API-008 | CMP-001 | Test |
| SV-FR-057 | Could | Implemented | ENT-013 | API-010, API-011 | CMP-001 | Test |
| SV-FR-058 | Could | Implemented | ENT-012, ENT-013 | API-009 | CMP-001 | Test |
| SV-FR-059 | Could | Partial | ENT-008 | API-055 | CMP-001, CMP-004 | Test |
| SV-IR-001 | Must | Implemented | — | API-001–API-058 | CMP-001 | Test |
| SV-IR-002 | Must | Implemented | — | EVT-001–EVT-030 | CMP-002 | Test |
| SV-IR-003 | Must | Partial | — | all API | CMP-001 | Test |
| SV-IR-004 | Must | Partial | ENT-004 | API-033, EVT-003 | CMP-001, CMP-002 | Test |
| SV-IR-005 | Should | Implemented | — | API-001–API-058 | CMP-001 | Demonstration |
| SV-IR-006 | Could | Implemented | — | API-056 | CMP-001, CMP-007 | Test |
| SV-NFR-001 | Must | Implemented | ENT-001 | API-001, API-002 | CMP-001 | Test |
| SV-NFR-002 | Must | Implemented | ENT-019 | API-002, API-003 | CMP-001 | Test |
| SV-NFR-003 | Must | Partial | ENT-021 | API-001–API-004, API-049 | CMP-001, CMP-005 | Test |
| SV-NFR-004 | Must | Implemented | — | — | CMP-001 | Demonstration |
| SV-NFR-005 | Must | Implemented | — | `/health` | CMP-001 | Test |
| SV-NFR-006 | Should | Implemented | — | — | CMP-001 | Inspection |
| SV-NFR-007 | Must | Implemented | ENT-001–ENT-018 | — | CMP-001, CMP-004 | Test |
| SV-NFR-008 | Must | Partial | — | — | CMP-010 | Inspection |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial server requirements SV-FR-001–SV-FR-059, SV-IR-001–SV-IR-006, SV-NFR-001–SV-NFR-008 |
