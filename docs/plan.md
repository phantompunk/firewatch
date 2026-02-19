# Anonymous Community Reporting Tool — Implementation Plan

**Project Type:** Non-Profit Internal Tool  
**Status:** Draft v0.1  
**Authors:** Engineering Team  
**Last Updated:** February 2026  
**Reference:** [Design Document](./design.md)

---

## Overview

This plan sequences the implementation of the reporting tool into seven phases. Each phase has a clear dependency on the one before it, with an early shippable milestone at the end of Phase 2. Phases 3–6 build out the self-service admin layer. Phase 7 hardens the system for production.

### Sequencing Summary

```
Phase 0 — Scaffold & Infrastructure
    └── Phase 1 — Authentication
            └── Phase 2 — Report Schema & Public Form   ← shippable MVP
                    ├── Phase 3 — Admin Form Editor
                    ├── Phase 4 — Settings
                    └── Phase 5 — Password Management
                            └── Phase 6 — User Management
                                    └── Phase 7 — Hardening & Production Readiness
```

Phase 2 is the earliest point the tool is genuinely useful — the public form works and reports land in an inbox. Everything after that is admin tooling. If there is time pressure, Phase 2 can ship with settings hardcoded in env vars and the schema seeded from a SQL file, then Phases 3–6 fill in the self-service admin layer.

---

## Phase 0 — Project Scaffold & Infrastructure

*Get a running Go server, database, and deployment pipeline before writing any feature code. Everything else builds on this.*

### 0.1 — Repository Setup

- Initialize Go module (`go mod init`)
- Create the full directory structure from the design document
- Add `.env.example` with all required vars documented
- Add `Dockerfile` (multi-stage build, distroless final image)
- Add `Caddyfile` (TLS termination, reverse proxy to Go on localhost)
- Add `Makefile` with targets: `run`, `build`, `migrate`, `test`, `lint`

### 0.2 — Database

- Provision PostgreSQL instance
- Write and test the following sequential migrations:

| # | Migration | Tables / Changes |
|---|---|---|
| 001 | Create admin users | `admin_users` (id, email, password_hash, role, status, created_at, last_login_at) |
| 002 | Create sessions | `sessions` (id, user_id, created_at, expires_at) |
| 003 | Create report schema | `report_schema` (id, version, is_live, schema JSONB, updated_at, updated_by) |
| 004 | Create settings | `settings` (id, data JSONB encrypted, updated_at) |
| 005 | Create audit log | `audit_log` (id, user_id, action, detail, created_at) |
| 006 | Create password reset tokens | `password_reset_tokens` (id, user_id, token_hash, expires_at, used) |
| 007 | Create invitation tokens | `invitation_tokens` (id, email, role, token_hash, expires_at, used) |

- Set up a migration runner (`golang-migrate` or a simple home-rolled sequential SQL runner in `cmd/migrate/`)

### 0.3 — Base HTTP Server

- Wire up `chi` router in `cmd/server/main.go`
- Serve `static/` files (embedded via `//go:embed`)
- Embed `templates/` via `//go:embed`
- Add `GET /api/health` handler — checks DB connectivity, returns `200 OK` with JSON status
- Confirm the binary builds and the health check responds

### 0.4 — Crypto Package

- Implement `internal/crypto/aes.go` — AES-256-GCM encrypt and decrypt functions
- Keys sourced from `SETTINGS_ENCRYPTION_KEY` environment variable
- Unit test encrypt → decrypt roundtrip
- Unit test that decrypting with the wrong key returns an error

> **Note:** This is a hard dependency for the settings store. Implement and test it before moving on.

---

## Phase 1 — Authentication

*Nothing in the admin is accessible without this. Build it in full before any admin views.*

### 1.1 — Password Utilities

- bcrypt hash and verify functions in `internal/auth/password.go`
- Minimum cost factor: 12
- Unit tests for hash, verify correct, verify incorrect

### 1.2 — Session Store

- `internal/store/session.go` — create, read, and delete session rows
- Session IDs are cryptographically random 32-byte hex strings
- `DeleteAllByUserID(userID)` function used by logout and password change to invalidate all devices

### 1.3 — Session Middleware

- `internal/middleware/session.go` — reads session cookie, validates against DB, attaches user to request context; redirects to `/admin/login` if session is missing or expired
- `internal/middleware/role.go` — reads role from request context, returns `403 Forbidden` if the user does not have the required role

### 1.4 — First Admin Seeding

- On startup, if the `admin_users` table is empty, check for `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` env vars
- If both are present, create the first `super_admin` user with a hashed password
- Log a warning and continue if vars are absent and no users exist (health check still passes)
- This is the only way to bootstrap the system; remove the seed vars from the environment after first login

