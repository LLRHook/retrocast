# Retrocast — Security Audit Baseline

Owned by the `security-audit` skill. Findings are filed as GitHub issues in LLRHook/retrocast (BUG column holds the issue number).

## Last run
- Date: 2026-11-26   SHA: d4b8d06 (release candidate = main after PR #94)   Depth: incremental (delta b1514a3..d4b8d06; authz, crypto, network and IAM changed, so every changed file and its boundary neighbours were reviewed; scanners ran repo-wide at 54d5403, and the 5-file delta 54d5403..d4b8d06 was re-checked separately)   Last full: 2026-11-26 @ b1514a3
- Tools: gitleaks@8.30.1 (history --all + tree), semgrep@1.178.0 (rules a5ad171), gosec@2.22.10, govulncheck@1.1.4 (source; binary mode on the built MinIO server), trivy@0.67.2 (fs, config, api/web/minio images), hadolint@2.12.0, zizmor@1.16.3; go test with Postgres/Redis (1268 pass, 0 fail, 2 opt-in stack tests skipped); gaps: compose — Checkov framework unsupported, reviewed manually; osv-scanner — excluded (external api.osv.dev not allowed); pnpm-audit — not re-run (no web dependency change in the delta; trivy fs over the lockfile: 0); golangci-lint@v2.9 — CI lint gate, not re-run. Delta 54d5403..d4b8d06: manual review, gitleaks tree (same 2 false positives), config and main go tests, and validator probes against the d4b8d06 API image.
- Result: crit 0 / high 0 / med 1 / low 6; new 2; resolved 4; suppressed 0; rejected in verification 0 (one severity lowered by the verifier: #96)
- Release gate (zero open crit/high): PASS for d4b8d06.
- Cloud: not run (self-hosted deployment model; no AWS)

## Threat model
Deployment: self-hosted group chat for a trusted 5–20 person team on one machine. Caddy is the only
public edge (80/443, TLS for $DOMAIN); it exposes /api/*, /gateway, /docs*, /health and /retrocast/*
(GET-only presigned MinIO), serves the SPA, 404s /metrics. Caddy caps request bodies (1 MiB; uploads
10 MiB + 64 KiB) and the API enforces the same limits with MaxBytesReader (#93). All other services
(Postgres, Redis, MinIO API/console, Prometheus) are internal; Grafana alone binds 127.0.0.1. LiveKit
publishes 7880/7881 + UDP 50000-50100 (accepted, SV-NFR-008).
Startup: the compose `env-check` service (`retrocast check-env`) must pass before any consumer starts,
and the API refuses placeholder (also when padded) or short (<32) secrets, a `password` query parameter in
DATABASE_URL/REDIS_URL, and a weak PGPASSWORD; `scripts/init-env.sh` generates them (#94). Upgraded
Grafana volumes need the documented mandatory admin reset.
MinIO: built from pinned, checksum-verified archived source (`Dockerfile.minio`, #78). Presigned URLs
are signed by a dedicated account limited to `s3:GetObject` on the bucket (#91); root credentials stay
in `minio` and `minio-init`.
Assets: message content and history, attachments, user credentials (Argon2id), session/refresh tokens,
guild/role/permission config, voice rooms.
Actors: unauthenticated internet client; authenticated member; a member with reduced permissions
(channel overrides); a banned/kicked former member; a malicious guild admin; the host operator (trusted).
Trust boundaries: browser↔Caddy (TLS), Caddy↔api/web/minio (internal), api↔Postgres/Redis/MinIO/LiveKit.
Auth: HS256 access JWT (15 min) + Redis refresh token with a rotation generation epoch; permissions are
a bitfield with owner/Administrator bypass and per-channel @everyone/role overrides. Ban/kick hierarchy
checks run under guild and member row locks (#76).
Top risks examined: secret strength at startup; permission bypass on message and voice surfaces;
session lifetime vs. token/password state; request-body DoS; moderation races; attachment URL forgery
and signer scope; supply-chain pinning and image CVEs; web edge hardening.
Out of scope / accepted: first-user bootstrap (ADR-014), 15-min access-token validity after password
change (SV-FR-061), LiveKit published ports (SV-NFR-008), DMs to any user, single-use refresh (no family
reuse detection, SV-FR-005), SPEAK not in LiveKit grants (known debt D-23). Known open: #51, #63, #71,
#81, #95.

## Open findings
| Fingerprint | Finding id | Dimension | Severity | Confidence | Package / location | Weakness | Fix | BUG |
|---|---|---|---|---|---|---|---|---|
| 14f8a76a6228 | CWE-613 | code | med | CONFIRMED | internal/gateway/manager.go:321 | Gateway session validated only at IDENTIFY; survives token expiry, password change, logout (re-verified at 54d5403) | Bind session to exp / re-validate; evict on password change via refresh-gen epoch | #82 |
| c3cfa524832b | CWE-693 | code | low | CONFIRMED | retrocast-web/nginx.conf:1 | No CSP, frame-ancestors, nosniff, HSTS or Referrer-Policy on the web edge (re-verified at 54d5403) | Add a restrictive CSP, nosniff, frame-ancestors 'none', HSTS | #85 |
| 070f30eb2318 | UNPINNED-IMAGES-ACTIONS | supply | low | CONFIRMED | retrocast/docker-compose.yml:58 | LiveKit/Prometheus/Grafana `:latest`, `caddy:2-alpine`, `redis:7-alpine` and CI actions on mutable tags (zizmor: 5 unpinned-uses) | Pin images/actions by digest; pin migrate; enable dependency updates | #86 |
| 780ffa84e978 | CWE-330 | code | low | CONFIRMED | internal/service/invite.go:244 | Invite codes carry 32 bits of entropy (public lookup 5/min per IP makes brute force impractical) | Use ≥64-bit invite codes | #87 |
| 5a8ae4e099b2 | OPS-HARDENING | iac | low | CONFIRMED | internal/service/voice.go:187 (was docker-compose.yml:110) | LiveKit token lasts 24 h with no revoke on kick/ban. The Grafana default-password facet is fixed in PR #94 (merge d4b8d06): new installs generate the password, and existing volumes follow the mandatory admin reset in docs/OPERATIONS.md | Shorten LiveKit token and revoke on kick/ban | #88 |
| 8d2a76fd4b89 | CWE-863 | code | low | CONFIRMED | internal/service/voice.go:66 | Voice join checks only CONNECT: a member denied VIEW_CHANNEL gets the private room's voice states and a LiveKit token for it; rises to med (CVSS 6.0) once voice audio ships (CAP-017) | Require VIEW_CHANNEL and CONNECT in JoinChannel; align SV-FR-048/SV-IR-006 | #96 |
| 2019db7af5f8 | MINIO-SOURCE-ADVISORIES | image | low | CONFIRMED | retrocast/Dockerfile.minio:5 | Archived MinIO source carries unfixed MinIO advisories and old Go modules (trivy 5 crit/64 high); none reachable through the unpublished, GET-only edge | Move to a maintained S3 store or fork; module bump only reduces noise | #97 |

## Suppressions (accepted risk)
| Fingerprint | Finding | Reason | Accepted by / date | Expires |
|---|---|---|---|---|

## Resolved
| Fingerprint | Finding | Resolved on | BUG |
|---|---|---|---|
| 9cc642bb06f6 | CWE-1188 placeholder secrets accepted at startup (forged JWT, MinIO presign and LiveKit admin token all refused once startup rejects placeholders) | 2026-11-26 — PR #94, merge d4b8d06 | #79 |
| 7999b20cedcd | CWE-863 VIEW_CHANNEL not enforced on REST message read/send/typing | 2026-11-26 — PR #92, merge f29fad2 | #80 |
| 91089178fd59 | CWE-770 no HTTP request body size limit | 2026-11-26 — PR #93, merge 6498a5e | #83 |
| 9a43586cbfe1 | CWE-862 banned/kicked member edits/deletes own messages | 2026-11-26 — PR #92, merge f29fad2 | #84 |
