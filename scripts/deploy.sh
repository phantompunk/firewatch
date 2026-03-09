#!/usr/bin/env bash
# =============================================================================
#  Firewatch — Production Deploy
#  Run this on a fresh Debian/Ubuntu server from inside the cloned repository.
#  Usage: sudo bash scripts/deploy.sh
# =============================================================================
set -euo pipefail

# ── Palette ───────────────────────────────────────────────────────────────────
AMBER='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

LINE="────────────────────────────────────────────────────────────────────────────"

# ── Paths ─────────────────────────────────────────────────────────────────────
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SECRETS_DIR="/etc/firewatch"

# ── Helpers ───────────────────────────────────────────────────────────────────
say()     { echo -e "  $*"; }
ok()      { echo -e "  ${GREEN}✓${NC}  $*"; }
skip()    { echo -e "  ${DIM}·  $* — already done, skipping${NC}"; }
hint()    { echo -e "  ${DIM}$*${NC}"; }
err()     { echo -e "\n  ${RED}✗  $*${NC}\n" >&2; exit 1; }

section() {
  echo
  echo -e "${AMBER}  ${LINE}${NC}"
  printf "${AMBER}  %-76s${NC}\n" "  $*"
  echo -e "${AMBER}  ${LINE}${NC}"
  echo
}

ask() {
  local message="$1"
  local default="${2:-}"
  local result

  if [[ -n "$default" ]]; then
    printf "  ${BOLD}%s${NC} ${DIM}[%s]${NC}: " "$message" "$default" >/dev/tty
  else
    printf "  ${BOLD}%s${NC}: " "$message" >/dev/tty
  fi

  IFS= read -r result </dev/tty
  echo "${result:-$default}"
}

ask_secret() {
  local message="$1"
  local result

  printf "  ${BOLD}%s${NC}: " "$message" >/dev/tty
  IFS= read -rs result </dev/tty
  echo >/dev/tty
  echo "$result"
}

confirm() {
  printf "  ${BOLD}%s${NC} ${DIM}[y/N]${NC}: " "$1"
  local answer
  IFS= read -r answer
  [[ "$answer" =~ ^[Yy]$ ]]
}

require_root() {
  [[ $EUID -eq 0 ]] || err "Run this script as root:  sudo bash scripts/deploy.sh"
}

