#!/bin/bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-.}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

mkdir -p "${BACKUP_DIR}"

# Database backup
DB_FILE="${BACKUP_DIR}/backup_${TIMESTAMP}.sql.gz"
echo "Backing up database..."
docker compose exec -T db pg_dump -U retrocast retrocast | gzip > "${DB_FILE}"
echo "Database backup saved: ${DB_FILE}"

# MinIO data backup
MINIO_FILE="${BACKUP_DIR}/minio_${TIMESTAMP}.tar.gz"
echo "Backing up MinIO data..."
docker compose exec -T minio sh -c 'cd /data && tar cf - .' | gzip > "${MINIO_FILE}"
echo "MinIO backup saved: ${MINIO_FILE}"

# Retention: delete backups older than 30 days
echo "Cleaning up backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -name "backup_*.sql.gz" -mtime +${RETENTION_DAYS} -delete
find "${BACKUP_DIR}" -name "minio_*.tar.gz" -mtime +${RETENTION_DAYS} -delete

# Summary
DB_SIZE=$(du -h "${DB_FILE}" | cut -f1)
MINIO_SIZE=$(du -h "${MINIO_FILE}" | cut -f1)
echo ""
echo "=== Backup Summary ==="
echo "Timestamp:     ${TIMESTAMP}"
echo "Database:      ${DB_FILE} (${DB_SIZE})"
echo "MinIO:         ${MINIO_FILE} (${MINIO_SIZE})"
echo "Retention:     ${RETENTION_DAYS} days"
echo "=== Backup Complete ==="
