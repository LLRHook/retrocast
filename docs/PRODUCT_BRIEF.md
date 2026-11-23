# Product Brief — Retrocast

Status: Gathering
Updated: 2026-11-23 · Sessions: 1 (unattended draft from repository evidence; no interviews held)
Feeds: docs/SRS.md via the `spec` skill. After `Specified`, the SRS is the source of truth
and this brief is not edited.

> **How this draft was made.** An agent wrote it without asking anyone. Every fact cites the
> repository file it came from as `(src: …)`. Nothing here records a decision by the project
> owner. Every unsettled product decision is a `Q-` item in §14, with options, the
> recommendation first, the evidence, and the assumption the SRS draft used meanwhile.

## 0. Readiness

| # | Check | State |
|---|---|---|
| 1 | Problem and who has it | ✅ (src: SYSTEM_DESIGN.md §Vision, README.md) |
| 2 | Goals with measures | ❌ Goals stated; no numeric measures (Q-031) |
| 3 | Buyer, primary user, affected party considered | ✅ Buyer = project owner; primary users = group members; affected party: none named in sources (Q-001 decides whether the public is one) |
| 4 | Each primary user has a capability | ✅ |
| 5 | Every capability has a release; none proposed | ❌ CAP-031–CAP-036 have Release `?` |
| 6 | Every solution idea classified | ✅ Classified from the existing code (built = `preference`, not built = `idea`); not yet confirmed by the owner |
| 7 | Constraints and non-goals answered | ✅ From the seed documents (src: SYSTEM_DESIGN.md) |
| 8 | Kill risks have a stance; no blocking question | ❌ Blocking: Q-001, Q-002, Q-003, Q-004; RISK-001 stance needs owner sign-off |

Readiness: 5/8. The gate needs a person, so this run stops before `Ready`.

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
| GOAL-001 | A friend group of 5–20 people can chat in text channels in real time on one self-hosted machine with no cloud service | TBD (Q-031) | src: SYSTEM_DESIGN.md:27 |
| GOAL-002 | Members can drop into persistent voice rooms with no call setup | TBD (Q-031) | src: SYSTEM_DESIGN.md:41 ("Persistent voice rooms — join/leave freely, no ringing, no call setup") |
| GOAL-003 | The feature set stays at early-Discord scope ("Simple, fast, focused") | The released capability set stays within §6 and §9 | src: SYSTEM_DESIGN.md:63 |
| GOAL-004 | The data stays on the operator's machine ("Your data, your server") | No core flow calls an external service; APNs push is the only optional exception | src: SYSTEM_DESIGN.md:27, :267 |

## 4. Stakeholders

| Id | Stakeholder | Role | Needs / concerns | Source |
|---|---|---|---|---|
| STK-001 | Project owner (Victor) | buyer | Decides scope, licence, hosting and spend; owns the repository | src: repository LLRHook/retrocast |
| STK-002 | Group member | primary user | Chats in text and voice with friends; no third-party account | src: SYSTEM_DESIGN.md:27, :265 |
| STK-003 | Server owner or admin | primary user | Creates servers, channels, roles and invites; moderates (kick, ban, roles) | src: SYSTEM_DESIGN.md:43, :61 |
| STK-004 | Instance operator | operator/admin | Runs the Docker Compose stack on one machine; backups, monitoring, upgrades | src: README.md:3, docs/deployment.md, retrocast/docker-compose.yml |
| STK-005 | Apple App Store review | gatekeeper/regulator | Applies only if the iOS app is distributed beyond development installs (Q-004) | src: retrocast-client/Retrocast/project.yml (bundle id com.retrocast.ios) |

## 5. Principles

- **PRN-001 — Self-hosted, data stays local:** Core functions need no external cloud
  service; "No cloud required. No accounts with a third party." (src: SYSTEM_DESIGN.md:27, :266)
- **PRN-002 — Early-Discord scope:** Features Discord lacked in 2015–2016 stay out
  (threads, bots, video and screen share, subscriptions, boosting, discovery).
  (src: SYSTEM_DESIGN.md:51–63)
- **PRN-003 — One protocol for every client:** "All clients share the same API surface
  (REST + WebSocket)." (src: SYSTEM_DESIGN.md:265)

## 6. Capabilities

Origin is `stated` when a repository document states the capability; nothing is `proposed`
or `accepted` because no owner answers exist yet. Release comes from the phase list in
SYSTEM_DESIGN.md §Feature Scope (Phase 1 is labelled "Core (MVP)", Phases 2–4 are `Later`).
A capability that the code or TODO.md has but no phase names has Release `?`.

### Accounts

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-001 | A member can register and sign in with a username and password, with no email (src: SYSTEM_DESIGN.md:232, :699) | STK-002 | GOAL-001, GOAL-004 | stated | MVP |
| CAP-002 | A member stays signed in between visits and can sign out (src: SYSTEM_DESIGN.md:234, :512) | STK-002 | GOAL-001 | stated | MVP |
| CAP-003 | A member can set a display name and an avatar; a server owner can set a server icon (src: SYSTEM_DESIGN.md:715, TODO.md "User settings (edit display name)") | STK-002, STK-003 | GOAL-001 | stated | Later |

