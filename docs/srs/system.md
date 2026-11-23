# SRS component — System (constraints and qualities)

Part of [docs/SRS.md](../SRS.md). Prefix `SYS-`. Covers requirements that no single component
owns: hosting, external dependencies, licence, capacity, latency, portability and backups.
Components are in [ARCHITECTURE_SPEC.md §5](../ARCHITECTURE_SPEC.md#5-components-c4-level-2).

Conventions for this file:
- **Evidence** paths are relative to the repository root.
- **Delivery** comes from reading the repository at commit `6bc385e`. No load, latency or
  restore measurement exists, so no quality target here is met yet.

## 3. Requirements

### 3.1 Functions
System-level functions are owned by the components: see [server.md](server.md),
[web.md](web.md) and [ios.md](ios.md).

### 3.2 Performance requirements

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-001 | Must | Capacity: the system shall serve 20 members connected to the gateway at once on one host. Measure: TBD-031 (latency, CPU and memory limits at 20 connections). | Given the compose stack on the operator's machine and 20 connected clients sending messages, when the load runs for 10 minutes, then the TBD-031 limits hold. | Analysis | Not started (no load harness exists; Stack Profile "Perf harness: none") | none (infra) | CMP-001, CMP-002, CMP-004, CMP-005 | GOAL-001 | SYSTEM_DESIGN.md:27 ("5–20 friends") |
| SYS-NFR-002 | Must | Latency: the system shall deliver a sent message to the other connected members of its channel within a stated time. Measure: TBD-031 (95th-percentile send-to-receive time). | Given two connected members on the same network, when one sends 100 messages, then the 95th-percentile delay meets TBD-031. | Analysis | Not started (no measurement exists) | none (infra) | API-022, EVT-009, CMP-002 | GOAL-001 | SYSTEM_DESIGN.md:63 ("Simple, fast, focused") |

### 3.3 Usability requirements
No usability or accessibility targets are stated (TBD-032).

### 3.4 Interface requirements
External interfaces (LiveKit, MinIO, APNs) are in
[DATA_CONTRACTS.md §7](../DATA_CONTRACTS.md#7-external-contracts).

### 3.5 Logical database requirements
See [server.md §3.5](server.md#35-logical-database-requirements).

### 3.6 Design constraints

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-CON-001 | Must | The system shall run as one Docker Compose stack on a single Mac or Linux machine with no cloud service. | Given a machine with Docker, when the operator runs `docker compose up -d` in `retrocast/` with a filled `.env`, then every service starts on that machine. | Inspection | Implemented | none (infra) | CMP-001–CMP-011 | LIM-001 | retrocast/docker-compose.yml; docs/deployment.md |
| SYS-CON-002 | Must | The system shall need no third-party account and no email address to register or use it. | Given the register form, when a user submits only a username and password, then the account is created. | Test | Implemented | web — Register page; ios — RegisterView | API-001 | LIM-002 | retrocast/internal/service/auth.go:54 |
| SYS-CON-003 | Must | The system shall keep the existing stack (one Go binary for REST and the gateway; PostgreSQL, Redis, MinIO, LiveKit; Caddy; a React web client; a SwiftUI iOS client). | Given a change proposal, when it replaces one of these parts, then it carries an ADR in ARCHITECTURE_SPEC §8. | Inspection | Implemented | none (infra) | CMP-001–CMP-011 | LIM-003 | ARCHITECTURE_SPEC §4.1; PR #22 proposes a Rust core (TBD-008) |
| SYS-CON-004 | Must | The repository shall state one licence consistently. | Given README.md and retrocast/docs/openapi.yaml, when they are compared, then they name the same licence. | Inspection | Partial (README.md says "unlicensed. All rights reserved."; openapi.yaml says `license: MIT`) | none (repository) | — | LIM-004 | README.md; retrocast/docs/openapi.yaml; TBD-007 |

### 3.7 Software system attributes

#### 3.7.1 Reliability
Recovery is covered by SYS-NFR-005.

#### 3.7.2 Availability

| ID | Priority | Requirement | Acceptance | Verification | Delivery | UI | Traces | Source | Evidence |
|---|---|---|---|---|---|---|---|---|---|
| SYS-NFR-005 | Must | Recoverability: the operator shall be able to restore the database and stored files from a backup. Measure: recovery point and recovery time (TBD-030). | Given a stack with messages and files, when the operator runs `scripts/backup.sh`, resets the volumes and runs `scripts/restore.sh`, then the messages and files return and the data lost is within the TBD-030 recovery point. | Demonstration | Partial (backup and restore scripts exist for PostgreSQL and MinIO; no schedule, no Redis backup and no restore drill is recorded) | none (infra) | CMP-004, CMP-006 | RISK-004 | retrocast/scripts/backup.sh:6 (30-day retention), :13, :19; retrocast/scripts/restore.sh |

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

Data retention and account deletion are TBD-021 and TBD-022.

#### 3.7.7 Observability
Server logging is SV-NFR-006. Metrics and dashboards wait for CAP-035 (TBD-039).

### 3.8 Supporting information
- The Stack Profile used for ARCHITECTURE_SPEC §4.1 is generated locally and is not committed.

## 5.1 Traceability matrix

| Req | Priority | Delivery | Entities | APIs / Events | Components | Verification |
|---|---|---|---|---|---|---|
| SYS-NFR-001 | Must | Not started | — | EVT-009 | CMP-001, CMP-002, CMP-004, CMP-005 | Analysis |
| SYS-NFR-002 | Must | Not started | ENT-008 | API-022, EVT-009 | CMP-002 | Analysis |
| SYS-CON-001 | Must | Implemented | — | — | CMP-001–CMP-011 | Inspection |
| SYS-CON-002 | Must | Implemented | ENT-001 | API-001 | CMP-001 | Test |
| SYS-CON-003 | Must | Implemented | — | — | CMP-001–CMP-011 | Inspection |
| SYS-CON-004 | Must | Partial | — | — | — | Inspection |
| SYS-NFR-003 | Must | Partial | — | — | CMP-001–CMP-010 | Demonstration |
| SYS-NFR-004 | Should | Partial | — | — | CMP-001–CMP-011 | Demonstration |
| SYS-NFR-005 | Must | Partial | ENT-001–ENT-018, ENT-022 | — | CMP-004, CMP-006 | Demonstration |

## Revision history
| Rev | Date | Commit | Mode | Summary |
|---|---|---|---|---|
| 1 | 2026-11-23T12:58:05-05:00 | 918f0a8471d3 | INIT | Initial system constraints and qualities SYS-CON-001–SYS-CON-004, SYS-NFR-001–SYS-NFR-005 |
