#!/bin/bash
set -euo pipefail
# Schedulers have a minimal PATH and no useful working directory.
export PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:${PATH:-}"
cd "$(dirname "$0")/.."
: "${BACKUP_DIR:?Set BACKUP_DIR to an absolute directory outside the checkout}"
case "$BACKUP_DIR" in /*|[A-Za-z]:/*) ;; *) echo 'BACKUP_DIR must be absolute' >&2; exit 1;; esac
umask 077
exec bash scripts/backup.sh
