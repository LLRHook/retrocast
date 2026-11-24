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
- **Rev 2 (2026-11-23).** Delivery was re-read at `origin/main` 55e50eb (PR #23 merged) and
  checked against the compose smoke run of 2026-11-23 on that tree. Where the smoke showed a
  requirement broken, Delivery says so. Requirements added in rev 2 record the owner's answers
  (PRODUCT_BRIEF §14) and are `Not started` unless code already does them.
- **UI** names the client screens: `web —` (retrocast-web) and `ios —` (retrocast-client).
- Priorities: MVP capabilities are `Must`; Later capabilities are `Could` (PRODUCT_BRIEF §10,
  Q-003). The iOS client follows the web client (Q-004), so the server's Must requirements are
  proved through the web client and API tests.
- Defect ids (D-nn) are listed in [ARCHITECTURE.md §9](../ARCHITECTURE.md#9-risks-and-technical-debt).

## 3. Requirements

### 3.1 Functions

#### 3.1.1 Accounts and sessions (CAP-001, CAP-002)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-001 | Must | The server shall create an account for an unused username matching `^[a-zA-Z0-9_]{2,32}$` with a password of 6–128 bytes in UTF-8. | Given registration mode `open` and an unused valid username and password, when the client calls API-001, then the server responds 200 with an access token, a refresh token and the user (display name equal to the username). | Test | Implemented | web — Register page; ios — RegisterView | ENT-001, ENT-019, API-001, CMP-001 | CAP-001 | Baseline evidence before PR #54: internal/service/auth.go:54; test: internal/api/auth_handler_test.go. The legacy docs/legacy/api.md said 201; the code returns 200. Changed: rev 2 — acceptance names mode `open`; the other modes are SV-FR-062–SV-FR-065 (TBD-012 resolved). The code behaves as `open` in every case today. Changed: rev 3 — the password length is stated in UTF-8 bytes, which is what auth.go:58 counts. PR #54 (including round 2): internal/service/auth.go Register; internal/api/auth_handler_test.go TestRegister_Success; atomicity_live_test.go bootstrap (open-mode owner registration). |
| SV-FR-002 | Must | The server shall refuse a username that already exists with 409 `USERNAME_TAKEN`. | Given a registered username, when API-001 is called with it, then the server responds 409 `USERNAME_TAKEN` and creates no account. | Test | Implemented | web — Register page; ios — RegisterView | ENT-001, API-001 | CAP-001 | internal/service/auth.go:80 (username lookup), :84 (conflict); concurrent unique constraint mapped at :135; unique constraint `users_username_key` (migrations/000001). Uniqueness is case-sensitive. |
| SV-FR-003 | Must | The server shall refuse a username or password outside the SV-FR-001 rules with 400 (`INVALID_USERNAME` or `INVALID_PASSWORD`). | Given a one-character username, when API-001 is called, then the server responds 400 `INVALID_USERNAME`; given a password of 5 bytes, then 400 `INVALID_PASSWORD`. | Test | Implemented | web — Register page; ios — RegisterView | API-001 | CAP-001 | internal/service/auth.go:73 (username validation), :76 (password validation); internal/api/auth_handler_test.go. |
| SV-FR-004 | Must | The server shall issue a new token pair for a correct username and password. | Given a registered user, when API-002 is called with the right password, then 200 with tokens; given a wrong password or an unknown username, then 401 `INVALID_CREDENTIALS` with the same message. | Test | Implemented | web — Login page; ios — LoginView | ENT-001, ENT-019, API-002 | CAP-001 | internal/service/auth.go Login (lines 148-162); internal/api/auth_handler_test.go login cases. |
| SV-FR-005 | Must | The server shall exchange a valid refresh token for a new token pair and invalidate the old refresh token. | Given a refresh token issued less than 7 days ago, when API-003 is called, then 200 with a new pair; when the old token is used again, then 401 `INVALID_TOKEN`; given two simultaneous API-003 calls with the same refresh token, then exactly one gets 200 and the other gets 401 `INVALID_TOKEN`. | Test | Implemented | none (system-only: clients call it on a 401) | ENT-019, API-003 | CAP-002 | Baseline evidence before PR #54: internal/service/auth.go:109–141; internal/auth/jwt.go PR #54 (including round 2): internal/redis/client.go ConsumeRefreshTokenUserID; internal/api/atomicity_live_test.go refresh_race (5 rounds of 256 callers). |
| SV-FR-006 | Must | The server shall invalidate the presented refresh token on logout. | Given a signed-in user, when API-004 is called with the refresh token, then 204 and that token no longer refreshes. | Test | Implemented | web — User settings, Log out; ios — UserSettingsView | ENT-019, API-004 | CAP-002 | internal/service/auth.go:144 |
| SV-FR-060 | Must | The server shall change the caller's password when the current password is correct and the new password meets the SV-FR-001 rules. | Given a signed-in user, when API-059 is called with the correct current password and a valid new one, then 200 and API-002 accepts only the new password; given a wrong current password, then 403 `INVALID_CREDENTIALS`; given a 5-character new password, then 400 `INVALID_PASSWORD`. | Test | Implemented | web — User settings, Password (WEB-FR-019); ios — none yet (Q-004) | ENT-001, API-059, CMP-001 | CAP-037 | Added rev 2 (Q-022). A wrong current password returns 403, not 401, so a client does not mistake it for an expired session. Rate limit: SV-NFR-003.  L25: account/session tests and local browser DOM evidence. |
| SV-FR-061 | Must | The server shall revoke every refresh token of a user whose password changes and return a new token pair to the caller. | Given a user signed in on two devices, when one calls API-059, then its response carries a new token pair and API-003 with the other device's refresh token returns 401 `INVALID_TOKEN`. | Test | Implemented | none (system-only) | ENT-019, API-059, API-003, CMP-005 | CAP-037, RISK-012 | Added rev 2 (Q-022). Access tokens already issued stay valid until they expire (15 minutes, SV-NFR-002).  L25: account/session tests and local browser DOM evidence. |

#### 3.1.2 Servers (CAP-004, CAP-006)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-007 | Must | The server shall create a guild owned by the caller when given a name of 2–100 characters. | Given a signed-in user, when API-013 is called with name "Friends", then 201 with the guild, `owner_id` equal to the caller, and EVT-012 is sent to the caller.  Injected failure leaves no guild rows or gateway publication. | Test | Implemented | web — Server list, Create server modal; ios — CreateGuildSheet | ENT-002, ENT-004, API-013, EVT-012, CMP-001 | CAP-004 | Baseline evidence before PR #54: internal/service/guild.go:47; test: internal/api/guild_handler_test.go. The steps are not in one transaction (ARCHITECTURE §9 D-12). PR #54 (including round 2): internal/api/atomicity_live_test.go guild_rollback (mid-write rollback and no publication). |
| SV-FR-008 | Must | The server shall give each new guild a default role set (an `@everyone` role with `DefaultEveryonePerms` and an `Admin` role with Administrator held by the owner). | Given a new guild, when its roles are listed, then `@everyone` (is_default, position 0, permissions 230147) and `Admin` (Administrator, position 1) exist and the owner holds both. | Test | Implemented | web — Guild settings, Roles; ios — GuildSettingsView | ENT-005, ENT-006 | CAP-004, CAP-014, CAP-015 | internal/service/guild.go:64–104; internal/permissions/bitfield.go:46. Changed: rev 2 — `@everyone` gets `DefaultEveryonePerms` = 230147 (View, Send, Read history, Connect, Speak, Create invite, Change nickname; Q-014, TBD-014 resolved). Guilds created before the fix keep their stored value (assumption, no data migration). PR #44: permission fixes covered by internal/api/permission_regression_test.go and permission_round2_test.go. |
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
| SV-FR-019 | Must | The server shall show an invite's guild name, member count and creator to anyone holding the code. | Given a valid code, when API-049 is called without a token, then 200 with `guild_name`, `member_count` (members at request time) and `creator_id`; given an expired code, then 404 `EXPIRED`. | Test | Implemented | web — Join server modal; ios — JoinGuildSheet | ENT-010, API-049 | CAP-005 | internal/service/invite.go:108–137. API-049 is limited to 5 requests per minute per client IP (PR #53). |
| SV-FR-020 | Must | The server shall add the caller to the guild when they accept a valid invite. | Given a valid code, when API-050 is called, then 200 with the guild, `uses` increases by 1 and EVT-018 reaches members; given an expired or used-up code, then 410 (`EXPIRED`, `MAX_USES`); given a member, then 409 `ALREADY_MEMBER`; given a banned user, then 403 `BANNED`.  Concurrent accepts commit at most max_uses memberships and uses. | Test | Implemented | web — Join server modal; ios — JoinGuildSheet | ENT-004, ENT-010, API-050, EVT-018 | CAP-005 | Baseline evidence before PR #54: internal/api/permission_regression_test.go; internal/service/. PR #54 (including round 2): internal/api/atomicity_live_test.go invite_race, commit_rollback_registration=false and guild_delete_race (bounded uses, commit rollback, parent-first lock order). |
| SV-FR-021 | Must | The server shall let a member with `MANAGE_GUILD` (or the invite's creator, for revoking) list and revoke a guild's invites. | Given a member without `MANAGE_GUILD`, when API-048 is called, then 403; given the creator, when API-051 is called, then 204. | Test | Implemented | web — Invite modal; ios — InviteSheet | ENT-010, API-048, API-051 | CAP-008 | internal/service/invite.go:92–105, :198–215 |
| SV-FR-022 | Must | The server shall let a member other than the owner leave a guild. | Given a member, when API-035 is called, then 204 and EVT-019 is sent; given the owner, then 403 `FORBIDDEN` ("transfer ownership or delete the guild"). | Test | Implemented | web — Guild settings, Leave; ios — GuildSettingsView | ENT-004, API-035, EVT-019 | CAP-005 | internal/service/member.go:221–240. Ownership transfer is Later (CAP-039, Q-026; TBD-026 resolved). |
| SV-FR-023 | Must | The server shall list a guild's members to its members in pages of 1–100 (default 50). | Given a guild with 60 members, when API-033 is called with no limit, then 50 members are returned with their role ids. | Test | Implemented | web — Member list; ios — MemberListView | ENT-004, ENT-005, API-033 | CAP-012 | internal/service/member.go:39–63; test: internal/api/member_handler_test.go |
| SV-FR-024 | Must | The server shall return one member's record to members of the same guild. | Given a non-member, when API-036 is called, then 404. | Test | Implemented | web — User profile popover; ios — UserProfilePopover | ENT-004, API-036 | CAP-012 | internal/service/member.go:66–84 |

#### 3.1.5 Messages (CAP-009, CAP-010, CAP-013)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-025 | Must | The server shall store a message of 1–2000 bytes from a member who has `SEND_MESSAGES` in the channel. | Given a member whose role is denied `SEND_MESSAGES` by a channel override, when API-022 is called, then 403; given a member with the permission, then 201 with the message and its author fields. | Test | Implemented | web — Chat area, message input; ios — MessageInput | ENT-008, API-022, CMP-001 | CAP-009 | internal/service/message.go:44–84; test: internal/api/message_handler_test.go |
| SV-FR-026 | Must | The server shall push each new guild message to the guild's connected members as EVT-009. | Given two members connected to the gateway, when one sends a message, then the other receives EVT-009 with the full message. | Test | Implemented | web — Chat area; ios — MessageListView | EVT-009, CMP-002 | CAP-009 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |
| SV-FR-027 | Must | The server shall return a channel's messages newest first in pages of 1–100 (default 50) before an optional message-id cursor. | Given a channel with 120 messages, when API-023 is called with `before` set to the 51st newest id, then the next 50 older messages are returned; given a member without `READ_MESSAGE_HISTORY`, then 403. | Test | Implemented | web — Message list, scroll back; ios — MessageListView | ENT-008, ENT-009, API-023 | CAP-010 | internal/api/message_handler.go:49–82; internal/database/message_repo.go:47; test: internal/database/message_repo_test.go |
| SV-FR-028 | Must | The server shall return a single message to members who can read the channel's history. | Given a message id in another channel, when API-024 is called, then 404. | Test | Implemented | none (system-only) | ENT-008, API-024 | CAP-010 | internal/service/message.go:110–131 |
| SV-FR-029 | Could | The server shall let only the author edit a message, setting `edited_at` and sending EVT-010. | Given another member's message, when API-025 is called, then 403; given the author with valid content, then 200 with `edited_at` set. | Test | Implemented | web — Message context menu, Edit | ENT-008, API-025, EVT-010 | CAP-013 | internal/service/message.go:134–180 |
| SV-FR-030 | Could | The server shall let the author or a member with `MANAGE_MESSAGES` delete a guild message, sending EVT-011. | Given a moderator with `MANAGE_MESSAGES`, when API-026 deletes another member's message, then 204 and EVT-011 is sent; in a DM, only the author may delete. | Test | Implemented | web — Message context menu, Delete | ENT-008, API-026, EVT-011 | CAP-013 | internal/service/message.go:183–222 |

#### 3.1.6 Typing and presence (CAP-011, CAP-012)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-031 | Must | The server shall send EVT-024 to a guild's connected members when a member who can send in the channel signals typing. | Given a member with `SEND_MESSAGES`, when API-028 is called, then other connected members receive EVT-024 with channel, guild and user; given a user who is not a member, then 403. | Test | Implemented | web — Typing indicator; ios — TypingIndicator | API-028, EVT-024, ENT-021 | CAP-011 | PR #44: caller SEND_MESSAGES check; PR #50: VIEW_CHANNEL fan-out and invalidation. Tests: internal/gateway/session_live_test.go and review_regression_test.go; internal/api/permission_regression_test.go. Changed rev 11: B3 typing delivery implemented. |
| SV-FR-032 | Must | The server shall mark a user online on IDENTIFY and send EVT-025 to members of every guild they share. | Given two members of one guild, when one identifies, then the other receives EVT-025 with status `online`. | Test | Implemented | web — Member list, presence dots; ios — PresenceDot | EVT-003, EVT-025, ENT-020 | CAP-012 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |
| SV-FR-033 | Must | The server shall accept a client status (`online`, `idle`, `dnd`, `invisible`) and show `invisible` to others as `offline`. | Given a connected user, when they send op 3 with `invisible`, then other members receive EVT-025 with `offline`. | Test | Implemented | web — User settings; ios — AppSettingsView | EVT-008, EVT-025, ENT-020 | CAP-012 | internal/gateway/manager.go:338–370; test: internal/gateway/manager_test.go |
| SV-FR-034 | Must | The server shall send EVT-025 with `offline` 10 seconds after a user's last connection closes unless they reconnect. | Given a connected user, when the connection drops and no new one opens within 10 s, then members receive `offline`. | Test | Implemented | web — Member list, presence dots | EVT-025, ENT-020 | CAP-012 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |

#### 3.1.7 Gateway session (CAP-009, CAP-002)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-035 | Must | The server shall send EVT-001 with a heartbeat interval of 41 250 ms on every new gateway connection. | Given a WebSocket client, when it connects to `/gateway`, then the first frame is op 10 with `heartbeat_interval` 41250. | Test | Implemented | none (system-only) | EVT-001, CMP-002 | CAP-009 | internal/gateway/server.go; internal/gateway/connection.go |
| SV-FR-036 | Must | The server shall answer a valid IDENTIFY with EVT-003 and close the connection on an invalid token. | Given a valid access token, when op 2 is sent, then EVT-003 arrives with session id, user id, guild ids and read states; given an expired token, then the server closes the socket. | Test | Implemented | none (system-only) | EVT-002, EVT-003, CMP-002 | CAP-009 | internal/gateway/manager.go:221–280. Guild ids are decimal strings (SV-IR-004, PR #45). |
| SV-FR-037 | Must | The server shall acknowledge each heartbeat and close a connection that sends none for the interval plus 10 seconds. | Given an identified client, when it sends op 1, then op 11 follows; when it stays silent for 51.25 s, then the server closes the socket. | Test | Implemented | none (system-only) | EVT-004, EVT-005 | CAP-009 | internal/gateway/connection.go:92–155 |
| SV-FR-038 | Must | The server shall replay the events a client missed when it resumes a session within the replay buffer. | Given a client that dropped after event N, when it sends op 6 with its valid token, session id and sequence N within the replay window, then later events replay in order with their original sequence numbers; given an unknown, expired or foreign session id, or a sequence outside the retained window, then READY arrives with a new session_id and sequence 1, full guilds, read states and a presence snapshot. Invalid tokens close the socket as IDENTIFY does. | Test | Implemented | none (system-only) | EVT-006, CMP-002 | CAP-009 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |
| SV-FR-039 | Won't | Status: Deprecated (rev 2, superseded by SV-FR-072). The server shall keep one gateway connection per user and send EVT-007 to the older connection when a new one identifies. | Given a user connected on web, when the same user identifies on iOS, then the web connection receives op 7 and is closed. | Test | Deprecated (rev 2, superseded by SV-FR-072; ADR-017 built in PR #50) | web — reconnect banner; ios — ErrorBanner | EVT-007, CMP-002 | CAP-009 | internal/gateway/manager.go; test: internal/gateway/manager_test.go. Changed: rev 2 — priority Must to Won't: Q-009 allows several connections per user (TBD-009 resolved). |
| SV-FR-071 | Must | The server shall keep each open gateway connection's guild subscriptions equal to its user's current memberships. | Given a user with an open connection, when they create a guild or accept an invite, then EVT-009 for that guild reaches the connection without a reconnect; when they leave, are kicked or are banned, then no later event of that guild reaches any of their connections. | Test | Implemented | web — Chat area; Server list (WEB-FR-023) | EVT-009, EVT-012, EVT-018, EVT-019, CMP-002 | CAP-009, CAP-005, CAP-031 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |
| SV-FR-072 | Must | The server shall keep every gateway connection a user opens (each receiving every event for that user) and mark the user offline only after the last one closes. | Given a user connected on web, when the same user identifies on iOS, then both connections stay open and each receives the next EVT-009; when the web connection closes, then no EVT-025 `offline` is sent; when the iOS one also closes, then SV-FR-034 applies. | Test | Implemented | web — Chat area; ios — ChatAreaView | EVT-003, EVT-009, EVT-025, CMP-002 | CAP-009, CAP-012 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |

#### 3.1.8 Roles and permissions (CAP-014, CAP-015, CAP-016)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-040 | Must | The server shall create a role only for a member who has `MANAGE_ROLES` and only at a position below that member's highest role. | Given a member without `MANAGE_ROLES`, when API-039 is called, then 403; given an admin whose highest role is at position 3, when the new role's position is 3 or more, then 403 `ROLE_HIERARCHY`. | Test | Implemented | web — Role editor; ios — RoleEditorView | ENT-006, API-039, EVT-021 | CAP-014, CAP-015 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-041 | Must | The server shall update a role only when it belongs to the guild in the path and the caller has `MANAGE_ROLES` above that role. | Given a role of guild A, when API-041 is called under guild B, then 404; given a member without `MANAGE_ROLES`, then 403. | Test | Implemented | web — Role editor; ios — RoleEditorView | ENT-006, API-041, EVT-022 | CAP-014, CAP-015 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-042 | Must | The server shall delete a role only under the same rules as SV-FR-041 and never delete `@everyone`. | Given the `@everyone` role, when API-042 is called by the owner, then 403 `CANNOT_DELETE`; given a member without `MANAGE_ROLES`, then 403. | Test | Implemented | web — Role editor | ENT-006, API-042, EVT-023 | CAP-014, CAP-015 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-043 | Must | The server shall assign or remove a role only for a caller with `MANAGE_ROLES` whose highest role is above the role. | Given a member without `MANAGE_ROLES`, when API-043 assigns the Admin role to themself, then 403. | Test | Implemented | web — Member list, role assignment; ios — MemberRow | ENT-005, API-043, API-044, API-037, EVT-020 | CAP-014, CAP-015 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-044 | Must | The server shall list a guild's roles only to its members. | Given a non-member, when API-040 is called, then 404. | Test | Implemented | web — Guild settings, Roles | ENT-006, API-040 | CAP-014 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-045 | Must | The server shall compute a member's channel permissions from the `@everyone` role, the member's roles and the channel's overrides in the order of DATA_CONTRACTS §3.2. | Given `@everyone` allowed `SEND_MESSAGES` and a channel override for `@everyone` that denies it, when a member without other roles sends there, then 403; given a role override that allows it, then 201. | Test | Implemented | none (system-only) | ENT-005, ENT-006, ENT-007, CMP-001 | CAP-015 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-046 | Must | The server shall grant every permission to the guild owner and to holders of a role with Administrator. | Given the owner with no roles, when any guild-scoped action is called, then it is allowed. | Test | Implemented | none (system-only) | ENT-002, ENT-006 | CAP-015 | internal/service/permission.go; test: internal/permissions/resolver_test.go |
| SV-FR-047 | Could | The server shall let only a member with `MANAGE_ROLES` set or delete a channel's override for a role of the same guild. | Given a member without `MANAGE_ROLES`, when API-045 is called, then 403; given a role of another guild, then 404. | Test | Implemented | web — Channel settings (none yet); ios — none | ENT-007, API-045, API-046 | CAP-016 | internal/api/permission_regression_test.go; implementation: internal/service/permission.go, role.go, member.go, guild.go. Fixed in PR #44. |
| SV-FR-078 | Must | The server shall enforce guild base permission grant limits for role managers (creation, update, assignment, removal, replacement and override allows; owner and Administrator bypass) with @everyone fixed at position 0 for all callers. | Given a MANAGE_ROLES and KICK_MEMBERS moderator at position 2, when they create an Administrator role, update @everyone to Administrator, assign or remove the Admin role at position 1, replace member roles with that role, or allow unheld permissions in an override, then 403 MISSING_PERMISSIONS and no role mutation; given a KICK_MEMBERS-only role below them, then create and assign to a lower member succeed; given an owner or Administrator, then all permission bits can be granted; given any caller moving @everyone away from 0, then 400 INVALID_POSITION. Given a role holding permissions outside the caller base, name-only or color-only updates, stripping its bits and deleting it are refused with 403 MISSING_PERMISSIONS, while roles left unchanged by a member role replacement are not checked. | Test | Implemented | web - Role editor | ENT-005, ENT-006, ENT-007, API-037, API-039, API-041, API-043, API-044, API-045 | CAP-014 | Added rev 5: driver decision after L11 F1, RISK-001. internal/service/permission.go (GuildBasePermissions, RequireGrantPermissions); internal/service/role.go; internal/service/member.go; test: internal/api/permission_round2_test.go. Pilot. Changed rev 6: current and new bits checked on update; deletion protected; member PATCH validates only changed roles before any write. Test: internal/api/permission_round3_test.go. |

#### 3.1.9 Voice (CAP-017)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-048 | Could | The server shall give a member with `CONNECT` on a voice channel a LiveKit token for room `voice-{channel id}` and record their voice state. | Given a member with `CONNECT`, when API-056 is called on a voice channel, then 200 with a token and the channel's voice states, and EVT-026 reaches members; given a text channel, then 400. | Test | Implemented | ios — VoiceChannelView; web — none (web voice is Later, audio only: Q-024) | ENT-016, API-056, EVT-026, CMP-007 | CAP-017 | internal/service/voice.go:53–112; test: internal/api/voice_handler_test.go. The token lasts 24 h and the response has no LiveKit URL. |
| SV-FR-049 | Could | The server shall remove the caller's voice state when they leave and send EVT-026. | Given a user in a voice channel, when API-057 is called, then 204 and members receive EVT-026 with that user's `channel_id` null. | Test | Implemented | ios — VoiceControlBar | ENT-016, API-057, EVT-026 | CAP-017 | internal/service/voice.go:115–149. A client that disconnects without calling it stays listed: no LiveKit webhook. |
| SV-FR-050 | Could | The server shall list the voice states of a voice channel to guild members. | Given two users in a voice channel, when API-058 is called, then both states are returned. | Test | Implemented | ios — VoiceChannelView | ENT-016, API-058 | CAP-017 | internal/service/voice.go:152 |

#### 3.1.10 Attachments (CAP-019)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-051 | Could | The server shall accept a file of at most 10 MB of an allowed type (JPEG, PNG, GIF, WebP, PDF, plain text) from a member with `ATTACH_FILES` through a pre-signed upload URL valid for 15 minutes. | Given a member with `ATTACH_FILES`, when they request an upload for a 1 MB PNG, upload it to the returned URL and send a message naming it, then the message carries the attachment; given an 11 MB file, then 400 `FILE_TOO_LARGE`; given an SVG, then 400 `INVALID_CONTENT_TYPE`; given an upload URL older than 15 minutes, then the upload is refused. | Test | Partial (storage and atomic linking fixed; Q-017 types enforced; pre-signed upload remains Later, ADR-019) | web — File upload, drag and drop | ENT-009, ENT-022, API-027, CMP-006 | CAP-019 | internal/service/upload.go; message.go; tests: internal/service/storage_integration_test.go, internal/service/upload_test.go, internal/api/storage_regression_test.go; migration 000020. D-09 and D-22 fixed. |
| SV-FR-052 | Could | The server shall include each message's attachments with a pre-signed download URL that is valid for 15 minutes through the public entry point. | Given a message with an attachment, when API-023 is called, then the attachment appears with an `https` URL on the public host that downloads the file for 15 minutes and is refused after that. | Test | Partial (attachments loaded in create/single/history/edit/search with existing internal HTTP URLs; public pre-signed URLs remain D-10) | web — Message list, image preview | ENT-009, API-023 | CAP-019 | internal/database/message_repo.go; internal/service/message.go and search.go; test: internal/service/storage_integration_test.go. |

#### 3.1.11 Profile (CAP-003)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-053 | Could | The server shall let a user read their profile and change their display name (1–32 characters) or avatar reference. | Given a signed-in user, when API-006 sets display name "Vic", then 200 with the new name; given a 33-character name, then 400. | Test | Partial (the avatar is a free-form hash string; no endpoint uploads an avatar or a server icon) | web — User settings; ios — UserSettingsView | ENT-001, API-005, API-006 | CAP-003 | internal/service/user.go:21–57; test: internal/api/user_handler_test.go |

#### 3.1.12 Direct messages (CAP-020, CAP-021)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-054 | Could | The server shall return the existing one-to-one DM between the caller and a recipient who shares a guild with the caller or create it. | Given no DM between A and B, when A calls API-008 with B's id, then 200 with a type-1 channel; when A calls it again, then the same channel is returned; given a user C who shares no guild with A, then 403 `NO_SHARED_GUILD`. | Test | Partial (any user id is accepted) | web — DM list, New DM; ios — DMListView | ENT-012, ENT-013, API-008 | CAP-020 | internal/service/dm.go:40–70; test: internal/api/dm_handler_test.go, internal/database/dm_repo_test.go. Changed: rev 2 — shared-guild rule from Q-013 (TBD-013 resolved). |
| SV-FR-055 | Could | The server shall store messages in a DM for its recipients and send them only to its recipients. | Given a DM between A and B, when A calls API-022 on it, then 201 and B receives EVT-009; given user C, then 403 or 404. | Test | Implemented | web — DM conversation; ios — DMListView | ENT-008, ENT-012, API-022, EVT-009 | CAP-020 | Migration 000020 validates and locks guild/DM parents and preserves delete cleanup; internal/service/message.go; test: internal/service/storage_integration_test.go (one-to-one/group storage, history, recipient-only fan-out, nonparticipant refusal), internal/database/storage_migration_test.go. D-08 fixed. |
| SV-FR-056 | Could | The server shall create a group DM of up to 10 participants owned by its creator when every recipient shares a guild with the creator. | Given a creator and 9 recipients, when API-008 is called with `recipient_ids`, then 200 with a type-3 channel; given 10 recipients, then 400; given a recipient who shares no guild with the creator, then 403 `NO_SHARED_GUILD`. | Test | Partial (any user id is accepted) | web — DM list; ios — DMListView | ENT-012, ENT-013, API-008 | CAP-021 | internal/service/dm.go:73–139; test: internal/api/group_dm_handler_test.go. Changed: rev 2 — shared-guild rule from Q-013. |
| SV-FR-057 | Could | The server shall let a group DM's owner add or remove participants and let any participant remove themself. | Given a participant who is not the owner, when API-010 adds a user, then 403; when API-011 removes themself, then 204; given the owner adding a user who shares no guild with the owner, then 403 `NO_SHARED_GUILD`. | Test | Partial (any user id can be added) | web — DM conversation header | ENT-013, API-010, API-011 | CAP-021 | internal/service/dm.go:142–258; test: internal/api/group_dm_handler_test.go. Changed: rev 2 — shared-guild rule from Q-013. |
| SV-FR-058 | Could | The server shall list the caller's DMs with their recipients. | Given a user in two DMs, when API-009 is called, then both are returned with recipient profiles. | Test | Implemented | web — DM list; ios — DMListView | ENT-012, ENT-013, API-009 | CAP-020 | internal/service/dm.go:261 |

#### 3.1.13 Search (CAP-028)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-059 | Could | The server shall return a guild's messages matching a text query (optional author and date filters) in pages of 1–100 (default 25) to members who can read the history. | Given messages containing "pizza", when API-055 is called with `q=pizza`, then those messages are returned newest first; given a non-member, then 404. | Test | Partial (channel VIEW_CHANNEL and READ_MESSAGE_HISTORY filtering fixed, PR #44; bounded pages can underfill after filtering; ranking still precedes time and non-members still receive 403) | ios — SearchView; web — none | ENT-008, API-055 | CAP-028 | internal/service/search.go:33; internal/database/message_repo.go:91; migrations/000015 (English stemming: Q-023, TBD-023 resolved); test: internal/api/search_handler_test.go; PR #44: permission fixes covered by internal/api/permission_regression_test.go and permission_round2_test.go. |

#### 3.1.14 Registration mode (CAP-040)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-062 | Must | The server shall read the registration mode from `REGISTRATION_MODE` (`open`, `invite` or `closed`; default `invite`) and refuse to start on any other value. | Given no `REGISTRATION_MODE`, when the server starts, then registration behaves as `invite`; given `REGISTRATION_MODE=public`, then the server exits with an error naming the key. | Test | Implemented | none (operator configuration) | API-001, CMP-001 | CAP-040, RISK-003 | Baseline evidence before PR #54: Added rev 2 (Q-012); DATA_CONTRACTS §1. ADR-014. PR #54 (including round 2): internal/config/config_test.go TestRegistrationMode (default, allowed modes, invalid mode rejection). |
| SV-FR-063 | Must | In `invite` mode the server shall create an account only with a valid invite code (consuming one use and adding the user to the invite's guild in one transaction). | Given mode `invite`, when API-001 is called without `invite_code`, then 403 `INVITE_REQUIRED`; given an expired, revoked or used-up code, then 403 `INVALID_INVITE` and no account is created; given a valid code with `max_uses` 1, then 200, the new user is a member of the invite's guild and a second registration with that code returns 403 `INVALID_INVITE`. | Test | Implemented | web — Register page, invite code field (WEB-FR-020); ios — none yet | ENT-001, ENT-004, ENT-010, API-001, CMP-001 | CAP-040, CAP-005 | Baseline evidence before PR #54: Added rev 2 (Q-012). ADR-014. PR #54 (including round 2): internal/api/atomicity_live_test.go registration_invite_and_rollback, commit_rollback_registration=true and guild_delete_race (valid/invalid/expired/exhausted code, account/member/use rollback). |
| SV-FR-064 | Must | In `closed` mode the server shall refuse every registration with 403 `REGISTRATION_CLOSED`. | Given mode `closed`, when API-001 is called with a valid unused username, then 403 `REGISTRATION_CLOSED` and no account is created; when an existing user calls API-002, then sign-in succeeds. | Test | Implemented | web — Register page, error message (WEB-FR-020) | API-001, API-002, CMP-001 | CAP-040 | Baseline evidence before PR #54: Added rev 2 (Q-012). PR #54 (including round 2): internal/api/atomicity_live_test.go closed_preserves_login. |
| SV-FR-065 | Must | In `invite` mode the server shall accept one registration without an invite code while the instance has no accounts. | Given an instance with no users in mode `invite`, when two registrations without a code are sent at once, then exactly one succeeds and the other returns 403 `INVITE_REQUIRED`. | Test | Implemented | web — Register page | ENT-001, API-001, CMP-001 | CAP-040, CAP-004 | Baseline evidence before PR #54: Added rev 2. ADR-014: the first user creates the first server and its invites. PR #54 (including round 2): internal/api/atomicity_live_test.go bootstrap and bootstrap_lock (HTTP singleton plus blocked competing transaction). |

#### 3.1.15 Moderation (CAP-031)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-066 | Must | The server shall let a member with `KICK_MEMBERS` remove another member from the guild within the SV-FR-070 rule. | Given a moderator with `KICK_MEMBERS` ranked above the target, when API-038 is called, then 204, the target is absent from API-033, EVT-019 reaches the guild and the target can rejoin with a valid invite; given a member without `KICK_MEMBERS`, then 403. | Test | Implemented | web — User profile popover, Kick (WEB-FR-021); ios — none yet | ENT-004, API-038, EVT-019, CMP-001 | CAP-031 | PR #50; internal/gateway/session_live_test.go and membership_live_test.go; service removal assertions in internal/api/member_handler_test.go and ban_handler_test.go. Changed rev 10: B3 gateway delivery; SV-FR-038 fallback specified by driver decision. |
| SV-FR-067 | Must | The server shall let a member with `BAN_MEMBERS` ban a user with an optional reason (removing their membership and blocking invite acceptance) within the SV-FR-070 rule. | Given a moderator ranked above the target, when API-053 is called with reason "spam", then 204, EVT-027 and EVT-019 reach the guild and the user's later API-050 returns 403 `BANNED`; given a member without `BAN_MEMBERS`, then 403. | Test | Implemented | web — User profile popover, Ban with reason (WEB-FR-021) | ENT-004, ENT-011, API-053, API-050, EVT-019, EVT-027, CMP-001 | CAP-031 | Added rev 2 (TBD-035 resolved). internal/service/ban.go:43–112; internal/service/invite.go:140–195 (`BANNED`) |
| SV-FR-068 | Must | The server shall list a guild's bans (user and reason) to members with `BAN_MEMBERS`. | Given two bans, when a moderator calls API-052, then both are returned with their reasons; given a member without `BAN_MEMBERS`, then 403. | Test | Implemented | web — Guild settings, Bans (WEB-FR-022) | ENT-011, API-052, CMP-001 | CAP-031 | Added rev 2. internal/service/ban.go |
| SV-FR-069 | Must | The server shall let a member with `BAN_MEMBERS` lift a ban. | Given a banned user, when a moderator calls API-054, then 204, EVT-028 reaches the guild and the user can accept a valid invite; given a member without `BAN_MEMBERS`, then 403. | Test | Implemented | web — Guild settings, Bans, Unban (WEB-FR-022) | ENT-011, API-054, EVT-028, CMP-001 | CAP-031 | Added rev 2. internal/service/ban.go |
| SV-FR-070 | Must | The server shall allow a kick or ban only when the caller is the guild owner or the caller's highest role position is strictly greater than the target's. | Given a caller and a target whose highest roles are both at position 2, when API-038 or API-053 is called, then 403 `ROLE_HIERARCHY`; given the owner as target, then 403 `FORBIDDEN`; given the caller as target, then 400 `CANNOT_KICK_SELF` or `CANNOT_BAN_SELF`; given a target with only `@everyone` (position 0) and a caller at position 1, then the action succeeds. | Test | Implemented | web — User profile popover (WEB-FR-021) | ENT-004, ENT-006, API-038, API-053, CMP-001 | CAP-031, CAP-015 | Added rev 2. internal/service/ban.go (`ROLE_HIERARCHY`, `CANNOT_BAN_SELF`, `highestPosition`); internal/service/member.go:226-275; PR #44: permission fixes covered by internal/api/permission_regression_test.go and permission_round2_test.go. |

#### 3.1.16 Reactions, unread state and nicknames (CAP-032, CAP-033, CAP-034; Later)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-074 | Could | The server shall let a member who can read a channel's history add or remove their own emoji reaction on a message (sending EVT-029 or EVT-030). | Given a member with `VIEW_CHANNEL` and `READ_MESSAGE_HISTORY`, when API-030 adds 👍, then 204 and EVT-029 reaches the channel's members; when API-031 removes it, then 204 and EVT-030 is sent; given an empty emoji, then 400. | Test | Implemented | web — Message reactions | ENT-014, API-030, API-031, EVT-029, EVT-030, CMP-001 | CAP-032 |Added rev 2 (Q-003: Later; TBD-036 resolved). internal/service/reaction.go D-31 fixed: remove/list validate message existence and route channel; test: internal/api/reaction_handler_test.go (TestReactionRouteBinding). |
| SV-FR-075 | Could | The server shall list the users who reacted to a message with one emoji. | Given two users who reacted with 👍, when API-032 is called, then both user ids are returned as strings. | Test | Implemented | web — Message reactions, tooltip | ENT-014, API-032, CMP-001 | CAP-032 | String ids implemented in REST v2 (PR #45); D-31 route binding fixed; tests: internal/api/reaction_handler_test.go. |
| SV-FR-076 | Could | The server shall record the last message a user read in each channel and return the user's read states. | Given a member, when API-029 acks message M in channel C, then API-012 and the next READY list C with `last_message_id` M. | Test | Implemented | web — Channel sidebar, unread marks | ENT-015, API-012, API-029, EVT-003, CMP-001 | CAP-033 | Added rev 2 (Q-003: Later; TBD-037 resolved). internal/service/read_state.go |
| SV-FR-077 | Could | The server shall let a member with `CHANGE_NICKNAME` set their own nickname (at most 32 characters) and a member with `MANAGE_NICKNAMES` set another member's. | Given a member whose roles lack `CHANGE_NICKNAME`, when API-034 sets a nickname, then 403; given a moderator with `MANAGE_NICKNAMES`, when API-037 sets another member's nickname to "Vic", then 200 and EVT-020 is sent; given 33 characters, then 400. | Test | Partial (API-034 checks no `CHANGE_NICKNAME`) | web — Member list, nickname | ENT-004, API-034, API-037, EVT-020, CMP-001 | CAP-034 | Added rev 2 (Q-003: Later; TBD-038 resolved). internal/service/member.go:87–190 |

#### 3.1.17 Admin CLI (CAP-036)

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-FR-073 | Should | The admin CLI shall exit 0 only when its command (`migrate`, `seed`, `health`, `version`) succeeds (2 for missing configuration, 3 for a connection failure, 1 otherwise). | Given no `DATABASE_URL`, when `retrocast-cli migrate` runs, then it exits 2; given a stopped server, when `retrocast-cli health` runs, then it exits 3; given a running stack, when `health` runs with `SERVER_URL` set to the public address, then it exits 0. | Test | Implemented | none (operator CLI) | CMP-003 | CAP-036 | Added rev 2 (Q-003: MVP; TBD-040 resolved). cmd/retrocast-cli/main.go. `SERVER_URL` defaults to `http://localhost:8080`, which stops working once 8080 is no longer published (SV-NFR-008); the operator sets `SERVER_URL=https://<host>`. The CLI is not in the server image. |

### 3.2 Performance requirements
Server performance targets are system-wide: see SYS-NFR-001 and SYS-NFR-002 in
[system.md](system.md).

### 3.3 Usability requirements
The server has no user interface; client usability is in [web.md](web.md) and [ios.md](ios.md).

### 3.4 Interface requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-IR-001 | Must | The server shall expose the REST operations API-001–API-059 under `/api/v1` with JSON bodies and bearer access tokens. | Given the OpenAPI document, when each operation is called as documented, then the status and body match DATA_CONTRACTS §4. | Test | Partial (API-059 does not exist yet) | none (system-only) | API-001–API-059 | LIM-003 | internal/api/router.go; docs/openapi.yaml. Changed: rev 2 — range extended to API-059; the body shape is SV-IR-007 (TBD-010 resolved). |
| SV-IR-002 | Must | The server shall frame every gateway message as one JSON object (`op`, `d`, `s`, `t`) per DATA_CONTRACTS §6. | Given a dispatch, when it is sent, then `op` is 0, `t` names the event and `s` increases per connection. | Test | Implemented | none (system-only) | EVT-001–EVT-030 | LIM-003 | internal/gateway/events.go; test: internal/gateway/manager_test.go |
| SV-IR-003 | Must | The server shall return every error as `{"error": {"code", "message"}}`. | Given a request without a token, when a protected route is called, then 401 with the error body. | Test | Implemented | none (system-only) | DATA_CONTRACTS §5 | LIM-003 | internal/api/response.go; internal/auth/middleware.go; Changed rev 7 - REST v2 and web contract fixes. |
| SV-IR-004 | Must | The server shall encode every snowflake id in JSON as a decimal string. | Given a member with roles, when API-033 is called, then each id in `roles` is a string; given READY, then each guild id is a string. | Test | Implemented | none (system-only) | ENT-004, EVT-003 | RISK-009 | internal/models/member.go; internal/gateway/events.go:78; Changed rev 7 - REST v2 and web contract fixes. |
| SV-IR-005 | Should | The server shall serve its OpenAPI document at `/docs/openapi.yaml` and a reference page at `/docs`. | Given a running server, when `/docs/openapi.yaml` is fetched, then the OpenAPI 3.0.3 document is returned. | Demonstration | Implemented | none (developer docs) | API-001–API-059 | LIM-003 | internal/api/router.go; docs/openapi.yaml. The `/docs` page loads Swagger UI from unpkg.com. |
| SV-IR-007 | Must | The server shall wrap every `/api/v1` success body as `{"data": …}` and send no body with a 204. | Given each operation in DATA_CONTRACTS §4 that returns a body, when it succeeds, then the body is a JSON object whose only key is `data`; given a 204 operation, then the body is empty; given an error, then the body has the SV-IR-003 shape. | Test | Partial (58 implemented operations conform; API-059 password change remains batch B5) | none (system-only) | API-001–API-059, CMP-001 | CAP-001, CAP-009, RISK-002 | Added rev 2 (Q-010; ADR-013). A breaking change: REST contract v2.0.0, path unchanged (DATA_CONTRACTS §9).; Changed rev 7 - REST v2 and web contract fixes. |
| SV-IR-008 | Must | The server's OpenAPI document shall declare `info.version` 2.0.0 and describe every success body inside `data`. | Given retrocast/docs/openapi.yaml, when it is read, then `info.version` is 2.0.0, it states no open-source licence, and every 2xx response with content has a schema whose only property is `data`. | Inspection | Implemented | none (developer docs) | API-001–API-059 | LIM-004, SYS-CON-004 | Added rev 2 (Q-007, Q-010). retrocast/docs/openapi.yaml; Changed rev 7 - REST v2 and web contract fixes. |
| SV-IR-006 | Could | The server shall mint LiveKit access tokens signed with the configured API secret that grant joining only the channel's room. | Given a join, when the token is decoded, then `iss` is the API key, `sub` is the user id and the grant is `roomJoin` for `voice-{channel id}`. | Test | Implemented | none (system-only) | DATA_CONTRACTS §7, CMP-007 | CAP-017 | internal/service/voice.go |

### 3.5 Logical database requirements
Entities ENT-001–ENT-022 are defined in [DATA_CONTRACTS.md §3](../DATA_CONTRACTS.md#3-entities).
Integrity rules that the server shall keep are on SV-FR-002 (unique usernames), SV-FR-012
(cascade on guild delete), SV-FR-055 (DM messages need a valid parent) and SV-FR-051
(attachments can remain pending until message creation). Messages and files are kept until their author or a moderator
deletes them; there is no automatic expiry (Q-021, TBD-021 resolved).

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
| SV-NFR-002 | Must | Security: the server shall accept an HS256 access token for 15 minutes and a single-use refresh token for 7 days. Measure: token lifetimes (`exp` − `iat` = 900 s; refresh TTL 604 800 s; refresh token of 32 random bytes). | Given a token issued 16 minutes ago, when a protected route is called, then 401; given two simultaneous refreshes with one refresh token, then exactly one succeeds. | Test | Implemented | none (system-only) | ENT-019 | RISK-012 | Baseline evidence before PR #54: internal/auth/jwt.go; test: internal/auth/jwt_test.go. No minimum length is enforced for `JWT_SECRET`. PR #54 (including round 2): internal/auth/jwt_test.go (token claims and lifetimes); internal/api/atomicity_live_test.go refresh_race (atomic single use). The JWT_SECRET minimum-length gap remains. |
| SV-NFR-003 | Must | Security: the server shall limit requests per client IP (SV-NFR-009) on unauthenticated routes (5 per minute per route) and per user on authenticated routes (50 per minute per route), answering 429 with `Retry-After`. Measure: fixed one-minute windows in Redis (unauthenticated routes: API-001–API-003, API-049; API-059 uses 5 per minute keyed by user). | Given 6 login attempts from one client within a minute, when the sixth is sent, then 429 with `Retry-After`; given 6 invite lookups (API-049) from one client within a minute, then the sixth gets 429; given 6 password-change attempts by one user within a minute, then the sixth gets 429. | Test | Partial (PR #53: API-001-API-003 and API-049 limited; API-059 remains absent) | none (system-only) | ENT-021, API-001–API-003, API-049, API-059 | RISK-003 | internal/api/ratelimit.go; internal/api/router.go:104–121; test: internal/api/ratelimit_test.go. Changed: rev 2 — invite lookup and password change added; keys from Q-015 (TBD-015 resolved). PR #53: anonymous outage and invite limit regressions in internal/api/ratelimit_test.go and router_test.go; API-059 remains separate. |
| SV-NFR-009 | Must | Security: the server shall take the client IP from `X-Forwarded-For` only as written by the Caddy hop. Measure: a client cannot move to another rate-limit bucket by sending its own `X-Forwarded-For`. | Given the compose stack, when one client sends 6 logins in a minute through Caddy with a different `X-Forwarded-For` on each, then the sixth gets 429; given the published ports, then the API port is not among them (SV-NFR-008). | Test | Implemented | none (system-only) | CMP-001, CMP-010 | RISK-003 | Added rev 2 (Q-015). internal/api/ratelimit.go:13–22 (`ClientIPExtractor`); internal/api/router.go:67–68; the compose smoke of 2026-11-23 saw separate buckets for two real clients through Caddy. ADR-015. PR #53: API unpublished in compose; forwarded-header bucket checks retained. |
| SV-NFR-010 | Must | Security: when Redis is unreachable the server shall refuse rate-limited unauthenticated requests with 503 `SERVICE_UNAVAILABLE` and serve authenticated requests without a limit. Measure: response to each route class while Redis is stopped. | Given Redis stopped, when API-002 or API-049 is called, then 503 `SERVICE_UNAVAILABLE`; when a signed-in member calls API-023 with a valid access token, then 200. | Test | Implemented | none (system-only) | CMP-001, CMP-005 | RISK-003 | Added rev 2 (Q-015). Reason (ADR-015): guessing and enumeration happen on the unauthenticated routes; sign-in already needs Redis for refresh tokens, so failing closed there costs no extra availability; access tokens are checked without Redis, so members already signed in keep chatting, and the 15-minute token lifetime and a small trusted group bound the abuse. PR #53: Redis outage policy tests; silent-connection deadline test in internal/redis/client_test.go. |
| SV-NFR-011 | Must | Security: the server shall serve cross-origin browser traffic (CORS requests, gateway upgrades) only for its own origin and the origins listed in `ALLOWED_ORIGINS`. Measure: responses that allow a foreign origin (target 0; `Access-Control-Allow-Origin` is never `*`). | Given Origin `https://other.example` not in the list, when it sends a CORS preflight or opens `/gateway`, then no `Access-Control-Allow-Origin` names it and the upgrade gets 403; given an Origin equal to the instance's own external origin (scheme, host and effective port; the scheme comes from the `X-Forwarded-Proto` that Caddy sets), then it is served; given an Origin that differs from it only in scheme or port and is not listed, then it is refused; given no Origin header (a native client), then it is served. | Test | Implemented | none (system-only) | CMP-001, CMP-002 | RISK-003 | Added rev 2 (Q-016). cmd/retrocast/main.go (CORS); internal/gateway/server.go:14. ADR-016. PR #53: shared origin policy; REST preflight and gateway handshake positive/negative tests. |
| SV-NFR-008 | Must | Security: every client connection shall use HTTPS or WSS through Caddy. Measure: published ports (only Caddy 80 and 443; LiveKit 7880, 7881 and UDP 50000–50100; Grafana on 127.0.0.1:3000). | Given the compose stack, when `docker compose ps` lists published ports, then only those appear and attachment URLs use `https`. | Inspection | Partial (PR #53 fixes D-26; D-10 internal HTTP attachment URLs remain) | none (infra) | CMP-010 | RISK-012, RISK-003 | docker-compose.yml; Caddyfile; SYSTEM_DESIGN.md:684. Changed: rev 2 — port list from Q-016 (TBD-016 resolved; ADR-016); the origin check moved to SV-NFR-011. LiveKit's ports stay published for voice (Later). PR #53: compose binding assertions; attachment URLs remain D-10. |
| SV-NFR-012 | Must | Security: the server shall serve `/metrics` only inside the compose network. Measure: `/metrics` is not reachable through Caddy or a published port. | Given the compose stack, when `https://<host>/metrics` is fetched, then no metrics body is returned; when Prometheus scrapes `api:8080/metrics`, then it succeeds. | Inspection | Implemented | none (infra) | CMP-001, CMP-010, CMP-011 | RISK-003 | Added rev 2 (Q-016). Caddyfile (routes `/api/*`, `/gateway`, `/health` only); internal/api/router.go:87–88 PR #53: Caddy /metrics 404 and successful internal Prometheus scrape. |

#### 3.7.4 Maintainability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-007 | Must | Maintainability: the server shall change the database schema only through numbered migrations applied at startup. Measure: every table and column exists in `migrations/`; startup applies pending ones before serving. | Given an empty database, when the server starts, then migrations 000001–000020 are applied and `/health` returns 200. | Test | Implemented | none (infra) | ENT-001–ENT-018 | LIM-003 | cmd/retrocast/main.go (golang-migrate `file://migrations`); CI "Run migrations" step |

#### 3.7.5 Portability
Covered system-wide by SYS-NFR-004.

#### 3.7.6 Privacy and compliance
Covered system-wide by SYS-CON-002 and SYS-NFR-003.

#### 3.7.7 Observability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SV-NFR-006 | Should | Observability: the server shall write one structured JSON log line per HTTP request. Measure: fields on each line (method, path, status, latency). | Given a request, when it completes, then one JSON line with those fields is written to stdout. | Inspection | Implemented | none (infra) | CMP-001 | LIM-003 | cmd/retrocast/main.go (slog JSON handler, Echo RequestLogger) |
| SV-NFR-013 | Must | Observability: the server shall expose request count and request latency in Prometheus format at `/metrics`. Measure: `retrocast_requests_total` (by status code) and `retrocast_request_duration_seconds` (histogram) are present. | Given a running server that has served one request, when `api:8080/metrics` is fetched inside the compose network, then both metric families are listed. | Test | Implemented | none (infra) | CMP-001, CMP-011 | CAP-035 | Added rev 2 (Q-003: MVP; TBD-039 resolved). internal/api/router.go:87–88 (`echoprometheus`, namespace `retrocast`); grafana/provisioning/dashboards/retrocast.json |

Scraping and dashboards are SYS-FR-001 in [system.md](system.md).

### 3.8 Supporting information
- Permission bits: 19 named bits plus Administrator (`1 << 31`)
  (internal/permissions/bitfield.go; DATA_CONTRACTS ENT-006).
- Routes outside `/api/v1`: `/health`, `/metrics`, `/docs`, `/docs/openapi.yaml`, `/gateway`.
- The admin CLI (`cmd/retrocast-cli`: `migrate`, `seed`, `health`, `version`) is SV-FR-073.
- Error codes added in rev 2 (`INVITE_REQUIRED`, `INVALID_INVITE`, `REGISTRATION_CLOSED`,
  `CANNOT_KICK_SELF`, `SERVICE_UNAVAILABLE`, `NO_SHARED_GUILD`) are listed in DATA_CONTRACTS §5.
- SV-FR-073 is numbered after the moderation block although it sits in §3.1.17; ids follow the
  order they were drafted.
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
| SV-FR-060 | Must | Implemented | ENT-001 | API-059 | CMP-001 | Test |
| SV-FR-061 | Must | Implemented | ENT-019 | API-059, API-003 | CMP-001, CMP-005 | Test |
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
| SV-FR-020 | Must | Implemented | ENT-004, ENT-010 | API-050, EVT-018 | CMP-001, CMP-002 | Test |
| SV-FR-021 | Must | Implemented | ENT-010 | API-048, API-051 | CMP-001 | Test |
| SV-FR-022 | Must | Implemented | ENT-004 | API-035, EVT-019 | CMP-001 | Test |
| SV-FR-023 | Must | Implemented | ENT-004, ENT-005 | API-033 | CMP-001 | Test |
| SV-FR-024 | Must | Implemented | ENT-004 | API-036 | CMP-001 | Test |
| SV-FR-025 | Must | Implemented | ENT-008 | API-022 | CMP-001 | Test |
| SV-FR-026 | Must | Implemented | — | EVT-009 | CMP-002 | Test |
| SV-FR-027 | Must | Implemented | ENT-008, ENT-009 | API-023 | CMP-001 | Test |
| SV-FR-028 | Must | Implemented | ENT-008 | API-024 | CMP-001 | Test |
| SV-FR-029 | Could | Implemented | ENT-008 | API-025, EVT-010 | CMP-001 | Test |
| SV-FR-030 | Could | Implemented | ENT-008 | API-026, EVT-011 | CMP-001 | Test |
| SV-FR-031 | Must | Implemented | ENT-021 | API-028, EVT-024 | CMP-002 | Test |
| SV-FR-032 | Must | Implemented | ENT-020 | EVT-003, EVT-025 | CMP-002 | Test |
| SV-FR-033 | Must | Implemented | ENT-020 | EVT-008, EVT-025 | CMP-002 | Test |
| SV-FR-034 | Must | Implemented | ENT-020 | EVT-025 | CMP-002 | Test |
| SV-FR-035 | Must | Implemented | — | EVT-001 | CMP-002 | Test |
| SV-FR-036 | Must | Implemented | — | EVT-002, EVT-003 | CMP-002 | Test |
| SV-FR-037 | Must | Implemented | — | EVT-004, EVT-005 | CMP-002 | Test |
| SV-FR-038 | Must | Implemented | — | EVT-006 | CMP-002 | Test |
| SV-FR-039 | Won't | Deprecated (rev 2, superseded by SV-FR-072) | — | EVT-007 | CMP-002 | Test |
| SV-FR-071 | Must | Implemented | ENT-004 | EVT-009, EVT-012, EVT-018, EVT-019 | CMP-002 | Test |
| SV-FR-072 | Must | Implemented | — | EVT-003, EVT-009, EVT-025 | CMP-002 | Test |
| SV-FR-040 | Must | Implemented | ENT-006 | API-039, EVT-021 | CMP-001 | Test |
| SV-FR-041 | Must | Implemented | ENT-006 | API-041, EVT-022 | CMP-001 | Test |
| SV-FR-042 | Must | Implemented | ENT-006 | API-042, EVT-023 | CMP-001 | Test |
| SV-FR-043 | Must | Implemented | ENT-005 | API-043, API-044, API-037, EVT-020 | CMP-001 | Test |
| SV-FR-044 | Must | Implemented | ENT-006 | API-040 | CMP-001 | Test |
| SV-FR-045 | Must | Implemented | ENT-005, ENT-006, ENT-007 | — | CMP-001 | Test |
| SV-FR-046 | Must | Implemented | ENT-002, ENT-006 | — | CMP-001 | Test |
| SV-FR-047 | Could | Implemented | ENT-007 | API-045, API-046 | CMP-001 | Test |
| SV-FR-078 | Must | Implemented | ENT-005, ENT-006, ENT-007 | API-037, API-039, API-041, API-043, API-044, API-045 | CMP-001 | Test |
| SV-FR-048 | Could | Implemented | ENT-016 | API-056, EVT-026 | CMP-001, CMP-007 | Test |
| SV-FR-049 | Could | Implemented | ENT-016 | API-057, EVT-026 | CMP-001 | Test |
| SV-FR-050 | Could | Implemented | ENT-016 | API-058 | CMP-001 | Test |
| SV-FR-051 | Could | Partial | ENT-009, ENT-022 | API-027 | CMP-001, CMP-006 | Test |
| SV-FR-052 | Could | Partial | ENT-009 | API-023 | CMP-001, CMP-006 | Test |
| SV-FR-053 | Could | Partial | ENT-001 | API-005, API-006 | CMP-001 | Test |
| SV-FR-054 | Could | Partial | ENT-012, ENT-013 | API-008 | CMP-001 | Test |
| SV-FR-055 | Could | Implemented | ENT-008, ENT-012 | API-022, EVT-009 | CMP-001, CMP-002 | Test |
| SV-FR-056 | Could | Partial | ENT-012, ENT-013 | API-008 | CMP-001 | Test |
| SV-FR-057 | Could | Partial | ENT-013 | API-010, API-011 | CMP-001 | Test |
| SV-FR-058 | Could | Implemented | ENT-012, ENT-013 | API-009 | CMP-001 | Test |
| SV-FR-059 | Could | Partial | ENT-008 | API-055 | CMP-001, CMP-004 | Test |
| SV-FR-062 | Must | Implemented | — | API-001 | CMP-001 | Test |
| SV-FR-063 | Must | Implemented | ENT-001, ENT-004, ENT-010 | API-001 | CMP-001 | Test |
| SV-FR-064 | Must | Implemented | — | API-001, API-002 | CMP-001 | Test |
| SV-FR-065 | Must | Implemented | ENT-001 | API-001 | CMP-001 | Test |
| SV-FR-066 | Must | Implemented | ENT-004 | API-038, EVT-019 | CMP-001 | Test |
| SV-FR-067 | Must | Implemented | ENT-004, ENT-011 | API-053, API-050, EVT-019, EVT-027 | CMP-001 | Test |
| SV-FR-068 | Must | Implemented | ENT-011 | API-052 | CMP-001 | Test |
| SV-FR-069 | Must | Implemented | ENT-011 | API-054, EVT-028 | CMP-001 | Test |
| SV-FR-070 | Must | Implemented | ENT-004, ENT-006 | API-038, API-053 | CMP-001 | Test |
| SV-FR-073 | Should | Implemented | — | — | CMP-003 | Test |
| SV-FR-074 | Could | Implemented | ENT-014 | API-030, API-031, EVT-029, EVT-030 | CMP-001 | Test |
| SV-FR-075 | Could | Implemented | ENT-014 | API-032 | CMP-001 | Test |
| SV-FR-076 | Could | Implemented | ENT-015 | API-012, API-029, EVT-003 | CMP-001 | Test |
| SV-FR-077 | Could | Partial | ENT-004 | API-034, API-037, EVT-020 | CMP-001 | Test |
| SV-IR-001 | Must | Partial | — | API-001–API-059 | CMP-001 | Test |
| SV-IR-002 | Must | Implemented | — | EVT-001–EVT-030 | CMP-002 | Test |
| SV-IR-003 | Must | Implemented | — | all API | CMP-001 | Test |
| SV-IR-004 | Must | Implemented | ENT-004 | API-033, EVT-003 | CMP-001, CMP-002 | Test |
| SV-IR-005 | Should | Implemented | — | API-001–API-059 | CMP-001 | Demonstration |
| SV-IR-006 | Could | Implemented | — | API-056 | CMP-001, CMP-007 | Test |
| SV-IR-007 | Must | Partial | — | API-001–API-059 | CMP-001 | Test |
| SV-IR-008 | Must | Implemented | — | API-001–API-059 | CMP-001 | Inspection |
| SV-NFR-001 | Must | Implemented | ENT-001 | API-001, API-002 | CMP-001 | Test |
| SV-NFR-002 | Must | Implemented | ENT-019 | API-002, API-003 | CMP-001 | Test |
| SV-NFR-003 | Must | Partial | ENT-021 | API-001–API-003, API-049, API-059 | CMP-001, CMP-005 | Test |
| SV-NFR-004 | Must | Implemented | — | — | CMP-001 | Demonstration |
| SV-NFR-005 | Must | Implemented | — | `/health` | CMP-001 | Test |
| SV-NFR-006 | Should | Implemented | — | — | CMP-001 | Inspection |
| SV-NFR-007 | Must | Implemented | ENT-001–ENT-018 | — | CMP-001, CMP-004 | Test |
| SV-NFR-008 | Must | Partial | — | — | CMP-010 | Inspection |
| SV-NFR-009 | Must | Implemented | — | API-001–API-003, API-049 | CMP-001, CMP-010 | Test |
| SV-NFR-010 | Must | Implemented | — | API-001–API-003, API-049 | CMP-001, CMP-005 | Test |
| SV-NFR-011 | Must | Implemented | — | all API, `/gateway` | CMP-001, CMP-002 | Test |
| SV-NFR-012 | Must | Implemented | — | `/metrics` | CMP-001, CMP-010, CMP-011 | Inspection |
| SV-NFR-013 | Must | Implemented | — | `/metrics` | CMP-001, CMP-011 | Test |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial server requirements SV-FR-001–SV-FR-059, SV-IR-001–SV-IR-006, SV-NFR-001–SV-NFR-008 |
| 2 | 2026-11-23T15:17:07-05:00 | 701009f47d68 | REVISION | Added SV-FR-060–SV-FR-077, SV-IR-007, SV-IR-008, SV-NFR-009–SV-NFR-013; SV-FR-039 deprecated (superseded by SV-FR-072); smoke-run deliveries set to Partial; REST v2.0.0 envelope |
| 3 | 2026-11-23T15:35:12-05:00 | 8cbf60de297e | REVISION | Review fixes: SV-FR-005 and SV-NFR-002 Partial (D-21, #43); password length in UTF-8 bytes; SV-NFR-011 same origin is scheme, host and port |
| 4 | 2026-11-23T15:52:00-05:00 | c350426f94c5 | REVISION | PR #44: permission management, default roles, channel search filtering, checked typing and kick hierarchy; residual B3 delivery gaps recorded |
| 5 | 2026-11-23T16:25:23-05:00 | 7af05226f577 | REVISION | PR #44 round 2: SV-FR-078 grant limits, fixed default-role position, API notes and restored evidence |
| 6 | 2026-11-23T16:57:25-05:00 | 3444c6dce8b6 | REVISION | PR #44 round 3: protect current role permissions on update/delete; member role deltas and validation before writes; evidence and UI corrections |
| 7 | 2026-11-23T17:24:36-05:00 | 001716422223 | REVISION | PR #45: REST v2 envelopes and string ids; web Pilot flow fixes; API-059 and role ordering remain partial |
| 8 | 2026-11-23T18:35:55-05:00 | 8c86806c1499 | REVISION | PR #45 round 2: optional configuration, unified error mapping, API-059/B5 remainder and unconditional local sign-out |
| 9 | 2026-11-23T19:23:21-05:00 | f3f6d6f7bafd | REVISION | B2b: migration 000020 repairs DM parent integrity and pending attachment linking; Q-017 types and D-31 route binding fixed; storage/retention limits and Later delivery retained |
| 10 | 2026-11-23T20:32:22-05:00 | c2f7642e7b8c | REVISION | PR #50: live subscriptions, VIEW_CHANNEL fan-out, multiple connections, session replay/fresh READY fallback and presence snapshot/TTL; ADR-017 built |
| 11 | 2026-11-23T21:36:57-05:00 | 42acac2a86f3 | REVISION | PR #50 round 2: fault-driven READY resynchronization, membership epoch retries, guild-wide channel create/delete metadata; SV-FR-031 delivery corrected |
| 12 | 2026-11-23T22:54:35-05:00 | f4e6409e92a2 | REVISION | PR #53: deployment exposure, origins, outage policy, invite limit, LiveKit defaults and raw OpenAPI; D-10 and API-059 remain separate |
| 13 | 2026-11-23T22:56:00-05:00 | 23fa99aabf34 | REVISION | Correct delivery columns, retain requirement statements and update PR #53 evidence |
| 14 | 2026-11-23T23:28:58-05:00 | 2f78d18e5e62 | REVISION | PR #53 round 2: supported LiveKit UDP range; local demo TLS and standalone proxy; deployment drift and requirement provenance repaired; Redis deadline regression |
| 15 | 2026-11-23T23:44:31-05:00 | ad03a1a80f06 | REVISION | PR #54: registration modes and bootstrap; atomic guild creation, invite acceptance and refresh consumption; delivery and evidence aligned |
| 16 | 2026-11-24T00:33:25-05:00 | 4bfa82018aa4 | REVISION | PR #54 round 2: restore requirement evidence, correct registration diagram and operator guidance, parent-first invite locks and stronger concurrency/commit regressions |
