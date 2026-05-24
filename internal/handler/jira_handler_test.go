package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock JiraService for testing
type mockJiraService struct {
	getIssuesFunc func(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error)
	getIssueFunc  func(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error)
}

func (m *mockJiraService) GetTicket(ctx interface{}, ticketID string) (*domain.JiraTicket, error) {
	return nil, nil
}

func (m *mockJiraService) UpdateTicketStatus(ctx interface{}, ticketID, status string) error {
	return nil
}

func (m *mockJiraService) AddComment(ctx interface{}, ticketID, comment string) error {
	return nil
}

func (m *mockJiraService) GetAuthenticatedUser(ctx interface{}) (*domain.User, error) {
	return nil, nil
}

func (m *mockJiraService) GetIssues(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
	if m.getIssuesFunc != nil {
		return m.getIssuesFunc(ctx, cloudID, accessToken, search, limit)
	}
	// Default implementation
	return []domain.JiraIssueResponse{
		{
			ID:          "10001",
			Key:         "PROJ-123",
			URL:         "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10001",
			TicketTitle: "Test Issue 1",
			Stakeholder: "alice@example.com",
			Status:      "In Progress",
			StatusColor: "blue",
		},
		{
			ID:          "10002",
			Key:         "PROJ-456",
			URL:         "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10002",
			TicketTitle: "Test Issue 2",
			Stakeholder: "charlie@example.com",
			Status:      "Done",
			StatusColor: "green",
		},
	}, nil
}

func (m *mockJiraService) GetIssue(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
	if m.getIssueFunc != nil {
		return m.getIssueFunc(ctx, cloudID, accessToken, issueID)
	}
	// Default implementation
	return &domain.JiraIssueDetailResponse{
		ID:            "10001",
		TicketNumber:  "PROJ-123",
		SelfLink:      "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10001",
		TicketTitle:   "Test Issue",
		Stakeholder:   "alice@example.com",
		Status:        "In Progress",
		StatusColor:   "blue",
		LastViewed:    "2026-05-20T11:38:10.480+0700",
		CreatorName:   "Test Creator",
		CreatorEmail:  "creator@example.com",
		AssigneeName:  "Test Assignee",
		AssigneeEmail: "assignee@example.com",
	}, nil
}

// TestGetIssues_Success tests successful retrieval of issues
func TestGetIssues_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
			return []domain.JiraIssueResponse{
				{
					ID:          "10001",
					Key:         "PROJ-123",
					URL:         "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10001",
					TicketTitle: "Test Issue",
					Stakeholder: "alice@example.com",
					Status:      "In Progress",
					StatusColor: "blue",
				},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 1)
}

// TestGetIssues_WithProjectFilter tests issues retrieval with project filter
func TestGetIssues_WithProjectFilter(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
			return []domain.JiraIssueResponse{
				{
					ID:          "10001",
					Key:         "PROJ-123",
					URL:         "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10001",
					TicketTitle: "Project Issue",
					Stakeholder: "alice@example.com",
					Status:      "Open",
					StatusColor: "blue",
				},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	// Create request with project filter
	params := url.Values{}
	params.Add("project", "PROJ")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues?"+params.Encode(), nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 1)

	issue := response[0].(map[string]interface{})
	assert.Equal(t, "Project Issue", issue["ticketTitle"])
}

// TestGetIssues_WithStatusFilter tests issues retrieval with status filter
func TestGetIssues_WithStatusFilter(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
			return []domain.JiraIssueResponse{
				{
					ID:          "10001",
					Key:         "PROJ-123",
					URL:         "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10001",
					TicketTitle: "In Progress Issue",
					Stakeholder: "alice@example.com",
					Status:      "In Progress",
					StatusColor: "blue",
				},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	// Create request with status filter
	params := url.Values{}
	params.Add("status", "In Progress")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues?"+params.Encode(), nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response []interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 1)

	issue := response[0].(map[string]interface{})
	assert.Equal(t, "In Progress Issue", issue["ticketTitle"])
	assert.Equal(t, "In Progress", issue["status"])
}

// TestGetIssues_TokenExpired tests issues retrieval with expired token
func TestGetIssues_TokenExpired(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
			return nil, ErrTokenExpired
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "JIRA_TOKEN_EXPIRED", errResp["code"])
	assert.Equal(t, "Jira access token has expired", errResp["message"])
}

// TestGetIssues_Unauthorized tests issues retrieval without authentication
func TestGetIssues_Unauthorized(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No X-User-Cloud-ID header set

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_CLOUD_ID", errResp["code"])
	assert.Equal(t, "X-User-Cloud-ID header is required", errResp["message"])
}

// TestGetIssues_JiraAPIError tests issues retrieval with Jira API error
func TestGetIssues_JiraAPIError(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error) {
			return nil, errors.New("Jira API connection failed")
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "JIRA_API_ERROR", errResp["code"])
	assert.Contains(t, errResp["message"].(string), "failed to fetch issues from Jira")
}