### Servers and channels

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-004 | A member can create a server and becomes its owner (src: SYSTEM_DESIGN.md:700) | STK-002 | GOAL-001 | stated | MVP |
| CAP-005 | A member can join a server through an invite link and leave it (src: SYSTEM_DESIGN.md:700, :706) | STK-002 | GOAL-001 | stated | MVP |
| CAP-006 | A server admin can create, rename and delete text and voice channels, and rename or delete the server (src: SYSTEM_DESIGN.md:701, :39–40) | STK-003 | GOAL-001 | stated | MVP |
| CAP-007 | A server admin can group channels into categories (src: SYSTEM_DESIGN.md:735) | STK-003 | GOAL-001 | stated | Later |
| CAP-008 | A server admin can create invite links with optional expiry and use limits, list them, and revoke them (src: SYSTEM_DESIGN.md:47, :706) | STK-003 | GOAL-001 | stated | MVP |

### Text chat and real-time

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-009 | A member can send messages in a text channel and see others' messages appear without reloading (src: SYSTEM_DESIGN.md:701) | STK-002 | GOAL-001 | stated | MVP |
| CAP-010 | A member can scroll back through a channel's message history (src: SYSTEM_DESIGN.md:702) | STK-002 | GOAL-001 | stated | MVP |
| CAP-011 | A member sees when others are typing (src: SYSTEM_DESIGN.md:704) | STK-002 | GOAL-001 | stated | MVP |
| CAP-012 | A member sees who is online, idle, on do-not-disturb or offline, and can appear invisible (src: SYSTEM_DESIGN.md:44, :705) | STK-002 | GOAL-001 | stated | MVP |
| CAP-013 | A member can edit and delete their own messages; a moderator can delete others' messages (src: SYSTEM_DESIGN.md:725) | STK-002, STK-003 | GOAL-001 | stated | Later |

### Roles and permissions

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-014 | A server admin can create roles with a name, colour and permission set, order them, and assign them to members (src: SYSTEM_DESIGN.md:43, :703) | STK-003 | GOAL-001 | stated | MVP |
| CAP-015 | Only members whose roles allow an action can perform it (src: SYSTEM_DESIGN.md:703, docs/permissions.md) | STK-002, STK-003 | GOAL-001 | stated | MVP |
| CAP-016 | A server admin can allow or deny permissions per channel for a role (src: SYSTEM_DESIGN.md:736) | STK-003 | GOAL-001 | stated | Later |

### Voice and media

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-017 | A member can join and leave a persistent voice channel and talk with whoever is in it (src: SYSTEM_DESIGN.md:41, :710) | STK-002 | GOAL-002 | stated | Later |
| CAP-018 | A member can adjust per-user volume, use push-to-talk or voice activity, and mute or deafen; a moderator can server-mute (src: SYSTEM_DESIGN.md:711–713) | STK-002, STK-003 | GOAL-002 | stated | Later |
| CAP-019 | A member can share files and images in a channel and see image previews inline (src: SYSTEM_DESIGN.md:46, :714) | STK-002 | GOAL-001 | stated | Later |

### Direct messages and notifications

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-020 | A member can message another member one-to-one (src: SYSTEM_DESIGN.md:42, :719) | STK-002 | GOAL-001 | stated | Later |
| CAP-021 | A member can hold a group conversation of up to 10 people outside any server (src: SYSTEM_DESIGN.md:42, :720) | STK-002 | GOAL-001 | stated | Later |
| CAP-022 | A member can mention a user, a role, @everyone or @here, and chooses which mentions notify them (src: SYSTEM_DESIGN.md:45, :721) | STK-002 | GOAL-001 | stated | Later |
| CAP-023 | A member gets a push notification on iPhone when mentioned or messaged while the app is closed (src: SYSTEM_DESIGN.md:267, :726) | STK-002 | GOAL-001 | stated | Later |

### Expression

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-024 | A member can insert Unicode emoji from a picker, and a server can add custom emoji (src: SYSTEM_DESIGN.md:722–723) | STK-002, STK-003 | GOAL-001 | stated | Later |
| CAP-025 | A posted link shows a preview with the page's metadata (src: SYSTEM_DESIGN.md:49, :724) | STK-002 | GOAL-001 | stated | Later |
| CAP-026 | The clients look like the iOS 7 era: flat design, translucent panels, clean type (src: SYSTEM_DESIGN.md:25, :727) | STK-002 | GOAL-003 | stated | Later |

