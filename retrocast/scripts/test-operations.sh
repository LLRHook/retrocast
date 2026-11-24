#!/bin/bash
# Hermetic regression: bash scripts/test-operations.sh (GNU test utilities).
set -euo pipefail
cd "$(dirname "$0")/.."
repo="$PWD"
tmp=$(mktemp -d -t l30-operations.XXXXXX)
trap 'rm -rf "$tmp"' EXIT
export CALLS="$tmp/calls" GZIP_CALLS="$tmp/gzip-calls" COMPOSE_ROOT="$repo"
export REAL_GZIP
REAL_GZIP=$(command -v gzip)
export PATH="$tmp:$PATH"
cat > "$tmp/gzip" <<'MOCK'
#!/bin/bash
printf '%s\n' "$*" >> "$GZIP_CALLS"
exec "$REAL_GZIP" "$@"
MOCK
cat > "$tmp/docker" <<'MOCK'
#!/bin/bash
set -euo pipefail
printf '%s\n' "$*" >> "$CALLS"
[ "$PWD" = "$COMPOSE_ROOT" ]
case "$*" in
  'compose --profile operations config --services')
    [ "${FAIL_CONFIG:-0}" != 1 ]
    printf '%s\n' api db minio
    if [ "${MISSING_TOOLS:-0}" != 1 ]; then echo backup-tools; fi;;
  *'pg_dump'*) printf '%s\n' '--' '-- PostgreSQL database dump' '--' 'SELECT 1;' '-- PostgreSQL database dump complete' '--';;
  *'psql'*) cat >/dev/null; [ "${FAIL_SQL:-0}" != 1 ];;
  *'backup-tools tar -C /data -cf - .'*) [ "${FAIL_ARCHIVE:-0}" != 1 ]; tar -cf - --files-from /dev/null;;
  *'backup-tools tar -C /data -xf -'*) cat >/dev/null;;
esac
MOCK
chmod +x "$tmp/docker" "$tmp/gzip"

assert_before() {
    local before after
    before=$(grep -nF -- "$1" "$CALLS" | cut -d: -f1)
    after=$(grep -nF -- "$2" "$CALLS" | cut -d: -f1)
    [ -n "$before" ]
    [ -n "$after" ]
    [ "$before" -lt "$after" ]
}
assert_refused() {
    : > "$CALLS"
    if bash scripts/restore.sh "$1" "$2" >/dev/null 2>&1; then exit 1; fi
    [ ! -s "$CALLS" ]
}

# A permissive parent umask makes removal of the wrapper's privacy setting fail.
umask 000
export BACKUP_DIR="$tmp/backups"
mkdir "$BACKUP_DIR"
mkdir "$BACKUP_DIR/nested"
for name in backup minio; do
    if [ "$name" = backup ]; then suffix=sql.gz; else suffix=tar.gz; fi
    touch -d '31 days ago' "$BACKUP_DIR/${name}_old.$suffix"
    touch -d '29 days ago' "$BACKUP_DIR/${name}_keep.$suffix"
    touch -d '30 days 23 hours ago' "$BACKUP_DIR/${name}_boundary.$suffix"
    touch -d '31 days ago' "$BACKUP_DIR/nested/${name}_nested.$suffix"
done
touch -d '31 days ago' "$BACKUP_DIR/unrelated.txt"
bash scripts/scheduled-backup.sh >/dev/null
[ ! -f "$BACKUP_DIR/backup_old.sql.gz" ]
[ ! -f "$BACKUP_DIR/minio_old.tar.gz" ]
[ -f "$BACKUP_DIR/backup_keep.sql.gz" ]
[ -f "$BACKUP_DIR/minio_keep.tar.gz" ]
[ -f "$BACKUP_DIR/backup_boundary.sql.gz" ]
[ -f "$BACKUP_DIR/minio_boundary.tar.gz" ]
[ -f "$BACKUP_DIR/nested/backup_nested.sql.gz" ]
[ -f "$BACKUP_DIR/nested/minio_nested.tar.gz" ]
[ -f "$BACKUP_DIR/unrelated.txt" ]
grep -q 'backup-tools tar -C /data -cf - .' "$CALLS"
echo 'PASS retention: 31d deleted, 29d and 30d23h retained, nested/non-archives untouched'
db=$(find "$BACKUP_DIR" -maxdepth 1 -name 'backup_20*' | head -1)
files=$(find "$BACKUP_DIR" -maxdepth 1 -name 'minio_20*' | head -1)
[ "$(stat -c '%a' "$db")" = 600 ]
[ "$(stat -c '%a' "$files")" = 600 ]
BACKUP_DIR="$tmp/private" bash scripts/scheduled-backup.sh >/dev/null
[ "$(stat -c '%a' "$tmp/private")" = 700 ]
echo 'PASS scheduled backup archive/directory permissions'

