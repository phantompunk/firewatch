# Firewatch Reports

Anonymous community safety reporting system. Submit structured incident reports using the SALUTE format (Size, Activity, Location, Uniform, Time, Equipment) — all processing happens in memory with no data stored.

## Privacy by Design

- **No IP logging** — server never records addresses
- **No cookies or analytics** — zero tracking
- **No database** — reports processed in memory, then discarded
- **Metadata stripping** — EXIF/GPS data removed from uploaded images
- **PGP encryption** — email reports optionally encrypted
- **Security headers** — CSP, HSTS, no-referrer

## Features

- SALUTE-format structured reporting
- English and Spanish interfaces
- File uploads (up to 5 files, 10MB each — JPEG, PNG, GIF, WebP, MP4, WebM)
- Honeypot + timestamp trap spam protection
- Global rate limiting (no per-IP tracking)
- Dark/light theme
- Drag-and-drop uploads

## Quick Start

```bash
cp .env.example .env
# Edit .env with your SMTP settings

make dev
# Server starts at http://localhost:8080
```

In development mode (no SMTP configured), emails print to the console.

## Build

```bash
make dev          # Live reload with Air
make run          # Build and run
make build        # Build to ./tmp/firewatch
make build-prod   # Static Linux binary → ./firewatch
```

The production build produces a statically linked Linux binary:

```
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o firewatch ./cmd/server
```

## Configuration

All configuration via environment variables (see `.env.example`):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `ENV` | `development` | `development` or `production` |
| `STATIC_DIR` | `./static` | Path to static files |
| `SMTP_HOST` | | SMTP server hostname |
| `SMTP_PORT` | `587` | SMTP server port |
| `SMTP_USER` | | SMTP username |
| `SMTP_PASS` | | SMTP password |
| `FROM_EMAIL` | `noreply@firewatch-reports.org` | Sender address |
| `RECIPIENT_EMAIL` | | Where reports are delivered |
| `PGP_PUBLIC_KEY_PATH` | | Path to PGP public key for encryption |
| `RATE_LIMIT_PER_MINUTE` | `10` | Global submission rate limit |
| `MAX_UPLOAD_SIZE_MB` | `50` | Max total upload size |

## API

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Static files |
| `GET` | `/api/health` | Health check |
| `POST` | `/api/submit` | Report submission |

## Testing

```bash
go test ./...
```

## Project Structure

```
cmd/server/          Entry point
internal/
  app/               HTTP handlers, routing, templates
  config/            Environment config loader
  email/             SMTP delivery, PGP encryption
  media/             Image metadata stripping
  models/            Data models
  security/          Rate limiting, security headers
static/              HTML, CSS, JS (vanilla, no frameworks)
```

## Deployment

Recommended setup:

1. Build the production binary
2. Run behind a reverse proxy (nginx/Caddy) with TLS
3. Disable access logging at the proxy level to preserve anonymity
4. Configure SMTP and optionally PGP encryption

## Tech Stack

- **Go 1.21** — stdlib-heavy, minimal dependencies
- **Vanilla HTML/CSS/JS** — no frontend frameworks
- **Dependencies**: `httprouter`, `godotenv`
