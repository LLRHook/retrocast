# Operations

Run these commands from `retrocast/` with Docker Compose v2, Bash, gzip, awk, tar and GNU
find available. Use the same `COMPOSE_PROJECT_NAME`, `COMPOSE_FILE` and `.env`
as the deployed stack. Backups contain password hashes and private conversations;
keep the destination private, outside the checkout, and monitor job exit status.
Redis is intentionally excluded (ADR-018). Backups remain on this host.

Attachment signing (SV-NFR-008, ADR-019): before upgrading, add
`MINIO_SIGNER_ACCESS_KEY`, `MINIO_SIGNER_SECRET_KEY` and `GRAFANA_ADMIN_PASSWORD`
to `.env`, then run the
usual `docker compose up -d --build`. Use a new account ID distinct from the
root user and `MINIO_ACCESS_KEY`. `.env.example` is a template only; its placeholders are refused. Run
`bash scripts/init-env.sh` for a new install.
The pinned mc client parses MC_HOST credentials literally, without URL decoding.
Root key IDs cannot contain colon, LF or ASCII control characters other than HTAB;
root passwords cannot contain colon or LF. Rotate incompatible MINIO_ACCESS_KEY
or MINIO_ROOT_PASSWORD with init-env-style values first. #91 refuses these in
init; #94 also refuses them in the shared startup validator before consumers start.
Every secret under 32 characters, including an otherwise strong 20?31-character
value, now refuses startup by variable name. Missing or empty signer credentials fail Compose and API storage initialization
in every environment; there is no root fallback.

The one-shot `minio-init` uses Compose's existing root credentials, waits for MinIO,
creates/updates the signer user and policy and attaches it. The API waits for
successful completion; rerunning `up` repeats initialization safely. The policy
allows only `s3:GetObject` on `arn:aws:s3:::retrocast/*`; unlisted actions and resources
are denied by default. Use a dedicated account with no other policies or groups. No list, put, delete, other-bucket reads or admin actions are allowed.
Uploads and deletion use the original write client. Neither secret is logged;
the signer access key ID appears in signed URLs as expected for SigV4.

Keep the two variables with the deployment configuration on restore. The MinIO
volume archive includes `.minio.sys` IAM configuration; backup/restore scripts
and the isolated restore-drill procedure remain unchanged. After restoring an
older archive without the account, run `docker compose up -d minio-init` before
starting the API. After rotation, rollback to `b1514a3` must keep the rotated
secrets: do not restore the old placeholder `.env`. Privately record the forward
MINIO_ACCESS_KEY identifier, set MINIO_ACCESS_KEY to `retrocast` (that revision's
hardcoded root user), check out `b1514a3`, then run
`docker compose up -d --build`. Rebuilding is required: the project API image tag
otherwise reuses the newer binary. Keep the same project, volumes and region.
Verify `/health` 200 and exact attachment bytes. To roll forward, check out the
approved new revision, restore the recorded MINIO_ACCESS_KEY identifier, then run
`docker compose up -d --build --force-recreate`; verify health and attachments
again. Postgres keeps its rotated role password throughout both directions.
Go unit tests use explicit fixture signer arguments; the opt-in stack test reads
signer variables from its environment. CI does not start MinIO or require signer
variables for ordinary Go tests.

Daily schedule (SYS-FR-002, ADR-018): install this in the operator's crontab after
replacing both absolute paths and the project name. The wrapper sets PATH, cwd
and umask. Docker Desktop must be running and the operator must have Docker access.

```cron
0 3 * * * BACKUP_DIR=/srv/backups/retrocast COMPOSE_PROJECT_NAME=retrocast /bin/bash /srv/retrocast/retrocast/scripts/scheduled-backup.sh >> /srv/backups/retrocast/backup.log 2>&1
```

Create `/srv/backups/retrocast` with mode 700 first. Run exactly the command after
the five cron fields once, then check both archives and exit status. Check the
log daily; a sleeping or powered-off Linux host misses cron runs. A daily successful
job gives an RPO of at most 24 hours; failures need operator attention.

On Linux hosts with systemd but no cron, use the user timer in
`scripts/retrocast-backup.service` and `scripts/retrocast-backup.timer`.
Replace every `/ABSOLUTE` path in the service and set `COMPOSE_PROJECT_NAME`
to the deployed project name. Create `BACKUP_DIR` with mode 700 first; both
output streams append to its `backup.log`. The wrapper sets PATH, cwd and umask.
For multiple compose files add `Environment="COMPOSE_FILE=/absolute/base.yml:/absolute/override.yml"`
and `Environment="COMPOSE_PATH_SEPARATOR=:"` under `[Service]`.
Copy both units to `~/.config/systemd/user/`, then run:

