# Jira Email Automation Backend

## Overview
Clean architecture Go backend for Jira OAuth integration, email automation, and follow-up tracking.

## Tech Stack
- **Framework**: Echo v4
- **Config**: Viper
- **Cache**: DragonflyDB (Redis-compatible)
- **Database**: PostgreSQL with pgx
- **Email**: IMAP v2
- **Scheduler**: Cron v3
- **Testing**: Testify

## Project Structure
```
/cmd/server          - Application entry point
/internal/domain     - Entities & interfaces (zero dependencies)
/internal/repository - Data access layer
/internal/service    - Business logic
/internal/handler    - HTTP handlers
/internal/cron       - Scheduled tasks
/internal/infra      - Infrastructure code
/config              - Configuration files
/tests               - Unit, integration, and repository tests
```

## Getting Started

### Prerequisites
- Go 1.21+
- DragonflyDB
- PostgreSQL

### Installation
```bash
go mod download
```

### Running
```bash
go run cmd/server/main.go
```

### Testing
```bash
go test ./...
```

## Development
See `CLAUDE.md` for development guidelines.
See `SPECS/` for technical specifications.