### Phase 4 ("Nice to Have")

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-027 | A client on the same local network finds the server without typing its address (src: SYSTEM_DESIGN.md:269, :731) | STK-002 | GOAL-001 | stated | Later |
| CAP-028 | A member can search a server's message history by text, author and date (src: SYSTEM_DESIGN.md:732) | STK-002 | GOAL-001 | stated | Later |
| CAP-029 | A member can use a desktop application (src: SYSTEM_DESIGN.md:733) | STK-002 | GOAL-001 | stated | Later |
| CAP-030 | A server admin can review a log of moderation and configuration actions (src: SYSTEM_DESIGN.md:734) | STK-003 | GOAL-001 | stated | Later |

### Built or listed, with no release in the sources

| Id | Capability (outcome, no mechanism) | For | Serves | Origin | Release |
|---|---|---|---|---|---|
| CAP-031 | A moderator can kick a member, ban a user (with an optional reason), list bans and lift a ban (src: SYSTEM_DESIGN.md:61 "just kick, ban, and role management"; TODO.md "Ban system") | STK-003 | GOAL-001 | stated | ? |
| CAP-032 | A member can react to a message with an emoji and see reaction counts (src: TODO.md "Message reactions/emojis") | STK-002 | GOAL-001 | stated | ? |
| CAP-033 | A member sees which channels have unread messages (src: TODO.md "Read states / unread badges") | STK-002 | GOAL-001 | stated | ? |
| CAP-034 | A member can set a per-server nickname; a moderator can change others' nicknames (src: retrocast/internal/service/member.go, docs/permissions.md) | STK-002, STK-003 | GOAL-001 | stated | ? |
| CAP-035 | The operator can see service health and metrics dashboards (src: TODO.md §Infra "Health check endpoint", "Prometheus metrics", "Grafana dashboards") | STK-004 | GOAL-004 | stated | ? |
| CAP-036 | The operator can back up and restore the database and stored files, apply migrations and load demo data from a command line (src: TODO.md §Infra "Backup automation", "CLI tool (retrocast-cli)", "Demo bootstrap script") | STK-004 | GOAL-004 | stated | ? |

Clients (web, iOS) are surfaces for these capabilities, not capabilities themselves; their
order is Q-004.

### Key scenarios

1. **First evening.** The operator runs the compose stack. A member registers (CAP-001),
   creates a server (CAP-004), gets a `general` text channel and a `General` voice channel,
   and sends invite links (CAP-008). Friends join (CAP-005) and chat in real time
   (CAP-009, CAP-011, CAP-012). Not binding.
2. **Moderation.** An admin creates a "Moderator" role (CAP-014). A moderator removes a
   member who is spamming (CAP-031), and the removed member stops receiving that server's
   messages.
3. **Failure: unauthorised action.** A member with no special role tries to create an
   administrator role or delete a channel. The server refuses (CAP-015). This path is
   broken today (RISK-001).
4. **Failure: expired invite.** A friend opens an invite after it expired or reached its use
   limit and is told it is no longer valid (CAP-005, CAP-008).
5. **Voice night (Later).** Members join the `General` voice channel and talk (CAP-017).

## 7. Domain terms

| Term | Meaning (plain language) | Relates to |
|---|---|---|
| Server (guild) | A space with its own channels, members and roles. The code and API call it a guild | CAP-004 |
| Channel | A text room, voice room or category inside a server | CAP-006, CAP-007 |
| Role | A named set of permissions with a colour and a rank; members can hold several | CAP-014 |
| @everyone | The default role every server has; its permissions apply to all members | CAP-015 |
| Permission override | A per-channel allow or deny for a role | CAP-016 |
| Invite | A short code that lets someone join a server, with an optional expiry and use limit | CAP-005, CAP-008 |
| Presence | Online, idle, do-not-disturb, invisible (shown as offline) or offline | CAP-012 |
| Gateway | The live connection that pushes new messages, typing and presence to clients | CAP-009 |
| DM, group DM | A conversation outside any server, with one person or with up to 10 people | CAP-020, CAP-021 |
| Read state | Where a member last read in a channel; drives unread badges | CAP-033 |
| Snowflake | A 64-bit, time-ordered id used for every object | — |
| Operator | The person who runs the server stack on their machine | STK-004 |

## 8. Constraints

| Id | Constraint | Kind | Hard / soft | Source |
|---|---|---|---|---|
| LIM-001 | Runs on a single Mac or Linux machine; no cloud required | hosting | hard | src: SYSTEM_DESIGN.md:27 |
| LIM-002 | No third-party account is needed to use the product; registration needs no email | regulation | hard | src: SYSTEM_DESIGN.md:27, :232 |
| LIM-003 | The existing system is reused: a Go server (one binary for REST and the gateway), PostgreSQL, Redis, MinIO and LiveKit, deployed with Docker Compose behind Caddy; a React web client; a SwiftUI iOS client | existing system | soft (a change is the owner's decision, see Q-008) | src: retrocast/go.mod, retrocast/docker-compose.yml, retrocast-web/package.json, retrocast-client/Retrocast/project.yml |
| LIM-004 | The licence is unresolved: README.md says "unlicensed. All rights reserved."; retrocast/docs/openapi.yaml says `license: MIT` | licence | hard once decided | Q-007 |

## 9. Non-goals

- Threads, bots and slash commands, screen sharing and video, paid subscriptions, server
  boosting, stage and forum channels, activities, stickers, polls, server discovery.
  Reason: Discord did not have them in 2015–2016, which is the targeted scope
  (src: SYSTEM_DESIGN.md:51–63).
- Email-based accounts and email password reset. Reason: "No email required (local network
  = trusted)" (src: SYSTEM_DESIGN.md:232).

