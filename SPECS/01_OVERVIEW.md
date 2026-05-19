# ARCHITECTURE.md

---

```
/cmd
  └── server/
      └── main.go              ← bootstrap: config, infra, routes, cron

/internal
  ├── domain/                  ← entities & interfaces only, zero dependencies
  │   ├── user.go
  │   ├── automation.go
  │   └── email.go
  │
  ├── repository/              ← data access layer
  │   ├── interfaces.go        ← all repo interfaces defined here
  │   └── dragonfly/           ← dragonfly implementations
  │       ├── user_repo.go
  │       ├── automation_repo.go
  │       └── email_repo.go
  │
  ├── service/                 ← business logic, depends on repo interfaces
  │   ├── interfaces.go        ← all service interfaces defined here
  │   ├── auth_service.go      ← jira oauth flow
  │   ├── automation_service.go
  │   ├── email_service.go     ← IMAP + SMTP logic
  │   └── jira_service.go
  │
  ├── handler/                 ← echo handlers, depends on service interfaces
  │   ├── auth_handler.go
  │   ├── automation_handler.go
  │   ├── email_handler.go
  │   └── jira_handler.go
  │
  ├── cron/
  │   └── scheduler.go         ← loads active rules, runs follow-ups on schedule
  │
  └── infra/
      ├── dragonfly.go         ← dragonfly client init
      ├── postgres.go          ← pgx client init (wired, unused)
      └── crypto.go            ← AES encrypt/decrypt helpers

/config
  └── config.go                ← viper loader, typed Config struct

/tests
  └── mocks/                   ← mock implementations of all interfaces
      ├── user_repo_mock.go
      ├── automation_repo_mock.go
      ├── email_repo_mock.go
      ├── auth_service_mock.go
      ├── automation_service_mock.go
      ├── email_service_mock.go
      └── jira_service_mock.go
```

---

## Dependency Rules

```
handler → service interface
service → repository interface
repository → infra (dragonfly/postgres)
domain ← depended on by all, depends on nothing
```

> ⚠️ Handlers must never import repository directly.
> ⚠️ Services must never import infra directly.
> ⚠️ Domain package must have zero internal imports.

---

## Pattern Conventions

**Repository Pattern**
- Every repo has an interface in `repository/interfaces.go`
- Dragonfly implementation lives under `repository/dragonfly/`
- Future PostgreSQL implementation drops in as `repository/postgres/` with zero changes to service layer

**Service Pattern**
- Every service has an interface in `service/interfaces.go`
- Services receive repo interfaces via constructor injection
- No global state

**Handler Pattern**
- Handlers receive service interfaces via constructor injection
- Handlers only do: parse request → call service → return response
- No business logic in handlers

---

## Key Design Decisions

| Decision | Reason |
|---|---|
| Interfaces at every layer | Easy to mock for TDD |
| All state in Dragonfly for MVP | Simplicity, no migration overhead |
| pgx wired but unused | Ready for persistence upgrade without restructure |
| Single app-level AES key | Simple for MVP, upgradeable to per-user key later |
| IMAP polling over webhooks | Simpler MVP setup, interval configurable via env |

