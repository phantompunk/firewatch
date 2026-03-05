#!/usr/bin/env bash
# =============================================================================
#  Firewatch — SQLite Backup
#  Backs up the database via the SQLite online backup API, compresses it,
#  rotates old backups, and sends an alert email on failure.
#
#  Usage: bash scripts/backup.sh [--backup-dir /path/to/backups]
#  Recommended: run daily via cron (registered automatically by deploy.sh)
# =============================================================================
set -euo pipefail

# ── Config ────────────────────────────────────────────────────────────────────
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/firewatch}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
CONTAINER="${CONTAINER_NAME:-firewatch-app-1}"
DB_PATH="${DB_PATH:-/data/firewatch.db}"
TIMESTAMP="$(date -u '+%Y%m%dT%H%M%SZ')"
BACKUP_FILE="$BACKUP_DIR/firewatch-$TIMESTAMP.db"
COMPRESSED="$BACKUP_FILE.gz"

# Alert email — reads from .env.docker if available, falls back to blank
ALERT_EMAIL=""
if [[ -f "$REPO_DIR/.env.docker" ]]; then
  ALERT_EMAIL="$(grep -E '^DESTINATION_EMAIL=' "$REPO_DIR/.env.docker" | cut -d= -f2- | tr -d '[:space:]')"
fi

# ── Helpers ───────────────────────────────────────────────────────────────────
log()  { echo "$(date -u '+%Y-%m-%dT%H:%M:%SZ')  $*"; }
fail() {
  log "ERROR: $*"
  if [[ -n "$ALERT_EMAIL" ]] && command -v mail &>/dev/null; then
    echo "Firewatch backup failed on $(hostname) at $(date -u).\n\nError: $*" \
      | mail -s "[Firewatch] Backup failed on $(hostname)" "$ALERT_EMAIL"
  fi
  exit 1
}

# ── Pre-flight ────────────────────────────────────────────────────────────────
mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

# Verify the container is running
docker inspect --format '{{.State.Running}}' "$CONTAINER" 2>/dev/null \
  | grep -q true \
  || fail "Container '$CONTAINER' is not running."

# ── Backup ────────────────────────────────────────────────────────────────────
log "Starting backup → $COMPRESSED"

# Use SQLite's .backup command — safe to run against a live database.
# The online backup API copies pages atomically without locking writers out.
docker exec "$CONTAINER" sqlite3 "$DB_PATH" ".backup /tmp/firewatch_backup.db" \
  || fail "sqlite3 .backup command failed inside container."

docker cp "$CONTAINER:/tmp/firewatch_backup.db" "$BACKUP_FILE" \
  || fail "docker cp failed to copy backup from container."

# Cleanup temp file inside container
docker exec "$CONTAINER" rm -f /tmp/firewatch_backup.db

# Compress
gzip -9 "$BACKUP_FILE" \
  || fail "gzip compression failed."

SIZE="$(du -sh "$COMPRESSED" | cut -f1)"
log "Backup complete: $COMPRESSED ($SIZE)"

# ── Optional offsite sync ─────────────────────────────────────────────────────
# Uncomment and configure if rclone is installed.
# RCLONE_REMOTE="${RCLONE_REMOTE:-}"
# if [[ -n "$RCLONE_REMOTE" ]] && command -v rclone &>/dev/null; then
#   log "Syncing to $RCLONE_REMOTE..."
#   rclone copy "$COMPRESSED" "$RCLONE_REMOTE/firewatch-backups/" \
#     || fail "rclone sync failed."
#   log "Offsite sync complete."
# fi

# ── Rotate ───────────────────────────────────────────────────────────────────
log "Rotating backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name 'firewatch-*.db.gz' \
  -mtime "+$RETENTION_DAYS" -delete

REMAINING="$(find "$BACKUP_DIR" -name 'firewatch-*.db.gz' | wc -l | tr -d ' ')"
log "Rotation done. $REMAINING backup(s) retained."
