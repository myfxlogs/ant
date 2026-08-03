#!/bin/bash
# PostgreSQL backup script for AlphaForge.
# Usage: ./scripts/backup-db.sh [--keep-days 7] [--keep-weeks 4]
# Intended to run via cron: 0 2 * * * /opt/ant/scripts/backup-db.sh
#
# Backups: local rotation (7d daily + 4w weekly) + Cloudflare R2 upload via rclone.
# R2 config: rclone remote "r2" must be pre-configured (rclone config).
# Env vars:
#   R2_REMOTE    - rclone remote name (default: r2)
#   R2_BUCKET    - R2 bucket name (default: alphaforge-backups)
#   RCLONE_ENABLED - set to "0" to skip R2 upload (default: "1")

set -euo pipefail

KEEP_DAYS=7
KEEP_WEEKS=4
BACKUP_DIR="${BACKUP_DIR:-/opt/ant/backups}"
COMPOSE_FILE="${COMPOSE_FILE:-/opt/ant/docker-compose.yml}"
CONTAINER="${CONTAINER:-postgres}"
DB_NAME="${DB_NAME:-ant}"
DB_USER="${DB_USER:-ant}"
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
R2_REMOTE="${R2_REMOTE:-r2}"
R2_BUCKET="${R2_BUCKET:-alphaforge-backups}"
RCLONE_ENABLED="${RCLONE_ENABLED:-1}"

while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-days) KEEP_DAYS="$2"; shift 2 ;;
        --keep-weeks) KEEP_WEEKS="$2"; shift 2 ;;
        *) echo "Unknown arg: $1"; exit 1 ;;
    esac
done

mkdir -p "$BACKUP_DIR"

BACKUP_FILE="${BACKUP_DIR}/ant_${TIMESTAMP}.sql.gz"

echo "[$(date -Iseconds)] Starting backup → $BACKUP_FILE"

docker compose -f "$COMPOSE_FILE" exec -T "$CONTAINER" \
    pg_dump -U "$DB_USER" -d "$DB_NAME" --no-owner --no-acl \
    --exclude-table='md_ticks_*' \
    | gzip > "$BACKUP_FILE"

if [[ -s "$BACKUP_FILE" ]]; then
    echo "[$(date -Iseconds)] Backup OK ($(du -h "$BACKUP_FILE" | cut -f1))"
else
    echo "[$(date -Iseconds)] Backup FAILED — empty file" >&2
    rm -f "$BACKUP_FILE"
    exit 1
fi

# Upload to Cloudflare R2 (if rclone is available and enabled)
if [[ "$RCLONE_ENABLED" == "1" ]] && command -v rclone &>/dev/null; then
    echo "[$(date -Iseconds)] Uploading to R2 (${R2_REMOTE}:${R2_BUCKET}/)..."
    if rclone copy "$BACKUP_FILE" "${R2_REMOTE}:${R2_BUCKET}/daily/" --quiet; then
        echo "[$(date -Iseconds)] R2 upload OK"
    else
        echo "[$(date -Iseconds)] R2 upload FAILED (non-fatal, local backup exists)" >&2
    fi
else
    echo "[$(date -Iseconds)] R2 upload skipped (rclone not found or disabled)"
fi

# Local rotation: keep 7 daily + 4 weekly (Sunday)
find "$BACKUP_DIR" -name "ant_*.sql.gz" -mtime +"$KEEP_DAYS" -delete 2>/dev/null || true

# Weekly archive: copy Sunday backups to weekly/ and keep 4 weeks
DOW=$(date -u +%u)  # 1=Mon, 7=Sun
if [[ "$DOW" == "7" ]]; then
    WEEKLY_DIR="${BACKUP_DIR}/weekly"
    mkdir -p "$WEEKLY_DIR"
    cp "$BACKUP_FILE" "${WEEKLY_DIR}/"
    find "$WEEKLY_DIR" -name "ant_*.sql.gz" -mtime +"$((KEEP_WEEKS * 7))" -delete 2>/dev/null || true
    # Upload weekly to R2
    if [[ "$RCLONE_ENABLED" == "1" ]] && command -v rclone &>/dev/null; then
        rclone copy "${WEEKLY_DIR}/$(basename "$BACKUP_FILE")" "${R2_REMOTE}:${R2_BUCKET}/weekly/" --quiet 2>/dev/null || true
    fi
fi

echo "[$(date -Iseconds)] Backup rotation done (keep ${KEEP_DAYS}d daily + ${KEEP_WEEKS}w weekly)"
