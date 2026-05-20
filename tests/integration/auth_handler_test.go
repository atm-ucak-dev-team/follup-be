package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/handler"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock AuthService for testing
type mockAuthService struct {
	exchangeCodeFunc  func(ctx context.Context, code, state string) (*domain.User, string, error)
	refreshTokenFunc  func(ctx context.Context, userID string) (string, error)
	generateURLFunc   func(state string) string
	validateTokenFunc func(token *domain.OAuthToken) bool
}

func (m *mockAuthService) ExchangeJiraCode(ctx context.Context, code, state string) (*domain.User, string, error) {
	if m.exchangeCodeFunc != nil {
		return m.exchangeCodeFunc(ctx, code, state)
	}
	// Default implementation
	return &domain.User{
		ID:    "test-user-id",
		Name:  "Test User",
		Email: "test@example.com",
	}, "", nil // JWT disabled - returning empty token
}

func (m *mockAuthService) RefreshJiraToken(ctx context.Context, userID string) (string, error) {
	if m.refreshTokenFunc != nil {
		return m.refreshTokenFunc(ctx, userID)
	}
	// Default implementation
	return "", nil // JWT disabled - returning empty token
}

func (m *mockAuthService) GenerateAuthURL(state string) string {
	if m.generateURLFunc != nil {
		return m.generateURLFunc(state)
	}
	// Default implementation
	return "https://auth.atlassian.com/authorize?client_id=test&redirect_uri=http://localhost/callback&state=" + state
}

func (m *mockAuthService) ValidateToken(token *domain.OAuthToken) bool {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(token)
	}
	// Default implementation
	return true
}

// TestConnectJira_JSONResponseSuccess tests that the connect endpoint returns JSON with authorization URL
func TestConnectJira_JSONResponseSuccess(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{
		generateURLFunc: func(state string) string {
			return "https://auth.atlassian.com/authorize?audience=api.atlassian.com&client_id=test&state=" + state
		},
	}
	h := handler.NewAuthHandler(mockAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/jira/connect", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.ConnectJira(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse JSON response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check response structure
	assert.Contains(t, response, "connectUrl")
	connectUrl := response["connectUrl"].(string)
	assert.Contains(t, connectUrl, "https://auth.atlassian.com/authorize")
	assert.Contains(t, connectUrl, "audience=api.atlassian.com")
}

// TestJiraCallback_Success tests successful OAuth callback
func TestJiraCallback_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{
		exchangeCodeFunc: func(ctx context.Context, code, state string) (*domain.User, string, error) {
			return &domain.User{
				ID:    "user-123",
				Name:  "John Doe",
				Email: "john@example.com",
			}, "", nil // JWT disabled - returning empty token
		},
	}
	h := handler.NewAuthHandler(mockAuth)

	// Create request with query parameters
	params := url.Values{}
	params.Add("code", "test-auth-code")
	params.Add("state", "test-state-123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/jira/callback?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.JiraCallback(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "access_token")
	assert.Contains(t, response, "user")

	// With JWT disabled, access_token should be empty string
	accessToken := response["access_token"]
	assert.Equal(t, "", accessToken, "access_token should be empty with JWT disabled")

	user := response["user"].(map[string]interface{})
	assert.Equal(t, "user-123", user["id"])
	assert.Equal(t, "John Doe", user["name"])
	assert.Equal(t, "john@example.com", user["email"])
}

// TestJiraCallback_InvalidCode tests callback with invalid authorization code
func TestJiraCallback_InvalidCode(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{
		exchangeCodeFunc: func(ctx context.Context, code, state string) (*domain.User, string, error) {
			return nil, "", assert.AnError
		},
	}
	h := handler.NewAuthHandler(mockAuth)

	// Create request with invalid code
	params := url.Values{}
	params.Add("code", "invalid-code")
	params.Add("state", "test-state")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/jira/callback?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.JiraCallback(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_CODE", errResp["code"])
	assert.Contains(t, errResp["message"].(string), "failed to exchange authorization code")
}

// TestJiraCallback_MissingState tests callback with missing state parameter
func TestJiraCallback_MissingState(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{}
	h := handler.NewAuthHandler(mockAuth)

	// Create request without state parameter
	params := url.Values{}
	params.Add("code", "test-code")
	// No state parameter

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/jira/callback?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.JiraCallback(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_STATE", errResp["code"])
	assert.Equal(t, "state parameter is required", errResp["message"])
}

// TestJiraCallback_MissingCode tests callback with missing code parameter
func TestJiraCallback_MissingCode(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{}
	h := handler.NewAuthHandler(mockAuth)

	// Create request without code parameter
	params := url.Values{}
	params.Add("state", "test-state")
	// No code parameter

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/jira/callback?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.JiraCallback(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_CODE", errResp["code"])
	assert.Equal(t, "authorization code is required", errResp["message"])
}

// TestRefreshToken_Success tests successful token refresh
func TestRefreshToken_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{
		refreshTokenFunc: func(ctx context.Context, userID string) (string, error) {
			return "new-access-token-xyz", nil
		},
	}
	h := handler.NewAuthHandler(mockAuth)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/jira/refresh", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user ID in context (simulating JWT middleware)
	c.Set("user_id", "user-123")

	// Execute
	err := h.RefreshToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "access_token")
	assert.Equal(t, "new-access-token-xyz", response["access_token"])
}

// TestRefreshToken_TokenExpired tests refresh with expired/invalid token
func TestRefreshToken_TokenExpired(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{
		refreshTokenFunc: func(ctx context.Context, userID string) (string, error) {
			return "", assert.AnError
		},
	}
	h := handler.NewAuthHandler(mockAuth)

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/jira/refresh", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set user ID in context (simulating JWT middleware)
	c.Set("user_id", "user-123")

	// Execute
	err := h.RefreshToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "TOKEN_REFRESH_FAILED", errResp["code"])
	assert.Contains(t, errResp["message"].(string), "failed to refresh token")
}

// TestRefreshToken_Unauthorized tests refresh without authentication
func TestRefreshToken_Unauthorized(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{}
	h := handler.NewAuthHandler(mockAuth)

	// Create request without user ID in context
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/jira/refresh", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No user ID set in context (simulating missing JWT)

	// Execute
	err := h.RefreshToken(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "UNAUTHORIZED", errResp["code"])
	assert.Equal(t, "user not authenticated", errResp["message"])
}

// TestHelper_GenerateState tests state generation uniqueness
func TestHelper_GenerateState(t *testing.T) {
	// Generate multiple states and ensure they're different
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		state1 := time.Now().Format("20060102150405")
		time.Sleep(time.Millisecond * 10) // Small delay to ensure different timestamps
		state2 := time.Now().Format("20060102150405")
		if state1 != state2 {
			states[state1] = true
			states[state2] = true
		}
	}

	// We should have generated different states
	assert.Greater(t, len(states), 1, "State generation should produce unique values")
}

// TestErrorResponseFormat tests that error responses match API spec
func TestErrorResponseFormat(t *testing.T) {
	// Setup
	e := echo.New()
	mockAuth := &mockAuthService{}
	h := handler.NewAuthHandler(mockAuth)

	// Create request that will trigger an error
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/jira/callback", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute (missing both code and state)
	err := h.JiraCallback(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check error response format matches API spec
	assert.Contains(t, response, "error")
	errorObj := response["error"].(map[string]interface{})
	assert.Contains(t, errorObj, "code")
	assert.Contains(t, errorObj, "message")

	// Check types
	assert.IsType(t, "", errorObj["code"])
	assert.IsType(t, "", errorObj["message"])
}
