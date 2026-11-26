# Restore drills

## D18 release rollback drill — 2026-11-26

PASS on DESKTOP-JRJNV5B, Windows 11 / Docker Desktop, Engine 29.9.3
linux/amd64, Compose v5.7.0. N-1:
`b1514a3699e05aa11f9e82d20bab92fc4ce23f74`; N:
`d4b8d065b2d9627b8931cbcaf2e4786505a72bc2`.
The final running revision was N-1, with the rotated secrets retained.

A separate GitHub clone used
`git -c core.autocrlf=false clone -c core.autocrlf=false https://github.com/LLRHook/retrocast.git "$ROOT"`.
The second option persists LF checkout settings. The successful run had zero
tracked CRLF files. Project `l59drill` retained the same volumes throughout.
Operator overrides bound Caddy HTTPS to `127.0.0.1:15943` and Grafana to
`127.0.0.1:15900` (checked free), removed LiveKit host ports, and gave N's MinIO
build its own `l59-minio:candidate` tag. Registration was open for this isolated
fixture; operator settings and region were retained during rotation.
BACKUP_DIR was private and outside the clone. No production resources were used.

| Check | N-1 start | After backup | N before upload | N after upload | Rollback N-1 |
|---|---:|---:|---:|---:|---:|
| Health HTTP | 200 | 200 | 200 | 200 | 200 |
| users | 2 | 2 | 2 | 2 | 2 |
| guilds | 1 | 1 | 1 | 1 | 1 |
| channels | 2 | 2 | 2 | 2 | 2 |
| messages | 4 | 4 | 4 | 5 | 5 |
| attachments | 1 | 1 | 1 | 2 | 2 |

REST seeded two users, one guild (including its text channel), three plain
messages and an attachment message. N added an attachment message, explaining
the only row-count increases. At every snapshot the query was:

```sql
SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM guilds),
       (SELECT count(*) FROM channels), (SELECT count(*) FROM messages),
       (SELECT count(*) FROM attachments);
```

Commands ran in Git Bash from `$ROOT/retrocast`; `$PRIVATE` denotes the owned
private temporary directory. Each commented check below was performed by the
captured probe or snapshot helper, not left for later execution.

```sh
export COMPOSE_PROJECT_NAME=l59drill COMPOSE_PATH_SEPARATOR=';'
export COMPOSE_FILE="$ROOT/retrocast/docker-compose.yml;$PRIVATE/override.yml"
export BACKUP_DIR="$PRIVATE/backups"
git checkout b1514a3699e05aa11f9e82d20bab92fc4ce23f74
cp .env.example .env
# Fixture-only REGISTRATION_MODE=open; retain placeholder secrets.
docker image inspect minio/minio:latest --format '{{.Id}}'
docker compose up -d --build --wait
# REST seed, signed download, health and SQL snapshot.
docker compose stop api livekit
bash scripts/scheduled-backup.sh
gzip -t "$BACKUP_DIR"/*.gz
tar -tzf /c/Users/Victor/AppData/Local/Temp/l59-private/backups/minio_*.tar.gz | wc -l
sha256sum "$BACKUP_DIR"/*.gz
docker compose start --wait api livekit
# Health and counts after backup.
```

The scheduled wrapper exited 0. Both gzip checks and tar listing exited 0;
tar listed 25 entries. API and LiveKit were quiescent during the paired backup.

| Archive | SHA-256 |
|---|---|
| `backup_20261126_084249.sql.gz` | `9ad3b02dc23220198274d0d00f1b7e32f9ceae523b84766a727e5b5cf84ceac4` |
| `minio_20261126_084249.tar.gz` | `da8fee96e5ff10cbbbeaef8eab61741dd1645f9af642a97aa6d62125c4f20169` |

OPERATIONS rotation steps 1–6 ran in order:

```sh
docker image inspect minio/minio:latest --format '{{.Id}}'
db_id=$(docker compose ps -q db)
redis_id=$(docker compose ps -q redis)
grafana_id=$(docker compose ps -q grafana)
docker compose stop api livekit
git checkout d4b8d065b2d9627b8931cbcaf2e4786505a72bc2
bash scripts/init-env.sh "$PRIVATE/new.env"
# Retain operator settings; old deployed .env stays until step 5.
docker exec -i "$db_id" psql -U retrocast retrocast
# Execute \password retrocast; replacement POSTGRES_PASSWORD twice on stdin.
docker exec "$redis_id" redis-cli FLUSHDB
docker exec -i "$grafana_id" grafana cli admin reset-admin-password --password-from-stdin
# Supply replacement GRAFANA_ADMIN_PASSWORD on stdin.
# Privately assert changed root, LiveKit and signer pairs; root/write secrets match.
cp "$PRIVATE/new.env" .env
chmod 600 .env
# Install N's image-tag override at the same override path.
docker compose up -d --build --force-recreate
docker compose up -d --wait
docker inspect l59drill-env-check-1 --format '{{.State.ExitCode}}'
# Step 6: b1514a3 has no dedicated signer to disable.
# Health, counts, sign in again, upload and exact signed downloads.
```

