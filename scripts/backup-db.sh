#!/bin/bash
# M6-2: PostgreSQL backup script for ant.
# Usage: ./scripts/backup-db.sh [--keep-days 7]
# Intended to run via cron: 0 2 * * * /opt/ant/scripts/backup-db.sh

set -euo pipefail

KEEP_DAYS=7
BACKUP_DIR="${BACKUP_DIR:-/opt/ant/backups}"
COMPOSE_FILE="${COMPOSE_FILE:-/opt/ant/docker-compose.yml}"
CONTAINER="${CONTAINER:-ant-postgres-1}"
DB_NAME="${DB_NAME:-ant}"
DB_USER="${DB_USER:-ant}"
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)

while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-days) KEEP_DAYS="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

mkdir -p "$BACKUP_DIR"

BACKUP_FILE="${BACKUP_DIR}/ant_${TIMESTAMP}.sql.gz"

echo "[$(date -Iseconds)] Starting backup → $BACKUP_FILE"

docker compose -f "$COMPOSE_FILE" exec -T "$CONTAINER" \
    pg_dump -U "$DB_USER" -d "$DB_NAME" --no-owner --no-acl \
    | gzip > "$BACKUP_FILE"

if [[ -s "$BACKUP_FILE" ]]; then
    echo "[$(date -Iseconds)] Backup OK ($(du -h "$BACKUP_FILE" | cut -f1))"
else
    echo "[$(date -Iseconds)] Backup FAILED — empty file" >&2
    rm -f "$BACKUP_FILE"
    exit 1
fi

# Cleanup old backups
find "$BACKUP_DIR" -name "ant_*.sql.gz" -mtime +"$KEEP_DAYS" -delete 2>/dev/null || true
echo "[$(date -Iseconds)] Backup rotation done (keep ${KEEP_DAYS}d)"
