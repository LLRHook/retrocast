#!/bin/bash
set -euo pipefail

BACKUP_FILE="${1:?Usage: restore.sh <backup_file.sql.gz> [minio_backup.tar.gz]}"
MINIO_FILE="${2:-}"

# Validate both inputs before interrupting the stack. Never resume writers after
# a failed restore: the operator must resolve it before starting the API.
gzip -t "${BACKUP_FILE}"
if [ -n "${MINIO_FILE}" ]; then gzip -t "${MINIO_FILE}"; fi
docker compose stop api

# Startup migrations already populated a fresh stack. Replace that database
# rather than merging a dump into its schema (psql otherwise ignores errors).
docker compose exec -T db dropdb -U retrocast --force --if-exists retrocast
docker compose exec -T db createdb -U retrocast -T template0 retrocast

echo "Restoring database from ${BACKUP_FILE}..."
gunzip -c "${BACKUP_FILE}" | docker compose exec -T db psql -v ON_ERROR_STOP=1 -U retrocast retrocast
echo "Database restored."

if [ -n "${MINIO_FILE}" ]; then
    echo "Restoring MinIO data from ${MINIO_FILE}..."
    docker compose stop minio
    gunzip -c "${MINIO_FILE}" | docker compose run --rm -T --no-deps backup-tools tar -C /data -xf -
    docker compose start minio
    echo "MinIO data restored."
fi

docker compose start api
docker compose exec -T api sh -c 'for i in $(seq 1 60); do curl -fsS http://localhost:8080/health && exit 0; sleep 1; done; exit 1'
echo "Restore complete."
