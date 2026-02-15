#!/bin/bash
set -euo pipefail

BACKUP_FILE="${1:?Usage: restore.sh <backup_file.sql.gz> [minio_backup.tar.gz]}"
MINIO_FILE="${2:-}"

echo "Restoring database from ${BACKUP_FILE}..."
gunzip -c "${BACKUP_FILE}" | docker compose exec -T db psql -U retrocast retrocast
echo "Database restored."

if [ -n "${MINIO_FILE}" ]; then
    echo "Restoring MinIO data from ${MINIO_FILE}..."
    gunzip -c "${MINIO_FILE}" | docker compose exec -T minio sh -c 'cd /data && tar xf -'
    echo "MinIO data restored."
fi

echo "Restore complete."
