package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing
type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, &domain.ValidationError{Field: "id", Message: "not found"}
	}
	return user, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, &domain.ValidationError{Field: "email", Message: "not found"}
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id string) error {
	delete(m.users, id)
	return nil
}

type mockOAuthTokenRepo struct {
	tokens map[string]*domain.OAuthToken // key: "userID:provider"
}

func newMockOAuthTokenRepo() *mockOAuthTokenRepo {
	return &mockOAuthTokenRepo{
		tokens: make(map[string]*domain.OAuthToken),
	}
}

func (m *mockOAuthTokenRepo) Create(ctx context.Context, token *domain.OAuthToken) error {
	key := token.UserID + ":" + token.Provider
	m.tokens[key] = token
	return nil
}

func (m *mockOAuthTokenRepo) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*domain.OAuthToken, error) {
	key := userID + ":" + provider
	token, ok := m.tokens[key]
	if !ok {
		return nil, &domain.ValidationError{Field: "token", Message: "not found"}
	}
	return token, nil
}

func (m *mockOAuthTokenRepo) Update(ctx context.Context, token *domain.OAuthToken) error {
	key := token.UserID + ":" + token.Provider
	m.tokens[key] = token
	return nil
}

func (m *mockOAuthTokenRepo) Delete(ctx context.Context, userID, provider string) error {
	key := userID + ":" + provider
	delete(m.tokens, key)
	return nil
}

type mockAutomationRepo struct {
	rules map[string]*domain.AutomationRule
}

func newMockAutomationRepo() *mockAutomationRepo {
	return &mockAutomationRepo{
		rules: make(map[string]*domain.AutomationRule),
	}
}

func (m *mockAutomationRepo) Create(ctx context.Context, rule *domain.AutomationRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockAutomationRepo) GetByID(ctx context.Context, id string) (*domain.AutomationRule, error) {
	rule, ok := m.rules[id]
	if !ok {
		return nil, &domain.ValidationError{Field: "id", Message: "not found"}
	}
	return rule, nil
}