## 10. Release scope

- **MVP:** CAP-001, CAP-002, CAP-004, CAP-005, CAP-006, CAP-008, CAP-009, CAP-010, CAP-011,
  CAP-012, CAP-014, CAP-015. It must prove that a group can sign up, form a server and chat
  in text in real time with working permissions. Source: SYSTEM_DESIGN.md Phase 1. Q-003
  asks the owner to confirm this scope.
- **Later — Voice and media (Phase 2):** CAP-003, CAP-017, CAP-018, CAP-019.
- **Later — Polish (Phase 3):** CAP-013, CAP-020, CAP-021, CAP-022, CAP-023, CAP-024,
  CAP-025, CAP-026.
- **Later — Nice to have (Phase 4):** CAP-007, CAP-016, CAP-027, CAP-028, CAP-029, CAP-030.
- **Release not set (`?`):** CAP-031, CAP-032, CAP-033, CAP-034, CAP-035, CAP-036 (Q-003).
- **Won't:** none decided. Q-020, Q-021, Q-027 and Q-028 propose candidates.

## 11. Risks and critical assumptions

| Id | Risk or assumption | Lens | Rating | Stance | Source |
|---|---|---|---|---|---|
| RISK-001 | Any server member can raise their own permissions: role create, update, delete, assign and channel-override endpoints check no permission | abuse of the product | kill (beyond a fully trusted group) | mitigate (fix before any wider use; owner sign-off pending) | src: retrocast/internal/service/role.go:48–277, retrocast/internal/api/router.go:159–168 |
| RISK-002 | TODO.md marks features done that fail at runtime: DM send and attachment upload hit foreign-key errors; web sign-in reads a response envelope the auth endpoints do not send | feasibility | major | validate before build (run the V&V protocol; fix) | src: retrocast/migrations/000007, 000008; retrocast/internal/service/upload.go:92; retrocast-web/src/lib/api.ts:113; evidence in this run's report |
| RISK-003 | An instance reachable from the internet accepts open registration, rate-limits by a client-supplied IP header, and allows any web origin. The seed assumed "local network = trusted" | abuse of the product | major | open (Q-001, Q-012, Q-015, Q-016) | src: SYSTEM_DESIGN.md:232; retrocast/internal/api/ratelimit.go; retrocast/cmd/retrocast/main.go (CORS); retrocast/internal/gateway/server.go:14 |
| RISK-004 | One machine is a single point of failure; disk loss loses messages and files | feasibility | minor at 5–20 users | mitigate (scheduled backups; restore drill) (Q-030) | src: LIM-001; retrocast/scripts/backup.sh |
| RISK-005 | The iOS app cannot be built or tested on the owner's Windows machine, has no tests and no CI job; device distribution needs an Apple developer account | gatekeeper · economics | major for iOS | open (Q-004) | src: retrocast-client/; .github/workflows/ci.yml |
| RISK-006 | Public material that calls the product a "Discord clone" or copies Discord's palette may raise trademark or trade-dress issues | legal/IP | minor while private; major if published | mitigate (no Discord branding in public material) (Q-005) | src: README.md:3; TODO.md "Dark theme (Discord palette)" |
| RISK-007 | Licence statements conflict (README vs OpenAPI) | legal/IP | minor | open (Q-007) | src: LIM-004 |
| RISK-008 | Browser voice needs TLS ("Required for WebRTC — browsers block insecure WebRTC connections") and LiveKit needs UDP ports 50000–50100 reachable | feasibility | major for voice | mitigate (Caddy TLS; open UDP range) (Q-024, Q-029) | src: SYSTEM_DESIGN.md:690; retrocast/docker-compose.yml |
| RISK-009 | Some ids are sent as JSON numbers (READY guild list, member role lists); 64-bit ids above 2^53 lose precision in JavaScript clients | feasibility | major | mitigate (send ids as strings everywhere) | src: retrocast/internal/models/member.go (`Roles []int64`), retrocast/internal/gateway (READY payload) |
| RISK-010 | PR #22 proposes a required Rust core for permission resolution, adding a second toolchain | feasibility | minor | open (Q-008) | src: driver brief for this run (PR not read) |
| RISK-011 | Content members post (message text, file names) could run script in other members' clients | abuse of the product | major | mitigate ("Message content sanitized (no raw HTML), file names sanitized") | src: SYSTEM_DESIGN.md:691 |
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
Built items are classified `preference` from the code; the owner has not confirmed this.

