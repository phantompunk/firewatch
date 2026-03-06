# Firewatch

Anonymous community incident reporting. Firewatch lets you deploy a private, self-hosted web form where people can submit reports without creating an account or exposing identifying information. Responses are encrypted then forwarded, never stored. No IP logging, no cookies, no analytics.

Forms, email settings, and users are managed through a built-in admin panel.

## Features

- Anonymous report submission with honeypot and timing-based bot protection
- Customizable web form fields (short text, long text, accordion sections)
- Multi-language support for forms (EN, ES, FR, DE, PT)
- Email notifications via SMTP with a customizable template
- Encrypted responses and settings (AES-256-GCM)
- Light / dark / system theme toggle
- Single static binary — no runtime dependencies

---

## Requirements

- A Debian or Ubuntu VPS (the deploy script handles everything else)
- A domain name pointed at the server
- SSH access as root

---

## Deploy

Clone the repository on your server and run the deploy script:

```bash
git clone https://github.com/your-org/firewatch.git
cd firewatch
sudo bash scripts/deploy.sh
```

The script will prompt you for:

| Prompt | Description |
|---|---|
| Domain name | The public domain, e.g. `reports.example.org` |
| SSH port | Used to configure UFW and Fail2ban (default: `22`) |
| Admin email | Used to create the first admin account |
| Admin password | Minimum 12 characters |
| SMTP settings | Can be skipped and configured later in the UI |

It then automatically:

1. Allocates swap (prevents OOM kills on small VPS)
2. Configures UFW (SSH, HTTP, HTTPS only)
3. Hardens SSH (disables password auth, restricts root login)
4. Installs and configures Fail2ban
5. Enables automatic security updates
6. Installs Docker and Caddy
7. Generates secret key files
8. Writes `.env.docker` and `docker-compose.yml`
9. Builds and starts the application
10. Registers a systemd service for auto-start on reboot

Once complete, Firewatch will be live at `https://<your-domain>` with TLS managed automatically by Caddy.

---

## Environment Variables

The deploy script generates `.env.docker` automatically. For manual or platform-based deployments (e.g. Dokploy), set these variables:

### Required

| Variable | Description |
|---|---|
| `DATABASE_URL` | SQLite connection string, e.g. `file:/data/firewatch.db?_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)` |
| `SESSION_SECRET_FILE` | Path to a 32-byte binary file used to sign sessions |
| `SETTINGS_ENCRYPTION_KEY_FILE` | Path to a 32-byte binary file used to encrypt stored settings |
| `EMAIL_HMAC_KEY_FILE` | Path to a 32-byte binary file used for email token signing |

### Server

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Port the app listens on |
| `ENV` | `development` | Set to `production` in production |
| `SECURE_COOKIES` | `false` | Set to `true` when serving over HTTPS |

### SMTP

| Variable | Description |
|---|---|
| `SMTP_HOST` | SMTP server hostname |
| `SMTP_PORT` | SMTP port (typically `587`) |
| `SMTP_USER` | SMTP username |
| `SMTP_PASS` | SMTP password |
| `SMTP_FROM_EMAIL` | From address for outgoing emails |
| `SMTP_FROM_NAME` | From name for outgoing emails |
| `DESTINATION_EMAIL` | Email address that receives report notifications |

### URLs

| Variable | Description |
|---|---|
| `ADMIN_INVITE_BASE_URL` | Base URL for admin invitation links, e.g. `https://reports.example.org` |
| `PASSWORD_RESET_BASE_URL` | Base URL for password reset links |

### First-run seed (remove after first login)

| Variable | Description |
|---|---|
| `SEED_ADMIN_EMAIL` | Email for the initial admin account |
| `SEED_ADMIN_PASSWORD` | Password for the initial admin account |

### Generating secret key files

```bash
mkdir -p /etc/firewatch
chmod 700 /etc/firewatch
openssl rand -out /etc/firewatch/session_secret 32
openssl rand -out /etc/firewatch/settings_encryption_key 32
openssl rand -out /etc/firewatch/email_hmac_key 32
chmod 600 /etc/firewatch/*
```

---

## Getting Started

1. Sign in at `https://<your-domain>/admin` with your admin email and password
2. Go to **Form Editor** to configure your report fields
3. Go to **Settings** to configure SMTP if you skipped it during setup
4. Remove `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` from `.env.docker` after your first login, then restart:

```bash
docker compose up -d
```

---

## Local Development

```bash
cp .env.example .env.local
# edit .env.local with your local settings
make dev       # runs with Air (hot reload)
```

---

## Backups

The deploy script registers a daily cron job at 2 AM that backs up the SQLite database to `/var/backups/firewatch/`. Backups are compressed and rotated after 14 days.

### Manual backup

```bash
sudo bash scripts/backup.sh
# or
make backup
```

Backups are written to `/var/backups/firewatch/firewatch-<timestamp>.db.gz`.

Logs are appended to `/var/log/firewatch-backup.log`.

### Restore

```bash
sudo bash scripts/restore.sh /var/backups/firewatch/firewatch-<timestamp>.db.gz
```

The restore script stops the container, decompresses the backup, runs an integrity check, copies it into the data volume, and restarts the container.

### Offsite sync (optional)

Edit `scripts/backup.sh` and uncomment the `rclone` block at the bottom. Set `RCLONE_REMOTE` to any configured rclone remote (S3, B2, GCS, etc.).

---

## Useful Commands

```bash
make up        # start with Docker Compose
make down      # stop
make build     # build binary to bin/server

# on the server
systemctl status firewatch
systemctl restart firewatch
docker compose logs -f
journalctl -u caddy -f
```
