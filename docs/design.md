# Firewatch Design Document

## Anonymous Agent Activity Reporting System

### 1. Overview

A privacy-focused web application enabling community members to anonymously report agent activity to a community organization. The system prioritizes reporter safety through maximum anonymity protections while collecting actionable intelligence.

**Key Principles:**
- Reporter anonymity is paramount
- No tracking, no logging, no fingerprinting
- Minimal data retention
- End-to-end security

---

### 2. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    STATIC FRONTEND                              │
│              (CDN / Static Host - No Logging)                   │
│                                                                 │
│  - Pure HTML/CSS/JS                                             │
│  - No analytics, no cookies                                     │
│  - Client-side form validation                                  │
│  - Client-side image metadata stripping                         │
│  - JavaScript optional (progressive enhancement)                │
└─────────────────────┬───────────────────────────────────────────┘
                      │ HTTPS POST (multipart/form-data)
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    GO BACKEND SERVER                            │
│              (Minimal footprint, no logging)                    │
│                                                                 │
│  - Receives form submissions                                    │
│  - Strips remaining image metadata (server-side backup)         │
│  - Validates and sanitizes input                                │
│  - Composes encrypted email                                     │
│  - No database, no persistence                                  │
│  - No IP logging, no request logging                            │
└─────────────────────┬───────────────────────────────────────────┘
                      │ SMTP/TLS
                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                 EMAIL DELIVERY                                  │
│           (Encrypted email to organization)                     │
│                                                                 │
│  - PGP/GPG encrypted body                                       │
│  - Attachments encrypted                                        │
│  - Organization decrypts locally                                │
└─────────────────────────────────────────────────────────────────┘
```

---

### 3. Data Collected (SALUTE Report Schema)

The form follows the SALUTE mnemonic (Size, Activity, Location, Uniform, Time, Equipment) for structured reporting of law enforcement activity. All fields optional except activity description.

| Category | Field | Type | Required | Description |
|----------|-------|------|----------|-------------|
| **S - Size** | `size` | text | No | Number of agents/officers and vehicles |
| **A - Activity** | `activity` | text | Yes | Description of observed activity/misconduct |
| **L - Location** | `location` | text | No | Where observed (intersection, landmark, address) |
| **U - Uniform** | `uniform` | text | No | Uniforms, badges, agency, visible ID |
| **T - Time** | `time` | text | No | When and how long (free text to avoid timezone leaks) |
| **E - Equipment** | `equipment` | text | No | Vehicles, weapons, and other equipment observed |
| **Other** | `media` | file[] | No | Photos/videos (max 5 files, 10MB each) |
| | `additional_info` | text | No | Witnesses, related incidents, other relevant info |

**Not Collected:**
- IP addresses
- Browser fingerprints
- Timestamps (server-side)
- Cookies or session identifiers
- Geolocation (automatic)

---

### 4. Privacy & Security Requirements

#### 4.1 Client-Side Protections

| Protection | Implementation |
|------------|----------------|
| No cookies | `<meta>` tags and CSP headers prevent cookie storage |
| No local storage | Application stores nothing locally |
| No analytics | Zero third-party scripts |
| No CDN tracking | Self-host all assets or use privacy-respecting CDN |
| EXIF stripping | Client-side JavaScript removes image metadata before upload |
| No JS required | No CAPTCHAs, no JavaScript requirements for basic submission |
| No referrer leaks | `Referrer-Policy: no-referrer` header |
| CSP hardening | Strict Content-Security-Policy |

#### 4.2 Server-Side Protections

| Protection | Implementation |
|------------|----------------|
| No IP logging | Disable all access logs, use `X-Forwarded-For` stripping |
| No request logging | Application logs errors only, no request details |
| Memory-only processing | Files processed in memory, never written to disk |
| Metadata stripping | Server-side EXIF removal as backup (using Go library) |
| Input sanitization | Prevent injection attacks, XSS in email rendering |
| Rate limiting | Per-submission delay (not per-IP) to prevent abuse |
| No database | Stateless processing, nothing persisted |

#### 4.3 Transport Security

| Protection | Implementation |
|------------|----------------|
| TLS 1.3 only | Modern cipher suites, disable older protocols |
| HSTS | Strict-Transport-Security header |
| Certificate pinning | Optional for mobile/app distribution |

#### 4.4 Email Security

| Protection | Implementation |
|------------|----------------|
| PGP encryption | All report content encrypted with org's public key |
| No plaintext | Subject line generic ("New Report"), body encrypted |
| Attachment encryption | Media files encrypted before attachment |
| SMTP TLS | Encrypted connection to mail server |

---

### 5. Frontend Design

#### 5.1 Technology Stack

- **HTML5** - Semantic, accessible markup
- **CSS3** - Minimal, no frameworks (reduce fingerprinting surface)
- **Vanilla JavaScript** - Progressive enhancement only
- **No build step** - Simple static files

#### 5.2 Pages

```
/
├── index.html          # Landing page with privacy info
├── report.html         # Report submission form
├── submitted.html      # Confirmation (no tracking params)
├── privacy.html        # Privacy policy / how we protect you
└── assets/
    ├── style.css
    ├── report.js       # Form handling, EXIF stripping
    └── exif-stripper.js # Client-side metadata removal
