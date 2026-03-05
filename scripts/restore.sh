#!/usr/bin/env bash
# =============================================================================
#  Firewatch — SQLite Restore
#  Restores a compressed backup (.db.gz) to the running container's data volume.
#
#  Usage: sudo bash scripts/restore.sh <path-to-backup.db.gz>
#
#  WARNING: This stops the application briefly during the restore.
# =============================================================================
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTAINER="${CONTAINER_NAME:-firewatch-app-1}"
DB_PATH="${DB_PATH:-/data/firewatch.db}"

# ── Helpers ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
AMBER='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

ok()  { echo -e "  ${GREEN}✓${NC}  $*"; }
err() { echo -e "\n  ${RED}✗  $*${NC}\n" >&2; exit 1; }
say() { echo -e "  $*"; }

# ── Validate args ─────────────────────────────────────────────────────────────
BACKUP_FILE="${1:-}"
[[ -n "$BACKUP_FILE" ]] || err "Usage: sudo bash scripts/restore.sh <path-to-backup.db.gz>"
[[ -f "$BACKUP_FILE" ]] || err "Backup file not found: $BACKUP_FILE"
[[ "$BACKUP_FILE" == *.gz ]] || err "Expected a .db.gz file. Got: $BACKUP_FILE"

[[ $EUID -eq 0 ]] || err "Run this script as root: sudo bash scripts/restore.sh <backup>"

echo
echo -e "${AMBER}  Firewatch — Database Restore${NC}"
echo
say "Backup file : ${BOLD}$BACKUP_FILE${NC}"
say "Target      : ${BOLD}$DB_PATH${NC} inside container ${BOLD}$CONTAINER${NC}"
echo
echo -e "  ${RED}WARNING: The application will be stopped briefly during the restore.${NC}"
echo
printf "  ${BOLD}Continue?${NC} [y/N]: "
read -r answer
[[ "$answer" =~ ^[Yy]$ ]] || { say "Cancelled."; exit 0; }
echo

# ── Stop container ────────────────────────────────────────────────────────────
say "Stopping Firewatch..."
cd "$REPO_DIR"
docker compose stop app
ok "Container stopped"

# ── Decompress to temp file ───────────────────────────────────────────────────
TMPFILE="$(mktemp /tmp/firewatch_restore_XXXXXX.db)"
trap 'rm -f "$TMPFILE"' EXIT

say "Decompressing backup..."
gunzip -c "$BACKUP_FILE" > "$TMPFILE" \
  || err "Failed to decompress backup file."
ok "Decompressed: $(du -sh "$TMPFILE" | cut -f1)"

# ── Validate the restored db is readable ─────────────────────────────────────
if command -v sqlite3 &>/dev/null; then
  sqlite3 "$TMPFILE" "PRAGMA integrity_check;" | grep -q "ok" \
    || err "Integrity check failed on the backup file. Aborting restore."
  ok "Integrity check passed"
fi

# ── Copy into the container's data volume ────────────────────────────────────
say "Copying database into container..."

# Start a temporary container to write into the named volume
VOLUME_NAME="$(docker compose config --format json 2>/dev/null \
  | python3 -c "import sys,json; cfg=json.load(sys.stdin); print(list(cfg.get('volumes',{}).keys())[0])" 2>/dev/null \
  || echo "firewatch_data")"

docker run --rm \
  -v "$VOLUME_NAME:/data" \
  -v "$TMPFILE:/restore/firewatch.db:ro" \
  alpine:3.19 \
  sh -c "cp /restore/firewatch.db /data/firewatch.db && chmod 640 /data/firewatch.db" \
  || err "Failed to copy database into volume."
ok "Database restored to volume"

# ── Restart container ─────────────────────────────────────────────────────────
say "Starting Firewatch..."
docker compose start app
ok "Container started"

echo
echo -e "${GREEN}  Restore complete.${NC}"
say "Verify the application at your domain and check logs:"
say "  docker compose logs -f"
echo
