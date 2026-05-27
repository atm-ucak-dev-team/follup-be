package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// MockEmailService for testing
type MockEmailService struct {
	saveCredentialFunc     func(ctx context.Context, userID, email, password, imapHost, smtpHost string) error
	getCredentialFunc      func(ctx interface{}, userID string) (*domain.EmailCredential, error)
	registerCredentialFunc func(ctx interface{}, cred *domain.EmailCredential) error
	sendFollowUpFunc       func(ctx interface{}, threadID, subject, body string, recipients []string) error
	checkForRepliesFunc    func(ctx interface{}) error
	decryptPasswordFunc    func(encryptedPassword string) (string, error)
	sendFollowUpAutoFunc   func(ctx context.Context, automationID string) error
	pollInboxFunc          func(ctx context.Context) error
}

func (m *MockEmailService) RegisterCredential(ctx interface{}, cred *domain.EmailCredential) error {
	if m.registerCredentialFunc != nil {
		return m.registerCredentialFunc(ctx, cred)
	}
	return nil
}

func (m *MockEmailService) GetCredential(ctx interface{}, userID string) (*domain.EmailCredential, error) {
	if m.getCredentialFunc != nil {
		return m.getCredentialFunc(ctx, userID)
	}
	return &domain.EmailCredential{
		UserID:            userID,
		EmailAddress:      "test@example.com",
		EncryptedPassword: "encrypted_password",
		IMAPHost:          "imap.example.com",
		SMTPHost:          "smtp.example.com",
	}, nil
}

func (m *MockEmailService) SendFollowUp(ctx interface{}, threadID, subject, body string, recipients []string) error {
	if m.sendFollowUpFunc != nil {
		return m.sendFollowUpFunc(ctx, threadID, subject, body, recipients)
	}
	return nil
}

func (m *MockEmailService) CheckForReplies(ctx interface{}) error {
	if m.checkForRepliesFunc != nil {
		return m.checkForRepliesFunc(ctx)
	}
	return nil
}

func (m *MockEmailService) DecryptPassword(encryptedPassword string) (string, error) {
	if m.decryptPasswordFunc != nil {
		return m.decryptPasswordFunc(encryptedPassword)
	}
	return "decrypted", nil
}

// Implement new interface methods with correct signatures
func (m *MockEmailService) SaveCredential(ctx context.Context, userID, email, password, imapHost, smtpHost string) error {
	if m.saveCredentialFunc != nil {
		return m.saveCredentialFunc(ctx, userID, email, password, imapHost, smtpHost)
	}
	return nil
}

func (m *MockEmailService) SendFollowUpByAutomation(ctx context.Context, automationID string) error {
	if m.sendFollowUpAutoFunc != nil {
		return m.sendFollowUpAutoFunc(ctx, automationID)
	}
	return nil
}

func (m *MockEmailService) PollInbox(ctx context.Context) error {
	if m.pollInboxFunc != nil {
		return m.pollInboxFunc(ctx)
	}
	return nil
}

// TestSaveCredentials_Success tests successful credential saving
func TestSaveCredentials_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{
		saveCredentialFunc: func(ctx context.Context, userID, email, password, imapHost, smtpHost string) error {
			return nil
		},
	}
	h := NewEmailHandler(mockService)

	// Create request body
	reqBody := map[string]string{
		"email_address": "test@example.com",
		"password":      "test_password",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/email/credentials", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context (simulating JWT middleware)
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.SaveCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Verify response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "email credential saved", resp["message"])
	assert.Equal(t, "test@example.com", resp["email_address"])
	assert.Nil(t, resp["password"]) // Password should never be in response
}

// TestSaveCredentials_MissingFields tests validation of missing required fields
func TestSaveCredentials_MissingEmail(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{}
	h := NewEmailHandler(mockService)

	// Create request body with missing email
	reqBody := map[string]string{
		"password": "test_password",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/email/credentials", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.SaveCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_EMAIL_ADDRESS", errObj["code"])
}

// TestSaveCredentials_MissingPassword tests validation of missing password
func TestSaveCredentials_MissingPassword(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{}
	h := NewEmailHandler(mockService)

	// Create request body with missing password
	reqBody := map[string]string{
		"email_address": "test@example.com",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/email/credentials", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.SaveCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_PASSWORD", errObj["code"])
}

