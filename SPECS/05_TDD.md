# TDD_GUIDE.md

---

## Philosophy

- Tests are written **before or alongside** implementation, never after
- Every service method must have a corresponding test
- Handlers tested via HTTP-level integration tests using Echo's test utilities
- Repository implementations tested with a real Dragonfly instance (via Docker in CI)
- **No business logic is tested through the handler layer** — handler tests only assert HTTP shape

---

## Test Layers

```
tests/
├── unit/                        ← service logic, pure functions
│   ├── automation_service_test.go
│   ├── email_service_test.go
│   ├── jira_service_test.go
│   └── auth_service_test.go
│
├── integration/                 ← handler + echo, mocked services
│   ├── auth_handler_test.go
│   ├── automation_handler_test.go
│   ├── email_handler_test.go
│   └── jira_handler_test.go
│
├── repository/                  ← dragonfly repo, real connection
│   ├── user_repo_test.go
│   ├── automation_repo_test.go
│   └── email_repo_test.go
│
└── mocks/                       ← shared mocks used across all layers
    ├── user_repo_mock.go
    ├── automation_repo_mock.go
    ├── email_repo_mock.go
    ├── auth_service_mock.go
    ├── automation_service_mock.go
    ├── email_service_mock.go
    └── jira_service_mock.go
```

---

## Mock Conventions

All mocks are generated from interfaces using `github.com/stretchr/testify/mock`.

**Mock naming:**
```
Interface name: AutomationService
Mock name:      MockAutomationService
File:           tests/mocks/automation_service_mock.go
```

**Mock structure pattern:**
```go
type MockAutomationService struct {
    mock.Mock
}

func (m *MockAutomationService) CreateRule(ctx context.Context, rule domain.AutomationRule) error {
    args := m.Called(ctx, rule)
    return args.Error(0)
}
```

---

## Unit Test Conventions

Each service test file follows this structure:

```
TestXxx_SuccessCase
TestXxx_ErrorCase_RepoFails
TestXxx_ErrorCase_ValidationFails
TestXxx_EdgeCase_Xxx
```

**Example: AutomationService**
```
TestCreateRule_Success
TestCreateRule_InvalidCron
TestCreateRule_InvalidRecipients
TestCreateRule_RepoFails

TestDeleteRule_Success
TestDeleteRule_NotFound
TestDeleteRule_NotOwnedByUser

TestTriggerRule_Success
TestTriggerRule_EmailCredentialMissing
TestTriggerRule_IMAPConnectionFailed
TestTriggerRule_JiraTokenExpired
```

**Example: EmailService**
```
TestSendFollowUp_Success
TestSendFollowUp_DecryptFails
TestSendFollowUp_SMTPConnectionFailed
TestSendFollowUp_CredentialNotFound

TestPollInbox_Success_NoNewReplies
TestPollInbox_Success_ReplyDetected
TestPollInbox_IMAPConnectionFailed
TestPollInbox_ThreadMatchFound
TestPollInbox_ThreadMatchNotFound
```

**Example: AuthService**
```
TestExchangeJiraCode_Success
TestExchangeJiraCode_InvalidCode
TestRefreshJiraToken_Success
TestRefreshJiraToken_TokenExpired
TestRefreshJiraToken_RefreshFailed_PausesAutomations
```

---

## Integration Test Conventions

Handler tests use Echo's `httptest` — services are fully mocked.

**Pattern:**
```go
func TestCreateAutomation_Success(t *testing.T) {
    // 1. setup mock service
    mockSvc := new(mocks.MockAutomationService)
    mockSvc.On("CreateRule", mock.Anything, mock.Anything).Return(expectedRule, nil)

    // 2. setup echo + handler
    e := echo.New()
    h := handler.NewAutomationHandler(mockSvc)
    e.POST("/automations", h.CreateAutomation)

    // 3. fire request
    req := httptest.NewRequest(http.MethodPost, "/automations", body)
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    // 4. assert HTTP shape only
    assert.Equal(t, http.StatusCreated, rec.Code)
    assert.JSONEq(t, expectedJSON, rec.Body.String())
    mockSvc.AssertExpectations(t)
}
```

**Handler test cases per endpoint:**
```
POST /automations
  → 201 valid request
  → 422 invalid cron expression
  → 422 empty recipients
  → 401 missing auth token
  → 502 upstream service error

DELETE /automations/:id
  → 204 success
  → 404 not found
  → 401 unauthorized

POST /automations/:id/trigger
  → 202 triggered
  → 404 rule not found
  → 502 email send failed
```

---

## Repository Test Conventions

Repo tests run against a **real Dragonfly instance** (Docker).

```go
func TestUserRepo_SaveAndGet(t *testing.T) {
    client := setupTestDragonfly(t)   // spins up or connects to test instance
    repo := dragonfly.NewUserRepo(client)

    user := domain.User{...}
    err := repo.Save(ctx, user)
    assert.NoError(t, err)

    got, err := repo.GetByID(ctx, user.ID)
    assert.NoError(t, err)
    assert.Equal(t, user, got)
}
```

**Setup helper:**
```
tests/repository/helper_test.go
  ├── setupTestDragonfly(t) *redis.Client
  └── cleanupKeys(t, client, keys...)   ← always deferred in each test
```

> Each test cleans up its own keys. Never share state between repo tests.

---

## Crypto Unit Tests

```
TestEncrypt_Success
TestDecrypt_Success
TestEncrypt_Decrypt_Roundtrip
TestDecrypt_TamperedCiphertext_Fails
TestDecrypt_WrongKey_Fails
```

> Crypto tests live in `infra/crypto_test.go` — no mocks needed, pure function testing.

---

## CI Test Execution Order

```
1. go vet ./...
2. go test ./tests/unit/...           ← no external deps
3. go test ./tests/integration/...    ← no external deps (mocked)
4. docker run dragonfly               ← spin up for repo tests
5. go test ./tests/repository/...
6. docker stop dragonfly
```

---

## Coverage Targets (MVP)

| Layer | Target |
|---|---|
| Service (unit) | ≥ 80% |
| Handler (integration) | ≥ 70% |
| Repository | ≥ 60% |
| Crypto (infra) | 100% |