```sh
systemd-analyze --user verify ~/.config/systemd/user/retrocast-backup.{service,timer}
systemctl --user daemon-reload
systemctl --user start retrocast-backup.service
systemctl --user show -p Result,ExecMainStatus retrocast-backup.service
# Expect Result=success and ExecMainStatus=0; inspect backup.log and both new archives.
gzip -t /absolute/backups/retrocast/backup_TIMESTAMP.sql.gz /absolute/backups/retrocast/minio_TIMESTAMP.tar.gz
tar -tzf /absolute/backups/retrocast/minio_TIMESTAMP.tar.gz
systemctl --user enable --now retrocast-backup.timer
systemctl --user list-timers retrocast-backup.timer
```

Use the actual destination and paired timestamp for the archive checks. The timer
runs daily at 03:00 local time; `Persistent=true` catches a missed run on wake.
User timers run only while the user has a session unless lingering is enabled
with `loginctl enable-linger`, which may need administrator rights. Docker must
remain available. Remove with `systemctl --user disable --now retrocast-backup.timer`,
delete both copied units, then run `systemctl --user daemon-reload`.

On macOS copy `scripts/org.retrocast.backup.plist` to
`~/Library/LaunchAgents/org.retrocast.backup.plist`, replace every `/ABSOLUTE`
path, set the deployed compose project name, create the private destination,
and run `plutil -lint` on it. Install with
`launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/org.retrocast.backup.plist`.
Test with `launchctl kickstart gui/$(id -u)/org.retrocast.backup`. It runs at 03:00
local time and launchd coalesces missed calendar runs on wake. Stay signed in
with Docker Desktop running. No cron or launchd daemon was installed by this PR.

Windows Task Scheduler: create a daily 03:00 task for the Docker Desktop user,
run only while that user is logged on, and enable "Run task as soon as possible
after a scheduled start is missed" and "Do not start a new instance". Program:
`C:\Program Files\Git\bin\bash.exe`; arguments: `-lc 'export BACKUP_DIR="C:/Backups/retrocast" COMPOSE_PROJECT_NAME=retrocast; bash "C:/src/retrocast/retrocast/scripts/scheduled-backup.sh"'`.
Use your real absolute paths. Test that identical Bash command once in Git Bash.
For multiple compose files on Windows set `COMPOSE_PATH_SEPARATOR=';'` and
`COMPOSE_FILE='C:/path/base.yml;C:/path/override.yml'` in the command environment.
Do not schedule overlapping backup/restore runs.

A backup dumps PostgreSQL and archives the MinIO named volume through the
on-demand `backup-tools` Alpine container to archive or restore the volume from a separate container; restore stops MinIO first.
Retention removes only top-level regular files matching `backup_*.sql.gz`
and `minio_*.tar.gz` whose mtime is over 30 whole days old; it leaves nested
archives and unrelated files alone; archive names alone do not determine age. This is a sequential
backup, not a transaction across PostgreSQL and MinIO. For a consistent recovery
point, arrange a maintenance window (stop the API and other writers, leave DB
and MinIO running) around the scheduled job. Do not delete attachments during
backup. Archives are staged as .partial files and published only after both exports
succeed; failed exports remove their partial files. Keep the same timestamp
pair together; verify both with `gzip -t` and the file archive with `tar -tzf`.

Restore (SYS-FR-004): **this replaces the target database**. Use a new, isolated
compose project with empty volumes and the same application/image version that
made the backup. Never aim a drill at the production project. Start its stack,
then run:

```sh
docker compose up -d --wait
bash scripts/restore.sh /absolute/backup_TIMESTAMP.sql.gz /absolute/minio_TIMESTAMP.tar.gz
curl -f http://localhost/health
```

Restore resolves both input paths against the caller's directory, then changes
to the server directory, so absolute or caller-relative archives also work when
invoked elsewhere. Before stopping the API it checks gzip integrity, the pg_dump
header and completion footer, tar block length and `tar -tzf`, and the resolved
compose configuration including the operations-profile archive helper. Empty
or row-boundary-truncated SQL dumps are refused before any Docker command.
Restore then stops the API, drops and recreates the database (removing startup
migrations), uses psql `--single-transaction` with `ON_ERROR_STOP`, stops MinIO,
unpacks its archive via `backup-tools`, starts MinIO and creates/starts the API
with `up -d --no-deps api`, and waits up to 60 seconds for health. On failure the script exits nonzero;
inspect logs and leave the API stopped until recovery is resolved. Invalid gzip, incomplete pg_dump framing, malformed tar, or missing compose
configuration/helper is refused before stopping anything. Later SQL/extraction
failures leave the API stopped; a failing SQL restore rolls back its transaction. Only restore trusted operator-made
archives; restore is not an untrusted archive importer. File restore is intended
for empty target volumes, not a merge into a used MinIO volume. Redis is not
restored: members sign in again. Repeat the restore only with the API quiescent.

