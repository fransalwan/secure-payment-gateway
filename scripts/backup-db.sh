#!/usr/bin/env bash
# ==============================================================================
# Enterprise PostgreSQL Automated Backup & Retention Script
# PT. Jiwa Muda Cipta Indonesia (JMC IT Consultant) - Standard Compliance Runbook
# ==============================================================================
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/secure-payment-gateway}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-spg-postgres}"
POSTGRES_DB="${POSTGRES_DB:-payment_gateway_db}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/spg_backup_${TIMESTAMP}.dump"

echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] Starting automated database backup for '${POSTGRES_DB}'..."

# Ensure target directory exists with strict permissions
mkdir -p "${BACKUP_DIR}"
chmod 700 "${BACKUP_DIR}"

# 1. Execute compressed custom-format pg_dump from within Docker container
docker exec -t "${POSTGRES_CONTAINER}" pg_dump \
  -U "${POSTGRES_USER}" \
  -d "${POSTGRES_DB}" \
  -F c \
  -b \
  -v > "${BACKUP_FILE}"

# Secure file permissions (owner read/write only)
chmod 600 "${BACKUP_FILE}"
FILESIZE=$(du -h "${BACKUP_FILE}" | cut -f1)

echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] Backup completed successfully: ${BACKUP_FILE} (${FILESIZE})"

# 2. Prune obsolete backups older than retention policy
echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] Pruning backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -type f -name "spg_backup_*.dump" -mtime +"${RETENTION_DAYS}" -exec rm -f {} \;

echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] Backup cycle finished with 0 errors."
