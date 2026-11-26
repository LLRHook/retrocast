# Operations

Run these commands from `retrocast/` with Docker Compose v2, Bash, gzip, awk, tar and GNU
find available. Use the same `COMPOSE_PROJECT_NAME`, `COMPOSE_FILE` and `.env`
as the deployed stack. Backups contain password hashes and private conversations;
keep the destination private, outside the checkout, and monitor job exit status.
Redis is intentionally excluded (ADR-018). Backups remain on this host.

Attachment signing (SV-NFR-008, ADR-019): before upgrading, add
`MINIO_SIGNER_ACCESS_KEY` and `MINIO_SIGNER_SECRET_KEY` to `.env`, then run the
usual `docker compose up -d --build`. Use a new account ID distinct from the
root user and `MINIO_ACCESS_KEY`. `.env.example` includes development placeholders.
Missing or empty signer credentials fail Compose and API storage initialization
in every environment; there is no root fallback.

The one-shot `minio-init` uses Compose's existing root credentials, waits for MinIO,
creates/updates the signer user and policy and attaches it. The API waits for
successful completion; rerunning `up` repeats initialization safely. The policy
allows only `s3:GetObject` on `arn:aws:s3:::retrocast/*`; explicit denies bound
other grants. No list, put, delete, other-bucket reads or admin actions are allowed.
Uploads and deletion use the original write client. Neither secret is logged;
the signer access key ID appears in signed URLs as expected for SigV4.

Keep the two variables with the deployment configuration on restore. The MinIO
volume archive includes `.minio.sys` IAM configuration; backup/restore scripts
and the isolated restore-drill procedure remain unchanged. After restoring an
older archive without the account, run `docker compose up -d minio-init` before
starting the API. Rollback to `b1514a3` ignores the new account and variables and
uses its original signer; keep original root credentials unchanged.
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
