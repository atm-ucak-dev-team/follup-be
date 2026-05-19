# DOMAIN.md

---

## Entities

### User
```
User
├── id          string    (uuid)
├── name        string
├── email       string    (unique)
└── created_at  time.Time
```

### OAuthToken *(per provider per user)*
```
OAuthToken
├── user_id       string
├── provider      string    ("jira")
├── access_token  string
├── refresh_token string
└── expires_at    time.Time
```

### EmailCredential
```
EmailCredential
├── user_id            string
├── email_address      string
├── encrypted_password string    (AES encrypted)
├── imap_host          string    (hardcoded at app level)
├── smtp_host          string    (hardcoded at app level)
└── created_at         time.Time
```

### AutomationRule
```
AutomationRule
├── id               string    (uuid)
├── user_id          string
├── jira_ticket_id   string
├── jira_ticket_key  string    (e.g. "PROJ-123")
├── recipients       []string  (email addresses)
├── cron_schedule    string    (cron expression, e.g. "0 9 * * 1")
├── status           string    ("active" | "paused")
├── last_run_at      *time.Time
└── created_at       time.Time
```

### EmailThread
```
EmailThread
├── id              string    (uuid)
├── user_id         string
├── automation_id   string
├── gmail_thread_id string    (IMAP thread/message ID)
├── ticket_id       string    (bound jira ticket)
├── status          string    ("open" | "replied" | "closed")
└── last_synced_at  time.Time
```

---

## Business Rules

### User & Credentials
- One user can have one `EmailCredential` only (upsert on re-register)
- Password is always stored encrypted, never in plaintext
- Password is decrypted only at runtime when IMAP/SMTP connection is needed

### OAuth & Jira
- One user can have one `OAuthToken` per provider
- Access token must be refreshed before expiry
- If refresh fails, automation rule is paused automatically

### AutomationRule
- One user can have many automation rules
- One rule is tied to exactly one Jira ticket
- `recipients` must have at least one entry
- `cron_schedule` must be a valid cron expression (5-part)
- Only `active` rules are picked up by the cron engine
- Deleting a rule stops its schedule immediately

### EmailThread
- A thread is created when a follow-up email is sent
- Incoming IMAP emails are matched to a thread via message threading headers (`In-Reply-To`, `Message-ID`)
- One ticket can have multiple threads over time (one per follow-up cycle)
- When a reply is detected, thread `status` → `"replied"`

---

## Storage Keys (DragonflyDB)

All data lives in Dragonfly as JSON-serialized structs. Key conventions:

```
user:{user_id}                          → User
oauth:{user_id}:{provider}              → OAuthToken
email_credential:{user_id}              → EmailCredential
automation:{automation_id}              → AutomationRule
automation:index:{user_id}              → []automation_id  (list of user's rules)
email_thread:{thread_id}                → EmailThread
email_thread:index:{automation_id}      → []thread_id
```

---

## Validation Rules

| Field | Rule |
|---|---|
| `cron_schedule` | Valid 5-part cron expression |
| `recipients` | Min 1, each must be valid email format |
| `jira_ticket_id` | Non-empty string |
| `email_address` | Valid email format |
| `status` (AutomationRule) | Only `"active"` or `"paused"` |
| `status` (EmailThread) | Only `"open"`, `"replied"`, `"closed"` |

