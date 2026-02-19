# Anonymous Community Reporting Tool — Design Document

**Project Type:** Non-Profit Internal Tool
**Status:** Draft v0.3
**Authors:** Engineering Team
**Last Updated:** February 2026

------

## Table of Contents

1. [Overview](#overview)
2. [Goals & Non-Goals](#goals--non-goals)
3. [Functional Requirements](#functional-requirements)
4. [Non-Functional Requirements](#non-functional-requirements)
5. [System Architecture](#system-architecture)
6. [Data Models](#data-models)
7. [API Endpoints](#api-endpoints)
8. [Frontend Design](#frontend-design)
9. [Security Design](#security-design)
10. [Infrastructure & Deployment](#infrastructure--deployment)
11. [Open Questions & Future Work](#open-questions--future-work)

------

## 1. Overview

This document describes the design for an **anonymous community reporting tool** built for a non-profit organization. The tool allows community members to submit structured reports without revealing their identity. Reports are securely forwarded via a configured SMTP provider to a designated inbox. Administrators can manage the report form schema, application settings, and other admin users through a protected dashboard.

The default report form follows the **SALUTE method** (Size, Activity, Location, Unit, Time, Equipment) commonly used for structured incident reporting, though the schema is fully configurable by administrators.

------

## 2. Goals & Non-Goals

### Goals

- Provide a simple, anonymous submission interface accessible to any community member without login or account creation.
- Encrypt all data in transit (TLS) and at rest.
- Forward submitted reports exclusively to the configured destination address via SMTP — no third-party analytics, ad networks, or data brokers.
- Allow administrators to fully customize the report form (fields, labels, descriptions, placeholders) without a code deployment.
- Support hundreds of daily report submissions reliably.
- Provide role-based access control for admin users, including a super-admin tier.

### Non-Goals

- This tool will **not** track, fingerprint, or identify submitters in any way.
- This tool will **not** store reports indefinitely — forwarding via SMTP is the system of record.
- This tool will **not** support file or media attachments in v1.
- This tool will **not** serve advertisements or integrate with any ad network.

------

## 3. Functional Requirements

### Public (Unauthenticated) Users

- View the current report form with its configured title, subtitle, and fields.
- Submit a completed report anonymously.
- Receive a generic confirmation that the report was submitted (no tracking token or identifier returned).

### Admin Users

- Log in and log out securely via session-based authentication.
- View the current report form schema.
- Edit any text field on the report form: titles, subtitles, field labels, descriptions, and placeholders.
- Add or remove fields from the report form.
- Update application settings (e.g., destination email address, SMTP credentials, notification preferences).
- Re-apply / re-deploy configuration changes without a full code deployment.
- Change their own password.
- Use a forgot-password / reset-password flow.

### Super Admins

All Admin capabilities, plus:

- View a list of all admin users.
- Invite new admin users.
- Update admin user roles or deactivate accounts.
- Delete admin user accounts.

------

## 4. Non-Functional Requirements

| Category         | Requirement                                                  |
| ---------------- | ------------------------------------------------------------ |
| **Privacy**      | No user tracking, no cookies beyond session auth, no analytics SDKs, no IP logging |
| **Encryption**   | TLS 1.2+ for all traffic in transit; secrets and settings encrypted at rest |
| **Data Routing** | Submitted reports forwarded only to the configured destination address via SMTP; no other external data egress |
| **Availability** | Target 99.9% uptime; capable of handling hundreds of submissions per day |
| **Performance**  | Report submission response < 2s under normal load            |
| **Auditability** | Admin actions (schema changes, settings updates, user management) logged internally |
| **Compliance**   | No PII collected from submitters; admin PII (email, hashed password) handled per applicable data protection standards |

------

## 5. System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Browser Client                      │
│         (Public Report Form  |  Admin Dashboard)         │
└───────────────────┬─────────────────────────────────────┘
                    │ HTTPS (TLS)
┌───────────────────▼─────────────────────────────────────┐
│                    Application Server                     │
│              (REST API + Session Management)              │
│                                                           │
│   ┌──────────────┐   ┌────────────────┐                  │
│   │  Report API  │   │   Admin API    │                  │
│   └──────┬───────┘   └───────┬────────┘                  │
│          │                   │                            │
│   ┌──────▼───────────────────▼────────┐                  │
│   │           Database                │                  │
│   │  (Schema | Settings | Admin Users)│                  │
│   └───────────────────────────────────┘                  │
│                                                           │
│   ┌───────────────────────────────────┐                  │
│   │          SMTP Forwarder           │                  │
│   │  (Formats & sends report email)   │                  │
│   └───────────────────────────────────┘                  │
└─────────────────────────────────────────────────────────┘
```

### Key Components

**Frontend** — Server-rendered HTML using Go's `html/template` package. HTMX handles all dynamic interactions (inline field editing, form submission, partial page updates) without a JavaScript build step. Sortable.js (vendored, single file) provides drag-to-reorder for the field list. A small vanilla JS handler (~20 lines, inline) powers the live email template preview. No Node.js, no bundler, no npm.

**Application Server** — A single Go binary serving both the HTML views and the JSON API. Handles form schema management, report submission, admin authentication, session management, settings management, and user management. Responsible for forwarding submissions via SMTP.

**Database** — PostgreSQL stores the report schema (as JSONB), application settings (encrypted), admin user accounts (hashed passwords), and sessions (keyed by user ID for all-device logout). Does **not** store submitted report content after forwarding.

**SMTP Forwarder** — An internal Go package that takes a submitted report payload, renders it using the configured email template, and sends it to the configured destination address via SMTP. This is the only external network call made on report submission. Any standard SMTP provider (e.g., SendGrid, Postmark, AWS SES, or a self-hosted relay) is supported via environment configuration.

------

## 6. Data Models

### Report Schema

The report schema is stored as a JSON document and versioned. It drives both the public form render and the admin editor.

```json
{
  "schemaVersion": 4,
  "updatedAt": "2026-02-19T10:00:00Z",
  "updatedBy": "admin@example.org",
  "page": {
    "title": "Community Incident Report",
    "subtitle": "All submissions are anonymous. No identifying information is collected.",
    "submitButtonLabel": "Submit Report"
  },
  "fields": [
    {
      "id": "field_001",
      "type": "textarea",
      "order": 1,
      "label": "Size",
      "description": "Describe the number of people or scale of the incident.",
      "placeholder": "e.g., Approximately 10–15 individuals...",
      "required": true
    },
    {
      "id": "field_002",
      "type": "textarea",
      "order": 2,
      "label": "Activity",
      "description": "What was happening? Describe the activity or behavior observed.",
      "placeholder": "e.g., A group was seen...",
      "required": true
    }
  ],
  "emailTemplate": "New Community Report\n\nSize:\n{{field_001}}\n\nActivity:\n{{field_002}}\n\n---\nThis report was submitted anonymously."
}
```

**Field types supported in v1:** `text`, `textarea`, `select` (with configurable options).

The `emailTemplate` is a plain-text string stored alongside the schema. It uses `{{field_id}}` tokens that are substituted with submitted values at send time. Admins edit this template in the form editor and can preview it with sample values before publishing (see [View 1: Report Form Editor](#view-1-report-form-editor)).

### Admin User

```json
{
  "id": "usr_abc123",
  "email": "admin@example.org",
  "role": "admin",
  "status": "active",
  "createdAt": "2026-01-01T00:00:00Z",
  "lastLoginAt": "2026-02-18T09:30:00Z"
}
```

**Roles:** `super_admin`, `admin`
**Statuses:** `active`, `inactive`
Passwords are stored as bcrypt hashes. Plaintext passwords are never stored or logged.

### Application Settings

Settings are stored encrypted at rest. The decrypted view:

```json
{
  "destinationEmail": "reports@example.org",
  "emailSubjectTemplate": "New Community Report",
  "smtpHost": "smtp.sendgrid.net",
  "smtpPort": 587,
  "smtpUser": "apikey",
  "smtpPass": "...",
  "smtpFromAddress": "no-reply@example.org",
  "smtpFromName": "Community Reports",
  "reportRetentionPolicy": "forward-only",
  "maintenanceMode": false
}
```

------

## 7. API Endpoints

All endpoints are prefixed with `/api`. Admin endpoints require a valid session cookie. Super Admin endpoints additionally require the `super_admin` role.

### Public Endpoints

| Method | Endpoint      | Description                                         | Auth   |
| ------ | ------------- | --------------------------------------------------- | ------ |
| `GET`  | `/api/report` | Returns the current published report schema         | Public |
| `POST` | `/api/report` | Submits a completed report; forwards to Proton Mail | Public |
| `GET`  | `/api/health` | Health check; returns server and dependency status  | Public |

#### `GET /api/report`

Returns the latest published schema version. The response includes all field definitions, page metadata (title, subtitle), and the schema version number. No auth required.

#### `POST /api/report`

Accepts a JSON body matching the current schema. Validates required fields, then renders the email template by substituting `{{field_id}}` tokens with submitted values and forwards the result to the configured destination address via SMTP. Returns a generic `202 Accepted` with no submission ID or token — intentional to prevent any form of tracking. No server-side timestamp is added to the forwarded email.

```json
// Request body example
{
  "schemaVersion": 4,
  "fields": {
    "field_001": "Approximately 10 individuals observed near the east gate.",
    "field_002": "Individuals were seen attempting to access a locked storage area."
  }
}
```

------

### Admin — Authentication

| Method | Endpoint                     | Description                                    | Auth                 |
| ------ | ---------------------------- | ---------------------------------------------- | -------------------- |
| `POST` | `/api/admin/login`           | Authenticates admin; creates session           | Public               |
| `POST` | `/api/admin/logout`          | Clears session; logs user out                  | Admin                |
| `POST` | `/api/admin/change-password` | Updates the authenticated admin's password     | Admin                |
| `POST` | `/api/admin/forgot-password` | Sends a password reset link to the given email | Public               |
| `POST` | `/api/admin/reset-password`  | Resets password using a valid reset token      | Public (token-gated) |

#### `POST /api/admin/login`

Accepts `email` and `password`. On success, issues an HTTP-only session cookie. Failed attempts are rate-limited. Plaintext credentials are never logged.

#### `POST /api/admin/forgot-password`

Accepts an `email` address. If the email matches an active admin account, a time-limited reset link is sent. The response is always `200 OK` regardless of whether the email exists (to prevent account enumeration).

------

### Admin — Report Schema

| Method | Endpoint                  | Description                                                  | Auth  |
| ------ | ------------------------- | ------------------------------------------------------------ | ----- |
| `GET`  | `/api/admin/report`       | Returns the current report schema including draft email template | Admin |
| `PUT`  | `/api/admin/report`       | Updates the report schema (fields, labels, page metadata, email template) | Admin |
| `POST` | `/api/admin/report/apply` | Publishes the current schema draft; triggers live reload     | Admin |

#### `PUT /api/admin/report`

Accepts a full or partial schema update. Changes are saved as a **draft** and do not affect the public form until `/apply` is called. This allows admins to preview changes before publishing.

#### `POST /api/admin/report/apply`

Atomically promotes the draft schema to the live schema. The public `GET /api/report` endpoint will immediately serve the new version. Logged for audit purposes.

------

### Admin — Settings

| Method | Endpoint                    | Description                                                  | Auth  |
| ------ | --------------------------- | ------------------------------------------------------------ | ----- |
| `GET`  | `/api/admin/settings`       | Returns current application settings (secrets masked)        | Admin |
| `PUT`  | `/api/admin/settings`       | Updates one or more application settings                     | Admin |
| `POST` | `/api/admin/settings/apply` | Re-applies settings (e.g., reconnects SMTP forwarder after credential change) | Admin |

Sensitive fields (e.g., `smtpPass`, `destinationEmail`) are returned masked in `GET` responses and only updated via `PUT`.

------

### Super Admin — User Management

| Method   | Endpoint               | Description                                       | Auth        |
| -------- | ---------------------- | ------------------------------------------------- | ----------- |
| `GET`    | `/api/admin/users`     | Lists all admin users                             | Super Admin |
| `POST`   | `/api/admin/users`     | Invites a new admin user (sends invitation email) | Super Admin |
| `PUT`    | `/api/admin/users/:id` | Updates a user's role or active status            | Super Admin |
| `DELETE` | `/api/admin/users/:id` | Permanently deletes an admin user account         | Super Admin |

#### `POST /api/admin/users`

Accepts `email` and `role`. Sends an invitation email with a time-limited sign-up link. The invited user sets their own password on first login. Super admins cannot be deleted or demoted via the API without another super admin performing the action.

------

## 8. Frontend Design

The frontend is server-rendered HTML delivered by the Go application server using `html/template`. There is no JavaScript build step, no npm, and no client-side framework. Interactive behaviour is handled by HTMX (vendored as a single static file) for server-round-trip interactions, Sortable.js (vendored) for drag-to-reorder, and a small inline vanilla JS block for the email template live preview.

Template files live in `internal/templates/` and are embedded into the Go binary at compile time using `//go:embed`, so deployment is always a single self-contained binary.

### Public Report Form

The public view is a standard HTML page rendered server-side from the live schema on every request. There is no client-side schema fetch — Go renders the form fields directly into the HTML response.

**Layout:**

- Full-width, centered single-column form.
- Page `title` and `subtitle` rendered at the top from the schema.
- Each field rendered in order with its `label`, `description` (helper text below the input), and `placeholder`.
- A single submit button labeled per the schema's `submitButtonLabel`.
- On submit, HTMX posts to `POST /api/report` and swaps in a generic success message in place of the form. No page reload. No confirmation ID or submission details displayed.

**Privacy design choices:**

- No analytics scripts loaded.
- No cookies set for public users.
- No autofill attributes that could expose browser-stored PII.
- Submit button disabled during in-flight request to prevent double submissions (HTMX `hx-disabled-elt` attribute).

------

### Admin Dashboard

The admin dashboard is a set of server-rendered HTML pages protected by session middleware in Go. Navigation between sections is standard `<a>` link navigation — full page loads, not a SPA. HTMX is used selectively within pages for partial updates (e.g., saving a field card without a full reload, confirming a delete inline). The top navigation bar is rendered as part of a shared base template.

#### Top Navigation

```
[ 📋 Report Form ]   [ ⚙️ Settings ]   [ 👥 Users ]          [ Logout ]
```

- **Report Form** — the default landing view after login.
- **Settings** — visible to all admins.
- **Users** — visible only to super admins; hidden from the nav for `admin` role.
- **Logout** — always visible, right-aligned. Posts to `POST /api/admin/logout` via a form, invalidates all sessions for the user, redirects to login.

------

#### View 1: Report Form Editor

This view has **two sub-tabs** within the main content area: **Form** and **Email Template**. Both tabs share the same Save Draft / Publish Changes action bar.

------

##### Tab 1: Form

A **split-panel layout** rendered server-side on page load. The right panel editor uses HTMX to POST individual field card changes back to the server without a full page reload. The left panel preview re-renders server-side in response (HTMX `hx-target` swaps the preview region).

**Left panel — Live Preview (60% width)** A server-rendered snapshot of the form as it currently appears in the draft schema. Refreshes via HTMX whenever the admin saves a change in the right panel. Labeled "Preview" with a visual indicator distinguishing it from the live public form.

**Right panel — Schema Editor (40% width)** A structured editing panel rendered from the draft schema. Not a raw JSON editor. Contains:

- **Page metadata section** at the top:
  - Editable `title` input
  - Editable `subtitle` input
  - Editable `submitButtonLabel` input
- **Field list** — one card per field, rendered in order, each containing:
  - Drag handle (Sortable.js). On drag-end, Sortable fires a small JS callback that POSTs the new field order to the server; the server responds with the updated field list HTML which HTMX swaps in.
  - `label` input
  - `description` textarea
  - `placeholder` input
  - `required` toggle
  - Delete field button — triggers an HTMX-powered inline confirmation before issuing the delete request.
- **"Add Field" button** at the bottom of the field list — HTMX swaps in a new blank field card at the bottom of the list without a full page reload.
- **Action bar** at the bottom of the right panel:
  - `Save Draft` — POSTs the full draft schema; server responds with a success indicator swapped in via HTMX.
  - `Publish Changes` — POSTs to `/apply`; server renders a confirmation modal fragment that HTMX swaps into the page before the admin confirms.

------

##### Tab 2: Email Template

This tab lets admins control exactly what the forwarded email looks like when a report is received.

**Left panel — Template Editor (50% width)**

A plain-text `<textarea>` containing the raw email template. Field tokens are inserted using `{{field_id}}` syntax. A reference list of available tokens (e.g., `{{field_001}} → Size`) is shown below the editor for convenience.

Example template:

```
New Community Report

Size:
{{field_001}}

Activity:
{{field_002}}

Location:
{{field_003}}

Unit:
{{field_004}}

Time:
{{field_005}}

Equipment:
{{field_006}}

---
This report was submitted anonymously.
```

**Right panel — Preview (50% width)**

A read-only `<pre>` block showing the template rendered with sample placeholder values substituted in for each `{{field_id}}` token. Sample values are drawn from each field's `placeholder` text, falling back to `[field label]` if no placeholder is set. The substitution is handled by a small inline vanilla JS `input` event listener on the textarea — no server round-trip needed since it is purely cosmetic.

```
New Community Report

Size:
e.g., Approximately 10–15 individuals...

Activity:
e.g., A group was seen...

---
This report was submitted anonymously.
```

The preview updates live as the admin types in the template editor on the left.

------

**Action bar** (shared across both tabs, pinned to the bottom of the editor panel):

- `Save Draft` — saves changes without publishing.
- `Publish Changes` — calls `/apply`; prompts confirmation before publishing. Shows a diff summary of what changed.

------

#### View 2: Settings

A simple form for managing application-level settings.

Sections:

**Mail Configuration**

- Destination email address (masked display, re-enter to update)
- Email subject line
- SMTP host, port, username, password (all masked on display)
- From address and display name
- "Send Test Email" button — sends a test message to the configured destination address to verify the connection

**General**

- Maintenance mode toggle (disables public form with a configurable message)

**Danger Zone**

- "Re-apply Settings" button — re-initializes the SMTP forwarder connection with current credentials. Useful after rotating SMTP credentials or changing providers.

------

#### View 3: User Management *(Super Admin only)*

A table listing all admin users with columns: Email, Role, Status, Last Login, Actions.

Actions per row:

- **Edit** — opens a modal to change role (`admin` / `super_admin`) or toggle active/inactive status.
- **Delete** — opens a confirmation dialog; requires typing the user's email to confirm.

**"Invite Admin" button** (top right of table) — opens a modal to enter an email and assign a role. Sends the invitation email.

**Own account protection:** Admins cannot delete or deactivate their own account from this view.

------

## 9. Security Design

### Authentication & Sessions

- Session-based auth with HTTP-only, Secure, SameSite=Strict cookies.
- Sessions expire after a configurable idle timeout (default: 60 minutes).
- Login attempts rate-limited per IP (e.g., 5 attempts per 10 minutes with exponential backoff).
- Passwords hashed with bcrypt (minimum cost factor 12).
- Password reset tokens are single-use and expire after 1 hour.
- **Logout invalidates all active sessions for that user** — implemented by storing sessions in the database keyed by user ID, so a logout or password change deletes all rows for that user. This is the simplest approach and ensures no stale sessions remain on other devices.

### Data in Transit

- TLS 1.2 minimum enforced; TLS 1.3 preferred.
- HSTS header set with long max-age.
- No mixed content.

### Data at Rest

- Application settings (including SMTP credentials and destination address) encrypted using AES-256-GCM with keys stored in environment variables or a secrets manager (e.g., HashiCorp Vault, AWS Secrets Manager).
- Admin passwords stored as bcrypt hashes only.
- **Submitted report content is never written to disk or database.** It is processed in memory, forwarded via SMTP, and discarded.

### Anonymity Guarantees

- No IP addresses logged for public submissions.
- No session tokens, cookies, or local storage used for public users.
- No user-agent or referrer logging for public submissions.
- The `POST /api/report` response returns no submission identifier.
- Report content is processed in memory and discarded immediately after the SMTP send completes — it is never written to the database or disk.

### Admin Audit Log

All admin actions are written to an internal audit log (not exposed via API in v1):

- Schema updates (who, when, what changed)
- Settings updates (field name only — not new values for sensitive fields)
- User management actions (invite, role change, delete)
- Login and logout events

------

## 10. Infrastructure & Deployment

### Chosen Stack

| Layer                  | Technology                                     | Notes                                                        |
| ---------------------- | ---------------------------------------------- | ------------------------------------------------------------ |
| Language               | Go                                             | Single binary deployment, strong stdlib, excellent long-term stability |
| HTTP Server            | Go `net/http` + `chi` router                   | Lightweight, no framework overhead                           |
| HTML Templating        | Go `html/template`                             | Compiled into the binary via `//go:embed`                    |
| Interactivity          | HTMX                                           | Vendored as a single static file — no npm, no build step     |
| Drag-to-reorder        | Sortable.js                                    | Vendored as a single static file                             |
| Email template preview | ~20 lines vanilla JS                           | Inline `<script>` block, no external dependency              |
| Database               | PostgreSQL                                     | Via `pgx` driver; schema/settings/users/sessions             |
| Mail Forwarding        | Any SMTP provider                              | Configured via env vars; Go `net/smtp` or `gomail` package   |
| TLS                    | Caddy                                          | Automatic Let's Encrypt, trivial reverse proxy config        |
| Hosting                | Any VPS (e.g., Hetzner, Linode) or self-hosted | Single binary + Caddy is all that needs to run               |

### Go Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go              # Entry point; wires together handlers, DB, SMTP
├── internal/
│   ├── handler/
│   │   ├── report.go            # GET /api/report, POST /api/report
│   │   ├── admin_report.go      # Admin schema view/edit/apply handlers
│   │   ├── admin_auth.go        # Login, logout, password reset handlers
│   │   ├── admin_settings.go    # Settings view/update/apply handlers
│   │   └── admin_users.go       # User management handlers
│   ├── middleware/
│   │   ├── session.go           # Session validation; redirects to login if invalid
│   │   └── role.go              # Super admin role guard
│   ├── model/
│   │   ├── schema.go            # ReportSchema, Field structs + JSON marshalling
│   │   ├── user.go              # AdminUser struct
│   │   └── settings.go          # AppSettings struct
│   ├── store/
│   │   ├── schema.go            # DB reads/writes for report schema
│   │   ├── user.go              # DB reads/writes for admin users
│   │   ├── session.go           # DB reads/writes for sessions
│   │   └── settings.go          # DB reads/writes for encrypted settings
│   ├── mailer/
│   │   └── smtp.go              # SMTP forwarder; renders template, sends email
│   └── crypto/
│       └── aes.go               # AES-256-GCM encrypt/decrypt for settings at rest
├── templates/
│   ├── base.html                # Shared layout: nav, head, footer
│   ├── report_form.html         # Public report submission form
│   ├── admin_login.html         # Login page
│   ├── admin_report.html        # Form editor (split panel + email template tab)
│   ├── admin_settings.html      # Settings page
│   ├── admin_users.html         # User management page
│   └── partials/
│       ├── field_card.html      # HTMX-swappable individual field editor card
│       ├── field_list.html      # HTMX-swappable full field list
│       ├── form_preview.html    # HTMX-swappable left-panel form preview
│       └── confirm_modal.html   # HTMX-swappable confirmation dialog fragment
├── static/
│   ├── htmx.min.js              # Vendored — never update unless intentional
│   ├── sortable.min.js          # Vendored — never update unless intentional
│   └── style.css                # Minimal hand-written CSS; no framework
├── migrations/
│   └── *.sql                    # Sequential SQL migration files
├── .env.example                 # Documented env var template (no secrets)
├── Dockerfile
├── Caddyfile
└── go.mod
```

### Vendoring the JS Files

Both `htmx.min.js` and `sortable.min.js` are checked directly into the `static/` directory and embedded into the binary at compile time alongside the templates. This means:

- No CDN dependency at runtime.
- No supply-chain risk from an external script tag.
- No version drift — the files only change when deliberately updated.
- The entire application ships as a single binary with zero runtime file system dependencies.

### Scaling for Hundreds of Daily Reports

At the expected volume (hundreds per day, not thousands per second), a single Go server process is more than sufficient — Go handles high concurrency natively. The primary scaling concern is SMTP send rate limits imposed by the chosen provider. If bursts exceed the provider's per-second or per-minute limits, a lightweight in-process job queue using a buffered Go channel can be added to the mailer package to smooth out spikes without dropping submissions. No external queue infrastructure is needed at this scale.

### Configuration via Environment Variables

```
DATABASE_URL=...
SESSION_SECRET=...
SETTINGS_ENCRYPTION_KEY=...
SMTP_HOST=...
SMTP_PORT=...
SMTP_USER=...
SMTP_PASS=...
SMTP_FROM_ADDRESS=...
SMTP_FROM_NAME=...
DESTINATION_EMAIL=...
ADMIN_INVITE_BASE_URL=...
PASSWORD_RESET_BASE_URL=...
```

No secrets should ever be committed to source control.

------

## 11. Decisions Log & Remaining Open Questions

### Resolved Decisions

| #    | Decision                                                     |
| ---- | ------------------------------------------------------------ |
| 1    | **Mail provider:** Standard SMTP provider (e.g., SendGrid, Postmark, AWS SES). Provider is configurable via environment variables — no vendor lock-in. |
| 2    | **Language:** Admin dashboard is English only. No localization needed. |
| 3    | **Email template preview:** Included as a tab in the Form Editor. A plain-text template editor on the left; a read-only preview with placeholder values substituted on the right. No shareable preview link. |
| 4    | **Timestamps:** No timestamp is added to forwarded emails — neither server-generated nor client-provided. |
| 5    | **File attachments:** Out of scope for v1.                   |
| 6    | **Report categories:** Out of scope for v1.                  |
| 7    | **Session logout scope:** Logout and password change invalidate **all active sessions for that user** across all devices. Implemented by storing sessions in the database keyed by user ID. |
| 8    | **Stack:** Go backend (`net/http` + `chi`), Go `html/template` for server-rendered HTML, HTMX for interactivity, Sortable.js for drag-to-reorder, vanilla JS for email preview. Both JS dependencies vendored into `static/`. No Node.js, no build step, single binary deployment behind Caddy. |

### Remaining Open Questions

| #    | Question                                                     | Priority            |
| ---- | ------------------------------------------------------------ | ------------------- |
| 1    | Which SMTP provider will be used? This determines how the `Send Test Email` button is implemented and what rate limits we need to plan around. | **High — blocking** |
| 2    | What email address will admin password reset / invite emails be sent from — same SMTP account as report forwarding, or a separate sender? | High                |

------

*This is a living document. Please comment or open a PR with questions, corrections, or additions before implementation begins.*