func (m *mockAutomationRepo) GetByUserID(ctx context.Context, userID string) ([]*domain.AutomationRule, error) {
	var rules []*domain.AutomationRule
	for _, rule := range m.rules {
		if rule.UserID == userID {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (m *mockAutomationRepo) GetActiveRules(ctx context.Context) ([]*domain.AutomationRule, error) {
	var rules []*domain.AutomationRule
	for _, rule := range m.rules {
		if rule.Status == domain.AutomationStatusActive {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (m *mockAutomationRepo) Update(ctx context.Context, rule *domain.AutomationRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockAutomationRepo) Delete(ctx context.Context, id string) error {
	delete(m.rules, id)
	return nil
}

// TestExchangeJiraCode_Success tests successful code exchange
func TestExchangeJiraCode_Success(t *testing.T) {
	// Setup test server responses
	tokenResp := map[string]interface{}{
		"access_token":  "test_access_token",
		"refresh_token": "test_refresh_token",
		"expires_in":    3600,
		"token_type":    "Bearer",
		"scope":         "read:jira-user read:jira-work offline_access",
	}

	accessibleResourcesResp := []map[string]interface{}{
		{
			"id":        "test-cloud-id-123",
			"name":      "Test Site",
			"url":       "https://test.atlassian.net",
			"scopes":    []string{"read:jira-user", "read:jira-work"},
			"avatarUrl": "https://example.com/site-avatar.png",
		},
	}

	userDetailsResp := map[string]interface{}{
		"accountId":    "user123",
		"displayName":  "Test User",
		"emailAddress": "test@example.com",
		"avatarUrls": map[string]string{
			"16x16": "https://example.com/avatar16.png",
			"24x24": "https://example.com/avatar24.png",
			"32x32": "https://example.com/avatar32.png",
			"48x48": "https://example.com/avatar48.png",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			// Handle POST request for token exchange
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(tokenResp)
		case "/oauth/token/accessible-resources":
			// Check Authorization header for API requests
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(accessibleResourcesResp)
		case "/ex/jira/test-cloud-id-123/rest/api/2/myself":
			// Check Authorization header for API requests
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(userDetailsResp)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Setup service
	config := &domain.Config{
		JiraAuthBaseURL:  server.URL,
		JiraAPIBaseURL:   server.URL,
		JiraClientID:     "test_client_id",
		JiraClientSecret: "test_secret",
		JiraRedirectURI:  "http://localhost/callback",
		JWTSecret:        "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	// Execute
	ctx := context.Background()
	user, tokenInfo, err := authService.(*AuthServiceImpl).ExchangeJiraCode(ctx, "test_code", "test_state")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user123", user.ID)
	assert.Equal(t, "Test User", user.Name)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "test-cloud-id-123", user.CloudID)
	assert.Equal(t, "https://example.com/avatar32.png", user.AvatarURL)

	// Verify token info is returned
	assert.NotNil(t, tokenInfo)
	assert.Equal(t, "test_access_token", tokenInfo.AccessToken)
	assert.Equal(t, "test_refresh_token", tokenInfo.RefreshToken)
	assert.Equal(t, int64(3600), tokenInfo.ExpiresIn)
	assert.Equal(t, "Bearer", tokenInfo.TokenType)
	assert.Contains(t, tokenInfo.Scope, "read:jira-user")

	// Verify OAuth token was saved
	oauthToken, err := oauthRepo.GetByUserIDAndProvider(ctx, "user123", "jira")
	require.NoError(t, err)
	assert.Equal(t, "test_access_token", oauthToken.AccessToken)
	assert.Equal(t, "test_refresh_token", oauthToken.RefreshToken)
}

// TestExchangeJiraCode_InvalidCode tests code exchange with invalid code
func TestExchangeJiraCode_InvalidCode(t *testing.T) {
	// Setup test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_code"})
	}))
	defer server.Close()

	// Setup service
	config := &domain.Config{
		JiraAuthBaseURL:  server.URL,
		JiraAPIBaseURL:   server.URL,
		JiraClientID:     "test_client_id",
		JiraClientSecret: "test_secret",
		JiraRedirectURI:  "http://localhost/callback",
		JWTSecret:        "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	// Execute
	ctx := context.Background()
	user, jwtToken, err := authService.(*AuthServiceImpl).ExchangeJiraCode(ctx, "invalid_code", "test_state")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Empty(t, jwtToken)
	assert.Contains(t, err.Error(), "failed to exchange code for token")
}

// TestRefreshJiraToken_Success tests successful token refresh
func TestRefreshJiraToken_Success(t *testing.T) {
	// Setup test server
	newTokenResp := map[string]interface{}{
		"access_token":  "new_access_token",
		"refresh_token": "new_refresh_token",
		"expires_in":    3600,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newTokenResp)
	}))
	defer server.Close()

	// Setup service
	config := &domain.Config{
		JiraAuthBaseURL:  server.URL,
		JiraAPIBaseURL:   server.URL,
		JiraClientID:     "test_client_id",
		JiraClientSecret: "test_secret",
		JWTSecret:        "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	// Create existing OAuth token
	ctx := context.Background()
	existingToken := &domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "old_access_token",
		RefreshToken: "old_refresh_token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
	}
	oauthRepo.Create(ctx, existingToken)

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	// Execute
	newAccessToken, err := authService.(*AuthServiceImpl).RefreshJiraToken(ctx, "user123")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "new_access_token", newAccessToken)

	// Verify token was updated
	updatedToken, err := oauthRepo.GetByUserIDAndProvider(ctx, "user123", "jira")
	require.NoError(t, err)
	assert.Equal(t, "new_access_token", updatedToken.AccessToken)
	assert.Equal(t, "new_refresh_token", updatedToken.RefreshToken)
}

// TestRefreshJiraToken_TokenExpired tests refresh with expired token
func TestRefreshJiraToken_TokenExpired(t *testing.T) {
	// Setup test server that returns unauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_refresh_token"})
	}))
	defer server.Close()

	// Setup service
	config := &domain.Config{
		JiraAuthBaseURL:  server.URL,
		JiraAPIBaseURL:   server.URL,
		JiraClientID:     "test_client_id",
		JiraClientSecret: "test_secret",
		JWTSecret:        "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	// Create existing OAuth token
	ctx := context.Background()
	existingToken := &domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "expired_access_token",
		RefreshToken: "expired_refresh_token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour), // Expired
	}
	oauthRepo.Create(ctx, existingToken)

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	// Execute
	newAccessToken, err := authService.(*AuthServiceImpl).RefreshJiraToken(ctx, "user123")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, newAccessToken)
	assert.Contains(t, err.Error(), "failed to refresh token")
}