```

#### 5.3 Form UX Principles

1. **Single page form** - No multi-step wizard (avoids state)
2. **No required fields except description** - Lower barrier to report
3. **Drag-and-drop media** - Easy photo upload
4. **Visual feedback** - Show when metadata is stripped from images
5. **No autosave** - Nothing stored until explicit submission
6. **Clear before close** - Warn and clear sensitive data on navigation

#### 5.4 Accessibility

- WCAG 2.1 AA compliance
- Screen reader compatible
- Keyboard navigation
- High contrast mode
- No time limits on form completion

---

### 6. Backend Design (Go)

#### 6.1 Project Structure

```
/cmd
    /server
        main.go             # Entry point
/internal
    /handler
        submit.go           # Form submission handler
        health.go           # Health check endpoint
    /email
        composer.go         # Email composition
        pgp.go              # PGP encryption
        sender.go           # SMTP sending
    /media
        stripper.go         # EXIF/metadata removal
        validator.go        # File type validation
    /security
    		headers.go          # Security headers middleware
        sanitizer.go        # Input sanitization
        ratelimit.go        # Submission rate limiting
		/config
    		config.go           # Environment configuration
```

#### 6.2 Dependencies (Minimal)

- `golang.org/x/crypto/openpgp` - PGP encryption
- `github.com/dsoprea/go-exif/v3` - EXIF stripping
- Standard library for HTTP, SMTP, etc.

#### 6.3 API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check (returns 200 OK, no logging) |
| `POST` | `/api/submit` | Form submission endpoint |

#### 6.4 Request Flow

```go
// Pseudocode for submission handler
func HandleSubmit(w http.ResponseWriter, r *http.Request) {
    // 1. Rate limit check (global counter, not per-IP)
    if !ratelimit.Allow() {
        // Return generic error, no details
        http.Error(w, "Please try again later", 429)
        return
    }

    // 2. Parse multipart form (memory only)
    r.ParseMultipartForm(50 << 20) // 50MB max

    // 3. Extract and sanitize text fields
    report := sanitize(extractFields(r))

    // 4. Process media attachments
    files := processMedia(r) // Strips metadata, validates type

    // 5. Compose report
    content := composeReport(report)

    // 6. Encrypt with PGP
    encrypted := pgp.Encrypt(content, orgPublicKey)

    // 7. Send email
    email.Send(encrypted, files)

    // 8. Return success (no tracking identifiers)
    http.Redirect(w, r, "/submitted.html", 303)
}
```

---

### 7. Email Format

#### 7.1 Email Structure

```
To: reports@organization.org
From: noreply@firewatch-reports.org
Subject: Community Report Received
Content-Type: multipart/encrypted; protocol="application/pgp-encrypted"

-----BEGIN PGP MESSAGE-----
[Encrypted report content]
-----END PGP MESSAGE-----

[Encrypted attachments if present]
```

#### 7.2 Decrypted Report Format (SALUTE)

```
=====================================
ANONYMOUS SALUTE REPORT
Received: [Date/Time - server timezone]
=====================================

[S] SIZE:
[size or "Not provided"]

[A] ACTIVITY:
[activity]

[L] LOCATION:
[location or "Not provided"]

[U] UNIFORM:
[uniform or "Not provided"]

[T] TIME:
[time or "Not provided"]

[E] EQUIPMENT:
[equipment or "Not provided"]

ADDITIONAL INFORMATION:
[additional_info or "None provided"]

