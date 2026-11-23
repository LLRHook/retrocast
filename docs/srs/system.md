# SRS component — System (constraints and qualities)

Part of [docs/SRS.md](../SRS.md). Prefix `SYS-`. Covers requirements that no single component
owns: hosting, external dependencies, licence, capacity, latency, portability and backups.
Components are in [ARCHITECTURE.md §5](../ARCHITECTURE.md#5-components-c4-level-2).

Conventions for this file:
- **Evidence** paths are relative to the repository root.
- **Delivery** comes from reading the repository at commit `6bc385e`, re-read at `origin/main`
  55e50eb in rev 2 and checked against the compose smoke run of 2026-11-23. No load, latency
  or restore measurement exists, so no quality target here is met yet.
- Rev 2 records the owner's answers Q-001, Q-030, Q-031 and Q-003 (operator health and
  backups are in the MVP).

## 3. Requirements

### 3.1 Functions
Client and server functions are in [server.md](server.md), [web.md](web.md) and
[ios.md](ios.md). The operator functions below cross components (CMP-011, CMP-012).

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-FR-001 | Must | The system shall scrape the server's metrics every 15 seconds and show them on a provisioned dashboard. | Given a fresh compose stack, when the operator opens Grafana on `127.0.0.1:3000` and signs in, then the "Retrocast API" dashboard shows request rate by status code, latency percentiles (p50, p95, p99), error rate and request rate by handler without manual setup. | Demonstration | Partial (scrape and dashboard are provisioned; Grafana is published on every interface, D-26) | none (operator dashboard: Grafana "Retrocast API") | CMP-011, SV-NFR-013 | CAP-035 | Added rev 2 (Q-003; TBD-039 resolved). retrocast/prometheus.yml (`scrape_interval: 15s`, target `api:8080`); retrocast/grafana/provisioning/; docker-compose.yml (`GRAFANA_ADMIN_PASSWORD`) |
| SYS-FR-002 | Must | The system shall back up the database and the stored files once a day without operator action after setup. | Given the documented host schedule installed, when 24 hours pass, then a new `backup_<timestamp>.sql.gz` and `minio_<timestamp>.tar.gz` exist in `BACKUP_DIR`. | Demonstration | Partial (retrocast/scripts/backup.sh makes both archives; nothing schedules it) | none (operator: host cron or launchd) | CMP-012, CMP-004, CMP-006 | CAP-036, RISK-004 | Added rev 2 (Q-030; TBD-040 resolved). ADR-018: a host cron (Linux) or launchd (macOS) job runs backup.sh; Redis is not backed up because it holds only sessions, presence, typing and counters. |
| SYS-FR-003 | Must | The system shall delete backup archives older than 30 days when a backup runs. | Given archives dated 31 and 29 days ago in `BACKUP_DIR`, when backup.sh runs, then the 31-day-old ones are gone and the 29-day-old ones remain. | Test | Implemented | none (operator script) | CMP-012 | CAP-036, RISK-004 | Added rev 2 (Q-030). retrocast/scripts/backup.sh:6 (`RETENTION_DAYS=30`), :24–25 (`find -mtime +30 -delete`) |
| SYS-FR-004 | Must | The system shall restore the database and stored files from one backup into a freshly started stack. | Given a backup taken from a stack with messages and attachments, when the operator starts a new stack with empty volumes and runs restore.sh with that backup, then the messages and files are back and `/health` returns 200. | Demonstration | Partial (retrocast/scripts/restore.sh restores both archives; no restore has been run or recorded) | none (operator script) | CMP-012, CMP-004, CMP-006 | CAP-036, RISK-004 | Added rev 2 (Q-030). retrocast/scripts/restore.sh |

