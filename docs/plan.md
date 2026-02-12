# Firewatch Build Plan

## Context

Build an anonymous community reporting system with a Go backend deployed on a VPS. The form uses the SALUTE method for collecting reports, encrypts them with PGP, and emails them to a nonprofit inbox. No data is stored. Existing static HTML/CSS/JS files in `static/` serve as the frontend starting point.

## Architecture

- **Frontend:** Vanilla HTML/CSS/JS served by Go from `static/` directory
- **Backend:** Go HTTP server (standard library + minimal dependencies)
- **Email:** Abstracted provider interface (SMTP first, SES/Postmark swappable later)
- **Encryption:** OpenPGP public key encryption
- **Deployment:** Single binary on a VPS behind a reverse proxy (nginx/caddy)

## Project Structure

```
firewatch/
├── cmd/server/
│   └── main.go                 # Entry point, server setup
├── internal/
│   ├── handler/
│   │   ├── submit.go           # POST /api/submit — main form handler
│   │   └── health.go           # GET /health
│   ├── email/
│   │   ├── provider.go         # EmailProvider interface
│   │   ├── smtp.go             # SMTP implementation
│   │   └── sender.go           # Retry logic + orchestration
│   ├── encryption/
│   │   └── pgp.go              # PGP encryption with golang.org/x/crypto
│   ├── validation/
│   │   ├── schema.go           # SALUTE field validation
│   │   └── spam.go             # Honeypot, rate limit, timestamp trap
│   ├── media/
│   │   └── metadata.go         # EXIF/GPS stripping from images
│   ├── security/
│   │   ├── headers.go          # Security headers middleware
│   │   └── ratelimit.go        # IP-hashed rate limiting
│   └── config/
│       └── config.go           # Env var loading + validation
├── static/                     # Served directly by Go
│   ├── index.html
│   ├── report.html             # English SALUTE form
│   ├── report-es.html          # Spanish form
│   ├── submitted.html          # Success page
│   ├── privacy.html            # Privacy policy
│   └── assets/
│       ├── style.css           # Dark/light theming, responsive
│       └── report.js           # Form handling, uploads, theme toggle
├── pubkey.asc                  # PGP public key
├── .env                        # Environment config
├── Makefile                    # Build/run/deploy commands
├── go.mod
└── go.sum
```

## Implementation Phases

### Phase 1: Project Setup
- Initialize Go module (`go mod init`)
- Create `cmd/server/main.go` — HTTP server with `net/http`
- Serve `static/` directory for all frontend files
- Set up config loader from env vars (`.env` file)
- Create `Makefile` with `build`, `run`, `dev` targets
- Update `report.js` to POST to `/api/submit` (currently points to `/api/submit`, just verify)

**Key files:** `cmd/server/main.go`, `internal/config/config.go`, `Makefile`

### Phase 2: Security Middleware
- Security headers middleware (CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, no X-Powered-By)
- Rate limiting by hashed IP (in-memory with cleanup goroutine)
- No IP logging — hash immediately, discard raw

**Key files:** `internal/security/headers.go`, `internal/security/ratelimit.go`

### Phase 3: Spam Protection
- Honeypot hidden field check (bots fill it, humans don't)
- Timestamp trap (reject < 3s or > 1hr since page load)
- Silent 302 redirect on spam (don't reveal detection)
- Add honeypot + timestamp hidden fields to `report.html` and `report-es.html`
- Add JS to set timestamp on page load in `report.js`

**Key files:** `internal/validation/spam.go`, `static/report.html`, `static/report-es.html`, `static/assets/report.js`

### Phase 4: Form Validation
- Parse multipart form data (SALUTE fields + file uploads)
- Validate required fields: Activity (required), all others optional
- Sanitize text inputs (trim, limit length)
- Validate file types and sizes (max 5 files, 10MB each)

**Key files:** `internal/validation/schema.go`, `internal/handler/submit.go`

### Phase 5: Image Metadata Stripping
- Strip EXIF/GPS data from JPEG and PNG uploads
- Use a Go EXIF library (e.g., `github.com/rwcarlsen/goexif` or `github.com/dsoprea/go-exif`)
- Videos pass through unmodified for V1
- Process files in memory, never write to disk

**Key files:** `internal/media/metadata.go`

### Phase 6: PGP Encryption
- Load public key from `pubkey.asc` on startup
- Encrypt the formatted report text with OpenPGP
- Encrypt file attachments as separate PGP binary blocks
- Use `golang.org/x/crypto/openpgp` or `github.com/ProtonMail/gopenpgp`

**Key files:** `internal/encryption/pgp.go`

### Phase 7: Email Layer
- Define `EmailProvider` interface: `Send(message) error`
- Implement `SMTPProvider` using `net/smtp` or `github.com/go-mail/mail`
- Build sender with retry + exponential backoff (configurable attempts)
- Provider factory reads `EMAIL_PROVIDER` env var
- Log send events only (no payload content)

**Key files:** `internal/email/provider.go`, `internal/email/smtp.go`, `internal/email/sender.go`

### Phase 8: Submit Handler (orchestration)
Wire everything together in `POST /api/submit`:
1. Parse multipart form data
2. Check honeypot + timestamp (spam protection)
3. Check rate limit
4. Validate SALUTE fields
5. Strip metadata from image uploads
6. Format report text from email template
7. Encrypt report + attachments with PGP
8. Send email with retry
9. Log event (no payload)
10. Redirect 302 to `/submitted.html`

**Key files:** `internal/handler/submit.go`

### Phase 9: Deployment
- Build static binary: `CGO_ENABLED=0 go build -o firewatch ./cmd/server`
- Create systemd service file for VPS
- Set up reverse proxy (Caddy recommended — automatic HTTPS)
- Configure `.env` on VPS with production SMTP credentials
- Test end-to-end

## Go Dependencies

| Package | Purpose |
|---------|---------|
| `golang.org/x/crypto` or `github.com/ProtonMail/gopenpgp` | PGP encryption |
| `github.com/go-mail/mail` | SMTP email sending |
| `github.com/joho/godotenv` | .env file loading |
| `github.com/rwcarlsen/goexif` | EXIF metadata stripping |

Minimal dependencies — most functionality uses the Go standard library.

## Frontend Changes (minimal)

- Add hidden honeypot field + timestamp field to both `report.html` and `report-es.html`
- Add JS in `report.js` to set timestamp on page load
- Verify form action points to `/api/submit`
- Everything else stays as-is

## Verification

1. `make run` — start server locally
2. Submit form, verify encrypted email arrives in Mailtrap
3. Upload image with GPS data, verify EXIF stripped
4. Fill honeypot field, verify silent redirect (no email sent)
5. Rapid-fire submissions, verify rate limit response
6. Test both EN/ES forms and dark/light theme
7. Build binary, deploy to VPS, repeat tests against production
