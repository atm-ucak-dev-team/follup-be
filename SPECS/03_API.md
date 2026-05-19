# API_SPEC.md

---

## General Conventions

- Base URL: `/api/v1`
- Content-Type: `application/json`
- Auth: Bearer token (JWT) in `Authorization` header — except auth endpoints
- All timestamps: ISO 8601 format
- Error shape always:
```json
{
  "error": {
    "code": "AUTOMATION_NOT_FOUND",
    "message": "automation rule not found"
  }
}
```

---

## Auth — Jira OAuth

### `GET /auth/jira/connect`
Redirect user to Jira OAuth consent page.

**Response:** `302 Redirect` → Jira OAuth URL

---

### `GET /auth/jira/callback`
Jira redirects here after user consent.

**Query Params:**
```
code   string  required
state  string  required
```

**Response `200`:**
```json
{
  "access_token": "jwt...",
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

---

### `POST /auth/jira/refresh`
Refresh Jira OAuth access token.

**Response `200`:**
```json
{
  "access_token": "jwt..."
}
```

---

## Email Credentials

### `POST /email/credentials`
Register or update user's IMAP/SMTP credentials.

**Request:**
```json
{
  "email_address": "john@gmail.com",
  "password": "plaintext_password"
}
```

**Response `201`:**
```json
{
  "message": "email credential saved",
  "email_address": "john@gmail.com"
}
```

> Password is encrypted server-side before storage. Never returned in any response.

---

### `GET /email/credentials`
Check if credential is registered (no password returned).

**Response `200`:**
```json
{
  "email_address": "john@gmail.com",
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

## Jira

### `GET /jira/issues`
Get authenticated user's Jira issues.

**Query Params:**
```
project   string   optional  filter by project key
status    string   optional  filter by status
```

**Response `200`:**
```json
{
  "issues": [
    {
      "id": "10001",
      "key": "PROJ-123",
      "summary": "Fix login bug",
      "status": "In Progress",
      "stakeholders": ["alice@example.com", "bob@example.com"]
    }
  ]
}
```

---

### `GET /jira/issues/:ticket_key`
Get single issue detail with stakeholder custom field.

**Response `200`:**
```json
{
  "id": "10001",
  "key": "PROJ-123",
  "summary": "Fix login bug",
  "status": "In Progress",
  "stakeholders": ["alice@example.com"]
}
```

---

## Automation Rules

### `POST /automations`
Create a new automation rule.

**Request:**
```json
{
  "jira_ticket_id": "10001",
  "jira_ticket_key": "PROJ-123",
  "recipients": ["alice@example.com", "bob@example.com"],
  "cron_schedule": "0 9 * * 1"
}
```

**Response `201`:**
```json
{
  "id": "uuid",
  "jira_ticket_key": "PROJ-123",
  "recipients": ["alice@example.com", "bob@example.com"],
  "cron_schedule": "0 9 * * 1",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### `GET /automations`
List all automation rules for authenticated user.

**Response `200`:**
```json
{
  "automations": [
    {
      "id": "uuid",
      "jira_ticket_key": "PROJ-123",
      "recipients": ["alice@example.com"],
      "cron_schedule": "0 9 * * 1",
      "status": "active",
      "last_run_at": "2024-01-08T09:00:00Z",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

### `GET /automations/:id`
Get single automation rule detail.

**Response `200`:**
```json
{
  "id": "uuid",
  "jira_ticket_key": "PROJ-123",
  "jira_ticket_id": "10001",
  "recipients": ["alice@example.com"],
  "cron_schedule": "0 9 * * 1",
  "status": "active",
  "last_run_at": null,
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### `PATCH /automations/:id`
Update automation rule.

**Request** *(all fields optional)*:
```json
{
  "recipients": ["new@example.com"],
  "cron_schedule": "0 10 * * *",
  "status": "paused"
}
```

**Response `200`:** Updated automation object (same shape as GET single)

---

### `DELETE /automations/:id`
Delete automation rule and stop its schedule.

**Response `204`:** No content

---

### `POST /automations/:id/trigger`
Manually trigger a follow-up for this rule (Swift app CTA).

**Response `202`:**
```json
{
  "message": "follow-up triggered",
  "automation_id": "uuid"
}
```

---

## Email Threads

### `GET /email/threads`
List email threads for authenticated user.

**Query Params:**
```
automation_id   string   optional  filter by rule
status          string   optional  open | replied | closed
```

**Response `200`:**
```json
{
  "threads": [
    {
      "id": "uuid",
      "automation_id": "uuid",
      "ticket_id": "10001",
      "status": "replied",
      "last_synced_at": "2024-01-08T10:30:00Z"
    }
  ]
}
```

---

## Error Codes

| Code | HTTP Status | Meaning |
|---|---|---|
| `UNAUTHORIZED` | 401 | Missing or invalid JWT |
| `JIRA_TOKEN_EXPIRED` | 401 | Jira OAuth token expired, re-auth needed |
| `EMAIL_CREDENTIAL_NOT_FOUND` | 404 | User has no registered email credential |
| `AUTOMATION_NOT_FOUND` | 404 | Rule not found or not owned by user |
| `INVALID_CRON` | 422 | Cron expression invalid |
| `INVALID_RECIPIENTS` | 422 | Recipients empty or malformed |
| `IMAP_CONNECTION_FAILED` | 502 | Could not connect to IMAP server |
| `SMTP_CONNECTION_FAILED` | 502 | Could not connect to SMTP server |
| `JIRA_API_ERROR` | 502 | Upstream Jira API error |

