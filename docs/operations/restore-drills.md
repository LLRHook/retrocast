# Restore drills

## v0.1.0 release restore drill — 2026-11-27

PASS: L66 round 2 ran the released application on a Windows 11 host with
Docker Desktop (Docker server 29.9.3). Release tag `v0.1.0` identifies commit
`75290b6138a98daa2ff43c16a0f8fb31d4b66a07`. Application trees:

| Tree | Git object ID |
|---|---|
| `retrocast/` | `1565e0d68e769c7b49931d8a21c73d6731129535` |
| `retrocast-web/` | `11e3d33d20359cf49ed375189bf6d60f2b3096f3` |

This is the dated SYS-NFR-006 record for v0.1.0. It was performed after
publication, on the release commit; it supplies the missing release-code drill
without claiming the check occurred before publication.

Setup followed the [README](../../README.md#run-with-docker-compose):

```sh
cd retrocast
bash scripts/init-env.sh
docker compose up --build
```

L66 exported the release with `git archive`, generated private credentials
(init-env exit 0, with its documented Windows ACL warning), and built API, web
and MinIO images (all exit 0). Owned `l66r2` project/image names and retained
`l66-common.yml`, `l66-full.yml` and `l66-restore.yml` overrides selected those
images and loopback Caddy ports 16643/16644, checked free first. LiveKit and
Grafana host bindings were removed. Startup used `up -d --no-build --wait`
(exit 0). No candidate application source changed.

Python orchestration reused the L42 backup/restore sequence, invoking the
released scripts unchanged. The REST fixture had two users, one guild and one
message with one attached 53-byte text file (in place of L42's 68-byte PNG).
With the source API stopped, `bash scripts/scheduled-backup.sh` exited 0 and
created this pair. Dates below are archive modification times in UTC; filenames
use the Windows host's local time.

| Archive | UTC date | Bytes | SHA-256 |
|---|---|---:|---|
| `backup_20261127_064350.sql.gz` | 2026-11-27T11:43:50.476081+00:00 | 4042 | `8118ab2533d167bf2fec3a84be9921993212f501280aeddb6b058aadf9ea75f1` |
| `minio_20261127_064350.tar.gz` | 2026-11-27T11:43:51.124728+00:00 | 5228 | `b01ad09822f73dd09455861c06101b9301a19319ccd4fb22cf2220a33a0b80a8` |

A separate Compose project started with empty volumes and normal minio-init,
then `bash scripts/restore.sh <sql-archive> <minio-archive>` exited 0.

| Snapshot | Users | Guilds | Messages |
|---|---:|---:|---:|
| Source before backup | 2 | 1 | 1 |
| Fresh target before restore | 0 | 0 | 0 |
| Target after restore | 2 | 1 | 1 |

The pre-backup message and its attachment returned. After backup, the source
received a marker message: source count 1, restored count 0. This establishes
recovery to the backup point and loss only of the later marker in this fixture;
the 24-hour RPO still depends on installing and monitoring the daily schedule.
A fresh sign-in and history request supplied a new HTTPS signed URL. Curl used
`--connect-to` to preserve its canonical host while reaching the target port.
The download matched all 53 input bytes; `/health` returned 200. The same pair
was restored a second time (exit 0), explicitly recording HTTPS GET 200 and
exact bytes. No signed URL or credential is published.

The documented cron line also fired once at 11:46 UTC in an owned sidecar with
its environment cleared, using L42 round 2's unchanged Go ptrace observer
(compiled with `golang:1.26.11`) and cached `docker:27.5.1-cli` tools. Only the
project, mounted paths and time fields for the near-term firing were substituted.
The actual child shell returned `exit_status=0 signaled=false`; `gzip -t` and
`tar -tzf` exited 0. This is cron mechanism evidence, not a 24-hour host schedule.

| Cron archive | Bytes | SHA-256 |
|---|---:|---|
| `backup_20261127_114600.sql.gz` | 4091 | `f30b9439a9b203399517c38b99920b865e7bd6d7b5487b637f549b95639831fa` |
| `minio_20261127_114600.tar.gz` | 10055 | `19f1f87cad6e862a1c74e86da972437b9059e1b521a424de01351c6212d879d5` |

Cleanup passed: all started projects received `down -v`; all L66 containers,
networks, volumes, cache volumes and image tags were removed. The exported
source, independent bare clone, private environment, helpers and both backup
directories were removed. Protected rollback images and unrelated resources
were preserved.

Source: accepted private L66 round-2 evidence `drill.json`,
`restored-download.json`, `backup.txt`, `restore.txt`, `cron-result.json`,
`cron-observed-exit.txt`, `cron-integrity.txt` and `cleanup.json`.
See the [seven release-code checks](../validation/release-2026-11-27.md#release-code-runtime-checks-v010).

## D19 systemd user timer observation — 2026-11-27

PASS on an Arch-based Linux host with systemd and no cron daemon.
Docker Engine and Compose versions were not captured during the run. The stack was
bound to loopback only, under Compose project `l51rc`; no production resources
were used. The REST fixture contained one user, one guild, one message and one
attachment (a 36-byte PDF linked from that message).

The tested application commit was
`f916c9060fd344d7f5e4f87536db5592e7b751e1`, with the two unit templates from
PR #90 copied in. Their SHA-256 values equal the blobs on release baseline
`7f018a3e2344f78bc6199eec1e982f342dce9acd`:

| Template | SHA-256 |
|---|---|
| `retrocast/scripts/retrocast-backup.service` | `f19fe36df09b702f630ae6263af4c12e67d832be9bd16087acd0d4ad4531c886` |
| `retrocast/scripts/retrocast-backup.timer` | `1744ec79cafb4095cbc0bde6b5a43fe3b7a03100c5dbe3217bd1381a7a373a9e` |

`scheduled-backup.sh` and `backup.sh` are unchanged between `f916c90` and
`7f018a3`; the Compose `backup-tools` service changed only in comments. Other
Compose services did change, so this is identity of the scheduled backup path,
not of the entire stack. The D18 rollback drill below ran
`scheduled-backup.sh` on `b1514a3` (N−1). L66 has now run the wrapper with
the v0.1.0 release Compose configuration, as recorded above. These checks make
this observation applicable to the release backup path.

Installation followed [OPERATIONS.md](../OPERATIONS.md), substituting only
paths and the project name, with its documented `COMPOSE_FILE` and
`COMPOSE_PATH_SEPARATOR` entries for the loopback override. Here `<clone>`
denotes the clone and `$BACKUP_DIR` the private destination outside it; paths
in command transcripts are redacted. The destination was created with mode
700; both units were copied to `~/.config/systemd/user/`. The service used
`COMPOSE_PROJECT_NAME=l51rc`,
`COMPOSE_FILE=<clone>/retrocast/docker-compose.yml:<clone>/l51-compose.yml`,
`COMPOSE_PATH_SEPARATOR=:`, and
`ExecStart=/bin/bash <clone>/retrocast/scripts/scheduled-backup.sh`.
Its destination and append-log paths were replaced consistently. `UMask=0077`
and the timer's daily 03:00 calendar and `Persistent=true` were unchanged.
The installed timer matched the template; the installed service differed only
by those documented substitutions and Compose entries.

```sh
systemd-analyze --user verify ~/.config/systemd/user/retrocast-backup.{service,timer}
systemctl --user daemon-reload
systemctl --user start retrocast-backup.service
systemctl --user show -p Result,ExecMainStatus retrocast-backup.service
gzip -t "$BACKUP_DIR/backup_20261126_042202.sql.gz" "$BACKUP_DIR/minio_20261126_042202.tar.gz"
tar -tzf "$BACKUP_DIR/minio_20261126_042202.tar.gz"
systemctl --user enable --now retrocast-backup.timer
systemctl --user list-timers retrocast-backup.timer
```

Unit verification, reload, manual start, gzip and tar checks exited 0. The
manual service ran at 2026-11-26 04:22:02–04:22:03 EST, returning
`Result=success` and `ExecMainStatus=0`. Its pair was timestamped
`20261126_042202`. Timer enablement started and completed at
2026-11-26 04:22:21 EST, exit 0, with next run 2026-11-27 03:00:00 EST.

At collection on 2026-11-27 04:36:04 EST, 87,223 wall-clock seconds
(24 h 13 m 43 s, 24.23 h) had passed since enablement. The records showed:

```text
ActiveState=active
NextElapseUSecRealtime=Sat 2026-11-28 03:00:00 EST
LastTriggerUSec=Fri 2026-11-27 03:00:00 EST
TriggeredBy=retrocast-backup.timer
Triggers=retrocast-backup.service
Result=success
ExecMainStartTimestamp=Fri 2026-11-27 03:00:00 EST
ExecMainExitTimestamp=Fri 2026-11-27 03:00:00 EST
ExecMainStatus=0
```

The user journal recorded service start and finish at 03:00:00. Attribution
rests on that matching activation, `LastTriggerUSec` and `TriggeredBy`/`Triggers`;
there was no separate literal timer-trigger line. No manual service start was
issued during collection. The new pair was:

| Archive | Bytes | Mode | SHA-256 |
|---|---:|---|---|
| `backup_20261127_030000.sql.gz` | 3922 | 600 | `4c616a39ae9368f32ef239c3a77e63cbb025fe33276cc62bb8128c1b378dc03c` |
| `minio_20261127_030000.tar.gz` | 10332 | 600 | `9a4243ace0e9b4dab3d6493fb7b3c9180192619ddedbda48e599270e5e7531ee` |

`$BACKUP_DIR` was mode 700 and `backup.log` mode 600. `gzip -t` on both
archives exited 0, `tar -tzf` passed, and `zcat` exited 0. Fixture checks returned:

```text
CREATE_TABLE_lines=19
COPY_public_lines=19
fixture_message_matching_lines=1
fixture_attachment_rows=1
attachment_object_key_directory_matches=1
attachment_object_xl_meta_matches=1
pair_integrity_and_fixture=PASS
```

The tar listing contained the fixture's object directory and `xl.meta`:
`./retrocast/attachments/251638554827034624/251638554843811840/l51-drill.pdf/`.
`backup.log` contained the manual and timer runs, each ending
`=== Backup Complete ===`. The log has no exit-code line: exit-0 evidence for
the timer run is systemd's `Result=success` / `ExecMainStatus=0` above.

The window includes a reboot and a reported 5 h 30 m 16 s clock jump. At
collection uptime was 13:41. With `Linger=yes`, the enabled timer restarted
at boot on November 26 at 09:25:08 and still fired at the configured 03:00.
The lead's sleep inhibitor did not survive the reboot. There was no explicit
completed suspend/resume record; that absence does not establish no sleep.
Continuous uptime and a monotonic 24-hour interval were not established.
The driver accepted this operator case and the systemd exit-status evidence
on 2026-11-27 (D19 / SYS-FR-002).

After the pair passed, `systemctl --user disable --now retrocast-backup.timer`
exited 0; the service was already inactive. Both copied units were removed
and `systemctl --user daemon-reload` exited 0. The timer inventory had no
Retrocast timer and `list-unit-files "retrocast*"` listed 0 unit files.
`docker compose down -v --remove-orphans` for `l51rc` exited 0, removing its
9 containers, 6 volumes and network. The three owned built image tags were
removed, exit 0. Filtered inventories found no owned Docker resources or
built images. The clone and backup destination were deleted, with
`exists_after=false` for both, completing at 2026-11-27T04:38:45-05:00.
Only private evidence remained; pulled dependency images and build cache were
retained by instruction. Other owners' inhibitors, Linger, packages and power
settings were unchanged.

Sources: L51 round-2 installation report and round-3 completion report,
checked and accepted by the driver on 2026-11-27. These private
reports are retained with the driver run; raw host evidence was not transferred
into the repository. Commands and outputs above are drawn from those reports
with private paths redacted. No restore or attachment download was performed
in this observation; fixture presence is established by SQL rows and the
archived object path. Restore coverage remains in the drills below.

## D18 release rollback drill — 2026-11-26

PASS on a Windows 11 host / Docker Desktop, Engine 29.9.3
linux/amd64, Compose v5.7.0. N-1:
`b1514a3699e05aa11f9e82d20bab92fc4ce23f74`; N:
`d4b8d065b2d9627b8931cbcaf2e4786505a72bc2`.
The final running revision was N-1, with the rotated secrets retained.
The record branch was rebased onto main
4910b846439ee164967dc3a66e05f1769f52e30c after the drill. Relative to N,
that main adds only SECURITY_AUDIT.md and its .gitignore line; application code
is identical. The tested N remains d4b8d06.

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
# Use the MSYS /c/ form of BACKUP_DIR for this listing.
tar -tzf "$BACKUP_DIR"/minio_*.tar.gz | wc -l
sha256sum "$BACKUP_DIR"/*.gz
docker compose start --wait api livekit
# Health and counts after backup.
```

The scheduled wrapper exited 0. Both gzip checks and tar listing exited 0;
tar listed 25 entries. API and LiveKit were quiescent during the paired backup.
The successful D18 listing used `tar -tzf` with an MSYS `/c/` path.
`tar --force-local` was checked separately on a one-file fixture; it was not
the command used to list this backup. The first drive-letter-path attempt
had treated the colon as a remote-host separator.

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
image tags remained present. Generated-secret/key-ID-shaped scan: 0 hits;
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


## Round 2 — 2026-11-24 (Windows 11 / Docker Desktop)

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

Candidate 0f7ee554a0b51d79252ace01ebcb213f5d2b5501, a Windows 11 host with Docker Desktop. Source l42pilot, fresh target l42restore. Both reused API image sha256:40f45ef0609ccd57988649a333fa76091565a07316b3df51c0fd4652959ad58c and web sha256:8d658ac5fdb4e931ea4ba8fa5d5579377a2f5d06b87554fc673911001554efe6. Documented scheduled wrapper and restore exited0. Paired backup_20261124_145041.sql.gz / minio_20261124_145041.tar.gz; before/after 2 users, 1 guild, 1 message; health200. Fresh invited browser downloaded restored PNG over HTTPS200, 68 bytes. A source message inserted after backup is absent from the target; the prebackup attachment message/file is present. Manual drill proves recovery to the backup point; no installed daily host schedule is claimed. Private scripts/output: evidence/L42/l42-drill.sh, restore.log, restore-counts-before/after.txt, rpo-source/restored.txt and restored-download.txt. See [full validation](../validation/pilot-2026-11-24.md).

## Pilot run 3 round 2 — 2026-11-24

Candidate `511b8cd241fc7fab6a15f9d62364ac8000378104`, a Windows 11 host with Docker Desktop. The original 0f7ee55 restore/RPO drill is carried by identical Go, storage, migration, compose, Caddy and Dockerfile object ids recorded in `evidence/L42/round2/carry.txt`; it was not rerun on the new candidate.

The documented cron entry itself was installed in an owned sidecar, changing only time fields to two minutes ahead and the documented project placeholder to l42r2open. Captured `crontab -l` uses BACKUP_DIR=/srv/backups/retrocast, the documented scheduled-backup.sh path and append redirection, with no wrapper job or added PATH/COMPOSE_FILE export. It fired at 21:44 UTC. Pair `backup_20261124_214403.sql.gz` / `minio_20261124_214403.tar.gz` passed gzip/tar integrity; the actual cron shell exited 0, observed without altering the job and recorded in backup.log. Private crontab/firing/exit/integrity evidence is under `evidence/L42/round2/`.

This establishes the operator line’s mechanism under cron’s minimal environment, not installation on the actual operator host or 24 elapsed hours. SYS-FR-002 remains Implemented; owner action is the actual daily host schedule and the first two paired backups observed over 24 hours. See [validation record](../validation/pilot-2026-11-24.md).