| Id | Idea | From | Status |
|---|---|---|---|
| SOL-001 | Go server with Echo; REST API and WebSocket gateway in one binary | SYSTEM_DESIGN.md:121, :353; retrocast/ | preference (built) |
| SOL-002 | PostgreSQL for all structured data and messages, with built-in full-text search | SYSTEM_DESIGN.md:136; retrocast/migrations | preference (built) |
| SOL-003 | Redis for sessions, presence, typing and rate limits | SYSTEM_DESIGN.md:79, :203 | preference (built) |
| SOL-004 | MinIO (S3-compatible) for attachments, avatars, icons and emoji | SYSTEM_DESIGN.md:194, :443 | preference (built for attachments only) |
| SOL-005 | LiveKit SFU for voice | SYSTEM_DESIGN.md:156 | preference (server token issuance built) |
| SOL-006 | Snowflake 64-bit ids | SYSTEM_DESIGN.md:561 | preference (built) |
| SOL-007 | Argon2id password hashing; JWT access tokens of 15 minutes; refresh tokens in Redis for 7 days | SYSTEM_DESIGN.md:233–234, :512 | preference (built) |
| SOL-008 | Pre-signed MinIO URLs so clients upload and download directly | SYSTEM_DESIGN.md:448–450 | idea (code uploads through the API instead; Q-011) |
| SOL-009 | SwiftUI native iOS client | SYSTEM_DESIGN.md:163–171 | preference (built) |
| SOL-010 | SvelteKit web client | SYSTEM_DESIGN.md:173–181 | idea (superseded in code by React 19 + Vite) |
| SOL-011 | React + Vite + TypeScript + Tailwind + Zustand web client | retrocast-web/package.json | preference (built) |
| SOL-012 | Tauri desktop app wrapping the web client | SYSTEM_DESIGN.md:183–190 | idea (Q-028) |
| SOL-013 | Caddy reverse proxy with automatic TLS | SYSTEM_DESIGN.md:156; retrocast/Caddyfile | preference (built) |
| SOL-014 | mkcert local CA for LAN TLS | SYSTEM_DESIGN.md:684, :690 | idea (Q-029) |
| SOL-015 | mDNS/Bonjour discovery via hashicorp/mdns and NWBrowser | SYSTEM_DESIGN.md:216–217 | idea (Q-029) |
| SOL-016 | Coturn TURN server | SYSTEM_DESIGN.md:268 | idea |
| SOL-017 | APNs notification worker | SYSTEM_DESIGN.md:292, :382 | idea (Q-025) |
| SOL-018 | Prometheus metrics and provisioned Grafana dashboards | TODO.md §Infra; retrocast/prometheus.yml | preference (built) |
| SOL-019 | Rust core for permission resolution (PR #22, unmerged) | driver brief | idea (Q-008) |

## 14. Open questions

Only open ones. Ordered by theme; within a theme, the question that changes the outcome most
comes first. Options and evidence follow the table.

| Id | Question | Why it matters (ids) | Owner | Blocking |
|---|---|---|---|---|
| Q-001 | Who is it for, and at what scale? | Every NFR, moderation need, hosting (GOAL-001, RISK-003) | Victor | yes |
| Q-002 | Self-hosted only, or also a hosted instance? | Hosting, cost, security posture (LIM-001, PRN-001) | Victor | yes |
| Q-003 | What does MVP "done" mean, and where do the `?` capabilities go? | Release scope (§10), release tiers | Victor | yes |
| Q-004 | Which client comes first: web or iOS? | Client priorities (STK-005, RISK-005) | Victor | yes |
| Q-005 | Should there be a public demo? | Cost, abuse, IP exposure (RISK-003, RISK-006) | Victor | no |
| Q-006 | Any monetisation? | Scope, licence, hosting | Victor | no |
| Q-007 | Which licence? | Distribution (LIM-004, RISK-007) | Victor | no |
| Q-008 | PR #22: adopt a required Rust core for permission resolution? | Stack, CI, contributors (LIM-003, RISK-010) | Victor | no |
| Q-009 | One live connection per user, or several devices at once? | Web and iOS used together (CAP-009, CAP-012) | Victor | no |
| Q-010 | One response shape for every endpoint? | Client compatibility; breaking change (RISK-002) | Victor | no |
| Q-011 | How should clients upload and fetch files? | CAP-019, security of stored files | Victor | no |
| Q-012 | Who may register on an instance? | Abuse (RISK-003, CAP-001) | Victor | no |
| Q-013 | Who may start a DM with whom? | Abuse (CAP-020) | Victor | no |
| Q-014 | What may @everyone do by default? | Voice, invites, nicknames for new members (CAP-015) | Victor | no |
| Q-015 | Which proxy may the server trust for client IPs, and how are requests rate-limited? | Abuse, lockout (RISK-003) | Victor | no |
| Q-016 | Which origins and ports are exposed? | Security (RISK-003) | Victor | no |
| Q-017 | Upload size limit and allowed file types? | CAP-019, security | Victor | no |
| Q-018 | Is kick, ban and role management enough moderation? | CAP-030, CAP-031 | Victor | no |
| Q-019 | End-to-end encryption? | Search, trust model (CAP-028) | Victor | no |
| Q-020 | Federation between instances? | Scope (PRN-002) | Victor | no |
| Q-021 | How long are messages and files kept? | Storage, privacy (RISK-004) | Victor | no |
| Q-022 | Password change and account deletion? | Privacy, account recovery (CAP-001) | Victor | no |
| Q-023 | Which language does message search assume? | CAP-028 | Victor | no |
| Q-024 | Voice in the web client, and video? | CAP-017, PRN-002 | Victor | no |
| Q-025 | Push notifications and mentions: when? | CAP-022, CAP-023 | Victor | no |
| Q-026 | Should a server owner be able to transfer ownership? | Owner cannot leave today (CAP-005) | Victor | no |
| Q-027 | Custom emoji, emoji picker and link previews: keep? | CAP-024, CAP-025 | Victor | no |
| Q-028 | Desktop app: keep in Later or drop? | CAP-029 | Victor | no |
| Q-029 | LAN discovery and LAN certificates: keep or drop? | CAP-027, RISK-008 | Victor | no |
| Q-030 | Availability and backup targets? | RISK-004, CAP-036 | Victor | no |
| Q-031 | Performance targets for the success measures? | GOAL-001, GOAL-002 | Victor | no |
| Q-032 | Accessibility target and supported browsers and devices? | Web and iOS clients | Victor | no |
| Q-033 | Keep the Go module path `github.com/victorivanov/retrocast`? | Imports, tooling | Victor | no |
| Q-034 | What happens to the older prose docs under docs/? | Documentation ownership; file-name clash | Victor | no |

### Options and evidence

Recommendation first. "Assumed" is what the SRS draft used until the owner answers.

**Product direction**

- **Q-001 — Audience and scale.** (a) **Recommended:** a private friend group of 5–20 on
  one instance. (b) Small communities of up to a few hundred per instance. (c) A public
  multi-tenant service. Evidence: "serving 5–20 friends" (SYSTEM_DESIGN.md:27); "a small
  friend group" (README.md:3). Assumed: (a).
- **Q-002 — Hosting.** (a) **Recommended:** self-hosted only; each group runs its own
  instance. (b) Self-hosted, plus one hosted instance run by the owner. (c) Hosted service
  only. Evidence: LIM-001; Docker Compose stack, no IaC (retrocast/docker-compose.yml).
  Assumed: (a).
- **Q-003 — MVP "done" and release tiers.** (a) **Recommended:** MVP = Phase 1 working end
  to end on the web client, with RISK-001 and RISK-002 fixed; kick and ban (CAP-031) join the
  MVP; reactions, unread badges and nicknames (CAP-032–CAP-034) are Later; operator health and
  backups (CAP-035–CAP-036) join the MVP. (b) MVP = everything TODO.md marks done, made to work.
  (c) MVP = Phase 1 plus voice. Evidence: SYSTEM_DESIGN.md §Feature Scope; TODO.md marks all
  items done; RISK-002. Assumed: Phase 1 = MVP; `?` capabilities left as TBD.
- **Q-004 — Client priority.** (a) **Recommended:** web first; iOS follows. (b) iOS first.
  (c) Both at once. Evidence: the web client is in CI; iOS has no CI job, no tests, and
  cannot build on the owner's Windows machine (RISK-005). Assumed: (a).
- **Q-005 — Public demo.** (a) **Recommended:** no public demo; `scripts/demo.sh` seeds a
  local one. (b) A time-limited public demo that resets daily. (c) A permanent public demo.
  Evidence: retrocast/scripts/demo.sh; RISK-003, RISK-006. Assumed: (a).
- **Q-006 — Monetisation.** (a) **Recommended:** none. (b) Donations. (c) A paid hosted
  tier. Evidence: README.md:3 (friend group); no billing code. Assumed: (a).
- **Q-007 — Licence.** (a) **Recommended:** keep "All rights reserved" (README.md) and
  remove `license: MIT` from openapi.yaml. (b) MIT for the whole repository, with a LICENSE
  file. (c) AGPL-3.0, so hosted forks must share changes. Evidence: README.md "unlicensed.
  All rights reserved."; retrocast/docs/openapi.yaml `license: MIT`. Assumed: unresolved.

**Architecture direction**

- **Q-008 — PR #22 Rust core.** (a) **Recommended:** keep permission resolution in Go and
  decide PR #22 after the Go authorisation defects (RISK-001) are fixed; those defects are
  missing checks in the service layer, not in the bit arithmetic. (b) Accept a required Rust
  core. (c) Accept it as optional or experimental. Evidence: driver brief (PR #22 unmerged,
  CI failing); retrocast/internal/service/role.go. The PR itself was not read in this run.
  Assumed: (a).
- **Q-009 — Connections per user.** (a) **Recommended:** allow several live connections per
  user (for example web and iPhone together). (b) Keep one; a new connection closes the older
  one. Evidence: the gateway closes the older connection with RECONNECT
  (retrocast/internal/gateway). Assumed: (b), the current behaviour.
- **Q-010 — Response shape.** (a) **Recommended:** wrap every success response in
  `{"data": …}` as docs/api.md and the web client expect, as a new major API version.
  (b) Use bare JSON everywhere and fix the web client. (c) Keep the mix and make clients
  accept both. Evidence: guild, channel, member and user handlers wrap; auth, message, role,
  invite, DM, ban, reaction, read-state, search, voice and upload handlers do not;
  retrocast-web/src/lib/api.ts:113 always unwraps `data`; the iOS client accepts both.
  Assumed: (c) recorded as the current contract.
- **Q-011 — File transfer.** (a) **Recommended:** short-lived pre-signed URLs for upload
  and download, as SYSTEM_DESIGN.md planned. (b) Upload and download through the API with a
  permission check. (c) A public bucket behind Caddy. Evidence: SYSTEM_DESIGN.md:448–450;
  code uploads through the API and returns `http://<minio endpoint>/<bucket>/<key>`
  (retrocast/internal/storage/minio.go), which clients outside the compose network cannot
  reach. Assumed: TBD.

**Trust, safety and security**

- **Q-012 — Registration.** (a) **Recommended:** the operator chooses open, invite-only or
  closed, default invite-only once the instance is reachable from the internet. (b) Always
  open (current). Evidence: "No email required (local network = trusted)"
  (SYSTEM_DESIGN.md:232); register needs no invite (retrocast/internal/service/auth.go).
  Assumed: (b).
- **Q-013 — DM policy.** (a) **Recommended:** only users who share a server can start a
  DM. (b) Anyone on the instance (current). (c) A friends list. Evidence: any user id is
  accepted (retrocast/internal/service/dm.go). Assumed: (b).
- **Q-014 — Default @everyone permissions.** (a) **Recommended:** View, Send, Read history,
  Connect, Speak, Create invite, Change nickname, as `DefaultEveryonePerms` and
  docs/permissions.md say. (b) Keep View, Send and Read history only (current code). Evidence:
  retrocast/internal/permissions/bitfield.go:46 versus retrocast/internal/service/guild.go:68.
  With (b), members without another role cannot join voice or create invites. Assumed: (b),
  flagged as a conflict.
- **Q-015 — Proxy trust and rate limits.** (a) **Recommended:** trust `X-Forwarded-For`
  only from the Caddy hop; limit unauthenticated routes by real client IP and authenticated
  routes by user; add a limit to the public invite lookup. (b) Ignore forwarded headers; all
  traffic through Caddy then shares one bucket. (c) Limit by user only. Evidence:
  retrocast/internal/api/ratelimit.go uses `c.RealIP()` with no trusted-proxy setting, so
  any client can choose its bucket; `GET /api/v1/invites/:code` has no limit
  (retrocast/internal/api/router.go). Assumed: (a) as the target, TBD.
- **Q-016 — Origins and exposed ports.** (a) **Recommended:** same origin behind Caddy, an
  operator-set list of extra allowed origins, and only Caddy's ports published (API port
  8080, `/metrics`, Prometheus 9090 and Grafana 3000 kept internal). (b) Keep allow-all
  origins and published ports. Evidence: CORS `AllowOrigins: ["*"]`
  (retrocast/cmd/retrocast/main.go); WebSocket `CheckOrigin` accepts all
  (retrocast/internal/gateway/server.go:14); compose publishes 8080, 9000, 9001, 9090 and
  3000. Assumed: TBD.
