package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bomanarakasura/jira-email-automation/internal/domain"
)

// MockOAuthTokenRepository is a mock implementation of OAuthTokenRepository
type MockOAuthTokenRepository struct {
	getByUserIDAndProviderFunc func(ctx context.Context, userID, provider string) (*domain.OAuthToken, error)
}

func (m *MockOAuthTokenRepository) Create(ctx context.Context, token *domain.OAuthToken) error {
	return nil
}

func (m *MockOAuthTokenRepository) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*domain.OAuthToken, error) {
	if m.getByUserIDAndProviderFunc != nil {
		return m.getByUserIDAndProviderFunc(ctx, userID, provider)
	}
	return &domain.OAuthToken{
		UserID:       userID,
		Provider:     provider,
		AccessToken:  "test-token",
		RefreshToken: "test-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}, nil
}

func (m *MockOAuthTokenRepository) Update(ctx context.Context, token *domain.OAuthToken) error {
	return nil
}

func (m *MockOAuthTokenRepository) Delete(ctx context.Context, userID, provider string) error {
	return nil
}

// TestGetIssues_Success tests successful retrieval of issues
func TestGetIssues_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		// Send mock response
		response := `{
			"startAt": 0,
			"maxResults": 50,
			"total": 1,
			"issues": [
				{
					"id": "12345",
					"key": "TEST-1",
					"fields": {
						"summary": "Test Issue",
						"status": {
							"name": "In Progress"
						}
					}
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create config
	config := &domain.Config{
		JiraBaseURL: server.URL,
	}

	// Create mock repository
	mockRepo := &MockOAuthTokenRepository{}

	// Create service
	jiraService := NewJiraService(mockRepo, config)

	// Test GetIssues
	ctx := context.Background()
	issues, err := jiraService.GetIssues(ctx, "user123", "TEST", "In Progress")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Key != "TEST-1" {
		t.Errorf("Expected issue key TEST-1, got %s", issues[0].Key)
	}

	if issues[0].Summary != "Test Issue" {
		t.Errorf("Expected summary 'Test Issue', got %s", issues[0].Summary)
	}

	if issues[0].Status != "In Progress" {
		t.Errorf("Expected status 'In Progress', got %s", issues[0].Status)
	}
}

// TestGetIssues_WithFilters tests JQL query building with filters
func TestGetIssues_WithFilters(t *testing.T) {
	// Create mock server that captures the query parameters
	var capturedJQL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")

		response := `{
			"startAt": 0,
			"maxResults": 50,
			"total": 0,
			"issues": []
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := &domain.Config{
		JiraBaseURL: server.URL,
	}

	mockRepo := &MockOAuthTokenRepository{}
	jiraService := NewJiraService(mockRepo, config)

	ctx := context.Background()
	_, _ = jiraService.GetIssues(ctx, "user123", "MYPROJECT", "Done")

	// Verify JQL query contains filters
	if capturedJQL == "" {
		t.Error("Expected JQL query to be set")
	}

	// Check that project and status filters are present
	if !contains(capturedJQL, "project") && !contains(capturedJQL, "MYPROJECT") {
		t.Error("Expected project filter in JQL query")
	}

	if !contains(capturedJQL, "status") && !contains(capturedJQL, "Done") {
		t.Error("Expected status filter in JQL query")
	}
}

// TestGetIssues_TokenExpired tests handling of expired/invalid tokens
func TestGetIssues_TokenExpired(t *testing.T) {
	// Create mock repository that returns error
	mockRepo := &MockOAuthTokenRepository{
		getByUserIDAndProviderFunc: func(ctx context.Context, userID, provider string) (*domain.OAuthToken, error) {
			return nil, &domain.ValidationError{Field: "token", Message: "not found"}
		},
	}

	config := &domain.Config{
		JiraBaseURL: "https://api.atlassian.com",
	}

	jiraService := NewJiraService(mockRepo, config)

	ctx := context.Background()
	_, err := jiraService.GetIssues(ctx, "user123", "", "")

	if err == nil {
		t.Error("Expected error for missing token, got nil")
	}

	if !contains(err.Error(), "failed to get OAuth token") {
		t.Errorf("Expected token error, got %v", err)
	}
}

// TestGetIssue_Success tests successful retrieval of a single issue
func TestGetIssue_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		response := `{
			"id": "12345",
			"key": "TEST-1",
			"fields": {
				"summary": "Single Test Issue",
				"status": {
					"name": "To Do"
				}
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	config := &domain.Config{
		JiraBaseURL: server.URL,
	}

	mockRepo := &MockOAuthTokenRepository{}
	jiraService := NewJiraService(mockRepo, config)

	ctx := context.Background()
	issue, err := jiraService.GetIssue(ctx, "user123", "TEST-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if issue == nil {
		t.Fatal("Expected issue, got nil")
	}

	if issue.Key != "TEST-1" {
		t.Errorf("Expected issue key TEST-1, got %s", issue.Key)
	}

	if issue.Summary != "Single Test Issue" {
		t.Errorf("Expected summary 'Single Test Issue', got %s", issue.Summary)
	}
}

// TestGetIssue_NotFound tests handling of non-existent issues
func TestGetIssue_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages": ["Issue does not exist"]}`))
	}))
	defer server.Close()

	config := &domain.Config{
		JiraBaseURL: server.URL,
	}

	mockRepo := &MockOAuthTokenRepository{}
	jiraService := NewJiraService(mockRepo, config)

	ctx := context.Background()
	_, err := jiraService.GetIssue(ctx, "user123", "NONEXISTENT-1")

	if err == nil {
		t.Error("Expected error for non-existent issue, got nil")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got %v", err)
	}
}

// TestGetIssues_APIError tests handling of Jira API errors
func TestGetIssues_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessages": ["Invalid query"]}`))
	}))
	defer server.Close()

	config := &domain.Config{
		JiraBaseURL: server.URL,
	}

	mockRepo := &MockOAuthTokenRepository{}
	jiraService := NewJiraService(mockRepo, config)

	ctx := context.Background()
	_, err := jiraService.GetIssues(ctx, "user123", "", "")

	if err == nil {
		t.Error("Expected error for invalid API request, got nil")
	}

	if !contains(err.Error(), "Jira API error") {
		t.Errorf("Expected Jira API error, got %v", err)
	}
}

// TestGetIssue_APIErrorWithFieldErrors tests handling of Jira API field errors
func TestGetIssue_APIErrorWithFieldErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors": {"field": "Invalid value"}}`))
	}))
	defer server.Close()

	config := &domain.Config{
		JiraBaseURL: server.URL,
	}

	mockRepo := &MockOAuthTokenRepository{}
	jiraService := NewJiraService(mockRepo, config)

	ctx := context.Background()
	_, err := jiraService.GetIssue(ctx, "user123", "TEST-1")

	if err == nil {
		t.Error("Expected error for invalid API request, got nil")
	}

	if !contains(err.Error(), "field") && !contains(err.Error(), "Jira API error") {
		t.Errorf("Expected field error, got %v", err)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}