// TestRefreshJiraToken_RefreshFailed_PausesAutomations tests automation pause on refresh failure
func TestRefreshJiraToken_RefreshFailed_PausesAutomations(t *testing.T) {
	// Setup test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_refresh_token"})
	}))
	defer server.Close()

	// Setup service
	config := &domain.Config{
		JiraAuthBaseURL:  server.URL,
		JiraAPIBaseURL:   server.URL,
		JiraClientID:     "test_client_id",
		JiraClientSecret: "test_secret",
		JWTSecret:        "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	// Create existing OAuth token
	ctx := context.Background()
	existingToken := &domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "expired_access_token",
		RefreshToken: "expired_refresh_token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}
	oauthRepo.Create(ctx, existingToken)

	// Create active automation rules
	automationRepo.Create(ctx, &domain.AutomationRule{
		ID:           "automation1",
		UserID:       "user123",
		JiraTicketID: "ticket1",
		Status:       domain.AutomationStatusActive,
		CreatedAt:    time.Now(),
	})

	automationRepo.Create(ctx, &domain.AutomationRule{
		ID:           "automation2",
		UserID:       "user123",
		JiraTicketID: "ticket2",
		Status:       domain.AutomationStatusActive,
		CreatedAt:    time.Now(),
	})

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	// Execute
	newAccessToken, err := authService.(*AuthServiceImpl).RefreshJiraToken(ctx, "user123")

	// Assert
	assert.Error(t, err)
	assert.Empty(t, newAccessToken)

	// Verify automations were paused
	rules, err := automationRepo.GetByUserID(ctx, "user123")
	require.NoError(t, err)
	assert.Len(t, rules, 2)

	for _, rule := range rules {
		assert.Equal(t, domain.AutomationStatusPaused, rule.Status, "Automation %s should be paused", rule.ID)
	}
}

// TestGenerateAuthURL tests authorization URL generation
func TestGenerateAuthURL(t *testing.T) {
	config := &domain.Config{
		JiraAuthBaseURL:  "https://auth.atlassian.com",
		JiraAPIBaseURL:   "https://api.atlassian.com",
		JiraClientID:     "test_client_id",
		JiraClientSecret: "test_secret",
		JiraRedirectURI:  "http://localhost/callback",
		JWTSecret:        "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	url := authService.(*AuthServiceImpl).GenerateAuthURL("test_state")

	// Check base URL (changed from /oauth/authorize to /authorize)
	assert.Contains(t, url, "https://auth.atlassian.com/authorize")

	// Check new parameters
	assert.Contains(t, url, "audience=api.atlassian.com")
	assert.Contains(t, url, "prompt=consent")

	// Check existing parameters
	assert.Contains(t, url, "client_id=test_client_id")
	assert.Contains(t, url, "redirect_uri=http%3A%2F%2Flocalhost%2Fcallback") // URL encoded
	assert.Contains(t, url, "state=test_state")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "scope=")
}

// TestValidateToken tests token validation
func TestValidateToken(t *testing.T) {
	config := &domain.Config{
		JWTSecret: "test-jwt-secret-minimum-32-chars",
	}

	userRepo := newMockUserRepo()
	oauthRepo := newMockOAuthTokenRepo()
	automationRepo := newMockAutomationRepo()

	authService := NewAuthService(userRepo, oauthRepo, automationRepo, config)

	// Test valid token
	validToken := &domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "valid_token",
		RefreshToken: "refresh_token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	assert.True(t, authService.(*AuthServiceImpl).ValidateToken(validToken))

	// Test expired token
	expiredToken := &domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "expired_token",
		RefreshToken: "refresh_token",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	}

	assert.False(t, authService.(*AuthServiceImpl).ValidateToken(expiredToken))
}
