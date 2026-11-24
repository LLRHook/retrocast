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
| SYS-FR-001 | Must | The system shall scrape the server's metrics every 15 seconds and show them on a provisioned dashboard. | Given a fresh compose stack, when the operator opens Grafana on `127.0.0.1:3000` and signs in, then the "Retrocast API" dashboard shows request rate by status code, latency percentiles (p50, p95, p99), error rate and request rate by handler without manual setup. | Demonstration | Implemented | none (operator dashboard: Grafana "Retrocast API") | CMP-011, SV-NFR-013 | CAP-035 | B5c: internal 15s scrape is up; provisioned dashboard datasource queries return live data; handler panel groups by the exported url label. See docs/operations/performance.md. |
| SYS-FR-002 | Must | The system shall back up the database and the stored files once a day without operator action after setup. | Given the documented host schedule installed, when 24 hours pass, then a new `backup_<timestamp>.sql.gz` and `minio_<timestamp>.tar.gz` exist in `BACKUP_DIR`. | Demonstration | Implemented | none (operator: host cron or launchd) | CMP-012, CMP-004, CMP-006 | CAP-036, RISK-004 | B5c: scripts/scheduled-backup.sh, cron command and launchd plist in docs/OPERATIONS.md; exact wrapper executed successfully on Windows Git Bash. Host schedule installation remains an operator setup step. |
| SYS-FR-003 | Must | The system shall delete backup archives older than 30 days when a backup runs. | Given archives dated 31 and 29 days ago in `BACKUP_DIR`, when backup.sh runs, then the 31-day-old ones are gone and the 29-day-old ones remain. | Test | Verified | none (operator script) | CMP-012 | CAP-036, RISK-004 | B5c: both 31-day-old archive types removed and both 29-day-old types retained in live run and scripts/test-operations.sh. |
| SYS-FR-004 | Must | The system shall restore the database and stored files from one backup into a freshly started stack. | Given a backup taken from a stack with messages and attachments, when the operator starts a new stack with empty volumes and runs restore.sh with that backup, then the messages and files are back and `/health` returns 200. | Demonstration | Verified | none (operator script) | CMP-012, CMP-004, CMP-006 | CAP-036, RISK-004 | B5c: API stopped, migrated target database replaced, ON_ERROR_STOP restore, MinIO archive helper, restart and health check. Matching counts and authenticated object download in docs/operations/restore-drills.md; D-10 client URL limitation remains. |

### 3.2 Performance requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-001 | Must | Capacity: the system shall serve 20 members connected to the gateway at once on one host. Measure: messages lost under a 10-minute load of 20 clients, each sending one message every 10 seconds (target 0, with SYS-NFR-002 holding). | Given the compose stack on the operator's machine and 20 connected clients each sending one message every 10 s, when the load runs for 10 minutes, then every message reaches every other client and SYS-NFR-002 holds. | Analysis | Verified | none (infra) | CMP-001, CMP-002, CMP-004, CMP-005 | GOAL-001 | B5c: docs/operations/performance.md, fixed API fdd8fa0; 20 clients for 600s, 1200 sends, 22800/22800 peer receipts, loss 0, load p95 8.706ms. Windows 11 / Docker Desktop. #58 read-deadline fix; no rate limits raised. |
| SYS-NFR-002 | Must | Latency: the system shall deliver a sent message to the other connected members of its channel within 1 second at the 95th percentile. Measure: send-to-receive time with 20 members connected (p95 ≤ 1 s). | Given 20 connected members on the operator's machine, when one sends 100 messages, then the 95th-percentile time from send to receipt at each other member is at most 1 s. | Analysis | Verified | none (infra) | API-022, EVT-009, CMP-002 | GOAL-001 | B5c: docs/operations/performance.md; 100 messages from one sender with 20 gateways, 1900/1900 receipts, p95 3.882ms; concurrent-load p95 8.706ms. Unfixed run retained separately; #58 fixed. |
| SYS-NFR-007 | Must | Latency: the system shall return a page of 50 history messages within 300 ms at the 95th percentile. Measure: API-023 response time with 20 members connected (p95 ≤ 300 ms). | Given a channel with 1 000 messages and 20 connected members, when API-023 is called 100 times with `limit=50` at random cursors, then the 95th-percentile response time is at most 300 ms. | Analysis | Verified | none (infra) | API-023, ENT-008, CMP-001, CMP-004 | GOAL-001 | B5c: docs/operations/performance.md; 100 random-cursor API-023 calls, 50 rows each, 1000 seeded messages and 20 READY members, p95 1.571ms. Through real Caddy on Windows 11 / Docker Desktop. |

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
| SYS-CON-001 | Must | The system shall run as one Docker Compose stack on a single Mac or Linux machine with no cloud service. | Given a machine with Docker, when the operator runs `docker compose up -d` in `retrocast/` with a filled `.env`, then every service starts on that machine and stays running. | Inspection | Implemented | none (infra) | CMP-001–CMP-012 | LIM-001 | retrocast/docker-compose.yml (`LIVEKIT_KEYS`); docs/legacy/deployment.md. Changed: rev 2 — "stays running" added; self-hosted only (Q-002, TBD-002 resolved). PR #53 round 2: valid LiveKit keys, UDP range 50000-50100 confirmed in startup configuration and listeners; zero restarts beyond two minutes. |
| SYS-CON-002 | Must | The system shall need no third-party account and no email address to register or use it. | Given the register form, when a user submits only a username and password, then the account is created. | Test | Implemented | web — Register page; ios — RegisterView | API-001 | LIM-002 | retrocast/internal/service/auth.go:54 |
| SYS-CON-003 | Must | The system shall keep the existing stack (one Go binary for REST and the gateway; PostgreSQL, Redis, MinIO, LiveKit; Caddy; a React web client; a SwiftUI iOS client). | Given a change proposal, when it replaces one of these parts, then it carries an ADR in ARCHITECTURE §8. | Inspection | Implemented | none (infra) | CMP-001–CMP-012 | LIM-003, LIM-005 | ARCHITECTURE §4.1. Permission resolution stays in Go; PR #22 (Rust core) is decided after the authorisation defects are fixed (Q-008, TBD-008 resolved; ADR-009). The Go module path stays (Q-033, ADR-020). |
| SYS-CON-004 | Must | The repository shall state one licence consistently. | Given README.md and retrocast/docs/openapi.yaml, when they are compared, then both say all rights reserved and neither names an open-source licence. | Inspection | Implemented | none (repository) | — | LIM-004 | README.md; retrocast/docs/openapi.yaml (`info.license` "Proprietary. All rights reserved." in rev 2). Changed: rev 2 — Q-007 (TBD-007 resolved). |