// TestSaveCredentials_Unauthorized tests authentication requirement
func TestSaveCredentials_Unauthorized(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{}
	h := NewEmailHandler(mockService)

	// Create request body
	reqBody := map[string]string{
		"email_address": "test@example.com",
		"password":      "test_password",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create request without user context
	req := httptest.NewRequest(http.MethodPost, "/email/credentials", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Note: NOT setting user_id to simulate missing auth

	// Execute
	err := h.SaveCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errObj["code"])
}

// TestSaveCredentials_ServiceError tests error handling from service layer
func TestSaveCredentials_ServiceError(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{
		saveCredentialFunc: func(ctx context.Context, userID, email, password, imapHost, smtpHost string) error {
			return assert.AnError
		},
	}
	h := NewEmailHandler(mockService)

	// Create request body
	reqBody := map[string]string{
		"email_address": "test@example.com",
		"password":      "test_password",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/email/credentials", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.SaveCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "CREDENTIAL_SAVE_FAILED", errObj["code"])
}

// TestGetCredentials_Success tests successful credential retrieval
func TestGetCredentials_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "encrypted_password",
				IMAPHost:          "imap.example.com",
				SMTPHost:          "smtp.example.com",
			}, nil
		},
	}
	h := NewEmailHandler(mockService)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/email/credentials", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var resp CredentialsResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, "test@example.com", resp.EmailAddress)
	assert.NotEmpty(t, resp.CreatedAt)

	// Ensure password is not exposed
	var respMap map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &respMap)
	_, passwordExposed := respMap["password"]
	assert.False(t, passwordExposed, "Password should never be exposed in response")
}

// TestGetCredentials_NotFound tests handling of missing credentials
func TestGetCredentials_NotFound(t *testing.T) {
	// Setup
	e := echo.New()

	// Create a proper not found error
	customNotFoundErr := &mockCredentialNotFoundError{message: "email credential not found"}

	mockService := &MockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return nil, customNotFoundErr
		},
	}
	h := NewEmailHandler(mockService)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/email/credentials", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "EMAIL_CREDENTIAL_NOT_FOUND", errObj["code"])
}

// TestGetCredentials_PasswordNotExposed ensures password field is never in response
func TestGetCredentials_PasswordNotExposed(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "super_secret_encrypted_password",
				IMAPHost:          "imap.example.com",
				SMTPHost:          "smtp.example.com",
			}, nil
		},
	}
	h := NewEmailHandler(mockService)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/email/credentials", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse response
	var respMap map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &respMap)

	// Ensure password field doesn't exist in response
	_, hasPasswordField := respMap["password"]
	assert.False(t, hasPasswordField, "Password field should not be present in response")

	_, hasEncryptedPasswordField := respMap["encrypted_password"]
	assert.False(t, hasEncryptedPasswordField, "Encrypted password field should not be present in response")

	// Verify expected fields are present
	assert.Contains(t, respMap, "email_address")
	assert.Contains(t, respMap, "created_at")
}

// TestGetCredentials_Unauthorized tests authentication requirement
func TestGetCredentials_Unauthorized(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{}
	h := NewEmailHandler(mockService)

	// Create request without user context
	req := httptest.NewRequest(http.MethodGet, "/email/credentials", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Note: NOT setting user_id to simulate missing auth

	// Execute
	err := h.GetCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errObj["code"])
}

// TestSaveCredentials_InvalidJSON tests handling of invalid JSON in request
func TestSaveCredentials_InvalidJSON(t *testing.T) {
	// Setup
	e := echo.New()
	mockService := &MockEmailService{}
	h := NewEmailHandler(mockService)

	// Create request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/email/credentials", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user context
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.SaveCredentials(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify error response
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Contains(t, resp, "error")
	errObj := resp["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUEST_BODY", errObj["code"])
}

// Custom error type for testing not found scenarios
type mockCredentialNotFoundError struct {
	message string
}

func (e *mockCredentialNotFoundError) Error() string {
	return e.message
}
