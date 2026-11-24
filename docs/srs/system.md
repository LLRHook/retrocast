# SRS component — System (constraints and qualities)

Part of [docs/SRS.md](../SRS.md). Prefix `SYS-`. Covers requirements that no single component
owns: hosting, external dependencies, licence, capacity, latency, portability and backups.
Components are in [ARCHITECTURE.md §5](../ARCHITECTURE.md#5-components-c4-level-2).

Conventions for this file:
- **Evidence** paths are relative to the repository root.
- **Delivery** comes from reading the repository at commit `6bc385e`, re-read at `origin/main`
  55e50eb in rev 2 and checked against the compose smoke run of 2026-11-23. B5c adds operator
  procedures and the recorded restore drill in
  docs/operations/restore-drills.md; measured capacity and latency are recorded in
  docs/operations/performance.md. Delivery describes the available evidence, not a
  promise that a host scheduler has already been installed.
- Rev 2 records the owner's answers Q-001, Q-030, Q-031 and Q-003 (operator health and
  backups are in the MVP).

## 3. Requirements

### 3.1 Functions
Client and server functions are in [server.md](server.md), [web.md](web.md) and
[ios.md](ios.md). The operator functions below cross components (CMP-011, CMP-012).

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-FR-001 | Must | The system shall scrape the server's metrics every 15 seconds and show them on a provisioned dashboard. | Given a fresh compose stack, when the operator opens Grafana on `127.0.0.1:3000` and signs in, then the "Retrocast API" dashboard shows request rate by status code, latency percentiles (p50, p95, p99), error rate and request rate by handler without manual setup. | Demonstration | Verified | none (operator dashboard: Grafana "Retrocast API") | CMP-011, SV-NFR-013 | CAP-035 | B5c: internal 15s scrape is up; provisioned dashboard datasource queries return live data; handler panel groups by the exported url label. See docs/operations/performance.md. L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-FR-002 | Must | The system shall back up the database and the stored files once a day without operator action after setup. | Given the documented host schedule installed, when 24 hours pass, then a new `backup_<timestamp>.sql.gz` and `minio_<timestamp>.tar.gz` exist in `BACKUP_DIR`. | Demonstration | Implemented | none (operator: host cron or launchd) | CMP-012, CMP-004, CMP-006 | CAP-036, RISK-004 |round2/crontab.txt exact documented line with only minute/hour and real project substitution; scheduled pair 20261124_214403, observed shell exit 0 in cron-backup.log, cron-integrity.txt PASS. Mechanism only: no operator host schedule installed and no 24 elapsed hours. Delivery Implemented; owner action.  L42 round 2: Not testable here; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-FR-003 | Must | The system shall delete backup archives older than 30 days when a backup runs. | Given archives dated 31 and 29 days ago in `BACKUP_DIR`, when backup.sh runs, then the 31-day-old ones are gone and the 29-day-old ones remain. | Test | Verified | none (operator script) | CMP-012 | CAP-036, RISK-004 | B5c round 2: independent fail-fast assertions cover both archive types at 29d, 30d23h and 31d; nested archives/non-archives are preserved. M06 and M06c are killed; round-1 hermetic retention assertions were insufficient (L31 F-1). L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-FR-004 | Must | The system shall restore the database and stored files from one backup into a freshly started stack. | Given a backup taken from a stack with messages and attachments, when the operator starts a new stack with empty volumes and runs restore.sh with that backup, then the messages and files are back and `/health` returns 200. | Demonstration | Verified | none (operator script) | CMP-012, CMP-004, CMP-006 | CAP-036, RISK-004 | B5c round 2: caller-relative paths resolve before cwd change; pg_dump framing, tar and compose helper preflight before interruption; single-transaction ON_ERROR_STOP restore, API creation and subsequent health wait. Matching counts and authenticated object download in docs/operations/restore-drills.md; D-10 was subsequently fixed by B6 public HTTPS signed GET URLs (PR #62). L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |

