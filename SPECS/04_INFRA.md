# INFRA.md

---

## Environment Variables

All config loaded via Viper from `.env` file or OS environment.

```env
# Server
APP_PORT=8080
APP_ENV=development

# Encryption
AES_SECRET_KEY=32-char-secret-key-here-exactly!!

# DragonflyDB
DRAGONFLY_ADDR=localhost:6379
DRAGONFLY_PASSWORD=
DRAGONFLY_DB=0

# PostgreSQL (wired, not active)
POSTGRES_DSN=postgres://user:pass@localhost:5432/dbname

# Jira OAuth
JIRA_CLIENT_ID=
JIRA_CLIENT_SECRET=
JIRA_REDIRECT_URI=https://yourapp.com/api/v1/auth/jira/callback
JIRA_BASE_URL=https://api.atlassian.com

# Email Provider (hardcoded single provider)
IMAP_HOST=imap.gmail.com
IMAP_PORT=993
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587

# IMAP Polling
IMAP_POLL_INTERVAL_SECONDS=300
```

---

## DragonflyDB

Dragonfly is Redis-compatible — client is `go-redis/v9`.

**Connection:**
```
infra/dragonfly.go
  └── NewDragonflyClient(cfg Config) *redis.Client
      ├── Addr     ← DRAGONFLY_ADDR
      ├── Password ← DRAGONFLY_PASSWORD
      └── DB       ← DRAGONFLY_DB
```

**Usage conventions:**
- All values stored as JSON-marshalled structs
- TTL only applied to `OAuthToken` keys (expire slightly before `expires_at`)
- No TTL on credentials, users, automations, threads

**Key TTL Rules:**
```
oauth:{user_id}:{provider}   → TTL = expires_at - now - 5min buffer
all other keys               → no TTL (persistent until deleted)
```

---

## PostgreSQL

Driver: `pgx/v5`

```
infra/postgres.go
  └── NewPostgresPool(cfg Config) *pgxpool.Pool
      └── DSN ← POSTGRES_DSN
```

> Not called anywhere in MVP. Wired so future repo implementations can inject `*pgxpool.Pool` without restructuring infra bootstrap.

---

## AES Encryption

**Algorithm:** AES-256-GCM
- Key: 32-byte string from `AES_SECRET_KEY` env
- Nonce: randomly generated per encryption, prepended to ciphertext
- Output: base64-encoded `nonce + ciphertext`

```
infra/crypto.go
  ├── Encrypt(plaintext string, key []byte) (string, error)
  └── Decrypt(ciphertext string, key []byte) (string, error)
```

**Flow for email password:**
```
Register  → plaintext password → Encrypt() → store in Dragonfly
Use       → fetch from Dragonfly → Decrypt() → pass to IMAP/SMTP dialer
```

> Key is never logged. Plaintext password is never stored or returned in any response.

---

## IMAP

**Library:** `github.com/emersion/go-imap/v2`

**Connection per poll cycle:**
```
email_service.go (PollInbox)
  ├── Decrypt credential password
  ├── Dial IMAP_HOST:IMAP_PORT (TLS)
  ├── Login with email + decrypted password
  ├── Select INBOX
  ├── Fetch unseen messages
  ├── Match In-Reply-To / Message-ID headers against known threads
  ├── Update EmailThread status → "replied"
  └── Logout & close connection
```

**Polling:**
- Triggered by a background goroutine on app start
- Interval: `IMAP_POLL_INTERVAL_SECONDS`
- Runs for all users who have active automations + registered credentials
- Each user gets their own IMAP session per poll cycle

---

## SMTP

**Library:** `net/smtp` (stdlib) with `STARTTLS`

**Connection per send:**
```
email_service.go (SendFollowUp)
  ├── Decrypt credential password
  ├── Dial SMTP_HOST:SMTP_PORT
  ├── STARTTLS upgrade
  ├── Auth with email + decrypted password
  ├── Compose follow-up email (To, Subject, Body)
  ├── Send
  ├── Create EmailThread record
  └── Close connection
```

---

## Cron Engine

**Library:** `github.com/robfig/cron/v3`

```
cron/scheduler.go
  ├── NewScheduler(automationService, emailService) *Scheduler
  ├── Start()
  │   ├── Load all active AutomationRules from repo
  │   ├── Register each rule as a cron job (using rule.cron_schedule)
  │   └── Start cron runner
  ├── AddRule(rule AutomationRule)    ← called on POST /automations
  ├── RemoveRule(automationID string) ← called on DELETE /automations/:id
  └── Stop()
```

**Job execution per rule:**
```
1. Fetch latest Jira ticket status
2. Compose follow-up email body with ticket context
3. Call email_service.SendFollowUp()
4. Update rule.last_run_at
5. Create EmailThread record
```

**Rule lifecycle sync:**
```
POST /automations        → scheduler.AddRule()
DELETE /automations/:id  → scheduler.RemoveRule()
PATCH /automations/:id   status:"paused"  → scheduler.RemoveRule()
PATCH /automations/:id   status:"active"  → scheduler.AddRule()
```

---

## Bootstrap Order (`main.go`)

```
1. Load config (Viper)
2. Init infra
   ├── NewDragonflyClient()
   └── NewPostgresPool()      ← init only, not used
3. Init repositories
   └── dragonfly implementations
4. Init services
   └── inject repos
5. Init handlers
   └── inject services
6. Init cron scheduler
   ├── inject services
   └── scheduler.Start()
7. Init IMAP poller goroutine
8. Register Echo routes
9. Start Echo server
```