- **Q-017 — Upload limit and types.** (a) **Recommended:** 10 MB; JPEG, PNG, GIF, WebP,
  PDF and plain text; no SVG. (b) Current: 10 MB and any `image/*`, including SVG. (c) A
  larger limit such as 25 MB. Evidence: retrocast/internal/service/upload.go; the seed says
  8 MB (SYSTEM_DESIGN.md:46). Assumed: (b).
- **Q-018 — Moderation depth.** (a) **Recommended:** kick, ban and role management only,
  as the seed says; audit log stays Later. (b) Add an audit log to the MVP. Evidence:
  SYSTEM_DESIGN.md:61, :734. Assumed: (a).
- **Q-019 — End-to-end encryption.** (a) **Recommended:** Won't; TLS in transit and a
  trusted operator. (b) Later, for DMs only. Evidence: server-side search needs plaintext
  (retrocast/migrations/000015). Assumed: (a).
- **Q-020 — Federation.** (a) **Recommended:** Won't. (b) Later. Evidence: single-instance
  design (SYSTEM_DESIGN.md:27). Assumed: (a).

**Data and privacy**

- **Q-021 — Retention.** (a) **Recommended:** keep messages and files until their author or
  a moderator deletes them; no automatic expiry. (b) An operator-set retention window.
  Evidence: no expiry code; backups keep 30 days (retrocast/scripts/backup.sh
  `RETENTION_DAYS=30`). Assumed: (a).