### 1.5 — Auth Handlers

- `GET /admin/login` — render `templates/admin_login.html`
- `POST /api/admin/login` — validate credentials against DB, create session row, set HTTP-only `Secure` `SameSite=Strict` cookie, redirect to `/admin/report`
- `POST /api/admin/logout` — delete all session rows for the authenticated user, clear cookie, redirect to `/admin/login`
- Rate limiting middleware on `POST /api/admin/login`: 5 attempts per 10 minutes per IP with exponential backoff; return `429 Too Many Requests` when exceeded

### 1.6 — Auth Templates

- `templates/base.html` — shared layout with top nav; nav items conditionally rendered based on session presence and role
- `templates/admin_login.html` — email and password form, error message partial

**Milestone:** An admin can log in, receive a session cookie, navigate to protected routes, and log out. All admin routes redirect to the login page without a valid session.

---

## Phase 2 — Report Schema & Public Form

*The core feature. The public form is the entire user-facing product.*

### 2.1 — Schema Store

- `internal/store/schema.go` — load latest live schema, load draft schema, save draft, promote draft to live (atomic update of `is_live` flag)
- On startup, if no schema rows exist, seed the DB with the default SALUTE schema (Size, Activity, Location, Unit, Time, Equipment) as both draft and live

### 2.2 — Schema Model

- `internal/model/schema.go` — Go structs for `ReportSchema`, `PageMeta`, `Field`
- JSON marshal and unmarshal with correct field tags
- Validation: required fields present, field IDs unique, field types valid (`text`, `textarea`, `select`)

### 2.3 — Public Report Form

- `GET /` — load live schema from DB, render `templates/report_form.html`
- `templates/report_form.html` — iterates over schema fields in order, renders each input type; no JavaScript required for the initial render
- No session, cookie, or tracking of any kind for public users

### 2.4 — SMTP Mailer

- `internal/mailer/smtp.go` — establishes SMTP connection using env vars; `Send(subject, body string) error` function
- `internal/mailer/template.go` — `RenderTemplate(template string, fields []Field, submission map[string]string) string` — substitutes `{{field_id}}` tokens with submitted values
- Unit test token substitution: known schema + known submission → expected output string
- Unit test missing token handling (field ID in template but not in submission — render as empty string)
- Integration test: send a real test email to a scratch address to confirm SMTP config works before Phase 4

### 2.5 — Report Submission Handler

- `POST /api/report` — validate JSON body against live schema (required fields present, no unknown field IDs), render email body via template substitution, call mailer
- Return `202 Accepted` with a generic success message body regardless of outcome
- If the SMTP send fails, log the error server-side but do not surface it to the submitter
- Never log the request body
- HTMX on the public form swaps the form element for the success message on `202`

**Milestone:** The public form renders from a live schema, a submission triggers an email to the configured address, and the submitter sees a generic confirmation. The core product works end to end.

---

## Phase 3 — Admin Form Editor

*Admins can view and modify the report schema and email template from the dashboard.*

### 3.1 — Static Assets & Base Admin Templates

- Drop `htmx.min.js` into `static/` — vendored, pinned version, do not reference a CDN
- Drop `sortable.min.js` into `static/` — vendored, pinned version
- `static/style.css` — minimal hand-written CSS for split panel, field cards, nav, tabs
- `templates/admin_report.html` — split-panel scaffold with Form / Email Template tab switcher
- `templates/partials/form_preview.html` — left-panel public form preview (re-rendered via HTMX on changes)
- `templates/partials/field_card.html` — a single editable field card
- `templates/partials/field_list.html` — full ordered field list (HTMX-swappable)
- `templates/partials/confirm_modal.html` — reusable inline confirmation dialog fragment

### 3.2 — Admin Schema Read Handler

- `GET /admin/report` — load draft schema, render `templates/admin_report.html` with split panel populated; session middleware applied

### 3.3 — Page Metadata Editing

- `PUT /api/admin/report/meta` — update `title`, `subtitle`, `submitButtonLabel` in the draft schema row; respond with updated `form_preview.html` partial
- HTMX `hx-put` on the metadata inputs, `hx-target` set to the preview region, `hx-trigger="change"` (not on every keypress)

### 3.4 — Field Editing

- `PUT /api/admin/report/fields/:id` — update a single field's `label`, `description`, `placeholder`, or `required` flag in the draft; respond with the updated `field_card.html` partial and trigger a preview refresh
- HTMX `hx-put` on each input within a field card, `hx-trigger="change"`

### 3.5 — Field Reordering

- `PUT /api/admin/report/fields/order` — accepts `{ "order": ["field_001", "field_003", "field_002"] }`, updates the `order` property of each field in the draft
- Sortable.js initialised on the field list container; on the `end` event, a small JS callback reads the new DOM order and POSTs field IDs to this endpoint; response HTML is swapped into the field list via HTMX

