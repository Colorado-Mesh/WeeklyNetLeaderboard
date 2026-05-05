#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-./data/meshmonday_dev.db}"
BACKUP_DIR="${2:-./backups}"
TIMESTAMP="$(date +"%Y%m%d-%H%M%S")"

mkdir -p "${BACKUP_DIR}"
cp "${DB_PATH}" "${BACKUP_DIR}/meshmonday-${TIMESTAMP}.db"
echo "backup written to ${BACKUP_DIR}/meshmonday-${TIMESTAMP}.db"
