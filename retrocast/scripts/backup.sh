#!/bin/bash
set -euo pipefail
BACKUP_DIR="${BACKUP_DIR:-.}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
docker compose exec -T db pg_dump -U retrocast retrocast | gzip > "${BACKUP_DIR}/backup_${TIMESTAMP}.sql.gz"
echo "Backup saved: ${BACKUP_DIR}/backup_${TIMESTAMP}.sql.gz"