### 3.6 — Add and Delete Fields

- `POST /api/admin/report/fields` — append a new blank field to the draft (auto-generate a unique `field_id`); respond with the new `field_card.html` partial; HTMX appends it to the field list
- `DELETE /api/admin/report/fields/:id` — remove field from draft; respond with empty `200`; HTMX removes the card from the DOM
- Delete uses an inline confirmation: clicking delete swaps the card footer for a "Are you sure? [Cancel] [Confirm]" fragment; Confirm triggers the `DELETE` request

### 3.7 — Save Draft and Publish

- `POST /api/admin/report/draft` — explicitly persist the current in-progress draft (the individual field/meta handlers already persist incrementally; this is a manual checkpoint save)
- `POST /api/admin/report/apply` — atomically promote the draft to the live schema; write to `audit_log`; respond with a success fragment swapped in by HTMX
- Before apply: clicking Publish swaps in `confirm_modal.html` with a summary of pending changes; Confirm triggers the apply POST

### 3.8 — Email Template Tab

- `PUT /api/admin/report/email-template` — update the `emailTemplate` string on the draft schema row
- Inline vanilla JS on the textarea `input` event — reads the current template text, iterates over the field list substituting each `{{field_id}}` with the field's `placeholder` value (falling back to `[label]`), writes the rendered string to the `<pre>` preview block
- No server round-trip for the preview; the substitution is entirely client-side
- Token reference list (e.g., `{{field_001}} → Size`) rendered server-side alongside the editor

**Milestone:** Admins can edit every part of the form schema and email template, preview changes live, and publish. Publishing immediately updates the public form.

---

## Phase 4 — Settings

*Admins can configure the SMTP connection and general settings from the UI without touching environment variables.*

### 4.1 — Settings Store

- `internal/store/settings.go` — load settings (decrypt JSONB on read using crypto package), save settings (encrypt JSONB on write)
- On first load, if no settings row exists, write defaults from environment variables so there is always a row

### 4.2 — Settings Model

- `internal/model/settings.go` — `AppSettings` struct with all fields: `DestinationEmail`, `EmailSubject`, `SMTPHost`, `SMTPPort`, `SMTPUser`, `SMTPPass`, `SMTPFromAddress`, `SMTPFromName`, `MaintenanceMode`

### 4.3 — Settings Handlers and Template

- `GET /admin/settings` — load and decrypt settings, render `templates/admin_settings.html`; sensitive fields (`SMTPPass`, `SMTPUser`, `DestinationEmail`) masked in the rendered HTML
- `PUT /api/admin/settings` — validate and save updated settings; encrypt before writing; write changed field names only (not values) to `audit_log`; respond with success indicator
- `POST /api/admin/settings/apply` — re-initialise the SMTP mailer singleton with current decrypted settings; respond with pass/fail fragment via HTMX
- `POST /api/admin/settings/test-email` — attempt to send a test message to `DestinationEmail` using current settings; respond with a pass/fail result fragment via HTMX; the "Send Test Email" button triggers this

**Milestone:** Admins can rotate SMTP credentials and change the destination address entirely from the UI without a redeployment.

---

## Phase 5 — Password Management

*Self-service password flows for admin accounts.*

### 5.1 — Change Password

- `GET /admin/change-password` — render form (requires valid session)
- `POST /api/admin/change-password` — verify current password, hash new password, update `admin_users` row, call `session.DeleteAllByUserID` to invalidate all devices, redirect to `/admin/login`

### 5.2 — Forgot Password Flow

- `GET /admin/forgot-password` — render email entry form (no session required)
- `POST /api/admin/forgot-password` — look up email in `admin_users`; if found and active, generate a cryptographically random token, store its hash in `password_reset_tokens` with a 1-hour expiry, send reset link via SMTP; always return `200 OK` regardless of whether the email matched (prevents account enumeration)
- `GET /admin/reset-password?token=...` — validate token (exists, not expired, not used); if invalid redirect to `/admin/forgot-password` with an error; if valid render the new password form
- `POST /api/admin/reset-password` — re-validate token, hash new password, update `admin_users`, mark token as used, call `session.DeleteAllByUserID`, redirect to `/admin/login`

**Milestone:** Admins can recover access independently. Super admin intervention is not required for password resets.

---

## Phase 6 — User Management

*Super admins can manage the admin roster entirely from the UI.*

### 6.1 — User Store

- `internal/store/user.go` — `ListAll`, `GetByID`, `GetByEmail`, `Create`, `UpdateRoleAndStatus`, `Delete`
- Guard in `Delete`: prevent deletion of the last `super_admin` account

