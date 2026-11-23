# Software Requirements Specification — Retrocast

Conforms to the SRS content outline of ISO/IEC/IEEE 29148:2018 §9.6.
Related: [Product Brief](PRODUCT_BRIEF.md) · [Data Contracts](DATA_CONTRACTS.md) · [Architecture](ARCHITECTURE_SPEC.md)
Status: Draft

> **Draft for owner review.** This set was written unattended from repository evidence
> (code, tests, migrations, OpenAPI, seed documents). No owner interview was held. Each
> requirement for existing behaviour cites its evidence; `Delivery` comes from reading the
> code and listing the tests, not from running the product. Open decisions are TBDs in §5.2.
> The architecture document is `ARCHITECTURE_SPEC.md` because `docs/architecture.md`
> (older prose) has the same name on case-insensitive file systems (TBD-034).

This is the index of a multi-component SRS. Requirements live in the component files listed
in §3; each id is defined in exactly one file.

## 1. Introduction

### 1.1 Purpose
This document specifies what Retrocast shall do: the server (REST API and real-time
gateway), the web client, the iOS client, and the system-wide constraints. It is read by the
project owner (to settle the TBDs), by `/groom` (to cut tickets) and by `release-readiness`
(to judge milestones).

### 1.2 Scope
Retrocast is a self-hosted chat product in the style of Discord in 2015–2016: servers
(guilds), text channels, voice channels, roles and permissions, invites, presence and
typing, for a group of 5–20 friends on one machine (PRODUCT_BRIEF GOAL-001–GOAL-004). It
will not include threads, bots, video, subscriptions, boosting or server discovery
(PRODUCT_BRIEF §9). Success measures are TBD-031.

