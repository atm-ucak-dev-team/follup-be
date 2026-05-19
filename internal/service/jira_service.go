package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/repository"
)

// JiraServiceImpl implements the JiraService interface
type JiraServiceImpl struct {
	oauthRepo repository.OAuthTokenRepository
	config    *domain.Config
	httpClient *http.Client
}

// NewJiraService creates a new JiraService instance
func NewJiraService(
	oauthRepo repository.OAuthTokenRepository,
	config *domain.Config,
) JiraService {
	return &JiraServiceImpl{
		oauthRepo: oauthRepo,
		config:    config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Jira API response structures
type jiraSearchResponse struct {
	StartAt    int           `json:"startAt"`
	MaxResults int           `json:"maxResults"`
	Total      int           `json:"total"`
	Issues     []jiraIssueItem `json:"issues"`
}

type jiraIssueItem struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		// Custom field for stakeholders - will be extracted dynamically
	} `json:"fields"`
}

type jiraIssueResponse struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status struct {
			Name string `json:"name"`
		} `json:"status"`
		// Custom field for stakeholders - will be extracted dynamically
	} `json:"fields"`
}

type jiraErrorResponse struct {
	ErrorMessages []string `json:"errorMessages"`
	Errors       map[string]string `json:"errors"`
}

// GetIssues retrieves Jira issues for a user with optional project and status filters
func (s *JiraServiceImpl) GetIssues(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
	// Get OAuth token for user
	token, err := s.getValidToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Build JQL query
	jql := s.buildJQLQuery(project, status)

	// Make API request
	searchURL := fmt.Sprintf("%s/rest/api/3/search", s.config.JiraBaseURL)
	req, err := s.createJiraRequest(ctx, "POST", searchURL, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add JQL query parameters
	params := url.Values{}
	params.Add("jql", jql)
	req.URL.RawQuery = params.Encode()

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		return nil, s.handleJiraError(resp)
	}

	// Parse response
	var searchResp jiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to domain entities
	issues := make([]domain.JiraIssue, len(searchResp.Issues))
	for i, item := range searchResp.Issues {
		issues[i] = domain.JiraIssue{
			ID:           item.ID,
			Key:          item.Key,
			Summary:      item.Fields.Summary,
			Status:       item.Fields.Status.Name,
			Stakeholders: []string{}, // Will be populated when custom field is known
		}
	}

	return issues, nil
}

// GetIssue retrieves a single Jira issue by ticket key
func (s *JiraServiceImpl) GetIssue(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error) {
	// Get OAuth token for user
	token, err := s.getValidToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Make API request
	issueURL := fmt.Sprintf("%s/rest/api/3/issue/%s", s.config.JiraBaseURL, ticketKey)
	req, err := s.createJiraRequest(ctx, "GET", issueURL, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Handle not found
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found: %s", ticketKey)
	}

	// Handle other errors
	if resp.StatusCode != http.StatusOK {
		return nil, s.handleJiraError(resp)
	}

	// Parse response
	var issueResp jiraIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to domain entity
	issue := &domain.JiraIssue{
		ID:           issueResp.ID,
		Key:          issueResp.Key,
		Summary:      issueResp.Fields.Summary,
		Status:       issueResp.Fields.Status.Name,
		Stakeholders: []string{}, // Will be populated when custom field is known
	}

	return issue, nil
}

// GetTicket retrieves a Jira ticket by ID (placeholder for compatibility)
func (s *JiraServiceImpl) GetTicket(ctx interface{}, ticketID string) (*domain.JiraTicket, error) {
	// This is a placeholder for the existing interface method
	// The main functionality is implemented in GetIssue
	return nil, fmt.Errorf("not implemented")
}

// UpdateTicketStatus updates the status of a Jira ticket (placeholder for compatibility)
func (s *JiraServiceImpl) UpdateTicketStatus(ctx interface{}, ticketID, status string) error {
	return fmt.Errorf("not implemented")
}

// AddComment adds a comment to a Jira ticket (placeholder for compatibility)
func (s *JiraServiceImpl) AddComment(ctx interface{}, ticketID, comment string) error {
	return fmt.Errorf("not implemented")
}

// GetAuthenticatedUser retrieves the authenticated Jira user (placeholder for compatibility)
func (s *JiraServiceImpl) GetAuthenticatedUser(ctx interface{}) (*domain.User, error) {
	return nil, fmt.Errorf("not implemented")
}

// Helper methods

// getValidToken gets a valid OAuth token for the user, refreshing if necessary
func (s *JiraServiceImpl) getValidToken(ctx context.Context, userID string) (string, error) {
	token, err := s.oauthRepo.GetByUserIDAndProvider(ctx, userID, "jira")
	if err != nil {
		return "", fmt.Errorf("failed to get OAuth token: %w", err)
	}

	// Check if token needs refresh (will be implemented with AuthService)
	// For now, just return the access token
	return token.AccessToken, nil
}

// buildJQLQuery constructs a JQL query based on filters
func (s *JiraServiceImpl) buildJQLQuery(project, status string) string {
	jql := "assignee = currentUser()"

	if project != "" {
		jql += fmt.Sprintf(" AND project = \"%s\"", project)
	}

	if status != "" {
		jql += fmt.Sprintf(" AND status = \"%s\"", status)
	}

	return jql
}

// createJiraRequest creates an authenticated HTTP request to Jira API
func (s *JiraServiceImpl) createJiraRequest(ctx context.Context, method, url, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// handleJiraError processes error responses from Jira API
func (s *JiraServiceImpl) handleJiraError(resp *http.Response) error {
	var errResp jiraErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return fmt.Errorf("Jira API error: status %d", resp.StatusCode)
	}

	if len(errResp.ErrorMessages) > 0 {
		return fmt.Errorf("Jira API error: %s", errResp.ErrorMessages[0])
	}

	if len(errResp.Errors) > 0 {
		for key, value := range errResp.Errors {
			return fmt.Errorf("Jira API error: %s: %s", key, value)
		}
	}

	return fmt.Errorf("Jira API error: status %d", resp.StatusCode)
}