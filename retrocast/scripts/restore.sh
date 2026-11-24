#!/bin/bash
set -euo pipefail

BACKUP_FILE="${1:?Usage: restore.sh <backup_file.sql.gz> [minio_backup.tar.gz]}"
MINIO_FILE="${2:-}"

# Resolve input paths against the caller's directory before changing to Compose.
absolute_file() {
    local directory filename
    directory=$(dirname -- "$1")
    filename=$(basename -- "$1")
    (cd -- "$directory" && printf '%s/%s\n' "$(pwd -P)" "$filename")
}
BACKUP_FILE=$(absolute_file "$BACKUP_FILE")
if [ -n "$MINIO_FILE" ]; then MINIO_FILE=$(absolute_file "$MINIO_FILE"); fi
cd "$(dirname "$0")/.."

# Validate archives completely before any Docker command. A valid gzip can still
# contain an empty/truncated SQL dump (psql accepts EOF at a COPY row boundary).
gzip -t "${BACKUP_FILE}"
gunzip -c "${BACKUP_FILE}" | awk '
    NR == 1 && $0 == "--" { next }
    !header {
        if ($0 != "-- PostgreSQL database dump") exit 1
        header = 1
        next
    }
    $0 == "-- PostgreSQL database dump complete" { complete = 1 }
    END { if (!header || !complete) exit 1 }
' || { echo 'Not a complete pg_dump SQL backup' >&2; exit 1; }
if [ -n "${MINIO_FILE}" ]; then
    gzip -t "${MINIO_FILE}"
    # Some tar implementations accept a short non-tar stream as empty input.
    archive_bytes=$(gunzip -c "${MINIO_FILE}" | wc -c)
    [ "$archive_bytes" -ge 1024 ]
    [ "$((archive_bytes % 512))" -eq 0 ]
    tar -tzf "${MINIO_FILE}" >/dev/null
fi

# Resolve configuration and the profile-only helper before interrupting writers.
services=$(docker compose --profile operations config --services)
for service in api db minio backup-tools; do
    if ! printf '%s\n' "$services" | grep -qx "$service"; then
        echo "Compose service missing: $service" >&2
        exit 1
    fi
done

docker compose stop api

# Startup migrations populated a fresh stack. Replace that database rather than
# merging a dump into its schema. On failure, leave the API stopped for recovery.
docker compose exec -T db dropdb -U retrocast --force --if-exists retrocast
docker compose exec -T db createdb -U retrocast -T template0 retrocast

echo "Restoring database from ${BACKUP_FILE}..."
gunzip -c "${BACKUP_FILE}" | docker compose exec -T db psql --single-transaction -v ON_ERROR_STOP=1 -U retrocast retrocast
echo "Database restored."

if [ -n "${MINIO_FILE}" ]; then
    echo "Restoring MinIO data from ${MINIO_FILE}..."
    docker compose stop minio
    gunzip -c "${MINIO_FILE}" | docker compose run --rm -T --no-deps backup-tools tar -C /data -xf -
    docker compose start minio
    echo "MinIO data restored."
fi

docker compose up -d --no-deps api
docker compose exec -T api sh -c 'for i in $(seq 1 60); do curl -fsS http://localhost:8080/health && exit 0; sleep 1; done; exit 1'
echo "Restore complete."
