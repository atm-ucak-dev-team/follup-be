package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
)

// JiraServiceImpl implements the JiraService interface
type JiraServiceImpl struct {
	config     *domain.Config
	httpClient *http.Client
}

// NewJiraService creates a new JiraService instance
func NewJiraService(
	config *domain.Config,
) JiraService {
	return &JiraServiceImpl{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Jira API response structures
type jiraSearchResponse struct {
	StartAt    int             `json:"startAt"`
	MaxResults int             `json:"maxResults"`
	Total      int             `json:"total"`
	Issues     []jiraIssueItem `json:"issues"`
}

// JQL search API response structures
type jiraJQLSearchResponse struct {
	Issues []jiraJQLIssueItem `json:"issues"`
	IsLast bool               `json:"isLast"`
}

type jiraJQLIssueItem struct {
	ID     string             `json:"id"`
	Self   string             `json:"self"`
	Key    string             `json:"key"`
	Fields jiraJQLIssueFields `json:"fields"`
}

type jiraJQLIssueFields struct {
	Summary          string        `json:"summary"`
	Customfield10072 string        `json:"customfield_10072"`
	Status           jiraJQLStatus `json:"status"`
}

type jiraJQLStatus struct {
	Name           string                `json:"name"`
	StatusCategory jiraJQLStatusCategory `json:"statusCategory"`
}

type jiraJQLStatusCategory struct {
	ColorName string `json:"colorName"`
}

type jiraIssueItem struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
		// Custom field for stakeholders - will be extracted dynamically
	} `json:"fields"`
}

// Detailed Jira issue response structures
type jiraIssueDetailResponse struct {
	ID     string                `json:"id"`
	Self   string                `json:"self"`
	Key    string                `json:"key"`
	Fields jiraIssueDetailFields `json:"fields"`
}

type jiraIssueDetailFields struct {
	Summary          string           `json:"summary"`
	Customfield10072 string           `json:"customfield_10072"`
	LastViewed       string           `json:"lastViewed"`
	Status           jiraDetailStatus `json:"status"`
	Creator          jiraDetailUser   `json:"creator"`
	Assignee         *jiraDetailUser  `json:"assignee"` // Pointer to handle null
}

type jiraDetailStatus struct {
	Name           string                   `json:"name"`
	StatusCategory jiraDetailStatusCategory `json:"statusCategory"`
}

type jiraDetailStatusCategory struct {
	ColorName string `json:"colorName"`
}

type jiraDetailUser struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

type jiraIssueResponse struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
		// Custom field for stakeholders - will be extracted dynamically
	} `json:"fields"`
}

type jiraErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

// GetIssues retrieves Jira issues using Atlassian Cloud API JQL search
func (s *JiraServiceImpl) GetIssues(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
	// Build JQL query
	jql := "assignee = currentUser()"
	if search != "" {
		jql += fmt.Sprintf(` and (key = "%s" or summary ~ "%s")`, search, search)
	}

	// Set default limit if not provided
	maxResults := "10" // default
	if limit != "" {
		maxResults = limit
	}

	// Build API request URL for JQL search endpoint
	searchURL := fmt.Sprintf("%s/ex/jira/%s/rest/api/2/search/jql", s.config.JiraAPIBaseURL, cloudID)

	// Create request with query parameters
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	// Add query parameters
	params := url.Values{}
	params.Add("jql", jql)
	params.Add("maxResults", maxResults)
	params.Add("fields", "summary,status,customfield_10072")
	req.URL.RawQuery = params.Encode()

	log.Println(req.URL.String())

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
	var jqlResp jiraJQLSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&jqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to domain entities
	issues := make([]domain.JiraIssueResponse, len(jqlResp.Issues))
	for i, item := range jqlResp.Issues {
		// Handle stakeholder field (might be null/empty)
		stakeholder := item.Fields.Customfield10072
		if stakeholder == "" {
			stakeholder = "Unassigned"
		}

		issues[i] = domain.JiraIssueResponse{
			ID:          item.ID,
			Key:         item.Key,
			URL:         item.Self,
			TicketTitle: item.Fields.Summary,
			Stakeholder: stakeholder,
			Status:      item.Fields.Status.Name,
			StatusColor: item.Fields.Status.StatusCategory.ColorName,
		}
	}

	return issues, nil
}

// GetIssue retrieves a single Jira issue by issue ID using Atlassian Cloud API
func (s *JiraServiceImpl) GetIssue(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
	// Build API request URL for issue detail endpoint
	issueURL := fmt.Sprintf("%s/ex/jira/%s/rest/api/2/issue/%s", s.config.JiraAPIBaseURL, cloudID, issueID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", issueURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Handle not found
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("issue not found: %s", issueID)
	}

	// Handle other errors
	if resp.StatusCode != http.StatusOK {
		return nil, s.handleJiraError(resp)
	}

	// Parse response using detailed structure
	var issueResp jiraIssueDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Handle stakeholder field (customfield_10072)
	stakeholder := issueResp.Fields.Customfield10072
	if stakeholder == "" {
		stakeholder = "Unassigned"
	}

	// Handle assignee fields (can be null)
	var assigneeName, assigneeEmail string
	if issueResp.Fields.Assignee != nil {
		assigneeName = issueResp.Fields.Assignee.DisplayName
		assigneeEmail = issueResp.Fields.Assignee.EmailAddress
	} else {
		assigneeName = "Unassigned"
		assigneeEmail = ""
	}

	// Map to domain entity
	issue := &domain.JiraIssueDetailResponse{
		ID:            issueResp.ID,
		TicketNumber:  issueResp.Key,
		SelfLink:      issueResp.Self,
		TicketTitle:   issueResp.Fields.Summary,
		Stakeholder:   stakeholder,
		Status:        issueResp.Fields.Status.Name,
		StatusColor:   issueResp.Fields.Status.StatusCategory.ColorName,
		LastViewed:    issueResp.Fields.LastViewed,
		CreatorName:   issueResp.Fields.Creator.DisplayName,
		CreatorEmail:  issueResp.Fields.Creator.EmailAddress,
		AssigneeName:  assigneeName,
		AssigneeEmail: assigneeEmail,
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
