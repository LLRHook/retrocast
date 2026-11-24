# Software Requirements Specification â€” Retrocast

Conforms to the SRS content outline of ISO/IEC/IEEE 29148:2018 Â§9.6.
Related: [Product Brief](PRODUCT_BRIEF.md) Â· [Data Contracts](DATA_CONTRACTS.md) Â· [Architecture](ARCHITECTURE.md)
Status: Baselined (rev 7)

> **Rev 2 (2026-11-23).** Rev 1 was written unattended from repository evidence. On
> 2026-11-23 Victor Ivanov accepted every recommendation of the product brief (Q-001â€“Q-034,
> PRODUCT_BRIEF Â§14). This revision records those decisions: every TBD in Â§5.2 is resolved,
> new requirements cover the capabilities the answers moved into the MVP, and `Delivery` was
> re-read at `origin/main` 55e50eb and checked against the compose smoke run of the same day.
> Design choices that implement an answer are the spec lead's or the driver's and are ADRs in
> [ARCHITECTURE.md Â§8](ARCHITECTURE.md#8-architecture-decisions); they are not attributed to
> the owner.

This is the index of a multi-component SRS. Requirements live in the component files listed
in Â§3; each id is defined in exactly one file.

## 1. Introduction

### 1.1 Purpose
This document specifies what Retrocast shall do: the server (REST API and real-time
gateway), the web client, the iOS client, and the system-wide constraints. It is read by the
project owner, by `/groom` (to cut tickets) and by `release-readiness` (to judge milestones).

### 1.2 Scope
Retrocast is a self-hosted chat product in the style of Discord in 2015â€“2016: servers
(guilds), text channels, voice channels, roles and permissions, invites, presence and
typing, for a private group of 5â€“20 friends on one machine (PRODUCT_BRIEF GOAL-001â€“GOAL-004,
Q-001). It will not include threads, bots, video, subscriptions, boosting, server discovery,
end-to-end encryption, federation or a hosted service (PRODUCT_BRIEF Â§9). Success measures are
SYS-NFR-001, SYS-NFR-002 and SYS-NFR-007 (Q-031).

### 1.3 Product perspective
A standalone system the operator runs as one Docker Compose stack: the Go server, PostgreSQL,
Redis, MinIO, LiveKit, the web client served by nginx, Caddy in front, plus Prometheus and
Grafana, with host-scheduled backups. Context diagram:
[ARCHITECTURE.md Â§3](ARCHITECTURE.md#3-system-context-c4-level-1).

#### 1.3.1 System interfaces
Clients reach the server only through Caddy: REST under `/api/v1`, the gateway at `/gateway`
(WebSocket), `/health` and the raw specification at `/docs`. Compose publishes only
Caddy 80/443 and LiveKit 7880/7881 plus UDP 50000-50100; Grafana binds to
127.0.0.1:3000. API 8080, MinIO and Prometheus stay internal (ADR-016, PR #53). The server calls PostgreSQL, Redis and MinIO, and mints LiveKit access
tokens that clients present to LiveKit (DATA_CONTRACTS Â§7).

#### 1.3.2 User interfaces
- **web:** browser single-page app (retrocast-web): server address, login and register
  pages; three-column app shell with server list, channel sidebar, chat area and member
  list; modals for servers, channels, invites, roles, bans, guild settings and user settings.
  The MVP ships on the web client (Q-004).
- **ios:** SwiftUI app (retrocast-client): the same flows in a `NavigationSplitView`, plus
  search and voice channel views. It follows the web client.
- Accessibility and platforms: WCAG 2.2 AA on the current and previous major versions of
  Chrome, Firefox, Safari and Edge (WEB-NFR-001, WEB-NFR-002); iOS 17 or later (IOS-NFR-001).

#### 1.3.3 Hardware interfaces
No hardware interfaces beyond the operator's host and the clients' microphones and speakers
for voice (Later).

#### 1.3.4 Software interfaces
REST operations API-001â€“API-059 (contract v2.0.0), gateway frames and events EVT-001â€“EVT-030,
LiveKit token contract and object storage contract (DATA_CONTRACTS Â§4, Â§6, Â§7).

#### 1.3.5 Communications interfaces
HTTPS and WSS through Caddy (automatic TLS for the configured `DOMAIN`, else `localhost`).
LiveKit signalling on 7880/7881 and media on UDP 50000â€“50100 (retrocast/docker-compose.yml).

#### 1.3.6 Memory constraints
No stated memory limit. Each login runs Argon2id with 64 MiB of memory (SV-NFR-001).

#### 1.3.7 Operations
One operator runs the stack (`docker compose up -d` in `retrocast/`). Migrations run at
server start. A host scheduler runs `retrocast/scripts/backup.sh` daily and archives older
than 30 days are deleted (SYS-FR-002, SYS-FR-003); `restore.sh` restores one backup
(SYS-FR-004), and a restore drill is recorded before each release (SYS-NFR-006). Availability
is best effort and there are no support hours: a private group (Q-001, Q-030).

#### 1.3.8 Site adaptation requirements
The operator sets secrets and endpoints in `retrocast/.env` (template `.env.example`) or
`retrocast.toml`, `DOMAIN` for Caddy, `REGISTRATION_MODE` (default `invite`) and
`ALLOWED_ORIGINS` (DATA_CONTRACTS Â§1, ARCHITECTURE Â§7.4).

### 1.4 Product functions
- Accounts and sessions: SV-FR-001â€“SV-FR-006, SV-FR-060, SV-FR-061, WEB-FR-001â€“WEB-FR-003, WEB-FR-019, WEB-FR-024, IOS-FR-001â€“IOS-FR-002
- Registration mode: SV-FR-062â€“SV-FR-065, WEB-FR-020
- Servers, channels, invites and membership: SV-FR-007â€“SV-FR-024, WEB-FR-005, WEB-FR-006, WEB-FR-012, WEB-FR-014, IOS-FR-004, IOS-FR-007
- Text chat, typing and presence: SV-FR-025â€“SV-FR-038, SV-FR-071, SV-FR-072, WEB-FR-004, WEB-FR-007â€“WEB-FR-011, IOS-FR-003, IOS-FR-005, IOS-FR-006, IOS-FR-012
- Roles and permissions: SV-FR-040â€“SV-FR-047, WEB-FR-013, IOS-FR-008; grant limits: SV-FR-078 (CAP-014).
- Moderation (kick and ban): SV-FR-066â€“SV-FR-070, WEB-FR-021â€“WEB-FR-023
- Operator health, metrics, backups and CLI: SV-NFR-005, SV-NFR-012, SV-NFR-013, SV-FR-073, SYS-FR-001â€“SYS-FR-004, SYS-NFR-005, SYS-NFR-006
- Voice (Later): SV-FR-048â€“SV-FR-050, IOS-FR-011
- Attachments: multipart upload/linking and public signed downloads built (SV-FR-052, D-10 fixed); upload presigning remains Later (SV-FR-051), and file-only sending remains partial (WEB-FR-017).
- Profiles (Later): SV-FR-053, WEB-FR-015
- Direct messages (Later): SV-FR-054â€“SV-FR-058, WEB-FR-016, IOS-FR-009
- Search (Later): SV-FR-059, IOS-FR-010
- Message editing and deletion (Later): SV-FR-029â€“SV-FR-030, WEB-FR-018
- Reactions, unread state and nicknames (Later): SV-FR-074â€“SV-FR-077
- System constraints and qualities: SYS-CON-001â€“SYS-CON-004, SYS-NFR-001â€“SYS-NFR-007, WEB-NFR-001, WEB-NFR-002, IOS-NFR-001

### 1.5 User characteristics
| User class | Description | Technical expertise | Permissions |
|---|---|---|---|
| Member (STK-002) | A friend in the group; uses web (iOS later) | Everyday app user | What the server's roles grant; @everyone by default |
| Server owner (STK-003) | Created the server | Everyday app user | All permissions in that server; only one who can delete it |
| Server admin or moderator (STK-003) | Holds a role with management permissions | Everyday app user | The role's permission bits (DATA_CONTRACTS ENT-006); acts only on members ranked below them (SV-FR-070) |
| Operator (STK-004) | Runs the stack on their machine | Comfortable with Docker and a shell | Host, database, secrets and the registration mode; no in-app admin role exists |

### 1.6 Limitations
- One machine, no high availability, no uptime target, no off-site backups (SYS-CON-001,
  Q-030).
- Registration is invite-only by default; the operator may open or close it (SV-FR-062).
- The REST contract moves to v2.0.0 on the same `/api/v1` path with no parallel v1 shape
  (SV-IR-007, ADR-013); clients must update together with the server.
- Known defects that block several requirements are listed in ARCHITECTURE Â§9.

### 1.7 Assumptions and dependencies
- The group stays at 5â€“20 people (Q-001). Larger groups change SYS-NFR-001 and moderation.
- Web is the first client; iOS follows (Q-004). Web requirements on MVP capabilities are
  `Must`; iOS ones are `Should` and sit outside the Pilot.
- Phase 1 of SYSTEM_DESIGN.md plus kick and ban, operator health and backups is the MVP
  (Q-003).
- Docker Compose, PostgreSQL 17, Redis 7, MinIO and LiveKit images stay available.
- Existing guilds keep their stored `@everyone` permissions when SV-FR-008 is fixed (spec
  lead's assumption; nothing has been released).

### 1.8 Apportioning of requirements
| Phase | Scope (FR/NFR ids) | Exit criteria |
|---|---|---|
| Pilot (MVP: the owner's group; SYSTEM_DESIGN Phase 1 on the web client, Q-003) | SV-FR-078, SV-FR-001â€“SV-FR-016, SV-FR-018â€“SV-FR-028, SV-FR-031â€“SV-FR-038, SV-FR-040â€“SV-FR-046, SV-FR-060â€“SV-FR-073; SV-IR-001â€“SV-IR-005, SV-IR-007, SV-IR-008; SV-NFR-001â€“SV-NFR-013; WEB-FR-001â€“WEB-FR-014, WEB-FR-019â€“WEB-FR-024; WEB-NFR-001, WEB-NFR-002 (Should); SYS-FR-001â€“SYS-FR-004; SYS-CON-001â€“SYS-CON-004; SYS-NFR-001â€“SYS-NFR-007 | Every Must requirement in this row Verified; the ARCHITECTURE Â§9 defects marked "Blocks Pilot" closed (D-08 and D-09 storage paths are repaired by migration 000020; RISK-002 still requires the complete Phase 1 paths as Q-003 requires); one restore drill recorded (SYS-NFR-006) |
| Production | No separate tier: the product stays a self-hosted instance for one private group (Q-001, Q-002), so the Pilot exit is the release | â€” |
| Later â€” iOS client (follows the web, Q-004) | IOS-FR-001â€“IOS-FR-008, IOS-FR-012, IOS-NFR-001; the iOS update to REST contract v2.0.0 | Scheduled by the owner |
| Later â€” Phase 2, voice and media | CAP-003, CAP-017 (web voice, audio only), CAP-018, CAP-019: SV-FR-048â€“SV-FR-053, SV-IR-006, WEB-FR-015, WEB-FR-017, IOS-FR-011 | Scheduled by the owner |
| Later â€” Phase 3, polish | CAP-013, CAP-020â€“CAP-024 (Unicode emoji picker only), CAP-026: SV-FR-029, SV-FR-030, SV-FR-054â€“SV-FR-058, WEB-FR-016, WEB-FR-018, IOS-FR-009 | Scheduled by the owner |
| Later â€” Phase 4, nice to have | CAP-007, CAP-016, CAP-028, CAP-030: SV-FR-017, SV-FR-047, SV-FR-059, IOS-FR-010 | Scheduled by the owner |
| Later â€” not phased (Q-003, Q-022, Q-026) | CAP-032â€“CAP-034 (reactions, unread state, nicknames; built): SV-FR-074â€“SV-FR-077; CAP-038 (account deletion) and CAP-039 (ownership transfer): no requirements yet | Scheduled by the owner |
| Won't | CAP-025 (link previews), CAP-027 (LAN discovery), CAP-029 (desktop app), custom emoji (Q-027â€“Q-029); SV-FR-039 (Deprecated rev 2, superseded by SV-FR-072) | â€” |

Later capabilities with no requirements yet: CAP-018 (voice controls), CAP-022 (mentions),
CAP-023 (push, with the iOS client), CAP-024 (Unicode emoji picker), CAP-026 (iOS 7-era look),
CAP-030 (audit log), CAP-038 and CAP-039. Their requirements are written when the owner
schedules them.

**Pilot validation 2026-11-24:** [full candidate matrix](validation/pilot-2026-11-24.md), candidate `26ebe80`: FAIL; 59 Must criteria remain without acceptance-complete candidate proof. Restore drill passes; issues #64, #65, #66 and coverage #67 recorded.

## 2. References
- ISO/IEC/IEEE 29148:2018, Systems and software engineering â€” Life cycle processes â€”
  Requirements engineering.
- [docs/PRODUCT_BRIEF.md](PRODUCT_BRIEF.md) (Status: Specified (SRS rev 2), 2026-11-23).
- SYSTEM_DESIGN.md, README.md, TODO.md, PLAN.md at commit `6bc385e` (seed and plans).
- retrocast/docs/openapi.yaml (OpenAPI 3.0.3, `info.version` 2.0.0): REST operations.
- docs/legacy/ (api.md, architecture.md, database.md, deployment.md, gateway.md,
  ios-architecture.md, permissions.md): older prose docs, kept for history. What they said
  that the code proves is folded into DATA_CONTRACTS and ARCHITECTURE (Q-034).
- retrocast/migrations 000001â€“000019: database schema.
- OWASP Application Security Verification Standard 4.0.3 (reference for SV-NFR security
  checks).
- W3C Web Content Accessibility Guidelines 2.2 (WEB-NFR-001).

## 3. Requirements

Requirements are split by component.

| Component | File | Prefix | Owner |
|---|---|---|---|
| Server (REST API, gateway, CLI) | [srs/server.md](srs/server.md) | SV- | Project owner (STK-001) |
| Web client | [srs/web.md](srs/web.md) | WEB- | Project owner (STK-001) |
| iOS client | [srs/ios.md](srs/ios.md) | IOS- | Project owner (STK-001) |
| System (cross-cutting functions, constraints and qualities) | [srs/system.md](srs/system.md) | SYS- | Project owner (STK-001) |

## 4. Verification
- **Test (server):** Go unit tests with mock repositories (`internal/api`, `auth`,
  `gateway`, `permissions`, `snowflake`) and integration tests against PostgreSQL
  (`internal/database`, skipped without `DATABASE_URL`). Run in CI by
  `go test -v -race ./...` after migrations, with PostgreSQL 17 and Redis 7 services.
- **Test (web):** none exists. The web client has lint (`pnpm lint`) and type-check
  (`tsc -b` in `pnpm build`) only. Web FRs use `Demonstration` until a runner exists;
  WEB-NFR-001 needs axe-core, which needs a browser test runner.
- **Test (iOS):** none exists; no CI job. iOS FRs use `Demonstration` on a device or
  simulator.
- **Demonstration:** the V&V protocol against a fresh compose stack (no `VERIFICATION.md`
  yet); restore drills (SYS-FR-004, SYS-NFR-006).
- **Inspection:** configuration and documents (compose file, Caddyfile, licence, drill
  records).
- **Analysis:** load and latency (no harness exists; SYS-NFR-001, SYS-NFR-002, SYS-NFR-007).
Per-requirement methods are on each requirement; matrices are in each component file's Â§5.1.

## 5. Appendices

### 5.1 Traceability matrix
One matrix per component file: [server](srs/server.md#51-traceability-matrix),
[web](srs/web.md#51-traceability-matrix), [ios](srs/ios.md#51-traceability-matrix),
[system](srs/system.md#51-traceability-matrix).

### 5.2 Open issues (TBD)
Every TBD is resolved in rev 2. Each mirrored a question in the brief (Q-) or a capability
with no release; the answers are Victor Ivanov's, 2026-11-23, accepting all recommendations in
PR #25 (PRODUCT_BRIEF Â§14). The rows stay for traceability.

| TBD | Question | Assumption used (rev 1) | Affects | Owner | Resolution (rev 2) |
|---|---|---|---|---|---|
| TBD-001 | Audience and scale (Q-001) | Private group of 5â€“20 per instance | SYS-NFR-001, SYS-NFR-002, all moderation | Victor | Resolved rev 2: a private group of 5â€“20 per instance; SYS-NFR-001 is sized for 20 |
| TBD-002 | Self-hosted only, or also a hosted instance (Q-002) | Self-hosted only | SYS-CON-001, ARCHITECTURE Â§6 | Victor | Resolved rev 2: self-hosted only; a hosted instance is a non-goal |
| TBD-003 | MVP "done" and release tiers (Q-003) | MVP = SYSTEM_DESIGN Phase 1 | Â§1.8 | Victor | Resolved rev 2: Phase 1 end to end on the web client with RISK-001 and RISK-002 fixed, plus kick and ban, operator health and backups; no separate production tier (Â§1.8) |
| TBD-004 | Client priority (Q-004) | Web first (Must), iOS second (Should) | WEB-FR-*, IOS-FR-* priorities | Victor | Resolved rev 2: web first, iOS follows; iOS rows moved to "Later â€” iOS client" |
| TBD-005 | Public demo (Q-005) | None | ARCHITECTURE Â§6 | Victor | Resolved rev 2: no public demo; `scripts/demo.sh` seeds a local one |
| TBD-006 | Monetisation (Q-006) | None | Scope | Victor | Resolved rev 2: none (non-goal) |
| TBD-007 | Licence (Q-007) | Unresolved | SYS-CON-004 | Victor | Resolved rev 2: all rights reserved everywhere; the MIT line is removed from openapi.yaml (SYS-CON-004, SV-IR-008) |
| TBD-008 | PR #22 Rust core for permission resolution (Q-008) | Permission resolution stays in Go | SV-FR-045, ARCHITECTURE ADR-009 | Victor | Resolved rev 2: resolution stays in Go; PR #22 is decided after the authorisation defects are fixed (SYS-CON-003, ADR-009) |
| TBD-009 | One gateway connection per user, or several (Q-009) | One (current) | SV-FR-039 | Victor | Resolved rev 2: several connections per user; SV-FR-072 supersedes SV-FR-039 (ADR-017) |
| TBD-010 | One response shape for every endpoint (Q-010) | Mixed (current), documented per operation | SV-IR-001, WEB-FR-002, WEB-FR-003, WEB-FR-007, WEB-FR-012, WEB-FR-013 | Victor | Resolved rev 2: every success body wrapped in `data` as REST contract v2.0.0 on the same `/api/v1` path, no parallel old shape (SV-IR-007, SV-IR-008; ADR-013, the driver's choice within the owner's answer) |
| TBD-011 | How clients upload and fetch files (Q-011) | Through the API (current); URL scheme open | SV-FR-051, SV-FR-052 | Victor | Resolved rev 2: pre-signed upload and download URLs valid 15 minutes (SV-FR-051, SV-FR-052; ADR-019); B6 implements download; upload presigning remains Later |
| TBD-012 | Registration policy (Q-012) | Open registration (current) | SV-FR-001 | Victor | Resolved rev 2: operator-chosen `open`, `invite` or `closed`, default `invite` (SV-FR-062â€“SV-FR-065, WEB-FR-020; ADR-014) |
| TBD-013 | Who may DM whom (Q-013) | Anyone on the instance (current) | SV-FR-054, SV-FR-056 | Victor | Resolved rev 2: only users who share a guild (SV-FR-054, SV-FR-056, SV-FR-057); Later |
| TBD-014 | Default @everyone permissions (Q-014) | View, Send, Read history (current code) | SV-FR-008 | Victor | Resolved rev 2: `DefaultEveryonePerms` = 230147 (SV-FR-008; D-27) |
| TBD-015 | Trusted proxy and rate-limit keys (Q-015) | Limits as coded; client IP source open | SV-NFR-003 | Victor | Resolved rev 2: trust `X-Forwarded-For` only from the Caddy hop; per client IP on unauthenticated routes including the invite lookup, per user on the rest; Redis failure fails closed only on unauthenticated routes (SV-NFR-003, SV-NFR-009, SV-NFR-010; ADR-015) |
| TBD-016 | Allowed origins and published ports (Q-016) | Allow-all (current) | SV-NFR-008, ARCHITECTURE Â§6 | Victor | Resolved rev 2: same origin plus `ALLOWED_ORIGINS`; only Caddy's ports (and LiveKit's, for voice) published; Grafana on loopback; `/metrics` internal (SV-NFR-008, SV-NFR-011, SV-NFR-012; ADR-016) |
| TBD-017 | Upload size limit and types (Q-017) | 10 MB, listed types plus any `image/*` (current) | SV-FR-051 | Victor | Resolved rev 2: 10 MB; JPEG, PNG, GIF, WebP, PDF, plain text; no SVG (SV-FR-051); Later |
| TBD-018 | Moderation depth; audit log (Q-018) | Kick, ban, roles; audit log Later | CAP-030 | Victor | Resolved rev 2: kick, ban and role management (SV-FR-066â€“SV-FR-070); audit log Later (CAP-030) |
| TBD-019 | End-to-end encryption (Q-019) | Not planned | SYS-NFR-003 | Victor | Resolved rev 2: Won't; TLS in transit and a trusted operator |
| TBD-020 | Federation (Q-020) | Not planned | Scope | Victor | Resolved rev 2: Won't |
| TBD-021 | Retention of messages and files (Q-021) | Kept until deleted | DATA_CONTRACTS Â§8 | Victor | Resolved rev 2: kept until their author or a moderator deletes them; no automatic expiry |
| TBD-022 | Password change and account deletion (Q-022) | Not specified | CAP-001, DATA_CONTRACTS Â§8 | Victor | Resolved rev 2: password change in the MVP (SV-FR-060, SV-FR-061, WEB-FR-019); account deletion Later (CAP-038) |
| TBD-023 | Search language (Q-023) | English stemming (current) | SV-FR-059 | Victor | Resolved rev 2: English stemming (SV-FR-059) |
| TBD-024 | Web voice; audio only or video (Q-024) | Web voice Later, audio only | CAP-017 | Victor | Resolved rev 2: web voice in Phase 2, audio only (CAP-017) |
| TBD-025 | Push notifications and mentions (Q-025) | Later | CAP-022, CAP-023 | Victor | Resolved rev 2: Later, together with the iOS client |
| TBD-026 | Ownership transfer (Q-026) | Not specified; owner cannot leave | SV-FR-022 | Victor | Resolved rev 2: added as a Later capability (CAP-039); until then the owner cannot leave (SV-FR-022) |
| TBD-027 | Emoji picker, custom emoji, link previews (Q-027) | Later, as the seed says | CAP-024, CAP-025 | Victor | Resolved rev 2: Unicode emoji picker Later; custom emoji and link previews Won't |
| TBD-028 | Desktop app (Q-028) | Later, as the seed says | CAP-029 | Victor | Resolved rev 2: Won't; the web client covers desktop |
| TBD-029 | LAN discovery and LAN certificates (Q-029) | Later, as the seed says | CAP-027 | Victor | Resolved rev 2: Won't; Caddy with a domain or `localhost` covers TLS |
| TBD-030 | Availability and backup targets (Q-030) | Best effort; daily backups kept 30 days | SYS-NFR-005 | Victor | Resolved rev 2: best effort; daily backups kept 30 days; a restore drill before each release (SYS-FR-002â€“SYS-FR-004, SYS-NFR-005, SYS-NFR-006; ADR-018) |
| TBD-031 | Performance targets (Q-031) | None until set | SYS-NFR-001, SYS-NFR-002 | Victor | Resolved rev 2: delivery p95 â‰¤ 1 s and a 50-message history page p95 â‰¤ 300 ms at 20 connected members (SYS-NFR-001, SYS-NFR-002, SYS-NFR-007) |
| TBD-032 | Accessibility target and supported platforms (Q-032) | iOS 17 or later; web target unset | Â§1.3.2, web and iOS FRs | Victor | Resolved rev 2: WCAG 2.2 AA on the current and previous major versions of Chrome, Firefox, Safari and Edge; iOS 17 or later (WEB-NFR-001, WEB-NFR-002, IOS-NFR-001) |
| TBD-033 | Go module path (Q-033) | Keep `github.com/victorivanov/retrocast` | Tooling | Victor | Resolved rev 2: keep it (LIM-005, ADR-020) |
| TBD-034 | Older prose docs and the architecture file name (Q-034) | Keep both; spec file named ARCHITECTURE_SPEC.md | docs/ | Victor | Resolved rev 2: their content is folded into the spec set, the originals moved to docs/legacy/, and ARCHITECTURE_SPEC.md renamed ARCHITECTURE.md |
| TBD-035 | Release of kick and ban, CAP-031 (built: retrocast/internal/service/member.go:191, ban.go:43â€“112; API-038, API-052â€“API-054) | No requirement until set | Â§1.8 | Victor | Resolved rev 2: MVP (SV-FR-066â€“SV-FR-070, WEB-FR-021â€“WEB-FR-023) |
| TBD-036 | Release of reactions, CAP-032 (built: retrocast/internal/service/reaction.go; API-030â€“API-032) | No requirement until set | Â§1.8 | Victor | Resolved rev 2: Later (SV-FR-074, SV-FR-075) |
| TBD-037 | Release of unread tracking, CAP-033 (built: retrocast/internal/service/read_state.go; API-012, API-029) | No requirement until set | Â§1.8 | Victor | Resolved rev 2: Later (SV-FR-076) |
| TBD-038 | Release of nicknames, CAP-034 (built: retrocast/internal/service/member.go:87â€“190; API-034, API-037) | No requirement until set | Â§1.8 | Victor | Resolved rev 2: Later (SV-FR-077) |
| TBD-039 | Release of operator monitoring, CAP-035 (built: `/metrics`, Prometheus, Grafana provisioning) | No requirement until set; `/health` is covered by SV-NFR-005 | Â§1.8 | Victor | Resolved rev 2: MVP (SV-NFR-012, SV-NFR-013, SYS-FR-001) |
| TBD-040 | Release of backups, CLI and demo data, CAP-036 (built: retrocast/scripts/*.sh, cmd/retrocast-cli) | No requirement until set; recoverability itself is SYS-NFR-005 (RISK-004) | Â§1.8 | Victor | Resolved rev 2: MVP (SV-FR-073, SYS-FR-002â€“SYS-FR-004, SYS-NFR-006) |

No new TBD was opened in rev 2. Assumptions the spec lead made while writing the
requirements are marked "spec lead's assumption" where they are used and listed in
ARCHITECTURE Â§8.

### 5.3 Acronyms, abbreviations, and definitions
| Term | Definition |
|---|---|
| Guild | The API's name for a server (PRODUCT_BRIEF Â§7) |
| Gateway | The WebSocket endpoint `/gateway` that pushes events (EVT-) to clients |
| Opcode | The `op` field of a gateway frame (DATA_CONTRACTS Â§6) |
| Snowflake | 64-bit time-ordered id (DATA_CONTRACTS Â§3.3) |
| JWT | JSON Web Token; access tokens are HS256-signed |
| SFU | Selective forwarding unit; LiveKit relays voice |
| MoSCoW | Must, Should, Could, Won't priorities |
| TBD | An open issue in Â§5.2 |
| V&V | Verification and validation |
| WCAG | Web Content Accessibility Guidelines |
| RPO | Recovery point objective: the most data, in time, a restore may lose |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial SRS index from repository evidence (unattended INIT); TBD-001â€“TBD-040 |
| 2 | 2026-11-23T15:17:07-05:00 | 701009f47d68 | REVISION | Owner decisions Q-001â€“Q-034 recorded (accepted all recommendations, PR #25); status Baselined; TBD-001â€“TBD-040 resolved; Pilot scope and exit criteria; legacy docs moved |
| 3 | 2026-11-23T15:35:12-05:00 | 8cbf60de297e | REVISION | Review fixes (PR #25): Pilot exit includes D-08 and D-09 (RISK-002, Q-003) |
| 5 | 2026-11-23T16:25:23-05:00 | 7af05226f577 | REVISION | PR #44 round 2: SV-FR-078 grant limits, fixed default-role position, API notes and restored evidence |
| 6 | 2026-11-23T19:23:21-05:00 | f3f6d6f7bafd | REVISION | B2b: migration 000020 repairs DM parent integrity and pending attachment linking; Q-017 types and D-31 route binding fixed; storage/retention limits and Later delivery retained |
| 7 | 2026-11-23T23:28:58-05:00 | 2f78d18e5e62 | REVISION | PR #53 round 2: supported LiveKit UDP range; local demo TLS and standalone proxy; deployment drift and requirement provenance repaired; Redis deadline regression |
| 8 | 2026-11-24T09:13:39-05:00 | 0edca0aaeafa | REVISION | PR #62 round 2: missing/deleted guild status and permission-before-lock; GET-only downloads and lowercase host; manager create positioning and no-room guidance; contract, operational and D-10 drift repaired |
| 9 | 2026-11-24T11:27:20-05:00 | b98bf925608b | REVISION | L36 full candidate validation: all 114 Pilot ids mapped, verified Delivery with evidence, explicit remaining gaps and successful candidate restore drill |
