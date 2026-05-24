package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
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
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-access-token" {
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
		JiraAPIBaseURL: server.URL,
	}

	// Create service
	jiraService := NewJiraService(config)

	// Test GetIssues
	ctx := context.Background()
	issues, err := jiraService.GetIssues(ctx, "test-cloud-id", "test-access-token", "status = 'In Progress'", "10")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Key != "TEST-1" {
		t.Errorf("Expected issue key TEST-1, got %s", issues[0].Key)
	}

	if issues[0].TicketTitle != "Test Issue" {
		t.Errorf("Expected title 'Test Issue', got %s", issues[0].TicketTitle)
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
		JiraAPIBaseURL: server.URL,
	}

	jiraService := NewJiraService(config)

	ctx := context.Background()
	_, _ = jiraService.GetIssues(ctx, "test-cloud-id", "test-access-token", "project = MYPROJECT AND status = Done", "10")

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
	// Create mock server that returns 401 Unauthorized (token expired)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "token expired"}`))
	}))
	defer server.Close()

	config := &domain.Config{
		JiraAPIBaseURL: server.URL,
	}

	jiraService := NewJiraService(config)

	ctx := context.Background()
	_, err := jiraService.GetIssues(ctx, "test-cloud-id", "test-access-token", "", "")

	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}

	// Check that error message contains information about the failed request (the error is "Jira API error: status 401")
	if !contains(err.Error(), "Jira API error") {
		t.Errorf("Expected error message to contain 'Jira API error', got %v", err)
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
		JiraAPIBaseURL: server.URL,
	}

	jiraService := NewJiraService(config)

	ctx := context.Background()
	issue, err := jiraService.GetIssue(ctx, "test-cloud-id", "test-access-token", "TEST-1")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if issue == nil {
		t.Fatal("Expected issue, got nil")
	}

	if issue.TicketNumber != "TEST-1" {
		t.Errorf("Expected ticket number TEST-1, got %s", issue.TicketNumber)
	}

	if issue.TicketTitle != "Single Test Issue" {
		t.Errorf("Expected title 'Single Test Issue', got %s", issue.TicketTitle)
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
		JiraAPIBaseURL: server.URL,
	}

	jiraService := NewJiraService(config)

	ctx := context.Background()
	_, err := jiraService.GetIssue(ctx, "test-cloud-id", "test-access-token", "NONEXISTENT-1")

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
		JiraAPIBaseURL: server.URL,
	}

	jiraService := NewJiraService(config)

	ctx := context.Background()
	_, err := jiraService.GetIssues(ctx, "test-cloud-id", "test-access-token", "", "")

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
		JiraAPIBaseURL: server.URL,
	}

	jiraService := NewJiraService(config)

	ctx := context.Background()
	_, err := jiraService.GetIssue(ctx, "test-cloud-id", "test-access-token", "TEST-1")

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