- **Q-022 — Account management.** (a) **Recommended:** password change in the MVP; account
  deletion Later. (b) Both in the MVP. (c) Both Later. Evidence: no endpoints exist
  (retrocast/internal/api/router.go). Assumed: TBD.
- **Q-023 — Search language.** (a) **Recommended:** English stemming (current). (b)
  Language-neutral matching. Evidence: `to_tsvector('english', …)`
  (retrocast/migrations/000015). Assumed: (a).

**Feature scope**

- **Q-024 — Web voice and video.** (a) **Recommended:** add voice to the web client in
  Later (Phase 2), audio only. (b) Voice in the MVP. (c) Voice on iOS only. Evidence: the
  server issues LiveKit tokens (retrocast/internal/service/voice.go); the web client has no
  voice code; the seed excludes video (SYSTEM_DESIGN.md:56) but TODO.md says "Voice/video via
  LiveKit integration". Assumed: (a).
- **Q-025 — Push and mentions.** (a) **Recommended:** Later, together with the iOS client.
  (b) MVP. (c) Won't. Evidence: the `device_tokens` table exists but nothing uses it; no
  mention parsing (retrocast/migrations, retrocast/internal/service). Assumed: (a).
- **Q-026 — Ownership transfer.** (a) **Recommended:** add it (Later). (b) Won't; the
  owner deletes the server instead. Evidence: the owner cannot leave ("transfer ownership or
  delete", retrocast/internal/service/member.go); no transfer endpoint. Assumed: TBD.
- **Q-027 — Emoji and link previews.** (a) **Recommended:** keep the Unicode emoji picker
  in Later; drop custom emoji and link previews (Won't). (b) Keep all three in Later.
  Evidence: SYSTEM_DESIGN.md:722–724; none built. Assumed: (b), as stated.
- **Q-028 — Desktop app.** (a) **Recommended:** Won't; the web client covers desktop.
  (b) Keep in Later. Evidence: SYSTEM_DESIGN.md:733; the planned web framework changed
  (SOL-010, SOL-011). Assumed: (b), as stated.
- **Q-029 — LAN discovery and certificates.** (a) **Recommended:** Won't; Caddy with a
  domain or `localhost` covers TLS. (b) Keep in Later. Evidence: SYSTEM_DESIGN.md:216, :690,
  :731; retrocast/Caddyfile `{$DOMAIN:localhost}`. Assumed: (b), as stated.

**Quality targets**

- **Q-030 — Availability and backups.** (a) **Recommended:** best effort on one machine;
  daily backups kept 30 days; a restore drill before each release. (b) A stated uptime target
  such as 99.5% a month, with off-site backups. Evidence: retrocast/scripts/backup.sh,
  restore.sh. Assumed: TBD.
- **Q-031 — Performance targets.** (a) **Recommended:** at 20 connected members, a sent
  message reaches every other online member within 1 s (p95) on the operator's machine, and
  history pages of 50 messages return within 300 ms (p95). (b) No targets until measured.
  Evidence: none stated; no benchmark harness. Assumed: TBD.
- **Q-032 — Accessibility and platforms.** (a) **Recommended:** web client meets WCAG 2.2
  AA on the current and previous major versions of Chrome, Firefox, Safari and Edge; iOS 17
  or later. (b) No formal target. Evidence: iOS deployment target 17.0
  (retrocast-client/Retrocast/project.yml); no web target stated. Assumed: TBD.

**Repository**

- **Q-033 — Go module path.** (a) **Recommended:** keep `github.com/victorivanov/retrocast`.
  (b) Rename to `github.com/LLRHook/retrocast`. Evidence: retrocast/go.mod; the repository is
  LLRHook/retrocast. Assumed: (a).
- **Q-034 — Older prose docs.** (a) **Recommended:** fold docs/architecture.md, database.md,
  permissions.md, api.md and gateway.md into the spec set, move the originals to
  `docs/legacy/`, and rename docs/ARCHITECTURE_SPEC.md to docs/ARCHITECTURE.md. (b) Keep both
  sets and mark the old ones as background. Evidence: the old docs disagree with the code
  (for example docs/architecture.md says there is no service layer); on Windows and macOS,
  docs/ARCHITECTURE.md and docs/architecture.md are the same file name, so the spec
  architecture file uses another name for now. Assumed: (b) until answered.
