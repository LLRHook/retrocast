# Operations

Run these commands from `retrocast/` with Docker Compose v2, Bash, gzip, awk, tar and GNU
find available. Use the same `COMPOSE_PROJECT_NAME`, `COMPOSE_FILE` and `.env`
as the deployed stack. Backups contain password hashes and private conversations;
keep the destination private, outside the checkout, and monitor job exit status.
Redis is intentionally excluded (ADR-018). Backups remain on this host.

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
on-demand `backup-tools` Alpine container, since MinIO's minimal image has no tar.
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
The current attachment URL limitation D-10 remains: an operator can verify stored
bytes with authenticated MinIO `mc cat`, but client-facing downloads remain a
separate requirement.

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