### 1.3 Product perspective
A standalone system the operator runs as one Docker Compose stack: the Go server, PostgreSQL,
Redis, MinIO, LiveKit, the web client served by nginx, Caddy in front, plus Prometheus and
Grafana. Context diagram: [ARCHITECTURE_SPEC.md §3](ARCHITECTURE_SPEC.md#3-system-context-c4-level-1).

#### 1.3.1 System interfaces
Clients are meant to reach the server only through Caddy: REST under `/api/v1`, the gateway
at `/gateway` (WebSocket) and `/health` (compose also publishes the API on 8080, SV-NFR-008). The server calls PostgreSQL, Redis and MinIO, and
mints LiveKit access tokens that clients present to LiveKit (DATA_CONTRACTS §7).

#### 1.3.2 User interfaces
- **web:** browser single-page app (retrocast-web): server address, login and register
  pages; three-column app shell with server list, channel sidebar, chat area and member
  list; modals for servers, channels, invites, roles, guild settings and user settings.
- **ios:** SwiftUI app (retrocast-client): the same flows in a `NavigationSplitView`, plus
  search and voice channel views.
- Accessibility target and supported browsers: TBD-032.

#### 1.3.3 Hardware interfaces
No hardware interfaces beyond the operator's host and the clients' microphones and speakers
for voice (Later).

#### 1.3.4 Software interfaces
REST operations API-001–API-058, gateway frames and events EVT-001–EVT-030, LiveKit token contract and
object storage contract (DATA_CONTRACTS §4, §6, §7).

#### 1.3.5 Communications interfaces
HTTPS and WSS through Caddy (automatic TLS for the configured `DOMAIN`, else `localhost`).
LiveKit signalling on 7880/7881 and media on UDP 50000–50100 (retrocast/docker-compose.yml).

#### 1.3.6 Memory constraints
No stated memory limit. Each login runs Argon2id with 64 MiB of memory (SV-NFR-001).

#### 1.3.7 Operations
One operator runs the stack (`docker compose up -d` in `retrocast/`). Migrations run at
server start. Backups and restores run from `retrocast/scripts/backup.sh` and
`restore.sh`. No support hours: a private group (TBD-001, TBD-030).

#### 1.3.8 Site adaptation requirements
The operator sets secrets and endpoints in `retrocast/.env` (template `.env.example`) or
`retrocast.toml`, and `DOMAIN` for Caddy (DATA_CONTRACTS §1, ARCHITECTURE_SPEC §7.4).

### 1.4 Product functions
- Accounts and sessions: SV-FR-001–SV-FR-006, WEB-FR-001–WEB-FR-003, IOS-FR-001–IOS-FR-002
- Servers, channels, invites and membership: SV-FR-007–SV-FR-024, WEB-FR-005, WEB-FR-006, WEB-FR-012, WEB-FR-014, IOS-FR-004, IOS-FR-007
- Text chat, typing and presence: SV-FR-025–SV-FR-039, WEB-FR-004, WEB-FR-007–WEB-FR-011, IOS-FR-003, IOS-FR-005, IOS-FR-006, IOS-FR-012
- Roles and permissions: SV-FR-040–SV-FR-047, WEB-FR-013, IOS-FR-008
- Voice (Later): SV-FR-048–SV-FR-050, IOS-FR-011
- Attachments (Later): SV-FR-051–SV-FR-052, WEB-FR-017
- Profiles (Later): SV-FR-053, WEB-FR-015
- Direct messages (Later): SV-FR-054–SV-FR-058, WEB-FR-016, IOS-FR-009
- Search (Later): SV-FR-059, IOS-FR-010
- Message editing and deletion (Later): SV-FR-029–SV-FR-030, WEB-FR-018
- System constraints and qualities: SYS-CON-001–SYS-CON-004, SYS-NFR-001–SYS-NFR-005

### 1.5 User characteristics
| User class | Description | Technical expertise | Permissions |
|---|---|---|---|
| Member (STK-002) | A friend in the group; uses web or iOS | Everyday app user | What the server's roles grant; @everyone by default |
| Server owner (STK-003) | Created the server | Everyday app user | All permissions in that server; only one who can delete it |
| Server admin or moderator (STK-003) | Holds a role with management permissions | Everyday app user | The role's permission bits (DATA_CONTRACTS ENT-006) |
| Operator (STK-004) | Runs the stack on their machine | Comfortable with Docker and a shell | Host, database and secrets; no in-app admin role exists |

### 1.6 Limitations
- One machine, no high availability (SYS-CON-001, TBD-030).
- Registration is open to anyone who can reach the server (TBD-012). The seed assumed
  "local network = trusted" (SYSTEM_DESIGN.md:232).
- Licence statements conflict (SYS-CON-004, TBD-007).
- Known defects that block several requirements are listed in ARCHITECTURE_SPEC §9.

### 1.7 Assumptions and dependencies
- The group stays at 5–20 people (TBD-001). Larger groups change SYS-NFR-001 and moderation.
- Web is the first client; iOS follows (TBD-004). Web requirements on MVP capabilities are
  `Must`; iOS ones are `Should`.
- Phase 1 of SYSTEM_DESIGN.md is the MVP (TBD-003).
- Docker Compose, PostgreSQL 17, Redis 7, MinIO and LiveKit images stay available.
- The response shape stays mixed until TBD-010 is answered; requirements describe the
  current contract.

### 1.8 Apportioning of requirements
| Phase | Scope (FR/NFR ids) | Exit criteria |
|---|---|---|
| Pilot (MVP: the owner's group; SYSTEM_DESIGN Phase 1) | SV-FR-001–SV-FR-016, SV-FR-018–SV-FR-028, SV-FR-031–SV-FR-046; SV-IR-001–SV-IR-005; SV-NFR-001–SV-NFR-008; SYS-CON-001–SYS-CON-004; SYS-NFR-001–SYS-NFR-005; WEB-FR-001–WEB-FR-014; IOS-FR-001–IOS-FR-008, IOS-FR-012 (Should) | Every Must requirement Verified; ARCHITECTURE_SPEC §9 defects marked Pilot closed; TBD-003 answered |
| Production | TBD-003 (no production tier stated) | TBD-003 |
| Later — Phase 2, voice and media | CAP-003, CAP-017, CAP-018, CAP-019: SV-FR-048–SV-FR-053, SV-IR-006, WEB-FR-015, WEB-FR-017, IOS-FR-011 | Scheduled by the owner |
| Later — Phase 3, polish | CAP-013, CAP-020–CAP-026: SV-FR-029, SV-FR-030, SV-FR-054–SV-FR-058, WEB-FR-016, WEB-FR-018, IOS-FR-009 | Scheduled by the owner |
| Later — Phase 4, nice to have | CAP-007, CAP-016, CAP-027–CAP-030: SV-FR-017, SV-FR-047, SV-FR-059, IOS-FR-010 | Scheduled by the owner |
| Release not set | CAP-031–CAP-036 (kick and ban, reactions, unread, nicknames, operator monitoring, backups and CLI): no requirements until TBD-035–TBD-040 are answered | — |

Later capabilities with no code and no requirements yet: CAP-018 (voice controls), CAP-022
(mentions), CAP-023 (push), CAP-024 (emoji picker, custom emoji), CAP-025 (link previews),
CAP-026 (iOS 7-era look), CAP-027 (LAN discovery), CAP-029 (desktop app), CAP-030 (audit log).
Their requirements are written when the owner schedules them.

## 2. References
- ISO/IEC/IEEE 29148:2018, Systems and software engineering — Life cycle processes —
  Requirements engineering.
- [docs/PRODUCT_BRIEF.md](PRODUCT_BRIEF.md) (Status: Gathering, 2026-11-23).
- SYSTEM_DESIGN.md, README.md, TODO.md, PLAN.md at commit `6bc385e` (seed and plans).
- retrocast/docs/openapi.yaml (OpenAPI 3.0.3, info.version 1.0.0): REST operations.
- docs/api.md, docs/gateway.md, docs/permissions.md, docs/database.md, docs/deployment.md
  (older prose docs; where they disagree with the code, the code is cited and the
  disagreement is noted).
- retrocast/migrations 000001–000019: database schema.
- OWASP Application Security Verification Standard 4.0.3 (reference for SV-NFR security
  checks).

## 3. Requirements

Requirements are split by component.

| Component | File | Prefix | Owner |
|---|---|---|---|
| Server (REST API, gateway, CLI) | [srs/server.md](srs/server.md) | SV- | Project owner (STK-001) |
| Web client | [srs/web.md](srs/web.md) | WEB- | Project owner (STK-001) |
| iOS client | [srs/ios.md](srs/ios.md) | IOS- | Project owner (STK-001) |
| System (cross-cutting constraints and qualities) | [srs/system.md](srs/system.md) | SYS- | Project owner (STK-001) |

## 4. Verification
- **Test (server):** Go unit tests with mock repositories (`internal/api`, `auth`,
  `gateway`, `permissions`, `snowflake`; 240 tests) and integration tests against PostgreSQL
  (`internal/database`; 69 tests, skipped without `DATABASE_URL`). Run in CI by
  `go test -v -race ./...` after migrations, with PostgreSQL 17 and Redis 7 services.
- **Test (web):** none exists. The web client has lint (`pnpm lint`) and type-check
  (`tsc -b` in `pnpm build`) only. Web FRs use `Demonstration` until a runner exists.
- **Test (iOS):** none exists; no CI job. iOS FRs use `Demonstration` on a device or
  simulator.
- **Demonstration:** the V&V protocol against a fresh compose stack (no `VERIFICATION.md`
  yet).
- **Inspection:** configuration and documents (compose file, Caddyfile, licence).
- **Analysis:** load and latency (no harness exists; SYS-NFR-001, SYS-NFR-002).
Per-requirement methods are on each requirement; matrices are in each component file's §5.1.

## 5. Appendices

### 5.1 Traceability matrix
One matrix per component file: [server](srs/server.md#51-traceability-matrix),
[web](srs/web.md#51-traceability-matrix), [ios](srs/ios.md#51-traceability-matrix),
[system](srs/system.md#51-traceability-matrix).

### 5.2 Open issues (TBD)
Each TBD mirrors an open question in the brief (Q-) or a capability with no release.
Options and evidence are in PRODUCT_BRIEF §14.

| TBD | Question | Assumption used | Affects | Owner |
|---|---|---|---|---|
| TBD-001 | Audience and scale (Q-001) | Private group of 5–20 per instance | SYS-NFR-001, SYS-NFR-002, all moderation | Victor |
| TBD-002 | Self-hosted only, or also a hosted instance (Q-002) | Self-hosted only | SYS-CON-001, ARCHITECTURE_SPEC §6 | Victor |
| TBD-003 | MVP "done" and release tiers (Q-003) | MVP = SYSTEM_DESIGN Phase 1 | §1.8 | Victor |
| TBD-004 | Client priority (Q-004) | Web first (Must), iOS second (Should) | WEB-FR-*, IOS-FR-* priorities | Victor |
| TBD-005 | Public demo (Q-005) | None | ARCHITECTURE_SPEC §6 | Victor |
| TBD-006 | Monetisation (Q-006) | None | Scope | Victor |
| TBD-007 | Licence (Q-007) | Unresolved | SYS-CON-004 | Victor |
| TBD-008 | PR #22 Rust core for permission resolution (Q-008) | Permission resolution stays in Go | SV-FR-045, ARCHITECTURE_SPEC ADR-009 | Victor |
| TBD-009 | One gateway connection per user, or several (Q-009) | One (current) | SV-FR-039 | Victor |
| TBD-010 | One response shape for every endpoint (Q-010) | Mixed (current), documented per operation | SV-IR-001, WEB-FR-002, WEB-FR-003, WEB-FR-007, WEB-FR-012, WEB-FR-013 | Victor |
| TBD-011 | How clients upload and fetch files (Q-011) | Through the API (current); URL scheme open | SV-FR-051, SV-FR-052 | Victor |
| TBD-012 | Registration policy (Q-012) | Open registration (current) | SV-FR-001 | Victor |
| TBD-013 | Who may DM whom (Q-013) | Anyone on the instance (current) | SV-FR-054, SV-FR-056 | Victor |
| TBD-014 | Default @everyone permissions (Q-014) | View, Send, Read history (current code) | SV-FR-008 | Victor |
| TBD-015 | Trusted proxy and rate-limit keys (Q-015) | Limits as coded; client IP source open | SV-NFR-003 | Victor |
| TBD-016 | Allowed origins and published ports (Q-016) | Allow-all (current) | SV-NFR-008, ARCHITECTURE_SPEC §6 | Victor |
| TBD-017 | Upload size limit and types (Q-017) | 10 MB, listed types plus any `image/*` (current) | SV-FR-051 | Victor |
| TBD-018 | Moderation depth; audit log (Q-018) | Kick, ban, roles; audit log Later | CAP-030 | Victor |
| TBD-019 | End-to-end encryption (Q-019) | Not planned | SYS-NFR-003 | Victor |
| TBD-020 | Federation (Q-020) | Not planned | Scope | Victor |
| TBD-021 | Retention of messages and files (Q-021) | Kept until deleted | DATA_CONTRACTS §8 | Victor |
| TBD-022 | Password change and account deletion (Q-022) | Not specified | CAP-001, DATA_CONTRACTS §8 | Victor |
| TBD-023 | Search language (Q-023) | English stemming (current) | SV-FR-059 | Victor |
| TBD-024 | Web voice; audio only or video (Q-024) | Web voice Later, audio only | CAP-017 | Victor |
| TBD-025 | Push notifications and mentions (Q-025) | Later | CAP-022, CAP-023 | Victor |
| TBD-026 | Ownership transfer (Q-026) | Not specified; owner cannot leave | SV-FR-022 | Victor |
| TBD-027 | Emoji picker, custom emoji, link previews (Q-027) | Later, as the seed says | CAP-024, CAP-025 | Victor |
| TBD-028 | Desktop app (Q-028) | Later, as the seed says | CAP-029 | Victor |
| TBD-029 | LAN discovery and LAN certificates (Q-029) | Later, as the seed says | CAP-027 | Victor |
| TBD-030 | Availability and backup targets (Q-030) | Best effort; daily backups kept 30 days | SYS-NFR-005 | Victor |
| TBD-031 | Performance targets (Q-031) | None until set | SYS-NFR-001, SYS-NFR-002 | Victor |
| TBD-032 | Accessibility target and supported platforms (Q-032) | iOS 17 or later; web target unset | §1.3.2, web and iOS FRs | Victor |
| TBD-033 | Go module path (Q-033) | Keep `github.com/victorivanov/retrocast` | Tooling | Victor |
| TBD-034 | Older prose docs and the architecture file name (Q-034) | Keep both; spec file named ARCHITECTURE_SPEC.md | docs/ | Victor |
| TBD-035 | Release of kick and ban, CAP-031 (built: retrocast/internal/service/member.go:191, ban.go:43–112; API-038, API-052–API-054) | No requirement until set | §1.8 | Victor |
| TBD-036 | Release of reactions, CAP-032 (built: retrocast/internal/service/reaction.go; API-030–API-032) | No requirement until set | §1.8 | Victor |
| TBD-037 | Release of unread tracking, CAP-033 (built: retrocast/internal/service/read_state.go; API-012, API-029) | No requirement until set | §1.8 | Victor |
| TBD-038 | Release of nicknames, CAP-034 (built: retrocast/internal/service/member.go:87–190; API-034, API-037) | No requirement until set | §1.8 | Victor |
| TBD-039 | Release of operator monitoring, CAP-035 (built: `/metrics`, Prometheus, Grafana provisioning) | No requirement until set; `/health` is covered by SV-NFR-005 | §1.8 | Victor |
| TBD-040 | Release of backups, CLI and demo data, CAP-036 (built: retrocast/scripts/*.sh, cmd/retrocast-cli) | No requirement until set; recoverability itself is SYS-NFR-005 (RISK-004) | §1.8 | Victor |

### 5.3 Acronyms, abbreviations, and definitions
| Term | Definition |
|---|---|
| Guild | The API's name for a server (PRODUCT_BRIEF §7) |
| Gateway | The WebSocket endpoint `/gateway` that pushes events (EVT-) to clients |
| Opcode | The `op` field of a gateway frame (DATA_CONTRACTS §6) |
| Snowflake | 64-bit time-ordered id (worker 1, process 1) |
| JWT | JSON Web Token; access tokens are HS256-signed |
| SFU | Selective forwarding unit; LiveKit relays voice |
| MoSCoW | Must, Should, Could, Won't priorities |
| TBD | An open issue in §5.2 |
| V&V | Verification and validation |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