ATTACHMENTS: [count] file(s)
=====================================
```

---

### 8. Deployment Architecture

#### 8.1 Recommended Setup

```
┌─────────────────────────────────────────────────────────────┐
│                     REVERSE PROXY                           │
│                  (Nginx / Caddy)                            │
│                                                             │
│  - TLS termination (Let's Encrypt)                          │
│  - Access logging DISABLED                                  │
│  - Static file serving                                      │
│  - Proxy to Go backend                                      │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
┌───────────────────┐     ┌───────────────────┐
│   STATIC FILES    │     │    GO BACKEND     │
│                   │     │                   │
│  /var/www/static  │     │  :8080 internal   │
└───────────────────┘     └───────────────────┘
```

#### 8.2 Hosting Requirements

- **VPS provider**: Choose provider that accepts anonymous payment
- **Jurisdiction**: Consider legal jurisdiction for data requests
- **No cloud functions**: Avoid AWS Lambda, Cloudflare Workers (logging)
- **Dedicated IP**: Avoid shared hosting

#### 8.3 Environment Variables

```bash
# Required
PGP_PUBLIC_KEY_PATH=/etc/firewatch/org-public.asc
SMTP_HOST=mail.provider.com
SMTP_PORT=587
SMTP_USER=reports@firewatch-reports.org
SMTP_PASS=<secure password>
RECIPIENT_EMAIL=reports@organization.org

# Optional
RATE_LIMIT_PER_MINUTE=10
MAX_UPLOAD_SIZE_MB=50
ALLOWED_MEDIA_TYPES=image/jpeg,image/png,video/mp4
```

---

### 9. Threat Model

#### 9.1 Threats Mitigated

| Threat | Mitigation |
|--------|------------|
| Network surveillance | HTTPS enforced |
| IP tracking | No server-side IP logging |
| Browser fingerprinting | Minimal JS, no third-party resources |
| Image metadata exposure | Client + server EXIF stripping |
| Timing correlation | No timestamps in user-visible responses |
| Email interception | PGP encryption end-to-end |
| Server compromise | No data at rest, memory-only processing |
| Subpoena for logs | No logs exist to produce |

#### 9.2 Threats NOT Mitigated (User Responsibility)

| Threat | User Action Required |
|--------|---------------------|
| Local device surveillance | Use public computer or Tails OS |
| Shoulder surfing | Submit in private location |
| Content-based identification | Don't include identifying details in report |
| Screenshot by malware | Use clean device |
| Network traffic analysis | Use VPN or privacy-focused browser |

#### 9.3 Organizational Threats

| Threat | Mitigation |
|--------|------------|
| Insider threat at org | PGP key held by limited personnel |
| Email account compromise | Strong authentication, limited access |
| Legal compulsion | Org cannot produce what doesn't exist |

---

### 10. Development Phases

#### Phase 1: Core MVP
- [ ] Static frontend with form
- [ ] Go backend with email sending
- [ ] Basic input validation
- [ ] HTTPS deployment

#### Phase 2: Privacy Hardening
- [ ] Client-side EXIF stripping
- [ ] Server-side metadata stripping
- [ ] PGP email encryption
- [ ] Security headers implementation

#### Phase 3: Polish
- [ ] Accessibility audit
- [ ] Mobile responsive design
- [ ] Privacy policy page
- [ ] Error handling improvements
- [ ] Rate limiting

#### Phase 4: Operations
- [ ] Deployment documentation
- [ ] Org onboarding guide (PGP setup)
- [ ] Incident response plan
- [ ] Security audit

---

### 11. Testing Strategy

#### 11.1 Security Testing

- [ ] No cookies set (browser dev tools)
- [ ] No local storage used
- [ ] All requests over HTTPS
- [ ] Headers validated (CSP, HSTS, etc.)
- [ ] EXIF data stripped from uploads
- [ ] PGP encryption verified
- [ ] No IP addresses in any logs

#### 11.2 Functional Testing

- [ ] Form submission works
- [ ] All field types handled
- [ ] File upload works
- [ ] Email received and decryptable
- [ ] Error states handled gracefully
- [ ] Rate limiting works

---

### 12. Legal Considerations

1. **Privacy Policy**: Clear documentation of what is/isn't collected
2. **No Promises**: Don't guarantee anonymity (edge cases exist)
3. **Jurisdiction**: Host in reporter-friendly jurisdiction
4. **Data Retention**: Document that no data is retained
5. **Organization Liability**: Org assumes responsibility for report handling

---

### 13. Future Considerations

- **Secure drop box**: Allow follow-up communication via anonymous ID
- **I2P support**: Alternative anonymous network
- **Mobile app**: Native app with additional protections
- **Multi-language**: Support for community languages
- **Report categories**: Structured activity type selection
- **Offline mode**: PWA for drafting reports offline

---

## Appendix A: Security Headers

```nginx
# Nginx configuration
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-Frame-Options "DENY" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "no-referrer" always;
add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;
add_header Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self';" always;

# Disable access logging
access_log off;
```

## Appendix B: PGP Setup for Organization

```bash
# Generate organization keypair
gpg --full-generate-key
# Select: RSA and RSA, 4096 bits, no expiration

# Export public key for server
gpg --armor --export reports@organization.org > org-public.asc

# Import on server
# Place org-public.asc at path specified in PGP_PUBLIC_KEY_PATH
```

---

*Document Version: 1.0*
*Last Updated: 2026-02-12*
