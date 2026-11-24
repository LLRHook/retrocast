#!/bin/bash
# Run with bash scripts/test-operations.sh (no live Docker required).
set -euo pipefail
cd "$(dirname "$0")/.."
tmp=$(mktemp -d -t l30-operations.XXXXXX)
trap 'rm -rf "$tmp"' EXIT
export CALLS="$tmp/calls" PATH="$tmp:$PATH"
cat > "$tmp/docker" <<'MOCK'
#!/bin/bash
set -euo pipefail
printf '%s\n' "$*" >> "$CALLS"
case "$*" in
  *'pg_dump'*) echo '-- fixture dump';;
  *'psql'*) cat >/dev/null; [ "${FAIL_SQL:-0}" != 1 ];;
  *'backup-tools tar -C /data -cf - .'*) tar -cf - --files-from /dev/null;;
  *'backup-tools tar -C /data -xf -'*) cat >/dev/null;;
esac
MOCK
chmod +x "$tmp/docker"
export BACKUP_DIR="$tmp/backups"
mkdir "$BACKUP_DIR"
for name in backup minio; do
  if [ "$name" = backup ]; then suffix=sql.gz; else suffix=tar.gz; fi
  touch -d '31 days ago' "$BACKUP_DIR/${name}_old.$suffix"
  touch -d '29 days ago' "$BACKUP_DIR/${name}_keep.$suffix"
done
bash scripts/scheduled-backup.sh > /dev/null
[ ! -f "$BACKUP_DIR/backup_old.sql.gz" ] && [ ! -f "$BACKUP_DIR/minio_old.tar.gz" ]
[ -f "$BACKUP_DIR/backup_keep.sql.gz" ] && [ -f "$BACKUP_DIR/minio_keep.tar.gz" ]
echo 'PASS retention and scheduled entrypoint'
db=$(find "$BACKUP_DIR" -name 'backup_20*' | head -1)
files=$(find "$BACKUP_DIR" -name 'minio_20*' | head -1)
: > "$CALLS"
bash scripts/restore.sh "$db" "$files" >/dev/null
sed -n '1p' "$CALLS" | grep -qx 'compose stop api'
grep -q 'dropdb -U retrocast --force --if-exists retrocast' "$CALLS"
grep -q 'createdb -U retrocast -T template0 retrocast' "$CALLS"
grep -q 'psql -v ON_ERROR_STOP=1' "$CALLS"
grep -q 'compose stop minio' "$CALLS"
grep -q 'backup-tools tar -C /data -xf -' "$CALLS"
grep -q 'compose start api' "$CALLS"
echo 'PASS restore stops writers, replaces schema, restores files, checks health'
: > "$CALLS"
if FAIL_SQL=1 bash scripts/restore.sh "$db" "$files" >/dev/null 2>&1; then exit 1; fi
! grep -q 'compose start api' "$CALLS"
echo 'PASS SQL failure leaves API stopped'
: > "$CALLS"
printf broken > "$tmp/broken.gz"
if bash scripts/restore.sh "$db" "$tmp/broken.gz" >/dev/null 2>&1; then exit 1; fi
[ ! -s "$CALLS" ]
echo 'PASS invalid archive rejected before any stack change'
if BACKUP_DIR=relative bash scripts/scheduled-backup.sh >/dev/null 2>&1; then exit 1; fi
echo 'PASS relative scheduled backup directory rejected'
