# SRS component — Web client

Part of [docs/SRS.md](../SRS.md). Prefix `WEB-`. Component: CMP-008 (retrocast-web, React
single-page app served by nginx). Server behaviour it relies on is in [server.md](server.md).

Conventions for this file:
- **Evidence** paths are relative to `retrocast-web/src/`.
- Web contract and unit tests use `node --test` through `pnpm test`; CI also runs lint and build. Browser journeys demonstrate rendered behavior.
- **Delivery** comes from reading the code at commit `6bc385e`; the client was not run in
  this review. Defect ids (D-nn) are listed in
  [ARCHITECTURE.md §9](../ARCHITECTURE.md#9-risks-and-technical-debt).
- **Rev 2 (2026-11-23).** Re-read at `origin/main` 55e50eb. In the compose smoke run of
  2026-11-23 the web sign-in failed after the server answered 200 (D-14, issue #32), so no
  later web screen was reached in that run; web Delivery below still comes from reading code.
- Rev 4: REST v2 success envelopes, empty 204 handling, refresh rotation and server
  sign-out are implemented. A local two-user browser journey covers register, sign-in,
  server/channel creation, messages, invite join/revoke, channel deletion, leave and
  refresh refusal after sign-out. Cold channel rendering uses stable store snapshots.
- The iOS client follows the web client (Q-004); web requirements on MVP capabilities are
  `Must`.

## 3. Requirements

### 3.1 Functions

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| WEB-FR-001 | Must | The web client shall ask for the server address before sign-in and keep it in browser storage. | Given a first visit, when the user enters `https://chat.example` on the server address page, then the client stores it and shows the login page; on the next visit, the address page is skipped. | Demonstration | Implemented | web — Server address page | CMP-008 | CAP-001 | pages/ServerAddressPage.tsx; stores/auth.ts (session restore, Web Locks, offline retry); lib/api.ts:31 |
| WEB-FR-002 | Must | The web client shall let a user register or sign in and keep the session's tokens. | Given a running server, when the user registers "alice" with a valid password, then the client shows the server list as alice. | Demonstration | Implemented | web — Login page; Register page | API-001, API-002 | CAP-001 | stores/auth.ts login/register/restore; contract.test.mjs password/invite and restore tests. |
| WEB-FR-003 | Must | The web client shall recover from an expired access token with one refresh (store the rotated refresh token, retry the request). | Given an access token older than 15 minutes, when the user opens a channel, then the request succeeds after one refresh and the new refresh token replaces the old one in storage. | Demonstration | Implemented | none (system-only) | API-003 | CAP-002 | stores/auth.ts refreshAccessToken/sessionLock/rotate; lib/api.ts refreshToken; contract.test.mjs cross-tab, no-Web-Locks, refresh coalescing and logout-drain tests. |
| WEB-FR-004 | Must | The web client shall hold a gateway session (identify, heartbeats, RESUME after a drop). | Given a signed-in user, when the network drops for 5 seconds, then the client reconnects and sends op 6 with its session id and last sequence. | Demonstration | Implemented | none (system-only) | EVT-001–EVT-007 | CAP-009 | lib/gateway.ts:7, :109, :193; lib/gateway-dispatcher.ts |
| WEB-FR-005 | Must | The web client shall list the user's servers and let them create a server or join one with an invite code. | Given an invite code, when the user joins with it, then the server appears in the list and is selected. | Demonstration | Implemented | web — Server list | API-007, API-013, API-050 | CAP-004, CAP-005 | components/ServerList.tsx:143–148; stores/guilds.ts:42, :51; Changed rev 4 - REST v2 and web contract fixes. |
| WEB-FR-006 | Must | The web client shall show a server's channels and let members with `MANAGE_CHANNELS` manage them (create, rename, delete). | Given an admin, when they create channel "random", then it appears in the sidebar for every connected member; when they delete it, then it disappears with no error shown. | Demonstration | Implemented | web — Channel sidebar; Channel modal | API-017–API-021, EVT-015–EVT-017 | CAP-006 | components/ChannelSidebar.tsx; components/ChannelModal.tsx:36–60; stores/channels.ts:76; Changed rev 4 - REST v2 and web contract fixes. |
| WEB-FR-007 | Must | The web client shall show a channel's history (newest at the bottom, older pages on scroll, live messages appended). | Given a channel with 120 messages, when the user scrolls to the top, then the next 50 older messages load above. | Demonstration | Implemented | web — Chat area; Message list | API-023, EVT-009–EVT-011 | CAP-009, CAP-010 | stores/messages.ts:88; components/MessageList.tsx; components/ChatArea.tsx; Changed rev 4 - REST v2 and web contract fixes. |
| WEB-FR-008 | Must | The web client shall provide a composer (Enter sends, Shift+Enter inserts a line break, typing is signalled at most once every 8 seconds). | Given a focused composer, when the user types "hi" and presses Enter, then API-022 is called and the composer clears; when they keep typing for 20 s, then API-028 is called at most 3 times. | Demonstration | Implemented | web — Message input | API-022, API-028 | CAP-009, CAP-011 | components/MessageInput.tsx:5 (`TYPING_THROTTLE` 8000 ms), :30, :68 |
| WEB-FR-009 | Must | The web client shall show who is typing in the open channel and clear each entry 8 seconds after their last signal. | Given another member typing, when EVT-024 arrives, then "bob is typing…" shows under the message list and disappears after 8 s without a new signal. | Demonstration | Implemented | web — Typing indicator | EVT-024 | CAP-011 | components/TypingIndicator.tsx; stores/typing.ts:14 (`TYPING_TIMEOUT` 8000 ms; the server keeps typing state 10 s) |
| WEB-FR-010 | Must | The web client shall list a server's members with a presence dot (online, idle, do-not-disturb, offline). | Given a member who goes idle, when EVT-025 arrives, then their dot changes to the idle colour. | Demonstration | Implemented | web — Member list; Presence dot | API-033, EVT-025 | CAP-012 | components/MemberList.tsx; components/PresenceDot.tsx; stores/presence.ts |
| WEB-FR-011 | Must | The web client shall render message Markdown (GitHub-flavoured) without rendering raw HTML from message content. | Given a message `<img src=x onerror=alert(1)>`, when it is displayed, then the text shows literally and no script runs. | Inspection | Implemented | web — Message list | EVT-009 | RISK-011 | components/MarkdownContent.tsx:90 (react-markdown 10 with remark-gfm; no rehype-raw; no `dangerouslySetInnerHTML`) |
| WEB-FR-012 | Must | The web client shall let members with `CREATE_INVITE` manage invite codes (create, copy, list, revoke). | Given an admin, when they create an invite, then the code is shown with a copy button; when they revoke it, then it disappears from the list. | Demonstration | Implemented | web — Invite modal | API-047, API-048, API-051 | CAP-008 | components/InviteModal.tsx:19, :28, :42; Changed rev 4 - REST v2 and web contract fixes. |
| WEB-FR-013 | Must | The web client shall let members who may manage roles edit a server's roles (create, edit, order, delete, choose permissions). | Given the owner, when they create role "Mod" with `MANAGE_MESSAGES`, then it appears in the role list. | Demonstration | Partial (role ordering UI remains absent; other role editing and unsigned permissions fixed) | web — Role editor | API-039–API-042, EVT-021–EVT-023 | CAP-014, CAP-015 | components/RoleEditor.tsx:118–142; stores/roles.ts:57; Changed rev 4 - REST v2 and web contract fixes. |
| WEB-FR-014 | Must | The web client shall let the owner rename or delete a server and let other members leave it. | Given a member who is not the owner, when they choose Leave, then the server disappears from their list. | Demonstration | Implemented | web — Guild settings modal | API-015, API-016, API-035 | CAP-005, CAP-006 | components/GuildSettingsModal.tsx:41, :56, :69; Changed rev 4 - REST v2 and web contract fixes. |
| WEB-FR-015 | Could | The web client shall let a user change their display name and avatar reference and sign out. | Given a signed-in user, when they save display name "Vic", then the new name shows on their messages. | Demonstration | Implemented (sign-out clears the tokens locally only; ending the server session is WEB-FR-024) | web — User settings modal; User profile popover | API-005, API-006, API-004 | CAP-003 | components/UserSettingsModal.tsx:29; components/UserProfilePopover.tsx |
| WEB-FR-016 | Could | The web client shall list the user's direct messages and open a conversation. | Given a DM with bob, when the user opens the DM list, then the conversation with bob is listed. | Demonstration | Implemented | web — DM list in the channel sidebar | API-008, API-009, API-022 | CAP-020 | components/ChannelSidebar.tsx:170; stores/dms.ts:23; REST v2 envelopes fixed in PR #45 and DM storage fixed by migration 000020; stores/dms.ts and components/ChannelSidebar.tsx list/open DMs. |
| WEB-FR-017 | Could | The web client shall upload a file (picker or drag and drop) and show image attachments inline. | Given a 1 MB PNG dropped on the chat area, when the upload ends, then the image shows in the channel. | Demonstration | Partial (text-plus-file upload/link/history works; file-only sends retained with a hint; public download URLs remain D-10) | web — File upload | API-027 | CAP-019 | components/MessageInput.tsx; MessageList.tsx; contract.test.mjs composer regression. |
| WEB-FR-018 | Could | The web client shall let an author edit or delete their message and a moderator delete any message from a context menu. | Given the author, when they choose Edit and save "fixed", then the message shows "fixed" with an edited mark; given a moderator with `MANAGE_MESSAGES` on another member's message, when they choose Delete, then the message is removed for everyone. | Demonstration | Partial (the context menu shows Edit and Delete only on the user's own messages, so a moderator has no Delete, components/MessageContextMenu.tsx:85) | web — Message context menu | API-025, API-026, EVT-010, EVT-011 | CAP-013 | components/MessageContextMenu.tsx:38; components/MessageList.tsx:174 |
| WEB-FR-019 | Must | The web client shall let a signed-in user change their password in user settings (current password, new password, confirmation). | Given the user settings modal, when the user enters the correct current password and a matching valid new one and saves, then API-059 is called, a success message shows and the returned tokens replace the stored ones; given a wrong current password, then the modal says the current password is incorrect and the user stays signed in. | Demonstration | Implemented | web — User settings modal, Password section | API-059 | CAP-037 | Added rev 2 (Q-022). components/UserSettingsModal.tsx  L25: account/session tests and local browser DOM evidence. |
| WEB-FR-020 | Must | The web client's register page shall take an optional invite code and show the server's registration errors in plain words. | Given mode `invite`, when the user registers with a valid invite code, then they land in that invite's server; given no code, then the page says an invite is required; given mode `closed`, then the page says registration is closed. | Demonstration | Implemented | web — Register page | API-001 | CAP-040, CAP-001 | Added rev 2 (Q-012). pages/RegisterPage.tsx  L25: account/session tests and local browser DOM evidence. |
| WEB-FR-021 | Must | The web client shall offer Kick and Ban (with an optional reason) in a member's profile popover only to members holding `KICK_MEMBERS` or `BAN_MEMBERS`. | Given a moderator with `BAN_MEMBERS`, when they open a member's popover and choose Ban with reason "spam", then API-053 is called and the member leaves the member list; given a member with neither permission, then neither action is shown; given a 403 `ROLE_HIERARCHY`, then the popover says the member ranks too high. | Demonstration | Not started | web — User profile popover | API-038, API-053 | CAP-031 | Added rev 2 (Q-003). components/UserProfilePopover.tsx |
| WEB-FR-022 | Must | The web client shall list a server's bans in a Bans tab of guild settings and let a member with `BAN_MEMBERS` lift each one. | Given two bans, when a moderator opens the Bans tab, then both users are listed with their reasons; when they choose Unban on one, then API-054 is called and it disappears from the list. | Demonstration | Not started (the modal has General and Roles tabs only) | web — Guild settings modal, Bans tab | API-052, API-054 | CAP-031 | Added rev 2 (Q-003). components/GuildSettingsModal.tsx |
| WEB-FR-023 | Should | The web client shall remove a server from the user's server list when the user is kicked or banned from it. | Given a member viewing server S, when EVT-019 for their own user id arrives, then S disappears from their list and the client shows another server or the empty state. | Demonstration | Partial (EVT-019 removes the user from the member store only, lib/gateway-dispatcher.ts:133–136; GUILD_BAN_ADD is ignored, :167) | web — Server list | EVT-019, EVT-027 | CAP-031 | Added rev 2. |
| WEB-FR-024 | Must | The web client shall always complete local sign-out with best-effort server-session revocation through API-004. | Given a signed-in user, when they choose Log out, then API-004 is called with their refresh token and a later API-003 with that token returns 401 when revocation succeeds; when the network or refresh fails, then local state and storage still clear and the client navigates to the server screen. | Demonstration | Implemented | web — User settings modal, Log out | API-004 | CAP-002 | Added rev 2 (found while re-reading WEB-FR-015). Changed rev 5 - server revocation is best-effort; local sign-out and navigation always complete. |

The web client has no screens for search (SV-FR-059; Later), reactions, voice (Later, Q-024)
or unread markers (Later, Q-003).

### 3.2 Performance requirements
No web-specific targets; system targets are SYS-NFR-001, SYS-NFR-002 and SYS-NFR-007
(Q-031, TBD-031 resolved).

### 3.3 Usability requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| WEB-NFR-001 | Should | Accessibility: the web client shall meet WCAG 2.2 level AA. Measure: axe-core violations at the WCAG 2.2 AA rule set on each page (target 0) plus a keyboard-only walkthrough of the key scenarios. | Given each page of the app (server address, login, register, app shell with each modal open), when axe-core runs with the WCAG 2.2 AA rules, then it reports 0 violations; when a tester signs in, sends a message and opens guild settings with the keyboard only, then every step is reachable with a visible focus. | Test | Not started (no audit recorded; no test runner) | web — every page | CMP-008 | GOAL-001 | Added rev 2 (Q-032, TBD-032 resolved). Should, not Must: a full audit is not a Pilot gate for a private group (spec lead's assumption). |
| WEB-NFR-002 | Should | Compatibility: the web client shall work on the current and previous major versions of four browsers (Chrome, Firefox, Safari, Edge). Measure: the key scenarios of PRODUCT_BRIEF §6 pass on each of the 8 browser versions. | Given each of the 8 browser versions, when a tester runs the first-evening and moderation scenarios, then each step succeeds. | Demonstration | Not started (no run recorded; Vite 7's default build target covers these browsers) | web — every page | CMP-008 | GOAL-001 | Added rev 2 (Q-032). |

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
  any script on the origin. Rev 2 keeps it there for the Pilot (spec lead's assumption): no raw
  HTML is rendered (WEB-FR-011) and only the same origin or the operator's list is allowed
  (SV-NFR-011, Q-016).
- Other attributes have no web-specific requirement.

### 3.8 Supporting information
- `retrocast-web/README.md` is the unchanged Vite template; it does not describe this client.

## 5.1 Traceability matrix

| Req | Priority | Delivery | Entities | APIs / Events | Components | Verification |
|---|---|---|---|---|---|---|
| WEB-FR-001 | Must | Implemented | — | — | CMP-008 | Demonstration |
| WEB-FR-002 | Must | Implemented | ENT-001 | API-001, API-002 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-003 | Must | Implemented | ENT-019 | API-003 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-004 | Must | Implemented | — | EVT-001–EVT-007 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-005 | Must | Implemented | ENT-002 | API-007, API-013, API-050 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-006 | Must | Implemented | ENT-003 | API-017–API-021, EVT-015–EVT-017 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-007 | Must | Implemented | ENT-008 | API-023, EVT-009–EVT-011 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-008 | Must | Implemented | ENT-008 | API-022, API-028 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-009 | Must | Implemented | — | EVT-024 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-010 | Must | Implemented | ENT-004 | API-033, EVT-025 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-011 | Must | Implemented | ENT-008 | EVT-009 | CMP-008 | Inspection |
| WEB-FR-012 | Must | Implemented | ENT-010 | API-047, API-048, API-051 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-013 | Must | Partial | ENT-006 | API-039–API-042, EVT-021–EVT-023 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-014 | Must | Implemented | ENT-002, ENT-004 | API-015, API-016, API-035 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-015 | Could | Implemented | ENT-001 | API-004, API-005, API-006 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-016 | Could | Implemented | ENT-012 | API-008, API-009, API-022 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-017 | Could | Partial | ENT-009 | API-027 | CMP-008, CMP-001, CMP-006 | Demonstration |
| WEB-FR-018 | Could | Partial | ENT-008 | API-025, API-026, EVT-010, EVT-011 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-019 | Must | Implemented | ENT-001 | API-059 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-020 | Must | Implemented | ENT-001, ENT-010 | API-001 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-021 | Must | Not started | ENT-004, ENT-011 | API-038, API-053 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-022 | Must | Not started | ENT-011 | API-052, API-054 | CMP-008, CMP-001 | Demonstration |
| WEB-FR-023 | Should | Partial | ENT-004 | EVT-019, EVT-027 | CMP-008, CMP-002 | Demonstration |
| WEB-FR-024 | Must | Implemented | ENT-019 | API-004 | CMP-008, CMP-001 | Demonstration |
| WEB-NFR-001 | Should | Not started | — | — | CMP-008 | Test |
| WEB-NFR-002 | Should | Not started | — | — | CMP-008 | Demonstration |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial web client requirements WEB-FR-001–WEB-FR-018 |
| 2 | 2026-11-23T15:17:07-05:00 | 701009f47d68 | REVISION | Added WEB-FR-019–WEB-FR-024, WEB-NFR-001, WEB-NFR-002; WEB-FR-006 and WEB-FR-014 Partial (D-29) |
| 3 | 2026-11-23T15:35:12-05:00 | 8cbf60de297e | REVISION | Review fix: WEB-FR-018 Partial (no moderator Delete in the context menu) |
| 4 | 2026-11-23T17:24:36-05:00 | 001716422223 | REVISION | PR #45: REST v2 envelopes and string ids; web Pilot flow fixes; API-059 and role ordering remain partial |
| 5 | 2026-11-23T18:35:55-05:00 | 8c86806c1499 | REVISION | PR #45 round 2: optional configuration, unified error mapping, API-059/B5 remainder and unconditional local sign-out |
| 6 | 2026-11-23T18:41:01-05:00 | 94bd59930c73 | REVISION | Clarify WEB-FR-024 as unconditional local sign-out; retain failure-path acceptance criteria |
| 7 | 2026-11-23T19:23:21-05:00 | f3f6d6f7bafd | REVISION | B2b: migration 000020 repairs DM parent integrity and pending attachment linking; Q-017 types and D-31 route binding fixed; storage/retention limits and Later delivery retained |
| 8 | 2026-11-23T20:11:25-05:00 | e40076472143 | REVISION | PR #48 round 2: down-refusal recovery, READ COMMITTED parent-delete requirement, pinned trigger lookup, updated DM delivery and uploader-delete blockers, repaired table rows |
| 9 | 2026-11-24T00:51:47-05:00 | 03b5c35200c3 | REVISION | PR #55: API-059, refresh generations, web session restoration, invite registration and password settings |