require_debian() {
  if [[ -f /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    [[ "${ID:-}" == "debian" || "${ID:-}" == "ubuntu" || "${ID_LIKE:-}" == *"debian"* ]] \
      || err "This script supports Debian/Ubuntu only (detected: ${ID:-unknown})."
  else
    err "Cannot determine OS. /etc/os-release not found."
  fi
}

validate_email() {
  [[ "$1" =~ ^[^@[:space:]]+@[^@[:space:]]+\.[^@[:space:]]+$ ]] \
    || err "\"$1\" doesn't look like a valid email address."
}

validate_domain() {
  [[ "$1" =~ ^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$ ]] \
    || err "\"$1\" doesn't look like a valid domain name."
}

# ── Welcome ───────────────────────────────────────────────────────────────────
clear
echo
echo -e "${AMBER}  🔥  Firewatch${NC}"
echo -e "      Production Setup — Fresh Install"
echo
say "This script installs Docker and Caddy, provisions secrets,"
say "writes your production config, and starts everything up."
echo
hint "Press Ctrl-C at any time to cancel without making changes."
echo

require_root
require_debian

# ── Gather config ─────────────────────────────────────────────────────────────
section "Let's start with your deployment details."

DOMAIN=$(ask "Domain name (e.g. reports.example.org)")
validate_domain "$DOMAIN"

ADMIN_EMAIL=$(ask "Admin email address")
validate_email "$ADMIN_EMAIL"

while true; do
  ADMIN_PASSWORD=$(ask_secret "Admin password (min 12 characters)")
  [[ ${#ADMIN_PASSWORD} -ge 12 ]] && break
  say "${RED}Password must be at least 12 characters. Try again.${NC}"
done

# Confirm before doing anything
echo
say "Ready to set up Firewatch on ${BOLD}${DOMAIN}${NC}."
echo
confirm "Continue?" || { say "Cancelled."; exit 0; }

# ── Docker ────────────────────────────────────────────────────────────────────
section "Step 1 of 5  ·  Docker"

if command -v docker &>/dev/null; then
  skip "Docker $(docker --version | grep -oP '\d+\.\d+\.\d+' | head -1) already installed"
else
  say "Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable --now docker
  ok "Docker installed"
fi

docker compose version &>/dev/null \
  || err "Docker Compose plugin not found. Install docker-compose-plugin and re-run."
ok "Docker Compose available"

if docker info --format '{{.Swarm.LocalNodeState}}' 2>/dev/null | grep -q "^active$"; then
  skip "Docker Swarm already initialised"
else
  docker swarm init --advertise-addr "$(hostname -I | awk '{print $1}')"
  ok "Docker Swarm initialised"
fi

# ── Caddy ─────────────────────────────────────────────────────────────────────
section "Step 2 of 5  ·  Caddy (TLS + reverse proxy)"

if command -v caddy &>/dev/null; then
  skip "Caddy $(caddy version | head -1) already installed"
else
  say "Installing Caddy..."
  apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
    | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
    | tee /etc/apt/sources.list.d/caddy-stable.list > /dev/null
  apt-get update -qq
  apt-get install -y -qq caddy
  ok "Caddy installed"
fi

# ── Secrets ───────────────────────────────────────────────────────────────────
section "Step 3 of 5  ·  Secrets"

mkdir -p "$SECRETS_DIR"
chmod 700 "$SECRETS_DIR"

gen_and_store_secret() {
  local name="$1"
  local path="$SECRETS_DIR/$name"

  if docker secret inspect "$name" &>/dev/null; then
    skip "Docker secret '$name' already exists"
    return
  fi

  openssl rand -out "$path" 32
  chmod 600 "$path"
  docker secret create "$name" "$path"
  rm -f "$path"
  ok "$name created in Docker Swarm"
}

gen_and_store_secret "session_secret"
gen_and_store_secret "settings_encryption_key"
gen_and_store_secret "email_hmac_key"

rmdir "$SECRETS_DIR" 2>/dev/null || true

# ── Config files ──────────────────────────────────────────────────────────────
section "Step 4 of 5  ·  Configuration"

# .env.docker
cat > "$REPO_DIR/.env.docker" <<EOF
# Generated by deploy.sh on $(date -u '+%Y-%m-%dT%H:%M:%SZ')
# ─────────────────────────────────────────────────────────────────────────────

# Database
DATABASE_URL=file:/data/firewatch.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)

# Secret key files (mounted into container via docker-compose.yml)
SESSION_SECRET_FILE=/run/secrets/session_secret
SETTINGS_ENCRYPTION_KEY_FILE=/run/secrets/settings_encryption_key
EMAIL_HMAC_KEY_FILE=/run/secrets/email_hmac_key

# Server
PORT=8080
ENV=production
SECURE_COOKIES=true

# First-run admin seed — remove these two lines after first login
SEED_ADMIN_USERNAME=admin
SEED_ADMIN_EMAIL=${ADMIN_EMAIL}
SEED_ADMIN_PASSWORD=${ADMIN_PASSWORD}

# Base URL for admin invitation emails
ADMIN_INVITE_BASE_URL=https://${DOMAIN}
EOF
chmod 600 "$REPO_DIR/.env.docker"
ok ".env.docker written"

# docker-compose.yml — Swarm stack; secrets injected from Docker Swarm, not host files
cat > "$REPO_DIR/docker-compose.yml" <<EOF
secrets:
  session_secret:
    external: true
  settings_encryption_key:
    external: true
  email_hmac_key:
    external: true

services:
  app:
    image: ghcr.io/phantompunk/firewatch:latest
    deploy:
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 5
    ports:
      - "0.0.0.0:8080:8080"
    env_file: .env.docker
    secrets:
      - session_secret
      - settings_encryption_key
      - email_hmac_key
    volumes:
      - firewatch_data:/data
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  firewatch_data:
EOF
ok "docker-compose.yml written"

# Caddyfile
# Security headers are set by the Go app middleware; only strip Server here.
cat > /etc/caddy/Caddyfile <<EOF
${DOMAIN} {
    reverse_proxy localhost:8080

    encode gzip

    header -Server
}
EOF
ok "Caddyfile written → /etc/caddy/Caddyfile"

# ── Start everything ──────────────────────────────────────────────────────────
section "Step 5 of 5  ·  Starting services"

say "Pulling latest Firewatch image..."
docker pull ghcr.io/phantompunk/firewatch:latest
docker stack deploy -c "$REPO_DIR/docker-compose.yml" firewatch
ok "Firewatch stack deployed"

# Systemd unit so the stack comes back after a reboot
cat > /etc/systemd/system/firewatch.service <<EOF
[Unit]
Description=Firewatch
Documentation=https://github.com/phantompunk/firewatch
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${REPO_DIR}
ExecStart=/usr/bin/docker stack deploy -c ${REPO_DIR}/docker-compose.yml firewatch
ExecStop=/usr/bin/docker stack rm firewatch
TimeoutStartSec=120

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable firewatch
ok "firewatch.service registered (auto-starts on reboot)"

# Reload Caddy with the new config
systemctl enable --now caddy
caddy reload --config /etc/caddy/Caddyfile
ok "Caddy started → https://${DOMAIN}"

# ── Backup scripts ────────────────────────────────────────────────────────────
SCRIPTS_BASE_URL="https://raw.githubusercontent.com/phantompunk/firewatch/main/scripts"

say "Fetching backup and restore scripts..."
curl -fsSL "$SCRIPTS_BASE_URL/backup.sh"  -o /usr/local/bin/firewatch-backup
curl -fsSL "$SCRIPTS_BASE_URL/restore.sh" -o /usr/local/bin/firewatch-restore
chmod +x /usr/local/bin/firewatch-backup /usr/local/bin/firewatch-restore
ok "firewatch-backup and firewatch-restore installed to /usr/local/bin/"

CRON_JOB="0 2 * * * firewatch-backup >> /var/log/firewatch-backup.log 2>&1"
if crontab -l 2>/dev/null | grep -qF "firewatch-backup"; then
  skip "Backup cron job already registered"
else
  (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
  ok "Daily backup cron registered (runs at 2 AM, logs to /var/log/firewatch-backup.log)"
fi

mkdir -p /var/backups/firewatch
chmod 700 /var/backups/firewatch
ok "Backup directory ready → /var/backups/firewatch"

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "${AMBER}  ${LINE}${NC}"
echo -e "${AMBER}  🔥  Firewatch is live.${NC}"
echo -e "${AMBER}  ${LINE}${NC}"
echo
ok "Running at   https://${DOMAIN}"
ok "TLS          managed automatically by Caddy"
ok "Persistence  Docker named volume  firewatch_data"
ok "Secrets      stored in Docker Swarm (encrypted at rest, not on host filesystem)"
ok "Auto-start   systemctl status firewatch"
echo
say "${BOLD}Next steps:${NC}"
echo
say "  1.  Sign in at https://${DOMAIN} with ${ADMIN_EMAIL}"
say "  2.  Open Settings → configure SMTP, PGP, and notification email"
say "  3.  Remove the seed credentials from .env.docker once you're in:"
echo
hint "      SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD"
echo
hint "      Then redeploy:  docker stack deploy -c ${REPO_DIR}/docker-compose.yml firewatch"
echo
say "${BOLD}Useful commands:${NC}"
echo
hint "  Logs      docker service logs -f firewatch_app"
hint "  Status    docker stack ps firewatch"
hint "  Redeploy  docker stack deploy -c ${REPO_DIR}/docker-compose.yml firewatch"
hint "  Restart   systemctl restart firewatch"
hint "  Caddy     journalctl -u caddy -f"
echo