: > "$CALLS"
: > "$GZIP_CALLS"
bash scripts/restore.sh "$db" "$files" >/dev/null
# Explicit checksum preflight is observable even though framing also decompresses.
grep -Fxq -- "-t $db" "$GZIP_CALLS"
grep -Fxq -- "-t $files" "$GZIP_CALLS"
assert_before 'compose --profile operations config --services' 'compose stop api'
assert_before 'compose stop api' 'compose exec -T db dropdb'
assert_before 'compose exec -T db dropdb' 'compose exec -T db createdb'
assert_before 'compose exec -T db createdb' 'compose exec -T db psql'
grep -q 'psql --single-transaction -v ON_ERROR_STOP=1' "$CALLS"
assert_before 'compose stop minio' 'backup-tools tar -C /data -xf -'
assert_before 'compose exec -T db psql' 'compose up -d --no-deps api'
assert_before 'compose up -d --no-deps api' 'compose exec -T api sh -c'
grep -q '/health' "$CALLS"
echo 'PASS ordered preflight, transactional restore, API creation and subsequent health wait'

: > "$CALLS"
if FAIL_SQL=1 bash scripts/restore.sh "$db" "$files" >/dev/null 2>&1; then exit 1; fi
if grep -Eq 'compose (start api|up -d --no-deps api)' "$CALLS"; then exit 1; fi
echo 'PASS SQL failure leaves API stopped'

printf broken > "$tmp/broken.gz"
assert_refused "$tmp/broken.gz" "$files"
echo 'PASS broken SQL gzip refused before any Docker call'
assert_refused "$db" "$tmp/broken.gz"
echo 'PASS broken MinIO gzip refused before any Docker call'
printf '' | gzip > "$tmp/empty.sql.gz"
assert_refused "$tmp/empty.sql.gz" "$files"
echo 'PASS empty SQL gzip refused before any Docker call'
printf '%s\n' '--' '-- PostgreSQL database dump' '--' 'COPY public.users (id) FROM stdin;' '1' | gzip > "$tmp/cut.sql.gz"
assert_refused "$tmp/cut.sql.gz" "$files"
echo 'PASS SQL cut at a COPY row boundary refused before any Docker call'
printf '%s\n' 'SELECT 1;' '-- PostgreSQL database dump complete' | gzip > "$tmp/no-header.sql.gz"
assert_refused "$tmp/no-header.sql.gz" "$files"
echo 'PASS non-pg_dump SQL refused before any Docker call'
printf 'not a tar archive' | gzip > "$tmp/short-tar.tar.gz"
assert_refused "$db" "$tmp/short-tar.tar.gz"
echo 'PASS short non-tar stream refused before any Docker call'
printf '%01024d' 0 | gzip > "$tmp/not-tar.tar.gz"
assert_refused "$db" "$tmp/not-tar.tar.gz"
echo 'PASS gzip-valid non-tar refused before any Docker call'

for setting in FAIL_CONFIG MISSING_TOOLS; do
    : > "$CALLS"
    if env "$setting=1" bash scripts/restore.sh "$db" "$files" >/dev/null 2>&1; then exit 1; fi
    [ "$(wc -l < "$CALLS")" -eq 1 ]
    grep -Fxq 'compose --profile operations config --services' "$CALLS"
done
echo 'PASS invalid Compose configuration/missing archive helper refused before mutation'

: > "$CALLS"
(cd "$tmp" && bash "$repo/scripts/restore.sh" "backups/$(basename "$db")" "backups/$(basename "$files")") >/dev/null
grep -q 'compose up -d --no-deps api' "$CALLS"
echo 'PASS restore from outside server directory with caller-relative input paths'
if BACKUP_DIR=relative bash scripts/scheduled-backup.sh >/dev/null 2>&1; then exit 1; fi
echo 'PASS relative scheduled backup directory rejected'
if BACKUP_DIR="$tmp/failed" FAIL_ARCHIVE=1 bash scripts/scheduled-backup.sh >/dev/null 2>&1; then exit 1; fi
[ -z "$(find "$tmp/failed" -type f -print -quit)" ]
echo 'PASS archive failure publishes no backup pair or partial files'