// TestGetIssue_Success tests successful retrieval of single issue
func TestGetIssue_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssueFunc: func(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
			assert.Equal(t, "test-cloud-123", cloudID)
			assert.Equal(t, "test-token-123", accessToken)
			assert.Equal(t, "PROJ-123", issueID)
			return &domain.JiraIssueDetailResponse{
				ID:            "10001",
				TicketNumber:  "PROJ-123",
				SelfLink:      "https://api.atlassian.com/ex/jira/test-cloud/rest/api/2/issue/10001",
				TicketTitle:   "Test Issue",
				Stakeholder:   "alice@example.com",
				Status:        "In Progress",
				StatusColor:   "blue",
				LastViewed:    "2026-05-20T11:38:10.480+0700",
				CreatorName:   "Test Creator",
				CreatorEmail:  "creator@example.com",
				AssigneeName:  "Test Assignee",
				AssigneeEmail: "assignee@example.com",
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response domain.JiraIssueDetailResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "10001", response.ID)
	assert.Equal(t, "PROJ-123", response.TicketNumber)
	assert.Equal(t, "Test Issue", response.TicketTitle)
	assert.Equal(t, "In Progress", response.Status)
	assert.Equal(t, "alice@example.com", response.Stakeholder)
}

// TestGetIssue_NotFound tests retrieval of non-existent issue
func TestGetIssue_NotFound(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssueFunc: func(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
			return nil, ErrIssueNotFound
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/NONEXIST-123", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("NONEXIST-123")

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "ISSUE_NOT_FOUND", errResp["code"])
	assert.Equal(t, "Jira issue not found", errResp["message"])
}

// TestGetIssue_MissingTicketKey tests issue retrieval with missing ticket key parameter
func TestGetIssue_MissingTicketKey(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No ticket_key parameter set

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_ISSUE_ID", errResp["code"])
	assert.Equal(t, "ticket_key parameter is required", errResp["message"])
}

// TestGetIssue_Unauthorized tests issue retrieval without authentication
func TestGetIssue_Unauthorized(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")
	// No user_id set in context

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "MISSING_CLOUD_ID", errResp["code"])
	assert.Equal(t, "X-User-Cloud-ID header is required", errResp["message"])
}

// TestGetIssue_JiraAPIError tests issue retrieval with Jira API error
func TestGetIssue_JiraAPIError(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssueFunc: func(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
			return nil, errors.New("Jira API connection failed")
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadGateway, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "JIRA_API_ERROR", errResp["code"])
	assert.Contains(t, errResp["message"].(string), "failed to fetch issue from Jira")
}

// TestGetIssue_TokenExpired tests issue retrieval with expired token
func TestGetIssue_TokenExpired(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssueFunc: func(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error) {
			return nil, ErrTokenExpired
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	req.Header.Set("X-User-Cloud-ID", "test-cloud-123")
	req.Header.Set("X-Jira-Access-Token", "test-token-123")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	errResp := response["error"].(map[string]interface{})
	assert.Equal(t, "JIRA_TOKEN_EXPIRED", errResp["code"])
	assert.Equal(t, "Jira access token has expired", errResp["message"])
}