### 3.2 Performance requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-001 | Must | Capacity: the system shall serve 20 members connected to the gateway at once on one host. Measure: messages lost under a 10-minute load of 20 clients, each sending one message every 10 seconds (target 0, with SYS-NFR-002 holding). | Given the compose stack on the operator's machine and 20 connected clients each sending one message every 10 s, when the load runs for 10 minutes, then every message reaches every other client and SYS-NFR-002 holds. | Analysis | Not started (no load harness exists; Stack Profile "Perf harness: none") | none (infra) | CMP-001, CMP-002, CMP-004, CMP-005 | GOAL-001 | SYSTEM_DESIGN.md:27 ("5–20 friends"). Changed: rev 2 — measure from Q-031 (TBD-031 resolved); the load profile (one message per client every 10 s) is the spec lead's assumption. |
| SYS-NFR-002 | Must | Latency: the system shall deliver a sent message to the other connected members of its channel within 1 second at the 95th percentile. Measure: send-to-receive time with 20 members connected (p95 ≤ 1 s). | Given 20 connected members on the operator's machine, when one sends 100 messages, then the 95th-percentile time from send to receipt at each other member is at most 1 s. | Analysis | Not started (no measurement exists) | none (infra) | API-022, EVT-009, CMP-002 | GOAL-001 | SYSTEM_DESIGN.md:63 ("Simple, fast, focused"). Changed: rev 2 — target from Q-031 (TBD-031 resolved). |
| SYS-NFR-007 | Must | Latency: the system shall return a page of 50 history messages within 300 ms at the 95th percentile. Measure: API-023 response time with 20 members connected (p95 ≤ 300 ms). | Given a channel with 1 000 messages and 20 connected members, when API-023 is called 100 times with `limit=50` at random cursors, then the 95th-percentile response time is at most 300 ms. | Analysis | Not started (no measurement exists) | none (infra) | API-023, ENT-008, CMP-001, CMP-004 | GOAL-001 | Added rev 2 (Q-031). |

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
| SYS-CON-001 | Must | The system shall run as one Docker Compose stack on a single Mac or Linux machine with no cloud service. | Given a machine with Docker, when the operator runs `docker compose up -d` in `retrocast/` with a filled `.env`, then every service starts on that machine and stays running. | Inspection | Partial (the compose smoke of 2026-11-23 saw LiveKit restart in a loop with the default key setting, D-25, issue #35) | none (infra) | CMP-001–CMP-012 | LIM-001 | retrocast/docker-compose.yml (`LIVEKIT_KEYS`); docs/legacy/deployment.md. Changed: rev 2 — "stays running" added; self-hosted only (Q-002, TBD-002 resolved). |
| SYS-CON-002 | Must | The system shall need no third-party account and no email address to register or use it. | Given the register form, when a user submits only a username and password, then the account is created. | Test | Implemented | web — Register page; ios — RegisterView | API-001 | LIM-002 | retrocast/internal/service/auth.go:54 |
| SYS-CON-003 | Must | The system shall keep the existing stack (one Go binary for REST and the gateway; PostgreSQL, Redis, MinIO, LiveKit; Caddy; a React web client; a SwiftUI iOS client). | Given a change proposal, when it replaces one of these parts, then it carries an ADR in ARCHITECTURE §8. | Inspection | Implemented | none (infra) | CMP-001–CMP-012 | LIM-003, LIM-005 | ARCHITECTURE §4.1. Permission resolution stays in Go; PR #22 (Rust core) is decided after the authorisation defects are fixed (Q-008, TBD-008 resolved; ADR-009). The Go module path stays (Q-033, ADR-020). |
| SYS-CON-004 | Must | The repository shall state one licence consistently. | Given README.md and retrocast/docs/openapi.yaml, when they are compared, then both say all rights reserved and neither names an open-source licence. | Inspection | Implemented | none (repository) | — | LIM-004 | README.md; retrocast/docs/openapi.yaml (`info.license` "Proprietary. All rights reserved." in rev 2). Changed: rev 2 — Q-007 (TBD-007 resolved). |

### 3.7 Software system attributes

#### 3.7.1 Reliability
Recovery is covered by SYS-NFR-005.

#### 3.7.2 Availability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-005 | Must | Recoverability: the operator shall be able to restore the database and stored files from a backup with at most 24 hours of data lost. Measure: recovery point objective 24 h; no recovery time target (best effort). | Given a stack with messages and files, when the operator runs `scripts/backup.sh`, resets the volumes and runs `scripts/restore.sh`, then the messages and files return and only data newer than the last daily backup is lost. | Demonstration | Partial (backup and restore scripts exist for PostgreSQL and MinIO; no schedule, SYS-FR-002; no restore drill is recorded, SYS-NFR-006) | none (infra) | CMP-004, CMP-006, CMP-012 | RISK-004 | retrocast/scripts/backup.sh:6 (30-day retention), :13, :19; retrocast/scripts/restore.sh. Changed: rev 2 — recovery point from Q-030 (TBD-030 resolved); Redis is not backed up (ADR-018). |
| SYS-NFR-006 | Must | Recoverability: a restore drill shall be run and recorded before each release. Measure: one recorded drill per release (backup file name, date, row counts of users, guilds and messages before and after). | Given a release candidate, when the release is prepared, then a drill record for that release exists with matching row counts before and after the restore. | Inspection | Not started (no drill is recorded) | none (operator procedure) | CMP-012 | CAP-036, RISK-004 | Added rev 2 (Q-030). |

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
| SYS-NFR-003 | Must | Privacy: no core flow shall call a service outside the operator's machine. Measure: outbound requests during core flows (sign-in, messaging, voice, file sharing) equal 0. | Given the stack on a machine without internet access, when a member signs in, sends a message and uploads a file, then each step succeeds. | Demonstration | Partial (core flows need no outside service; the `/docs` page loads Swagger UI from unpkg.com) | none (infra) | CMP-001–CMP-010 | PRN-001 | retrocast/internal/api/router.go:21, :26; GOAL-004 |

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
| SYS-FR-001 | Must | Partial | — | `/metrics` | CMP-011 | Demonstration |
| SYS-FR-002 | Must | Partial | ENT-001–ENT-018, ENT-022 | — | CMP-012, CMP-004, CMP-006 | Demonstration |
| SYS-FR-003 | Must | Implemented | — | — | CMP-012 | Test |
| SYS-FR-004 | Must | Partial | ENT-001–ENT-018, ENT-022 | — | CMP-012, CMP-004, CMP-006 | Demonstration |
| SYS-NFR-001 | Must | Not started | — | EVT-009 | CMP-001, CMP-002, CMP-004, CMP-005 | Analysis |
| SYS-NFR-002 | Must | Not started | ENT-008 | API-022, EVT-009 | CMP-002 | Analysis |
| SYS-CON-001 | Must | Partial | — | — | CMP-001–CMP-012 | Inspection |
| SYS-CON-002 | Must | Implemented | ENT-001 | API-001 | CMP-001 | Test |
| SYS-CON-003 | Must | Implemented | — | — | CMP-001–CMP-012 | Inspection |
| SYS-CON-004 | Must | Implemented | — | — | — | Inspection |
| SYS-NFR-003 | Must | Partial | — | — | CMP-001–CMP-010 | Demonstration |
| SYS-NFR-004 | Should | Partial | — | — | CMP-001–CMP-011 | Demonstration |
| SYS-NFR-005 | Must | Partial | ENT-001–ENT-018, ENT-022 | — | CMP-004, CMP-006, CMP-012 | Demonstration |
| SYS-NFR-006 | Must | Not started | — | — | CMP-012 | Inspection |
| SYS-NFR-007 | Must | Not started | ENT-008 | API-023 | CMP-001, CMP-004 | Analysis |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial system constraints and qualities SYS-CON-001–SYS-CON-004, SYS-NFR-001–SYS-NFR-005 |