Before each release record date, paired backup names, users/guilds/messages row
counts before and after, health and an attachment download in
`docs/operations/restore-drills.md` (SYS-NFR-006). Use
`SELECT (SELECT count(*) FROM users), (SELECT count(*) FROM guilds), (SELECT count(*) FROM messages);`.
Verify restored stored bytes with authenticated MinIO `mc cat`, and verify client-facing
downloads through fresh 15-minute HTTPS signed GET URLs from message history.
D-10 is fixed by B6; expired pre-restore links should be renewed by fetching history.

Repeat script regressions with `bash scripts/test-operations.sh` and
`python scripts/test-monitoring.py` from `retrocast/`.

Performance (SYS-NFR-001/002/007): on an isolated disposable stack configured for
open registration and Caddy configured for the URL under test, run
`go run ./cmd/perf -base http://localhost:18300 -duration 10m`
from `retrocast/`, or the same command in `golang:1.26.11` with that directory
mounted at `/app`, `-w /app`, and the stack's compose network, using
`-base http://caddy`. For that isolated HTTP fixture set Caddy environment
`DOMAIN: ":80"` in a compose override (the default localhost site uses HTTPS).
The complete measurement override is in docs/operations/performance.md. The HTTP
and gateway traffic goes through real Caddy.
It creates 20 unique users and one guild, accepts invites, seeds 1000 messages
through REST, waits for 20 READY events, measures 100 random-cursor pages of 50
(seed 1), and runs 20 simultaneous senders every 10 seconds for 10 minutes.
Each attempted message requires 19 distinct peer receipts; errors, duplicate
receipts, disconnects and loss fail the command, as do p95 >1s or history >300ms.
Latency starts before POST and ends when each peer receives MESSAGE_CREATE,
including HTTP and gateway time; the sender's own echo is excluded. A 5s final
drain counts late receipts in latency. Setup honors 429 Retry-After, distributes
seed writes across users and refreshes credentials before measurement. Setup rejects credentials whose
remaining lifetime cannot cover the measured run plus one minute. Measured calls
never retry or change server limits. An additional 100-message phase from one
sender, spaced 1.3s apart after a minute cooldown, checks SYS-NFR-002 explicitly.
Credential refresh before measurement also honors Retry-After; the single sender
refreshes again before its separate phase. Default setup plus load takes about 23
minutes. Use a fresh stack for repeated runs: fixture users/data are left behind.
Results and limitations belong in `docs/operations/performance.md`.