### 3.2 Performance requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-001 | Must | Capacity: the system shall serve 20 members connected to the gateway at once on one host. Measure: messages lost under a 10-minute load of 20 clients, each sending one message every 10 seconds (target 0, with SYS-NFR-002 holding). | Given the compose stack on the operator's machine and 20 connected clients each sending one message every 10 s, when the load runs for 10 minutes, then every message reaches every other client and SYS-NFR-002 holds. | Analysis | Verified | none (infra) | CMP-001, CMP-002, CMP-004, CMP-005 | GOAL-001 |Run 3 at 0f7ee55, CARRIED: lost 0; 22,800/22,800 receipts in 600 seconds; single-sender p95 4.912 ms; history p95 1.595 ms. performance-result.json/perf.log exit 0. round2/carry.txt proves all backend/build/deployment trees identical; only web/docs changed.  L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-NFR-002 | Must | Latency: the system shall deliver a sent message to the other connected members of its channel within 1 second at the 95th percentile. Measure: send-to-receive time with 20 members connected (p95 ≤ 1 s). | Given 20 connected members on the operator's machine, when one sends 100 messages, then the 95th-percentile time from send to receipt at each other member is at most 1 s. | Analysis | Verified | none (infra) | API-022, EVT-009, CMP-002 | GOAL-001 |Run 3 at 0f7ee55, CARRIED: lost 0; 22,800/22,800 receipts in 600 seconds; single-sender p95 4.912 ms; history p95 1.595 ms. performance-result.json/perf.log exit 0. round2/carry.txt proves all backend/build/deployment trees identical; only web/docs changed.  L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-NFR-007 | Must | Latency: the system shall return a page of 50 history messages within 300 ms at the 95th percentile. Measure: API-023 response time with 20 members connected (p95 ≤ 300 ms). | Given a channel with 1 000 messages and 20 connected members, when API-023 is called 100 times with `limit=50` at random cursors, then the 95th-percentile response time is at most 300 ms. | Analysis | Verified | none (infra) | API-023, ENT-008, CMP-001, CMP-004 | GOAL-001 |Run 3 at 0f7ee55, CARRIED: lost 0; 22,800/22,800 receipts in 600 seconds; single-sender p95 4.912 ms; history p95 1.595 ms. performance-result.json/perf.log exit 0. round2/carry.txt proves all backend/build/deployment trees identical; only web/docs changed.  L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |

### 3.3 Usability requirements
Usability and platform targets are on the clients: WEB-NFR-001, WEB-NFR-002 and IOS-NFR-001
(Q-032, TBD-032 resolved).

