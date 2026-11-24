# Restore drills

## B5c candidate â€” 2026-11-24 (Windows 11, Docker Desktop)

Source baseline: `140c236b08b311b382ac4489771b84ef16fd0aca`; procedure implemented
in `eb4b921`. Owned compose project: `l30drill`. This is a development drill,
not a production backup or an installation of a host scheduler.
The complete empty-volume drill was repeated at 2026-11-24 05:31
America/New_York using the API rebuilt from fdd8fa0 (the #58 gateway fix):
the same 1/1/1 counts, exact PDF bytes and HTTP 200 health held again.

Paired files: `backup_20261124_043945.sql.gz` and
`minio_20261124_043945.tar.gz`, created 2026-11-24 04:39:45 America/New_York.
The fixture was created via REST: register one user, create one guild, upload
`drill.pdf`, then send a message linking its attachment ID.

| Row count | Before backup | After restore |
|---|---:|---:|
| users | 1 | 1 |
| guilds | 1 | 1 |
| messages | 1 | 1 |

Commands (from repository root, Git Bash; replace ROOT with its absolute Windows
path and use your own isolated project/ports). The private override supplied
API port 18408 and MinIO port 18490 on loopback, image `l30-api`, and open
registration. No production stack or volume was touched.

```sh
export COMPOSE_PROJECT_NAME=l30drill COMPOSE_PATH_SEPARATOR=';'
export COMPOSE_FILE="$ROOT/retrocast/docker-compose.yml;$ROOT/l30-drill.yml"
export BACKUP_DIR="$ROOT/l30-backups"
bash "$ROOT/retrocast/scripts/scheduled-backup.sh"
# Record the row-count query in docs/OPERATIONS.md, then remove ONLY this project.
docker compose down -v
docker compose up -d --wait api minio
bash "$ROOT/retrocast/scripts/restore.sh" \
  "$BACKUP_DIR/backup_20261124_043945.sql.gz" \
  "$BACKUP_DIR/minio_20261124_043945.tar.gz"
curl -f http://127.0.0.1:18408/health
```

The fresh API had already run all migrations. Baseline restore emitted
"already exists" SQL errors and still exited zero. Fixed restore stopped the
API, recreated the database, applied the dump with ON_ERROR_STOP, restored the
MinIO volume while its server was stopped, and restarted the services. Health
returned HTTP 200 with `{"status":"ok"}`. Counts matched as above.

The restored PDF downloaded using authenticated MinIO `mc cat` from the restored
object path and matched the original 38 bytes exactly. Its SHA-256 is
`10afbab7fb3a602c939a27f262f731646584016521885a9eefee4d88e5339a0c`. This verifies stored file
recovery. That B5c drill used direct storage access; B6 subsequently fixed D-10
with public HTTPS signed GET URLs. Its browser PNG/PDF evidence is recorded separately
in PR #62; this historical restore drill does not itself claim a browser download.

The exact scheduled wrapper command ran successfully, producing both archives.
Retention fixture: both archive types with mtimes 31 days old were removed;
both with mtimes 29 days old remained. L31 subsequently found that the original
hermetic script could falsely pass retention and SQL-failure restart assertions.
Round 2 replaces them with independent fail-fast assertions, explicit early-start
refusal, malformed-SQL/MinIO cases and an ordered health-wait check. The revised
regression also retains 30d23h archives, nested archives and non-archive files;
checks private modes; and refuses empty/row-boundary-truncated SQL and invalid tar
before any Docker command. Selected mutations and their kill counts are recorded
in the round-2 entry below.

Repeat this drill before each release and append its actual timestamped backup
names, counts and download result here (SYS-NFR-006). Host cron, launchd and
Windows task installation and 24-hour wall-clock scheduling were not performed
on this Windows test host; their commands and templates are in docs/OPERATIONS.md.


## Round 2 ? 2026-11-24 (Windows 11 / Docker Desktop)

Restore implementation: `c0dd2a599760424deb04b1d3bf93a90bdd717c77`.
Initial full drill completed at 2026-11-24T07:00:16-0500. Source project `l30r2src`,
empty-volume target `l30r2dst`. Paired archives:
`backup_20261124_065959.sql.gz` and `minio_20261124_065959.tar.gz`.

| Row count | Before backup | After fresh-target restore |
|---|---:|---:|
| users | 3 | 3 |
| guilds | 1 | 1 |
| messages | 7 | 7 |

Original user sign-in succeeded on the restored stack; all seven history rows
matched. The restored PDF downloaded through authenticated MinIO, matching all
42 bytes and SHA-256
`7c4849ac7ba93f0d56371b4cce7c1b53f6dd2fd9a35857ec05d5e24a8c3f80e5`.
Health returned HTTP 200. The API had already migrated the fresh target before
restore. The invocation started from outside `retrocast/`, with caller-relative
archive paths, and the script resolved them before changing directory.

Repeat using the earlier command form with these paired names and an isolated
project/ports; caller-relative inputs also work when invoking the script by its
absolute path from another directory. Release verification repeats the complete
drill after the final revision stamp. It additionally exercises a target started
with only `db redis minio` (no API container), malformed inputs and transactional
SQL failure/recovery; private evidence records the exact final tested head.

The corrected hermetic suite has 15 PASS groups and asserts each retention file
independently. It keeps both 29-day and 30d23h archives, removes both 31-day
archives, preserves nested/non-archive files, checks modes 600/700, and orders API
creation before the health wait. Broken SQL gzip, broken MinIO gzip, empty SQL,
COPY-row-boundary truncation, missing pg_dump header, short non-tar input and a
full-block non-tar archive must all fail before any Docker call. Config/helper
failures must precede mutation; SQL errors must not restart the API.

Mutation verification: M01, M06, M06c, M09, M10 and M11 are killed (6/6, 100%).
M01 observes the explicit checksum preflight; framing validation independently
also rejects corrupt SQL. G5 is killed (1/1) by the production heartbeat/read-wait
relation test. These replace the false-positive round-1 regression claims found
by L31. Perf main exit-gate mutants P4/P5 remain a documented test limitation;
no perf or production gateway behavior changed, so the accepted performance run
was retained without re-running it.


## L36 Pilot candidate — 2026-11-24

Candidate `26ebe80e934be286300d7838f2f728e2eb0933e6`, Windows 11 / Docker Desktop; source l36pilot, isolated empty-volume target l36restore, unchanged candidate application source. The target Compose invocation rebuilt its project images rather than reusing source image tags; target image IDs were API `sha256:86395706d7033065f7d9737924c148df014838313666fcd0d826addf359364c7` and web `sha256:772cf19e08c2f8d1ae146618440f06c14ea6117ae117c0c2409f1419b1cd70fb`. These differ from the source image IDs; this drill proves source-version equivalence, not reuse of identical image digests. Paired backups `backup_20261124_112000.sql.gz` and `minio_20261124_112000.tar.gz`; backup taken with API stopped, using documented scripts/backup.sh and scripts/restore.sh. Backup artifacts were private temporary files, removed after the drill.

| Check | Before backup | After restore |
|---|---|---|
| Users | 2 | 2 |
| Guilds | 1 | 1 |
| Messages | 1 | 1 |
| Health | 200 | 200 |

Restore script exit 0. A newly invited test member registered after these counts were compared; history hydrated the original message and its PNG, decoded inline and downloaded 68 bytes with HTTP 200 over HTTPS Caddy using a fresh signed URL. Subsequent history-paging fixtures are not part of the before/after comparison. No user stack or volume was touched. No daily scheduler or 24-hour RPO measurement was performed. Raw evidence: `.git/agent-handoffs/driver-rc-revive/evidence/L36/restore.log`, `restore-counts-before.txt`, `restore-counts-after.txt`, `restored-client-download.txt/png`, `restored-download.txt`, `l36-drill.sh`. See [full validation](../validation/pilot-2026-11-24.md).