### 6.2 — User Management Handlers and Template

- `GET /admin/users` — list all users, render `templates/admin_users.html`; super admin role middleware applied
- `PUT /api/admin/users/:id` — update role or status; write to `audit_log`; respond with updated table row HTML via HTMX swap
- `DELETE /api/admin/users/:id` — delete user and all their sessions; guard against self-deletion; respond with HTMX row removal swap; write to `audit_log`
- Self-deletion guard: if `id` matches the authenticated user's ID, return `400 Bad Request`

### 6.3 — Invite Flow

- `POST /api/admin/users` — validate email not already registered; generate a cryptographically random invitation token; store hash in `invitation_tokens` with role and 48-hour expiry; send invite email via SMTP with link; write to `audit_log`; respond with new pending-user row HTML via HTMX
- `GET /admin/accept-invite?token=...` — validate token (exists, not expired, not used); if invalid render an expiry error page; if valid render a set-password form with email pre-filled
- `POST /api/admin/accept-invite` — re-validate token; hash password; create `admin_users` row with the role from the token; mark token used; redirect to `/admin/login`

**Milestone:** Super admins can invite new admins, update roles, deactivate accounts, and remove users entirely from the UI.

---

## Phase 7 — Hardening & Production Readiness

*The features are done. Now make the system safe, observable, and easy to operate.*

### 7.1 — Security Headers

Middleware applied to all responses:

| Header | Value |
|---|---|
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains` |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | `default-src 'self'; script-src 'self' 'nonce-{generated}'` |
| `Referrer-Policy` | `no-referrer` |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=()` |

### 7.2 — Rate Limiting

- Confirm login rate limiting from Phase 1 is correct under load
- `POST /api/report` — 10 submissions per minute per IP (IP used only for rate limiting, never logged)
- `POST /api/admin/forgot-password` — 3 requests per hour per IP
- All rate limit state held in-process (a simple token bucket); no Redis or external state required at this scale

### 7.3 — SMTP Send Queue

- Add a buffered Go channel in `internal/mailer/smtp.go` as an outbound queue
- A dedicated goroutine drains the channel and handles retries with exponential backoff (max 3 retries)
- `POST /api/report` enqueues the send and returns `202 Accepted` immediately; the caller never waits on the SMTP round-trip
- Log send failures with structured fields (no report content in the log)

### 7.4 — Audit Log Viewer

- `GET /admin/audit-log` — super admin only; paginated table of audit events (action, user, timestamp); read-only; no editing or deletion

### 7.5 — Error Handling and Structured Logging

- Structured JSON logging via Go stdlib `log/slog` — no external dependency
- Log levels: `DEBUG` (dev), `INFO` (prod default), `ERROR` for unexpected failures
- Custom error pages rendered from templates: `404`, `403`, `500` — no stack traces exposed to the browser
- Confirm no request body content appears anywhere in logs; audit with a grep over log output in tests

### 7.6 — End-to-End Test Checklist

| Scenario | Expected Result |
|---|---|
| Submit valid report | Email received at destination address |
| Submit report with missing required field | `400` returned, no email sent |
| Login with correct credentials | Session cookie set, redirect to `/admin/report` |
| Login with wrong password 6 times | `429` returned on 6th attempt |
| Session cookie expired | Redirect to `/admin/login` |
| Logout | All sessions invalidated, cookie cleared |
| Edit field label and publish | Public form reflects new label |
| Change SMTP credentials and apply | Test email sends successfully |
| Invite new admin | Invite email sent, user can set password and log in |
| Super admin deletes another admin | User row removed, their sessions invalidated |
| Admin attempts to access `/admin/users` | `403 Forbidden` |
| Forgot password flow | Reset email received, password updated, old sessions invalidated |

### 7.7 — Deployment

- Finalise `Dockerfile` — multi-stage build; final image is distroless or `scratch` with the Go binary and no shell
- Finalise `Caddyfile` — automatic TLS via Let's Encrypt, reverse proxy to `localhost:8080`
- Write `README.md` covering: environment variable reference, first-run seeding, running migrations, deploying with Docker + Caddy
- Confirm production checklist:
  - `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` env vars removed after first login
  - `SETTINGS_ENCRYPTION_KEY` is a randomly generated 32-byte value, stored in the host environment only
  - `SESSION_SECRET` is a randomly generated value, stored in the host environment only
  - No secrets in source control or Docker image
  - TLS termination confirmed active
  - Health check endpoint responding

**Milestone:** The system is production-ready. A single `docker compose up` (or equivalent) starts the Go binary and Caddy. The application is fully operational from a blank server.

---

*This is a living document. Update phase status and decisions as implementation progresses.*