MinIO image (issue #77): Compose builds `Dockerfile.minio` from the official
server `RELEASE.2025-10-15T17-29-55Z` and mc `RELEASE.2025-08-13T08-35-41Z`
source releases. Their exact module versions and checksums are pinned in that
file and verified through the Go sum database. Docker Hub's removed MinIO images
are not required. Run `docker compose up -d --build` on upgrade; the existing
`minio-data` volume, root credentials, region and internal ports remain in use.
CI builds this image to catch unavailable sources or build failures. Source
builds retain MinIO's AGPLv3 license. Upstream is archived; choosing a maintained
replacement is a separate product decision. Backup and restore commands above
still archive and restore the complete data volume, including IAM configuration.

The signer is a regular MinIO user, not a root-derived service account. `mc cat`
may fail its bucket-location discovery because s3:GetBucketLocation is deliberately
not granted. The API sets the region explicitly and signs GET locally; do not
broaden the signer policy to support diagnostic client discovery.

Use the ordinary Allow statement without NotAction: the pinned server can accept
a NotAction policy at runtime but fails to reload it from disk. Restart and restore
checks must include a real signer GET, not only root-based /health.

Startup secret policy (SV-NFR-002, #79): every configured secret must have at
least 32 characters and must not equal a committed placeholder. The API checks
JWT_SECRET, MINIO_SECRET_KEY, MINIO_SIGNER_SECRET_KEY and LIVEKIT_API_SECRET,
plus passwords embedded in DATABASE_URL or REDIS_URL. URL userinfo requires a
password. Only postgres/postgresql and redis/rediss connection URLs are supported;
keyword DSNs are refused by variable name. Key IDs are
identifiers, not secrets. The Compose `env-check` subcommand additionally checks
POSTGRES_PASSWORD, MINIO_ROOT_PASSWORD and GRAFANA_ADMIN_PASSWORD, requires the
MinIO root/write secret pair to match, and gates API, Postgres, MinIO, LiveKit and
Grafana startup. It runs the same repository API image without opening connections.
Errors name variables only. No development bypass exists. Committed CI/test secrets
are also refused; positive startup tests use fresh random values and the generator test proves real generated values work.

New installations, from `retrocast/`:

```sh
bash scripts/init-env.sh
docker compose up -d --build
```

The script requires Bash and openssl, refuses an existing destination, prints no
values and writes `.env` with mode 600. It generates 33 characters (a letter plus 32 hexadecimal characters)
for each secret and 20 for key IDs. These fit the pinned MinIO credential rules
and LiveKit's YAML string map. MINIO_ROOT_USER in both MinIO services comes from
the same MINIO_ACCESS_KEY used by the API write client. The dedicated signer
uses MINIO_SIGNER_ACCESS_KEY / MINIO_SIGNER_SECRET_KEY. Protect the environment
file and its backups. MSYS, MINGW and Cygwin cannot enforce mode 600 through
NTFS: the script prints a warning there; restrict the file ACL to the operator.

Existing deployments whose secrets already have at least 32 characters and are
not committed values keep them; do not run the init script over the deployed
environment. Placeholder deployments now refuse to start. Add the required
GRAFANA_ADMIN_PASSWORD even when Grafana was initialized previously. Plan a
maintenance window and take a private backup before rotation. Run the following
in order, using the same project, volume names and Compose overrides throughout:

1. On the running old revision, record the existing service container IDs before
   changing code: `db_id=$(docker compose ps -q db)`,
   `redis_id=$(docker compose ps -q redis)` and
   `grafana_id=$(docker compose ps -q grafana)`. Stop API and LiveKit with
   `docker compose stop api livekit`; stop other application writers. Check out
   the approved new revision while those data services keep running. Prepare a
   private replacement file with `bash scripts/init-env.sh /private/new.env`;
   retain operator settings (DOMAIN, origins, registration, region and endpoints).
   Keep the deployed old `.env` in place until step 5. Do not invoke the new
   Compose configuration with that old file: interpolation and env-check refuse it.
2. Postgres: POSTGRES_PASSWORD applies only on volume initialization. Connect to
   the already running container with `docker exec -it "$db_id" psql -U retrocast retrocast`
   and run `\password retrocast`. Enter the replacement POSTGRES_PASSWORD privately
   twice; this issues ALTER ROLE inside the existing volume. Update any external
   DATABASE_URL to the same password. Do not recreate the database volume.
3. JWT: the new JWT_SECRET ends access-token sessions. Clear refresh tokens on
   the running old Redis with `docker exec "$redis_id" redis-cli FLUSHDB`.
   This database is dedicated to Retrocast transient state (ADR-018); all users
   must sign in again. Grafana's environment password only initializes a new
   database: run `docker exec -i "$grafana_id" grafana cli admin reset-admin-password --password-from-stdin`
   and supply the replacement GRAFANA_ADMIN_PASSWORD privately on stdin. This
   covers #88's default Grafana password; its LiveKit token lifetime remains open.
4. Confirm the replacement file changes MINIO_ACCESS_KEY and both
   MINIO_SECRET_KEY / MINIO_ROOT_PASSWORD together (the two secrets must match),
   plus LIVEKIT_API_KEY / LIVEKIT_API_SECRET and the dedicated signer pair.
   The pinned October MinIO release reads root credentials on startup even on
   existing volumes. LiveKit's recreated service rejects old tokens; voice sessions
   may reconnect. Record the previous signer key ID if this version had one.
5. Install the replacement as `.env` with private permissions, then run
   `docker compose up -d --build --force-recreate`. This recreates MinIO and
   minio-init together with the new root credentials; init updates the new signer
   and policy. Check env-check exits 0. Do not start a partial new configuration
   before installing the replacement file.
6. If the signer key ID changed, disable the old account now using
   `mc admin user disable local <old-signer-key-id>` from a private admin context
   authenticated with the new root credentials. Do not put secrets on mc argv;
   use MC_HOST_local in the environment with the raw supported credentials.
   A b1514a3 installation has no dedicated signer to disable. Confirm `/health`
   200, sign in again, upload an attachment and verify exact signed-download bytes.

The Postgres and MinIO rotations and rollback/forward sequence are verified on
volumes created under b1514a3's placeholder configuration. Changing `.env` alone
does not rotate an existing Postgres or Grafana database password.
