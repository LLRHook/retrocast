# Product Brief — Retrocast

Status: Specified (SRS rev 2)
Updated: 2026-11-23 · Sessions: 2 (1: unattended draft from repository evidence; 2: the owner's answers to Q-001–Q-034)
Feeds: docs/SRS.md via the `spec` skill. After `Specified`, the SRS is the source of truth
and this brief is not edited.

> **How this brief was made.** An agent drafted it from the repository without asking anyone;
> every fact cites the file it came from as `(src: …)`. On 2026-11-23 Victor Ivanov (STK-001)
> read the draft in PR #25 and accepted every recommendation in §14 (option (a) of Q-001–Q-034).
> Those answers are folded into the sections below and logged in §14, attributed to him.
> Design details that implement an answer are decisions of the driver or the spec lead and live
> in the SRS and the architecture decisions, not here.

## 0. Readiness

| # | Check | State |
|---|---|---|
| 1 | Problem and who has it | ✅ (src: SYSTEM_DESIGN.md §Vision, README.md) |
| 2 | Goals with measures | ✅ GOAL-001 measured by Q-031; GOAL-003 and GOAL-004 have measures; GOAL-002 has none until voice is scheduled (Later, Q-024) |
| 3 | Buyer, primary user, affected party considered | ✅ Buyer = project owner; primary users = group members; affected party: none (a private group, Q-001) |
| 4 | Each primary user has a capability | ✅ |
| 5 | Every capability has a release; none proposed | ✅ Q-003, Q-022, Q-026–Q-029 set the last ones |
| 6 | Every solution idea classified | ✅ Confirmed or declined by Q-008, Q-011, Q-028, Q-029 |
| 7 | Constraints and non-goals answered | ✅ Q-002, Q-005–Q-007, Q-019–Q-021, Q-033 |
| 8 | Kill risks have a stance; no blocking question | ✅ RISK-001 stance accepted (Q-003); no open question |

Readiness: 8/8. Victor accepted the brief on 2026-11-23; SRS rev 2 records the answers.

## 1. Seed

> Build a self-hosted messaging and voice platform that captures the spirit of early Discord — before Nitro, before Activities, before bloat. Just servers, channels, voice chat, and friends. The aesthetic targets the iOS 7 era: flat design, translucent panels, clean typography, and a focused feature set.
>
> **Target deployment**: A single Mac or Linux machine on a local network, serving 5–20 friends. No cloud required. No accounts with a third party. Your data, your server.

(src: SYSTEM_DESIGN.md:25–27)

> A self-hosted Discord clone inspired by the simplicity of Discord circa 2015-2016. Text channels, voice chat, roles, permissions — no bloat. Built for a small friend group to run on a single machine.

(src: README.md:3)

## 2. Problem

- **Problem:** A small group of friends wants text and voice chat organised into servers and
  channels, like early Discord, without a third-party account and without its data leaving
  a machine they control. Hosted chat products add features the group does not want
  ("before Nitro, before Activities, before bloat"). (src: SYSTEM_DESIGN.md:25–27)
- **Today:** Not stated in the sources.
- **Why now:** Not stated.
- **Evidence:** The seed documents only. No user research, usage data or interviews.
  The codebase exists: a Go server, a React web client and a SwiftUI iOS client
  (src: README.md, TODO.md).

## 3. Goals

| Id | Goal | Success measure | Source |
|---|---|---|---|
| GOAL-001 | A friend group of 5–20 people can chat in text channels in real time on one self-hosted machine with no cloud service | With 20 members connected, a sent message reaches every other online member within 1 s (95th percentile) on the operator's machine, and a page of 50 history messages returns within 300 ms (95th percentile) (Q-031) | src: SYSTEM_DESIGN.md:27; Victor (Q-001, Q-031) |
| GOAL-002 | Members can drop into persistent voice rooms with no call setup | None set: voice is Later (Q-024); a measure is set when voice is scheduled | src: SYSTEM_DESIGN.md:41 ("Persistent voice rooms — join/leave freely, no ringing, no call setup") |
| GOAL-003 | The feature set stays at early-Discord scope ("Simple, fast, focused") | The released capability set stays within §6 and §9 | src: SYSTEM_DESIGN.md:63 |
| GOAL-004 | The data stays on the operator's machine ("Your data, your server") | No core flow calls an external service; APNs push is the only optional exception | src: SYSTEM_DESIGN.md:27, :267 |

## 4. Stakeholders

| Id | Stakeholder | Role | Needs / concerns | Source |
|---|---|---|---|---|
| STK-001 | Project owner (Victor) | buyer | Decides scope, licence, hosting and spend; owns the repository | src: repository LLRHook/retrocast |
| STK-002 | Group member | primary user | Chats in text and voice with friends; no third-party account | src: SYSTEM_DESIGN.md:27, :265 |
| STK-003 | Server owner or admin | primary user | Creates servers, channels, roles and invites; moderates (kick, ban, roles) | src: SYSTEM_DESIGN.md:43, :61 |
| STK-004 | Instance operator | operator/admin | Runs the Docker Compose stack on one machine; chooses who may register; backups, monitoring, upgrades | src: README.md:3, retrocast/docker-compose.yml; Victor (Q-012, Q-030) |
| STK-005 | Apple App Store review | gatekeeper/regulator | Applies only if the iOS app is distributed beyond development installs; iOS follows the web client (Q-004) | src: retrocast-client/Retrocast/project.yml (bundle id com.retrocast.ios) |

## 5. Principles

- **PRN-001 — Self-hosted, data stays local:** Core functions need no external cloud
  service; "No cloud required. No accounts with a third party." (src: SYSTEM_DESIGN.md:27, :266)
- **PRN-002 — Early-Discord scope:** Features Discord lacked in 2015–2016 stay out
  (threads, bots, video and screen share, subscriptions, boosting, discovery).
  (src: SYSTEM_DESIGN.md:51–63)
- **PRN-003 — One protocol for every client:** "All clients share the same API surface
  (REST + WebSocket)." (src: SYSTEM_DESIGN.md:265)

## 6. Capabilities

Origin is `stated` when a repository document states the capability and `accepted` when it
came from an option Victor accepted in §14. Release comes from the phase list in
SYSTEM_DESIGN.md §Feature Scope (Phase 1 is labelled "Core (MVP)", Phases 2–4 are `Later`),
changed where an answer in §14 says so.

### Accounts

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-001 | A member can register and sign in with a username and password, with no email (src: SYSTEM_DESIGN.md:232, :699) | STK-002 | GOAL-001, GOAL-004 | stated | MVP |
| CAP-002 | A member stays signed in between visits and can sign out (src: SYSTEM_DESIGN.md:234, :512) | STK-002 | GOAL-001 | stated | MVP |
| CAP-003 | A member can set a display name and an avatar; a server owner can set a server icon (src: SYSTEM_DESIGN.md:715, TODO.md "User settings (edit display name)") | STK-002, STK-003 | GOAL-001 | stated | Later |
| CAP-037 | A member can change their password (Q-022) | STK-002 | GOAL-001 | accepted | MVP |
| CAP-038 | A member can delete their account (Q-022) | STK-002 | GOAL-004 | accepted | Later |
| CAP-040 | The operator can choose who may register: anyone, only people holding an invite, or nobody; invite-only is the default (Q-012) | STK-004 | GOAL-001 | accepted | MVP |

### Servers and channels

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-004 | A member can create a server and becomes its owner (src: SYSTEM_DESIGN.md:700) | STK-002 | GOAL-001 | stated | MVP |
| CAP-005 | A member can join a server through an invite link and leave it (src: SYSTEM_DESIGN.md:700, :706) | STK-002 | GOAL-001 | stated | MVP |
| CAP-006 | A server admin can create, rename and delete text and voice channels, and rename or delete the server (src: SYSTEM_DESIGN.md:701, :39–40) | STK-003 | GOAL-001 | stated | MVP |
| CAP-007 | A server admin can group channels into categories (src: SYSTEM_DESIGN.md:735) | STK-003 | GOAL-001 | stated | Later |
| CAP-008 | A server admin can create invite links with optional expiry and use limits, list them, and revoke them (src: SYSTEM_DESIGN.md:47, :706) | STK-003 | GOAL-001 | stated | MVP |
| CAP-039 | A server owner can hand ownership of the server to another member (Q-026) | STK-003 | GOAL-001 | accepted | Later |

### Text chat and real-time

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-009 | A member can send messages in a text channel and see others' messages appear without reloading, on several devices at once (src: SYSTEM_DESIGN.md:701; Q-009) | STK-002 | GOAL-001 | stated | MVP |
| CAP-010 | A member can scroll back through a channel's message history (src: SYSTEM_DESIGN.md:702) | STK-002 | GOAL-001 | stated | MVP |
| CAP-011 | A member sees when others are typing (src: SYSTEM_DESIGN.md:704) | STK-002 | GOAL-001 | stated | MVP |
| CAP-012 | A member sees who is online, idle, on do-not-disturb or offline, and can appear invisible (src: SYSTEM_DESIGN.md:44, :705) | STK-002 | GOAL-001 | stated | MVP |
| CAP-013 | A member can edit and delete their own messages; a moderator can delete others' messages (src: SYSTEM_DESIGN.md:725) | STK-002, STK-003 | GOAL-001 | stated | Later |

### Roles and permissions

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-014 | A server admin can create roles with a name, colour and permission set, order them, and assign them to members (src: SYSTEM_DESIGN.md:43, :703) | STK-003 | GOAL-001 | stated | MVP |
| CAP-015 | Only members whose roles allow an action can perform it; a new member can by default view, send, read history, join voice, speak, create invites and change their nickname (src: SYSTEM_DESIGN.md:703, docs/legacy/permissions.md; Q-014) | STK-002, STK-003 | GOAL-001 | stated | MVP |
| CAP-016 | A server admin can allow or deny permissions per channel for a role (src: SYSTEM_DESIGN.md:736) | STK-003 | GOAL-001 | stated | Later |

### Voice and media

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-017 | A member can join and leave a persistent voice channel and talk with whoever is in it, from iPhone and from the browser, audio only (src: SYSTEM_DESIGN.md:41, :710; Q-024) | STK-002 | GOAL-002 | stated | Later |
| CAP-018 | A member can adjust per-user volume, use push-to-talk or voice activity, and mute or deafen; a moderator can server-mute (src: SYSTEM_DESIGN.md:711–713) | STK-002, STK-003 | GOAL-002 | stated | Later |
| CAP-019 | A member can share files and images (up to 10 MB; JPEG, PNG, GIF, WebP, PDF, plain text) in a channel and see image previews inline (src: SYSTEM_DESIGN.md:46, :714; Q-017) | STK-002 | GOAL-001 | stated | Later |

### Direct messages and notifications

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-020 | A member can message another member they share a server with, one-to-one (src: SYSTEM_DESIGN.md:42, :719; Q-013) | STK-002 | GOAL-001 | stated | Later |
| CAP-021 | A member can hold a group conversation of up to 10 people outside any server (src: SYSTEM_DESIGN.md:42, :720) | STK-002 | GOAL-001 | stated | Later |
| CAP-022 | A member can mention a user, a role, @everyone or @here, and chooses which mentions notify them (src: SYSTEM_DESIGN.md:45, :721) | STK-002 | GOAL-001 | stated | Later |
| CAP-023 | A member gets a push notification on iPhone when mentioned or messaged while the app is closed; ships with the iOS client (src: SYSTEM_DESIGN.md:267, :726; Q-025) | STK-002 | GOAL-001 | stated | Later |

### Expression

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-024 | A member can insert Unicode emoji from a picker. Custom server emoji: Won't (Q-027) (src: SYSTEM_DESIGN.md:722–723) | STK-002 | GOAL-001 | stated | Later |
| CAP-025 | A posted link shows a preview with the page's metadata (src: SYSTEM_DESIGN.md:49, :724) | STK-002 | GOAL-001 | stated | Won't (Q-027) |
| CAP-026 | The clients look like the iOS 7 era: flat design, translucent panels, clean type (src: SYSTEM_DESIGN.md:25, :727) | STK-002 | GOAL-003 | stated | Later |

### Phase 4 ("Nice to Have")

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-027 | A client on the same local network finds the server without typing its address (src: SYSTEM_DESIGN.md:269, :731) | STK-002 | GOAL-001 | stated | Won't (Q-029) |
| CAP-028 | A member can search a server's message history by text, author and date, with English word matching (src: SYSTEM_DESIGN.md:732; Q-023) | STK-002 | GOAL-001 | stated | Later |
| CAP-029 | A member can use a desktop application (src: SYSTEM_DESIGN.md:733) | STK-002 | GOAL-001 | stated | Won't (Q-028: the web client covers desktop) |
| CAP-030 | A server admin can review a log of moderation and configuration actions (src: SYSTEM_DESIGN.md:734; Q-018) | STK-003 | GOAL-001 | stated | Later |

### Built or listed outside the seed phases (releases set by Q-003)

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-031 | A moderator can kick a member, ban a user (with an optional reason), list bans and lift a ban; a moderator cannot act on someone whose role ranks at or above their own (src: SYSTEM_DESIGN.md:61 "just kick, ban, and role management"; TODO.md "Ban system"; Q-018) | STK-003 | GOAL-001 | stated | MVP |
| CAP-032 | A member can react to a message with an emoji and see reaction counts (src: TODO.md "Message reactions/emojis") | STK-002 | GOAL-001 | stated | Later |
| CAP-033 | A member sees which channels have unread messages (src: TODO.md "Read states / unread badges") | STK-002 | GOAL-001 | stated | Later |
| CAP-034 | A member can set a per-server nickname; a moderator can change others' nicknames (src: retrocast/internal/service/member.go, docs/legacy/permissions.md) | STK-002, STK-003 | GOAL-001 | stated | Later |
| CAP-035 | The operator can see service health and metrics dashboards (src: TODO.md §Infra "Health check endpoint", "Prometheus metrics", "Grafana dashboards") | STK-004 | GOAL-004 | stated | MVP |
| CAP-036 | The operator gets a daily backup of the database and stored files kept 30 days, can restore it, and can apply migrations and load demo data from a command line (src: TODO.md §Infra "Backup automation", "CLI tool (retrocast-cli)", "Demo bootstrap script"; Q-030) | STK-004 | GOAL-004 | stated | MVP |

Clients (web, iOS) are surfaces for these capabilities, not capabilities themselves: the MVP
ships on the web client and iOS follows (Q-004).

### Key scenarios

1. **First evening.** The operator runs the compose stack. A member registers (CAP-001),
   creates a server (CAP-004), gets a `general` text channel and a `General` voice channel,
   and sends invite links (CAP-008). Friends register with their invite, since registration is
   invite-only by default (CAP-040), join (CAP-005) and chat in real time from browser and
   phone (CAP-009, CAP-011, CAP-012). Not binding.
2. **Moderation.** An admin creates a "Moderator" role (CAP-014). A moderator removes a
   member who is spamming (CAP-031), and the removed member stops receiving that server's
   messages.
3. **Failure: unauthorised action.** A member with no special role tries to create an
   administrator role or delete a channel. The server refuses (CAP-015). This path is
   broken today (RISK-001).
4. **Failure: expired invite.** A friend opens an invite after it expired or reached its use
   limit and is told it is no longer valid (CAP-005, CAP-008).
5. **Recovery.** The operator's disk fails. They restore last night's backup on a new machine
   and the group loses at most one day of messages (CAP-036).
6. **Voice night (Later).** Members join the `General` voice channel and talk (CAP-017).

## 7. Domain terms

| Term | Meaning (plain language) | Relates to |
|---|---|---|
| Server (guild) | A space with its own channels, members and roles. The code and API call it a guild | CAP-004 |
| Channel | A text room, voice room or category inside a server | CAP-006, CAP-007 |
| Role | A named set of permissions with a colour and a rank; members can hold several | CAP-014 |
| @everyone | The default role every server has; its permissions apply to all members | CAP-015 |
| Permission override | A per-channel allow or deny for a role | CAP-016 |
| Invite | A short code that lets someone join a server, with an optional expiry and use limit | CAP-005, CAP-008 |
| Registration mode | The operator's choice of who may create an account: open, invite-only or closed | CAP-040 |
| Kick, ban | Kick removes a member, who may rejoin with an invite; ban removes them and stops them rejoining until the ban is lifted | CAP-031 |
| Presence | Online, idle, do-not-disturb, invisible (shown as offline) or offline | CAP-012 |
| Gateway | The live connection that pushes new messages, typing and presence to clients | CAP-009 |
| DM, group DM | A conversation outside any server, with one person or with up to 10 people | CAP-020, CAP-021 |
| Read state | Where a member last read in a channel; drives unread badges | CAP-033 |
| Snowflake | A 64-bit, time-ordered id used for every object | — |
| Operator | The person who runs the server stack on their machine | STK-004 |

## 8. Constraints

| Id | Constraint | Kind | Hard / soft | Source |
|---|---|---|---|---|
| LIM-001 | Runs on a single Mac or Linux machine; no cloud required; self-hosted only, each group runs its own instance | hosting | hard | src: SYSTEM_DESIGN.md:27; Victor (Q-002) |
| LIM-002 | No third-party account is needed to use the product; registration needs no email | regulation | hard | src: SYSTEM_DESIGN.md:27, :232 |
| LIM-003 | The existing system is reused: a Go server (one binary for REST and the gateway), PostgreSQL, Redis, MinIO and LiveKit, deployed with Docker Compose behind Caddy; a React web client; a SwiftUI iOS client. Permission resolution stays in Go (Q-008) | existing system | soft (a change is the owner's decision) | src: retrocast/go.mod, retrocast/docker-compose.yml, retrocast-web/package.json, retrocast-client/Retrocast/project.yml; Victor (Q-008) |
| LIM-004 | The repository is unlicensed: "All rights reserved" everywhere; no open-source licence | licence | hard | src: README.md; Victor (Q-007) |
| LIM-005 | The Go module path stays `github.com/victorivanov/retrocast` | existing system | soft | src: retrocast/go.mod; Victor (Q-033) |

## 9. Non-goals

- Threads, bots and slash commands, screen sharing and video, paid subscriptions, server
  boosting, stage and forum channels, activities, stickers, polls, server discovery.
  Reason: Discord did not have them in 2015–2016, which is the targeted scope
  (src: SYSTEM_DESIGN.md:51–63). Video stays out of voice too (Q-024).
- Email-based accounts and email password reset. Reason: "No email required (local network
  = trusted)" (src: SYSTEM_DESIGN.md:232).
- A hosted instance or hosted service run by the owner (Q-002), a public demo (Q-005; a local
  demo comes from `retrocast/scripts/demo.sh`), and any monetisation (Q-006).
- End-to-end encryption: TLS in transit and a trusted operator instead (Q-019).
- Federation between instances (Q-020).
- Automatic expiry of messages and files: they stay until their author or a moderator
  deletes them (Q-021).
- A stated uptime target or off-site backups: availability is best effort on one machine
  (Q-030).
- Declined: custom server emoji and link previews (Q-027); a desktop application (Q-028);
  LAN discovery and LAN certificates (Q-029). Declined 2026-11-23; do not re-propose.

## 10. Release scope

- **MVP (Pilot):** CAP-001, CAP-002, CAP-004, CAP-005, CAP-006, CAP-008, CAP-009, CAP-010,
  CAP-011, CAP-012, CAP-014, CAP-015, CAP-031, CAP-035, CAP-036, CAP-037, CAP-040. It must
  prove that SYSTEM_DESIGN Phase 1 works end to end on the web client: a group can sign up,
  form a server and chat in text in real time with working permissions, with RISK-001 and
  RISK-002 fixed, kick and ban, operator health and monitoring, and daily backups with a
  restore drill. The web client ships first; the iOS client follows (Q-003, Q-004).
- **Later — Voice and media (Phase 2):** CAP-003, CAP-017 (web voice, audio only, Q-024),
  CAP-018, CAP-019.
- **Later — Polish (Phase 3):** CAP-013, CAP-020, CAP-021, CAP-022, CAP-023 (with the iOS
  client, Q-025), CAP-024 (Unicode picker only), CAP-026.
- **Later — Nice to have (Phase 4):** CAP-007, CAP-016, CAP-028, CAP-030.
- **Later — not phased:** CAP-032, CAP-033, CAP-034 (built, held back from the MVP, Q-003);
  CAP-038 (Q-022); CAP-039 (Q-026).
- **Won't:** CAP-025 (Q-027), CAP-027 (Q-029), CAP-029 (Q-028); custom emoji (Q-027).

## 11. Risks and critical assumptions

| Id | Risk or assumption | Lens | Rating | Stance | Source |
|---|---|---|---|---|---|
| RISK-001 | Any server member can raise their own permissions: role create, update, delete, assign and channel-override endpoints check no permission | abuse of the product | kill (beyond a fully trusted group) | mitigate: fixed before the MVP ships (Q-003) | src: retrocast/internal/service/role.go:48–277, retrocast/internal/api/router.go:159–168 |
| RISK-002 | TODO.md marks features done that fail at runtime: DM send and attachment upload hit foreign-key errors; web sign-in reads a response envelope the auth endpoints do not send | feasibility | major | mitigate: one response envelope for every endpoint (Q-010); every RISK-002 defect (sign-in envelope, DM send, attachment upload) fixed before the MVP ships (Q-003); the DM and upload capabilities stay Later | src: retrocast/migrations/000007, 000008; retrocast/internal/service/upload.go:92; retrocast-web/src/lib/api.ts:113 |
| RISK-003 | An instance reachable from the internet accepts open registration, rate-limits by a client-supplied IP header, and allows any web origin. The seed assumed "local network = trusted" | abuse of the product | major | mitigate: invite-only registration by default (Q-012); client IP trusted only from the proxy hop and a limit on the invite lookup (Q-015); same-origin requests plus an operator list, only Caddy's ports published (Q-016) | src: SYSTEM_DESIGN.md:232; retrocast/internal/api/ratelimit.go; retrocast/cmd/retrocast/main.go (CORS); retrocast/internal/gateway/server.go:14 |
| RISK-004 | One machine is a single point of failure; disk loss loses messages and files | feasibility | minor at 5–20 users | mitigate: daily backups kept 30 days; a restore drill before each release (Q-030) | src: LIM-001; retrocast/scripts/backup.sh |
| RISK-005 | The iOS app cannot be built or tested on the owner's Windows machine, has no tests and no CI job; device distribution needs an Apple developer account | gatekeeper · economics | major for iOS | accept for the MVP: web first, iOS follows (Q-004) | src: retrocast-client/; .github/workflows/ci.yml |
| RISK-006 | Public material that calls the product a "Discord clone" or copies Discord's palette may raise trademark or trade-dress issues | legal/IP | minor while private; major if published | mitigate: no public demo (Q-005); no Discord branding in public material | src: README.md:3; TODO.md "Dark theme (Discord palette)" |
| RISK-007 | Licence statements conflict (README vs OpenAPI) | legal/IP | minor | mitigate: "All rights reserved" everywhere (Q-007) | src: LIM-004 |
| RISK-008 | Browser voice needs TLS ("Required for WebRTC — browsers block insecure WebRTC connections") and LiveKit needs UDP ports 50000–50100 reachable | feasibility | major for voice | mitigate when voice ships (Later): Caddy TLS with a domain or `localhost` (Q-029); open the UDP range | src: SYSTEM_DESIGN.md:690; retrocast/docker-compose.yml |
| RISK-009 | Some ids are sent as JSON numbers (READY guild list, member role lists, reaction users); 64-bit ids above 2^53 lose precision in JavaScript clients | feasibility | major | mitigate (send ids as strings everywhere) | src: retrocast/internal/models/member.go (`Roles []int64`), retrocast/internal/gateway (READY payload); issue #36 |
| RISK-010 | PR #22 proposes a required Rust core for permission resolution, adding a second toolchain | feasibility | minor | accept: resolution stays in Go; PR #22 is decided after RISK-001 is fixed (Q-008) | src: driver brief (PR #22 unmerged) |
| RISK-011 | Content members post (message text, file names) could run script in other members' clients | abuse of the product | major | mitigate ("Message content sanitized (no raw HTML), file names sanitized"; no SVG uploads, Q-017) | src: SYSTEM_DESIGN.md:691 |
| RISK-012 | Traffic or stored credentials are read in transit or from a stolen database | regulation/privacy | major | mitigate (HTTPS/WSS for all client connections; Argon2id password hashes; opaque refresh tokens) | src: SYSTEM_DESIGN.md:683–685 |

## 12. Prior art

The seed uses Discord as it was in 2015–2016 as the reference product
(src: SYSTEM_DESIGN.md §What Discord Was). No other prior art was researched: research is
not offered in unattended runs.

| Product / project | What it does | Gap or difference for us | Source |
|---|---|---|---|
| Discord (2015–2016) | Hosted servers, text and voice channels, roles | Retrocast is self-hosted and keeps the early scope | SYSTEM_DESIGN.md:30–63 |

## 13. Solution ideas

Status: `mandatory` (moved to §8 as LIM-) · `preference` (spec weighs it) · `idea`.

| Id | Idea | From | Status |
|---|---|---|---|
| SOL-001 | Go server with Echo; REST API and WebSocket gateway in one binary | SYSTEM_DESIGN.md:121, :353; retrocast/ | preference (built) |
| SOL-002 | PostgreSQL for all structured data and messages, with built-in full-text search | SYSTEM_DESIGN.md:136; retrocast/migrations | preference (built; English stemming, Q-023) |
| SOL-003 | Redis for sessions, presence, typing and rate limits | SYSTEM_DESIGN.md:79, :203 | preference (built) |
| SOL-004 | MinIO (S3-compatible) for attachments, avatars, icons and emoji | SYSTEM_DESIGN.md:194, :443 | preference (built for attachments only) |
| SOL-005 | LiveKit SFU for voice | SYSTEM_DESIGN.md:156 | preference (server token issuance built) |
| SOL-006 | Snowflake 64-bit ids | SYSTEM_DESIGN.md:561 | preference (built) |
| SOL-007 | Argon2id password hashing; JWT access tokens of 15 minutes; refresh tokens in Redis for 7 days | SYSTEM_DESIGN.md:233–234, :512 | preference (built) |
| SOL-008 | Short-lived pre-signed MinIO URLs so clients upload and download directly | SYSTEM_DESIGN.md:448–450 | preference (accepted, Q-011; the code uploads through the API today) |
| SOL-009 | SwiftUI native iOS client | SYSTEM_DESIGN.md:163–171 | preference (built) |
| SOL-010 | SvelteKit web client | SYSTEM_DESIGN.md:173–181 | idea (superseded in code by React 19 + Vite) |
| SOL-011 | React + Vite + TypeScript + Tailwind + Zustand web client | retrocast-web/package.json | preference (built) |
| SOL-012 | Tauri desktop app wrapping the web client | SYSTEM_DESIGN.md:183–190 | idea (not planned: desktop app Won't, Q-028) |
| SOL-013 | Caddy reverse proxy with automatic TLS | SYSTEM_DESIGN.md:156; retrocast/Caddyfile | preference (built) |
| SOL-014 | mkcert local CA for LAN TLS | SYSTEM_DESIGN.md:684, :690 | idea (not planned: Won't, Q-029) |
| SOL-015 | mDNS/Bonjour discovery via hashicorp/mdns and NWBrowser | SYSTEM_DESIGN.md:216–217 | idea (not planned: Won't, Q-029) |
| SOL-016 | Coturn TURN server | SYSTEM_DESIGN.md:268 | idea |
| SOL-017 | APNs notification worker | SYSTEM_DESIGN.md:292, :382 | idea (Later, with the iOS client, Q-025) |
| SOL-018 | Prometheus metrics and provisioned Grafana dashboards | TODO.md §Infra; retrocast/prometheus.yml | preference (built) |
| SOL-019 | Rust core for permission resolution (PR #22, unmerged) | driver brief | idea (parked: resolution stays in Go; decided after RISK-001 is fixed, Q-008) |
| SOL-020 | One response shape: every success body wrapped as `{"data": …}`, as a new major API version | docs/legacy/api.md; retrocast-web/src/lib/api.ts | preference (accepted, Q-010) |

## 14. Open questions

None. Every question below was answered on 2026-11-23 by **Victor Ivanov, who accepted all
recommendations in PR #25** (option (a) of each). Each answer is folded into the section named
in the last column; the options he did not choose are in this file's git history (commit
`037473c`).

### Answered (decision log)

| Id | Question | Decision (option (a), Victor Ivanov, 2026-11-23) | Folded into |
|---|---|---|---|
| Q-001 | Who is it for, and at what scale? | A private friend group of 5–20 people on one instance | §0, GOAL-001 |
| Q-002 | Self-hosted only, or also a hosted instance? | Self-hosted only; each group runs its own instance | LIM-001, §9 |
| Q-003 | What does MVP "done" mean, and where do the `?` capabilities go? | MVP = Phase 1 working end to end on the web client, with RISK-001 and RISK-002 fixed; kick and ban (CAP-031), operator health (CAP-035) and backups (CAP-036) join the MVP; reactions, unread badges and nicknames (CAP-032–CAP-034) are Later | §10 |
| Q-004 | Which client comes first: web or iOS? | Web first; iOS follows | §10, RISK-005 |
| Q-005 | Should there be a public demo? | No public demo; `scripts/demo.sh` seeds a local one | §9, RISK-006 |
| Q-006 | Any monetisation? | None | §9 |
| Q-007 | Which licence? | Keep "All rights reserved" (README.md) and remove `license: MIT` from openapi.yaml | LIM-004, RISK-007 |
| Q-008 | PR #22: adopt a required Rust core for permission resolution? | Keep permission resolution in Go; decide PR #22 after the Go authorisation defects (RISK-001) are fixed | LIM-003, SOL-019, RISK-010 |
| Q-009 | One live connection per user, or several devices at once? | Allow several live connections per user (for example web and iPhone together) | CAP-009 |
| Q-010 | One response shape for every endpoint? | Wrap every success response in `{"data": …}`, as a new major API version | SOL-020, RISK-002 |
| Q-011 | How should clients upload and fetch files? | Short-lived pre-signed URLs for upload and download | SOL-008 |
| Q-012 | Who may register on an instance? | The operator chooses open, invite-only or closed; default invite-only | CAP-040, RISK-003 |
| Q-013 | Who may start a DM with whom? | Only users who share a server can start a DM | CAP-020 |
| Q-014 | What may @everyone do by default? | View, Send, Read history, Connect, Speak, Create invite, Change nickname (`DefaultEveryonePerms`) | CAP-015 |
| Q-015 | Which proxy may the server trust for client IPs, and how are requests rate-limited? | Trust `X-Forwarded-For` only from the Caddy hop; limit unauthenticated routes by real client IP and authenticated routes by user; add a limit to the public invite lookup | RISK-003 |
| Q-016 | Which origins and ports are exposed? | Same origin behind Caddy, an operator-set list of extra allowed origins, and only Caddy's ports published (API port 8080, `/metrics`, Prometheus 9090 and Grafana 3000 kept internal) | RISK-003 |
| Q-017 | Upload size limit and allowed file types? | 10 MB; JPEG, PNG, GIF, WebP, PDF and plain text; no SVG | CAP-019, RISK-011 |
| Q-018 | Is kick, ban and role management enough moderation? | Kick, ban and role management only; the audit log stays Later | CAP-030, CAP-031 |
| Q-019 | End-to-end encryption? | Won't; TLS in transit and a trusted operator | §9 |
| Q-020 | Federation between instances? | Won't | §9 |
| Q-021 | How long are messages and files kept? | Until their author or a moderator deletes them; no automatic expiry | §9 |
| Q-022 | Password change and account deletion? | Password change in the MVP; account deletion Later | CAP-037, CAP-038 |
| Q-023 | Which language does message search assume? | English stemming (current) | CAP-028, SOL-002 |
| Q-024 | Voice in the web client, and video? | Add voice to the web client in Later (Phase 2), audio only | CAP-017, §9 |
| Q-025 | Push notifications and mentions: when? | Later, together with the iOS client | CAP-022, CAP-023, SOL-017 |
| Q-026 | Should a server owner be able to transfer ownership? | Add it (Later) | CAP-039 |
| Q-027 | Custom emoji, emoji picker and link previews: keep? | Keep the Unicode emoji picker in Later; drop custom emoji and link previews (Won't) | CAP-024, CAP-025, §9 |
| Q-028 | Desktop app: keep in Later or drop? | Won't; the web client covers desktop | CAP-029, SOL-012 |
| Q-029 | LAN discovery and LAN certificates: keep or drop? | Won't; Caddy with a domain or `localhost` covers TLS | CAP-027, SOL-014, SOL-015, RISK-008 |
| Q-030 | Availability and backup targets? | Best effort on one machine; daily backups kept 30 days; a restore drill before each release | CAP-036, RISK-004, §9 |
| Q-031 | Performance targets for the success measures? | At 20 connected members, a sent message reaches every other online member within 1 s (p95) on the operator's machine, and history pages of 50 messages return within 300 ms (p95) | GOAL-001 |
| Q-032 | Accessibility target and supported browsers and devices? | The web client meets WCAG 2.2 AA on the current and previous major versions of Chrome, Firefox, Safari and Edge; iOS 17 or later | SRS (web and iOS quality requirements) |
| Q-033 | Keep the Go module path `github.com/victorivanov/retrocast`? | Keep it | LIM-005 |
| Q-034 | What happens to the older prose docs under docs/? | Fold docs/architecture.md, database.md, permissions.md, api.md and gateway.md into the spec set, move the originals to `docs/legacy/`, and rename docs/ARCHITECTURE_SPEC.md to docs/ARCHITECTURE.md | done in SRS rev 2 (docs/legacy/) |
