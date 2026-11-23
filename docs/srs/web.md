# SRS component — Web client

Part of [docs/SRS.md](../SRS.md). Prefix `WEB-`. Component: CMP-008 (retrocast-web, React
single-page app served by nginx). Server behaviour it relies on is in [server.md](server.md).

Conventions for this file:
- **Evidence** paths are relative to `retrocast-web/src/`.
- The web client has no test runner (Stack Profile flag 5), so requirements are verified by
  demonstration against a running stack. CI runs `pnpm lint` and `pnpm build` only.
- **Delivery** comes from reading the code at commit `6bc385e`; the client was not run in
  this review. Defect ids (D-nn) are listed in
  [ARCHITECTURE_SPEC.md §9](../ARCHITECTURE_SPEC.md#9-risks-and-technical-debt).
- `lib/api.ts:113` returns `json.data` for every successful response. The server wraps only
  guild, channel, member and user responses in `data` (SV-IR-001, TBD-010), so every client
  call to an unwrapped endpoint receives `undefined` (D-14).

## 3. Requirements

### 3.1 Functions

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| WEB-FR-001 | Must | The web client shall ask for the server address before sign-in and keep it in browser storage. | Given a first visit, when the user enters `https://chat.example` on the server address page, then the client stores it and shows the login page; on the next visit, the address page is skipped. | Demonstration | Implemented | web — Server address page | CMP-008 | CAP-001 | pages/ServerAddressPage.tsx; stores/auth.ts:79; lib/api.ts:31 |
| WEB-FR-002 | Must | The web client shall let a user register or sign in and keep the session's tokens. | Given a running server, when the user registers "alice" with a valid password, then the client shows the server list as alice. | Demonstration | Partial (the auth responses are not wrapped in `data`, so `data.refresh_token` is read from `undefined` and sign-in fails, D-14) | web — Login page; Register page | API-001, API-002 | CAP-001 | stores/auth.ts:41–60; lib/api.ts:113 |
| WEB-FR-003 | Must | The web client shall recover from an expired access token with one refresh (store the rotated refresh token, retry the request). | Given an access token older than 15 minutes, when the user opens a channel, then the request succeeds after one refresh and the new refresh token replaces the old one in storage. | Demonstration | Partial (refresh reads the unwrapped response as `data`, D-14; only the access token is kept, so the next refresh sends a token the server has already invalidated, D-15) | none (system-only) | API-003 | CAP-002 | lib/api.ts:64; stores/auth.ts:83–93 |
| WEB-FR-004 | Must | The web client shall hold a gateway session (identify, heartbeats, RESUME after a drop). | Given a signed-in user, when the network drops for 5 seconds, then the client reconnects and sends op 6 with its session id and last sequence. | Demonstration | Implemented | none (system-only) | EVT-001–EVT-007 | CAP-009 | lib/gateway.ts:7, :109, :193; lib/gateway-dispatcher.ts |
| WEB-FR-005 | Must | The web client shall list the user's servers and let them create a server or join one with an invite code. | Given an invite code, when the user joins with it, then the server appears in the list and is selected. | Demonstration | Partial (create and list work through wrapped endpoints; join expects `{guild_id}` but API-050 returns the guild unwrapped, D-14) | web — Server list | API-007, API-013, API-050 | CAP-004, CAP-005 | components/ServerList.tsx:143–148; stores/guilds.ts:42, :51 |
| WEB-FR-006 | Must | The web client shall show a server's channels and let members with `MANAGE_CHANNELS` manage them (create, rename, delete). | Given an admin, when they create channel "random", then it appears in the sidebar for every connected member. | Demonstration | Implemented | web — Channel sidebar; Channel modal | API-017–API-021, EVT-015–EVT-017 | CAP-006 | components/ChannelSidebar.tsx; components/ChannelModal.tsx:36–60; stores/channels.ts:76 |
| WEB-FR-007 | Must | The web client shall show a channel's history (newest at the bottom, older pages on scroll, live messages appended). | Given a channel with 120 messages, when the user scrolls to the top, then the next 50 older messages load above. | Demonstration | Partial (message lists are not wrapped in `data`, so history loads as empty, D-14) | web — Chat area; Message list | API-023, EVT-009–EVT-011 | CAP-009, CAP-010 | stores/messages.ts:88; components/MessageList.tsx; components/ChatArea.tsx |
| WEB-FR-008 | Must | The web client shall provide a composer (Enter sends, Shift+Enter inserts a line break, typing is signalled at most once every 8 seconds). | Given a focused composer, when the user types "hi" and presses Enter, then API-022 is called and the composer clears; when they keep typing for 20 s, then API-028 is called at most 3 times. | Demonstration | Implemented | web — Message input | API-022, API-028 | CAP-009, CAP-011 | components/MessageInput.tsx:5 (`TYPING_THROTTLE` 8000 ms), :30, :68 |
| WEB-FR-009 | Must | The web client shall show who is typing in the open channel and clear each entry 8 seconds after their last signal. | Given another member typing, when EVT-024 arrives, then "bob is typing…" shows under the message list and disappears after 8 s without a new signal. | Demonstration | Implemented | web — Typing indicator | EVT-024 | CAP-011 | components/TypingIndicator.tsx; stores/typing.ts:14 (`TYPING_TIMEOUT` 8000 ms; the server keeps typing state 10 s) |
| WEB-FR-010 | Must | The web client shall list a server's members with a presence dot (online, idle, do-not-disturb, offline). | Given a member who goes idle, when EVT-025 arrives, then their dot changes to the idle colour. | Demonstration | Implemented | web — Member list; Presence dot | API-033, EVT-025 | CAP-012 | components/MemberList.tsx; components/PresenceDot.tsx; stores/presence.ts |
| WEB-FR-011 | Must | The web client shall render message Markdown (GitHub-flavoured) without rendering raw HTML from message content. | Given a message `<img src=x onerror=alert(1)>`, when it is displayed, then the text shows literally and no script runs. | Inspection | Implemented | web — Message list | EVT-009 | RISK-011 | components/MarkdownContent.tsx:90 (react-markdown 10 with remark-gfm; no rehype-raw; no `dangerouslySetInnerHTML`) |
| WEB-FR-012 | Must | The web client shall let members with `CREATE_INVITE` manage invite codes (create, copy, list, revoke). | Given an admin, when they create an invite, then the code is shown with a copy button; when they revoke it, then it disappears from the list. | Demonstration | Partial (invite responses are not wrapped in `data`, D-14; revoke calls `/invites/{code}` without the `/api/v1` prefix, D-16) | web — Invite modal | API-047, API-048, API-051 | CAP-008 | components/InviteModal.tsx:19, :28, :42 |
| WEB-FR-013 | Must | The web client shall let members who may manage roles edit a server's roles (create, edit, order, delete, choose permissions). | Given the owner, when they create role "Mod" with `MANAGE_MESSAGES`, then it appears in the role list. | Demonstration | Partial (role responses are not wrapped in `data`, so the role list loads as empty, D-14) | web — Role editor | API-039–API-042, EVT-021–EVT-023 | CAP-014, CAP-015 | components/RoleEditor.tsx:118–142; stores/roles.ts:57 |
| WEB-FR-014 | Must | The web client shall let the owner rename or delete a server and let other members leave it. | Given a member who is not the owner, when they choose Leave, then the server disappears from their list. | Demonstration | Implemented | web — Guild settings modal | API-015, API-016, API-035 | CAP-005, CAP-006 | components/GuildSettingsModal.tsx:41, :56, :69 |
| WEB-FR-015 | Could | The web client shall let a user change their display name and avatar reference and sign out. | Given a signed-in user, when they save display name "Vic", then the new name shows on their messages. | Demonstration | Implemented | web — User settings modal; User profile popover | API-005, API-006, API-004 | CAP-003 | components/UserSettingsModal.tsx:29; components/UserProfilePopover.tsx |
| WEB-FR-016 | Could | The web client shall list the user's direct messages and open a conversation. | Given a DM with bob, when the user opens the DM list, then the conversation with bob is listed. | Demonstration | Partial (DM responses are not wrapped in `data`, D-14; DM messages fail on the server, D-08) | web — DM list in the channel sidebar | API-008, API-009, API-022 | CAP-020 | components/ChannelSidebar.tsx:170; stores/dms.ts:23 |
| WEB-FR-017 | Could | The web client shall upload a file (picker or drag and drop) and show image attachments inline. | Given a 1 MB PNG dropped on the chat area, when the upload ends, then the image shows in the channel. | Demonstration | Partial (uploads fail on the server, D-09; attachment URLs point inside the compose network, D-10) | web — File upload | API-027 | CAP-019 | components/FileUpload.tsx; components/MessageList.tsx:61 |
| WEB-FR-018 | Could | The web client shall let an author edit or delete their message and a moderator delete any message from a context menu. | Given the author, when they choose Edit and save "fixed", then the message shows "fixed" with an edited mark. | Demonstration | Implemented | web — Message context menu | API-025, API-026, EVT-010, EVT-011 | CAP-013 | components/MessageContextMenu.tsx:38; components/MessageList.tsx:174 |

The web client has no screens for search (SV-FR-059), reactions, voice (TBD-024), unread
markers or bans.

### 3.2 Performance requirements
No web-specific targets are stated; system targets are SYS-NFR-001 and SYS-NFR-002 (TBD-031).

### 3.3 Usability requirements
No usability or accessibility targets are stated (TBD-032).

### 3.4 Interface requirements
The client uses the REST and gateway contracts of SV-IR-001–SV-IR-004 through nginx, which
proxies `/api` and `/gateway` to the server (`retrocast-web/nginx.conf`).

### 3.5 Logical database requirements
The client keeps only the server address and the refresh token in browser storage
(WEB-FR-001, WEB-FR-003); it holds no other persistent data.

### 3.6 Design constraints
Bound by SYS-CON-003 (existing stack: React 19, Vite 7, TypeScript, Tailwind 4, React Router
7, Zustand 5, pnpm 10).

### 3.7 Software system attributes
- Security: WEB-FR-011 (no raw HTML). The refresh token sits in `localStorage`, readable by
  any script on the origin; whether that is acceptable is part of TBD-016.
- Other attributes have no web-specific requirement.

### 3.8 Supporting information
- `retrocast-web/README.md` is the unchanged Vite template; it does not describe this client.

## 5.1 Traceability matrix

| Req | Priority | Delivery | Entities | APIs / Events | Components | Verification |
|---|---|---|---|---|---|---|
| WEB-FR-001 | Must | Implemented | — | — | CMP-008 | Demonstration |
| WEB-FR-002 | Must | Partial | ENT-001 | API-001, API-002 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-003 | Must | Partial | ENT-019 | API-003 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-004 | Must | Implemented | — | EVT-001–EVT-007 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-005 | Must | Partial | ENT-002 | API-007, API-013, API-050 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-006 | Must | Implemented | ENT-003 | API-017–API-021, EVT-015–EVT-017 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-007 | Must | Partial | ENT-008 | API-023, EVT-009–EVT-011 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-008 | Must | Implemented | ENT-008 | API-022, API-028 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-009 | Must | Implemented | — | EVT-024 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-010 | Must | Implemented | ENT-004 | API-033, EVT-025 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-011 | Must | Implemented | ENT-008 | EVT-009 | CMP-008 | Inspection |
| WEB-FR-012 | Must | Partial | ENT-010 | API-047, API-048, API-051 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-013 | Must | Partial | ENT-006 | API-039–API-042, EVT-021–EVT-023 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-014 | Must | Implemented | ENT-002, ENT-004 | API-015, API-016, API-035 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-015 | Could | Implemented | ENT-001 | API-004, API-005, API-006 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-016 | Could | Partial | ENT-012 | API-008, API-009, API-022 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-017 | Could | Partial | ENT-009 | API-027 | CMP-008, CMP-001, CMP-006 | Demonstration |
| WEB-FR-018 | Could | Implemented | ENT-008 | API-025, API-026, EVT-010, EVT-011 | CMP-008, CMP-001 | Demonstration |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial web client requirements WEB-FR-001–WEB-FR-018 |