### 3.7 Software system attributes

#### 3.7.1 Reliability
Recovery is covered by SYS-NFR-005.

#### 3.7.2 Availability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-005 | Must | Recoverability: the operator shall be able to restore the database and stored files from a backup with at most 24 hours of data lost. Measure: recovery point objective 24 h; no recovery time target (best effort). | Given a stack with messages and files, when the operator runs `scripts/backup.sh`, resets the volumes and runs `scripts/restore.sh`, then the messages and files return and only data newer than the last daily backup is lost. | Demonstration | Implemented | none (infra) | CMP-004, CMP-006, CMP-012 | RISK-004 | B5c: documented daily host schedules plus matching DB and file restore drill. RPO depends on installing and monitoring the daily job; no 24-hour wall-clock schedule test was run. |
| SYS-NFR-006 | Must | Recoverability: a restore drill shall be run and recorded before each release. Measure: one recorded drill per release (backup file name, date, row counts of users, guilds and messages before and after). | Given a release candidate, when the release is prepared, then a drill record for that release exists with matching row counts before and after the restore. | Inspection | Verified | none (operator procedure) | CMP-012 | CAP-036, RISK-004 | B5c: docs/operations/restore-drills.md records paired archive names, date, users/guilds/messages counts, health and stored attachment retrieval. Repeat before each release. |

#### 3.7.3 Security
Security requirements are on the server: SV-NFR-001–SV-NFR-003 and SV-NFR-008.

#### 3.7.4 Maintainability
See SV-NFR-007 (migrations). No formatter gate exists in CI (Stack Profile flag 6).

#### 3.7.5 Portability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-004 | Should | Portability: the stack shall run on macOS and on Linux. Measure: the compose stack starts and `/health` returns 200 on both. | Given a Mac and a Linux machine with Docker, when the stack is started on each, then `/health` returns 200. | Demonstration | Partial (CI builds images only on ubuntu-latest with no platform list; no run on macOS or on arm64 is recorded) | none (infra) | CMP-001–CMP-011 | LIM-001 | .github/workflows/ci.yml; retrocast/Dockerfile |

#### 3.7.6 Privacy and compliance

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-003 | Must | Privacy: no core flow shall call a service outside the operator's machine. Measure: outbound requests during core flows (sign-in, messaging, voice, file sharing) equal 0. | Given the stack on a machine without internet access, when a member signs in, sends a message and uploads a file, then each step succeeds. | Demonstration | Implemented | none (infra) | CMP-001–CMP-010 | PRN-001 | retrocast/internal/api/router.go; GOAL-004. PR #53: embedded raw OpenAPI at /docs; TestDocsOffline passes with container networking disabled. |

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
| SYS-FR-001 | Must | Implemented | — | `/metrics` | CMP-011 | Demonstration |
| SYS-FR-002 | Must | Implemented | ENT-001–ENT-018, ENT-022 | — | CMP-012, CMP-004, CMP-006 | Demonstration |
| SYS-FR-003 | Must | Verified | — | — | CMP-012 | Test |
| SYS-FR-004 | Must | Verified | ENT-001–ENT-018, ENT-022 | — | CMP-012, CMP-004, CMP-006 | Demonstration |
| SYS-NFR-001 | Must | Verified | — | EVT-009 | CMP-001, CMP-002, CMP-004, CMP-005 | Analysis |
| SYS-NFR-002 | Must | Verified | ENT-008 | API-022, EVT-009 | CMP-002 | Analysis |
| SYS-CON-001 | Must | Implemented | — | — | CMP-001–CMP-012 | Inspection |
| SYS-CON-002 | Must | Implemented | ENT-001 | API-001 | CMP-001 | Test |
| SYS-CON-003 | Must | Implemented | — | — | CMP-001–CMP-012 | Inspection |
| SYS-CON-004 | Must | Implemented | — | — | — | Inspection |
| SYS-NFR-003 | Must | Implemented | — | — | CMP-001–CMP-010 | Demonstration |
| SYS-NFR-004 | Should | Partial | — | — | CMP-001–CMP-011 | Demonstration |
| SYS-NFR-005 | Must | Implemented | ENT-001–ENT-018, ENT-022 | — | CMP-004, CMP-006, CMP-012 | Demonstration |
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
| 3 | 2026-11-24T05:50:46-05:00 | 591c4747c5e7 | REVISION | B5c: host schedules, MinIO archive helper, restore drill, live monitoring, gateway deadline fix #58 and measured capacity/latency |