The automated psql session used `-i` in place of interactive `-it`, providing
both private responses on stdin to the same `\password` command in the existing
container. No secret reached argv or evidence. FLUSHDB returned OK; Grafana's
CLI reported a successful password reset; env-check exited 0. After recreation,
Grafana `/api/user` refused the committed default (401) and accepted the new
password (200). The old dedicated-signer disable command was inapplicable.

| Attachment | SHA-256 | N-1 GET | N GET | Rollback GET |
|---|---|---|---|---|
| Initial fixture | `b47e6ddc4d0f057460eeae0da3acefe26a392866cef0de8baad6d38fd3b40911` | 200, exact, `retrocast` | 200, exact, dedicated signer | 200, exact, `retrocast` |
| N upload | `eb528450312080a6a43253b12b8896eca09a70c96335d3c918458da1668452b7` | Not yet created | upload 201; GET 200, exact, dedicated signer | 200, exact, `retrocast` |

The probe fetched fresh signed HTTPS URLs from authenticated history, classified
the credential internally and checked each response against the original fixture
SHA-256. It saved neither URLs nor tokens. Only the isolated localhost probe
accepted Caddy's development certificate.

```sh
# Privately record forward MINIO_ACCESS_KEY; set MINIO_ACCESS_KEY=retrocast.
git -c core.autocrlf=false checkout b1514a3699e05aa11f9e82d20bab92fc4ce23f74
# Restore N-1's override at the same path; retain rotated secrets and region.
docker compose up -d --build
docker compose up -d --wait
# Health 200, unchanged counts, 2/2 exact signed downloads by retrocast.
docker compose down -v --remove-orphans
docker image rm l59drill-api:latest l59drill-web:latest l59-minio:candidate
docker volume rm l59-gocache
```

The drill stopped at N-1. All owned containers, networks, volumes and image tags,
the clone, private files and paired archives were removed. Both protected MinIO
image tags remained present. Generated-secret/key-ID scan: 0 exact matches;
concrete token-shaped scan: 0 hits.

OPERATIONS corrections: none. Before the successful clean run, harness incidents
were a 201 expectation for valid 200 registration, Windows tar treating `C:` as a
remote archive, an override missing a newline, and a command-local autocrlf
setting that did not persist for later checkouts. Those resources were removed
and the whole drill repeated from an empty-volume, persistently LF clone.
The successful run above supersedes that attempt.

Private evidence: `.git/agent-handoffs/driver-rc-revive/evidence/L59/` contains
`01-start.log`, `02-backup.log`, `04-rotate.log`, `06-grafana.log`,
`07-rollback.log`, `08-cleanup.log`, `metadata.txt`, `secret-scan.txt` and
`scripts/`; superseded logs are in `attempt1/`. This drill does not claim an
archive restore into another stack, a host scheduler installation or a 24-hour
RPO observation.


## B5c candidate — 2026-11-24 (Windows 11, Docker Desktop)

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


## Pilot run 3 — 2026-11-24

Candidate 0f7ee554a0b51d79252ace01ebcb213f5d2b5501, Windows11 Docker Desktop DESKTOP-JRJNV5B. Source l42pilot, fresh target l42restore. Both reused API image sha256:40f45ef0609ccd57988649a333fa76091565a07316b3df51c0fd4652959ad58c and web sha256:8d658ac5fdb4e931ea4ba8fa5d5579377a2f5d06b87554fc673911001554efe6. Documented scheduled wrapper and restore exited0. Paired backup_20261124_145041.sql.gz / minio_20261124_145041.tar.gz; before/after 2 users, 1 guild, 1 message; health200. Fresh invited browser downloaded restored PNG over HTTPS200, 68 bytes. A source message inserted after backup is absent from the target; the prebackup attachment message/file is present. Manual drill proves recovery to the backup point; no installed daily host schedule is claimed. Private scripts/output: evidence/L42/l42-drill.sh, restore.log, restore-counts-before/after.txt, rpo-source/restored.txt and restored-download.txt. See [full validation](../validation/pilot-2026-11-24.md).

## Pilot run 3 round 2 — 2026-11-24

Candidate `511b8cd241fc7fab6a15f9d62364ac8000378104`, DESKTOP-JRJNV5B, Windows 11 / Docker Desktop. The original 0f7ee55 restore/RPO drill is carried by identical Go, storage, migration, compose, Caddy and Dockerfile object ids recorded in `evidence/L42/round2/carry.txt`; it was not rerun on the new candidate.

The documented cron entry itself was installed in an owned sidecar, changing only time fields to two minutes ahead and the documented project placeholder to l42r2open. Captured `crontab -l` uses BACKUP_DIR=/srv/backups/retrocast, the documented scheduled-backup.sh path and append redirection, with no wrapper job or added PATH/COMPOSE_FILE export. It fired at 21:44 UTC. Pair `backup_20261124_214403.sql.gz` / `minio_20261124_214403.tar.gz` passed gzip/tar integrity; the actual cron shell exited 0, observed without altering the job and recorded in backup.log. Private crontab/firing/exit/integrity evidence is under `evidence/L42/round2/`.

This establishes the operator line’s mechanism under cron’s minimal environment, not installation on the actual operator host or 24 elapsed hours. SYS-FR-002 remains Implemented; owner action is the actual daily host schedule and the first two paired backups observed over 24 hours. See [validation record](../validation/pilot-2026-11-24.md).
