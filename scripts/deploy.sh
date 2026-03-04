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
SECRETS_DIR="$REPO_DIR/secrets"

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

echo
say "SMTP settings are used to send report notifications, password reset"
say "emails, and admin invitations. You can also set these in the Settings"
say "UI after first login."
echo

SMTP_HOST=""
SMTP_PORT="587"
SMTP_USER=""
SMTP_PASS=""
SMTP_FROM_EMAIL=""
SMTP_FROM_NAME="Firewatch"
DESTINATION_EMAIL=""

if confirm "Configure SMTP now?"; then
  echo
  SMTP_HOST=$(ask        "SMTP host"                   "smtp.brevo.com")
  SMTP_PORT=$(ask        "SMTP port"                   "587")
  SMTP_USER=$(ask        "SMTP username")
  SMTP_PASS=$(ask_secret "SMTP password")
  SMTP_FROM_EMAIL=$(ask  "From address"                "no-reply@${DOMAIN}")
  SMTP_FROM_NAME=$(ask   "From name"                   "Firewatch")
  DESTINATION_EMAIL=$(ask "Report destination email"   "$ADMIN_EMAIL")
  validate_email "$SMTP_FROM_EMAIL"
  validate_email "$DESTINATION_EMAIL"
fi

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

gen_secret() {
  local name="$1"
  local path="$SECRETS_DIR/$name"

  if [[ -f "$path" ]]; then
    skip "$name already exists (existing key preserved)"
  else
    openssl rand -out "$path" 32
    chmod 600 "$path"
    ok "$name generated"
  fi
}

gen_secret "session_secret"
gen_secret "settings_encryption_key"
gen_secret "email_hmac_key"

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

# SMTP — update via Settings UI after first login if left blank
SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USER=${SMTP_USER}
SMTP_PASS=${SMTP_PASS}
SMTP_FROM_EMAIL=${SMTP_FROM_EMAIL}
SMTP_FROM_NAME=${SMTP_FROM_NAME}
DESTINATION_EMAIL=${DESTINATION_EMAIL}

# Server
PORT=8080
ENV=production
SECURE_COOKIES=true
LISTEN_ADDR=:8080

# First-run admin seed — remove these two lines after first login
SEED_ADMIN_USERNAME=admin
SEED_ADMIN_EMAIL=${ADMIN_EMAIL}
SEED_ADMIN_PASSWORD=${ADMIN_PASSWORD}

# Transactional email base URLs
ADMIN_INVITE_BASE_URL=https://${DOMAIN}
PASSWORD_RESET_BASE_URL=https://${DOMAIN}
EOF
chmod 600 "$REPO_DIR/.env.docker"
ok ".env.docker written"

# docker-compose.yml — bind to loopback only so Caddy is the only entry point
cat > "$REPO_DIR/docker-compose.yml" <<EOF
services:
  app:
    build: .
    restart: unless-stopped
    ports:
      - "0.0.0.0:8080:8080"
    env_file: .env.docker
    volumes:
      - firewatch_data:/data
      - ./secrets/session_secret:/run/secrets/session_secret:ro
      - ./secrets/settings_encryption_key:/run/secrets/settings_encryption_key:ro
      - ./secrets/email_hmac_key:/run/secrets/email_hmac_key:ro
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  firewatch_data:
EOF
ok "docker-compose.yml updated (bound to 127.0.0.1)"

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

say "Building and starting Firewatch (this takes a minute)..."
cd "$REPO_DIR"
docker compose up -d --build
ok "Firewatch is running"

# Systemd unit so the containers come back after a reboot
cat > /etc/systemd/system/firewatch.service <<EOF
[Unit]
Description=Firewatch
Documentation=https://github.com/your-org/firewatch
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${REPO_DIR}
ExecStart=/usr/bin/docker compose up -d
ExecStop=/usr/bin/docker compose down
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

# ── Done ──────────────────────────────────────────────────────────────────────
echo
echo -e "${AMBER}  ${LINE}${NC}"
echo -e "${AMBER}  🔥  Firewatch is live.${NC}"
echo -e "${AMBER}  ${LINE}${NC}"
echo
ok "Running at   https://${DOMAIN}"
ok "TLS          managed automatically by Caddy"
ok "Persistence  Docker named volume  firewatch_data"
ok "Auto-start   systemctl status firewatch"
echo
say "${BOLD}Next steps:${NC}"
echo
say "  1.  Sign in at https://${DOMAIN} with ${ADMIN_EMAIL}"
say "  2.  Open Settings → configure SMTP if you skipped it above"
say "  3.  Remove the seed credentials from .env.docker once you're in:"
echo
hint "      SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD"
echo
hint "      Then run:  docker compose up -d   (picks up the new env file)"
echo
say "${BOLD}Useful commands:${NC}"
echo
hint "  Logs      docker compose -f ${REPO_DIR}/docker-compose.yml logs -f"
hint "  Restart   systemctl restart firewatch"
hint "  Status    systemctl status firewatch"
hint "  Caddy     journalctl -u caddy -f"
echo
