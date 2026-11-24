# Restore drills

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
recovery; D-10 (compose-internal client attachment URLs) remains outside B5c.
No client-facing download claim is made.

The exact scheduled wrapper command ran successfully, producing both archives.
Retention fixture: both archive types with mtimes 31 days old were removed;
both with mtimes 29 days old remained. The committed regression reproduces this
with `bash scripts/test-operations.sh`. The same suite confirms SQL failure
leaves the API stopped and invalid gzip input changes no service state.

Repeat this drill before each release and append its actual timestamped backup
names, counts and download result here (SYS-NFR-006). Host cron, launchd and
Windows task installation and 24-hour wall-clock scheduling were not performed
on this Windows test host; their commands and templates are in docs/OPERATIONS.md.