### 3.4 Interface requirements
External interfaces (LiveKit, MinIO, APNs) are in
[DATA_CONTRACTS.md §7](../DATA_CONTRACTS.md#7-external-contracts).

### 3.5 Logical database requirements
See [server.md §3.5](server.md#35-logical-database-requirements).

### 3.6 Design constraints

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-CON-001 | Must | The system shall run as one Docker Compose stack on a single Mac or Linux machine with no cloud service. | Given a machine with Docker, when the operator runs `docker compose up -d` in `retrocast/` with a filled `.env`, then every service starts on that machine and stays running. | Inspection | Verified | none (infra) | CMP-001–CMP-012 | LIM-001 | retrocast/docker-compose.yml (`LIVEKIT_KEYS`); docs/legacy/deployment.md. Changed: rev 2 — "stays running" added; self-hosted only (Q-002, TBD-002 resolved). PR #53 round 2: valid LiveKit keys, UDP range 50000-50100 confirmed in startup configuration and listeners; zero restarts beyond two minutes. L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-CON-002 | Must | The system shall need no third-party account and no email address to register or use it. | Given the register form, when a user submits only a username and password, then the account is created. | Test | Verified | web — Register page; ios — RegisterView | API-001 | LIM-002 | retrocast/internal/service/auth.go:54 L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-CON-003 | Must | The system shall keep the existing stack (one Go binary for REST and the gateway; PostgreSQL, Redis, MinIO, LiveKit; Caddy; a React web client; a SwiftUI iOS client). | Given a change proposal, when it replaces one of these parts, then it carries an ADR in ARCHITECTURE §8. | Inspection | Verified | none (infra) | CMP-001–CMP-012 | LIM-003, LIM-005 | ARCHITECTURE §4.1. Permission resolution stays in Go; PR #22 (Rust core) is decided after the authorisation defects are fixed (Q-008, TBD-008 resolved; ADR-009). The Go module path stays (Q-033, ADR-020). L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-CON-004 | Must | The repository shall state one licence consistently. | Given README.md and retrocast/docs/openapi.yaml, when they are compared, then both say all rights reserved and neither names an open-source licence. | Inspection | Verified | none (repository) | — | LIM-004 | README.md; retrocast/docs/openapi.yaml (`info.license` "Proprietary. All rights reserved." in rev 2). Changed: rev 2 — Q-007 (TBD-007 resolved). L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |

### 3.7 Software system attributes

#### 3.7.1 Reliability
Recovery is covered by SYS-NFR-005.

#### 3.7.2 Availability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-005 | Must | Recoverability: the operator shall be able to restore the database and stored files from a backup with at most 24 hours of data lost. Measure: recovery point objective 24 h; no recovery time target (best effort). | Given a stack with messages and files, when the operator runs `scripts/backup.sh`, resets the volumes and runs `scripts/restore.sh`, then the messages and files return and only data newer than the last daily backup is lost. | Demonstration | Verified | none (infra) | CMP-004, CMP-006, CMP-012 | RISK-004 | B5c: documented daily host schedules plus matching DB and file restore drill. RPO depends on installing and monitoring the daily job; no 24-hour wall-clock schedule test was run. L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |
| SYS-NFR-006 | Must | Recoverability: a restore drill shall be run and recorded before each release. Measure: one recorded drill per release (backup file name, date, row counts of users, guilds and messages before and after). | Given a release candidate, when the release is prepared, then a drill record for that release exists with matching row counts before and after the restore. | Inspection | Verified | none (operator procedure) | CMP-012 | CAP-036, RISK-004 | B5c: docs/operations/restore-drills.md records paired archive names, date, users/guilds/messages counts, health and stored attachment retrieval. Repeat before each release. L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |

#### 3.7.3 Security
Security requirements are on the server: SV-NFR-001–SV-NFR-003 and SV-NFR-008.

#### 3.7.4 Maintainability
See SV-NFR-007 (migrations). No formatter gate exists in CI (Stack Profile flag 6).

#### 3.7.5 Portability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-004 | Should | Portability: the stack shall run on macOS and on Linux. Measure: the compose stack starts and `/health` returns 200 on both. | Given a Mac and a Linux machine with Docker, when the stack is started on each, then `/health` returns 200. | Demonstration | Partial (CI builds images only on ubuntu-latest with no platform list; no run on macOS or on arm64 is recorded) | none (infra) | CMP-001–CMP-011 | LIM-001 | .github/workflows/ci.yml; retrocast/Dockerfile L42 round 2: Not testable here; [candidate evidence](../validation/pilot-2026-11-24.md). |

#### 3.7.6 Privacy and compliance

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-003 | Must | Privacy: no core flow shall call a service outside the operator's machine. Measure: outbound requests during core flows (sign-in, messaging, voice, file sharing) equal 0. | Given the stack on a machine without internet access, when a member signs in, sends a message and uploads a file, then each step succeeds. | Demonstration | Verified | none (infra) | CMP-001–CMP-010 | PRN-001 | retrocast/internal/api/router.go; GOAL-004. PR #53: embedded raw OpenAPI at /docs; TestDocsOffline passes with container networking disabled. L42 round 2: Verified; [candidate evidence](../validation/pilot-2026-11-24.md). |

Messages and files are kept until their author or a moderator deletes them (Q-021). Account
deletion is Later (CAP-038, Q-022). End-to-end encryption and federation are non-goals
(Q-019, Q-020).

#### 3.7.7 Observability
Server logging is SV-NFR-006; metrics are SV-NFR-013; scraping and dashboards are SYS-FR-001.

### 3.8 Supporting information
- The Stack Profile used for ARCHITECTURE §4.1 is generated locally and is not committed.
- Availability is best effort on one machine; no uptime target and no off-site backups
  (Q-030, PRODUCT_BRIEF §9).

## 5.1 Traceability matrix

| Req | Priority | Delivery | Entities | APIs / Events | Components | Verification |
|---|---|---|---|---|---|---|
| SYS-FR-001 | Must | Verified | — | `/metrics` | CMP-011 | Demonstration |
| SYS-FR-002 | Must | Implemented | ENT-001–ENT-018, ENT-022 | — | CMP-012, CMP-004, CMP-006 | Demonstration |
| SYS-FR-003 | Must | Verified | — | — | CMP-012 | Test |
| SYS-FR-004 | Must | Verified | ENT-001–ENT-018, ENT-022 | — | CMP-012, CMP-004, CMP-006 | Demonstration |
| SYS-NFR-001 | Must | Verified | — | EVT-009 | CMP-001, CMP-002, CMP-004, CMP-005 | Analysis |
| SYS-NFR-002 | Must | Verified | ENT-008 | API-022, EVT-009 | CMP-002 | Analysis |
| SYS-CON-001 | Must | Verified | — | — | CMP-001–CMP-012 | Inspection |
| SYS-CON-002 | Must | Verified | ENT-001 | API-001 | CMP-001 | Test |
| SYS-CON-003 | Must | Verified | — | — | CMP-001–CMP-012 | Inspection |
| SYS-CON-004 | Must | Verified | — | — | — | Inspection |
| SYS-NFR-003 | Must | Verified | — | — | CMP-001–CMP-010 | Demonstration |
| SYS-NFR-004 | Should | Partial | — | — | CMP-001–CMP-011 | Demonstration |
| SYS-NFR-005 | Must | Verified | ENT-001–ENT-018, ENT-022 | — | CMP-004, CMP-006, CMP-012 | Demonstration |
| SYS-NFR-006 | Must | Verified | — | — | CMP-012 | Inspection |
| SYS-NFR-007 | Must | Verified | ENT-008 | API-023 | CMP-001, CMP-004 | Analysis |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial system constraints and qualities SYS-CON-001–SYS-CON-004, SYS-NFR-001–SYS-NFR-005 |
| 2 | 2026-11-23T15:17:07-05:00 | 701009f47d68 | REVISION | Added SYS-FR-001–SYS-FR-004, SYS-NFR-006, SYS-NFR-007; performance, RPO and licence decisions; SYS-CON-001 Partial (D-25) |
| 3 | 2026-11-23T22:54:35-05:00 | f4e6409e92a2 | REVISION | PR #53: deployment exposure, origins, outage policy, invite limit, LiveKit defaults and raw OpenAPI; D-10 and API-059 remain separate |
| 4 | 2026-11-23T22:56:00-05:00 | 23fa99aabf34 | REVISION | Correct delivery columns, retain requirement statements and update PR #53 evidence |
| 5 | 2026-11-23T23:28:58-05:00 | 2f78d18e5e62 | REVISION | PR #53 round 2: supported LiveKit UDP range; local demo TLS and standalone proxy; deployment drift and requirement provenance repaired; Redis deadline regression |
| 6 | 2026-11-24T05:50:46-05:00 | 591c4747c5e7 | REVISION | B5c: host schedules, MinIO archive helper, restore drill, live monitoring, gateway deadline fix #58 and measured capacity/latency |
| 7 | 2026-11-24T07:02:55-05:00 | 0f5851ab8ae2 | REVISION | PR #59 round 2: fail-fast retention/restore regressions, complete archive and compose preflight, transactional restore and evidence corrections |
| 8 | 2026-11-24T09:13:39-05:00 | 0edca0aaeafa | REVISION | PR #62 round 2: missing/deleted guild status and permission-before-lock; GET-only downloads and lowercase host; manager create positioning and no-room guidance; contract, operational and D-10 drift repaired |
| 9 | 2026-11-24T11:27:20-05:00 | b98bf925608b | REVISION | L36 full candidate validation: all 114 Pilot ids mapped, verified Delivery with evidence, explicit remaining gaps and successful candidate restore drill |
| 10 | 2026-11-24T15:12:09-05:00 | 33dc349ad91a | REVISION | L42 round 1 candidate 0f7ee55: provisional all 107 Must claim, corrected by round 2 with acceptance-mapped CI/runtime evidence; accelerated cron, restore/RPO, offline flows and 10-minute performance; explicit Should and readiness gaps |
| 11 | 2026-11-24T16:49:53-05:00 | 3c40e51fe9a3 | REVISION | L42 round 2 candidate 511b8cd: 106 Musts Verified; SYS-FR-002 Implemented pending operator host schedule over 24 hours; exact cron line, deployed invite/channel/history/refresh probes, explicit backend carry and SRS readiness corrections |
