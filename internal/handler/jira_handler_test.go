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
	getIssuesFunc func(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error)
	getIssueFunc  func(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error)
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

func (m *mockJiraService) GetIssues(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
	if m.getIssuesFunc != nil {
		return m.getIssuesFunc(ctx, userID, project, status)
	}
	// Default implementation
	return []domain.JiraIssue{
		{
			ID:      "10001",
			Key:     "PROJ-123",
			Summary: "Test Issue 1",
			Status:  "In Progress",
			Stakeholders: []string{
				"alice@example.com",
				"bob@example.com",
			},
		},
		{
			ID:      "10002",
			Key:     "PROJ-456",
			Summary: "Test Issue 2",
			Status:  "Done",
			Stakeholders: []string{
				"charlie@example.com",
			},
		},
	}, nil
}

func (m *mockJiraService) GetIssue(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error) {
	if m.getIssueFunc != nil {
		return m.getIssueFunc(ctx, userID, ticketKey)
	}
	// Default implementation
	return &domain.JiraIssue{
		ID:      "10001",
		Key:     "PROJ-123",
		Summary: "Test Issue",
		Status:  "In Progress",
		Stakeholders: []string{
			"alice@example.com",
			"bob@example.com",
		},
	}, nil
}

// TestGetIssues_Success tests successful retrieval of issues
func TestGetIssues_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
			return []domain.JiraIssue{
				{
					ID:           "10001",
					Key:          "PROJ-123",
					Summary:      "Test Issue",
					Status:       "In Progress",
					Stakeholders: []string{"alice@example.com"},
				},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "issues")
	issues := response["issues"].([]interface{})
	assert.Len(t, issues, 1)
}

// TestGetIssues_WithProjectFilter tests issues retrieval with project filter
func TestGetIssues_WithProjectFilter(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
			assert.Equal(t, "PROJ", project)
			return []domain.JiraIssue{
				{
					ID:           "10001",
					Key:          "PROJ-123",
					Summary:      "Project Issue",
					Status:       "Open",
					Stakeholders: []string{"alice@example.com"},
				},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	// Create request with project filter
	params := url.Values{}
	params.Add("project", "PROJ")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "issues")
	issues := response["issues"].([]interface{})
	assert.Len(t, issues, 1)

	issue := issues[0].(map[string]interface{})
	assert.Equal(t, "Project Issue", issue["summary"])
}

// TestGetIssues_WithStatusFilter tests issues retrieval with status filter
func TestGetIssues_WithStatusFilter(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
			assert.Equal(t, "In Progress", status)
			return []domain.JiraIssue{
				{
					ID:           "10001",
					Key:          "PROJ-123",
					Summary:      "In Progress Issue",
					Status:       "In Progress",
					Stakeholders: []string{"alice@example.com"},
				},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	// Create request with status filter
	params := url.Values{}
	params.Add("status", "In Progress")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues?"+params.Encode(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetIssues(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "issues")
	issues := response["issues"].([]interface{})
	assert.Len(t, issues, 1)

	issue := issues[0].(map[string]interface{})
	assert.Equal(t, "In Progress Issue", issue["summary"])
	assert.Equal(t, "In Progress", issue["status"])
}

// TestGetIssues_TokenExpired tests issues retrieval with expired token
func TestGetIssues_TokenExpired(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
			return nil, ErrTokenExpired
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "test-user-123")

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
	// No user_id set in context

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
	assert.Equal(t, "UNAUTHORIZED", errResp["code"])
	assert.Equal(t, "user not authenticated", errResp["message"])
}

// TestGetIssues_JiraAPIError tests issues retrieval with Jira API error
func TestGetIssues_JiraAPIError(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssuesFunc: func(ctx context.Context, userID, project, status string) ([]domain.JiraIssue, error) {
			return nil, errors.New("Jira API connection failed")
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "test-user-123")

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
		getIssueFunc: func(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error) {
			assert.Equal(t, "PROJ-123", ticketKey)
			return &domain.JiraIssue{
				ID:           "10001",
				Key:          "PROJ-123",
				Summary:      "Test Issue",
				Status:       "In Progress",
				Stakeholders: []string{"alice@example.com", "bob@example.com"},
			}, nil
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")
	c.Set("user_id", "test-user-123")

	// Execute
	err := h.GetIssue(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response domain.JiraIssue
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "10001", response.ID)
	assert.Equal(t, "PROJ-123", response.Key)
	assert.Equal(t, "Test Issue", response.Summary)
	assert.Equal(t, "In Progress", response.Status)
	assert.Len(t, response.Stakeholders, 2)
}

// TestGetIssue_NotFound tests retrieval of non-existent issue
func TestGetIssue_NotFound(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssueFunc: func(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error) {
			return nil, ErrIssueNotFound
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/NONEXIST-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("NONEXIST-123")
	c.Set("user_id", "test-user-123")

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
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No ticket_key parameter set
	c.Set("user_id", "test-user-123")

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
	assert.Equal(t, "MISSING_TICKET_KEY", errResp["code"])
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
	assert.Equal(t, "UNAUTHORIZED", errResp["code"])
	assert.Equal(t, "user not authenticated", errResp["message"])
}

// TestGetIssue_JiraAPIError tests issue retrieval with Jira API error
func TestGetIssue_JiraAPIError(t *testing.T) {
	// Setup
	e := echo.New()
	mockJira := &mockJiraService{
		getIssueFunc: func(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error) {
			return nil, errors.New("Jira API connection failed")
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")
	c.Set("user_id", "test-user-123")

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
		getIssueFunc: func(ctx context.Context, userID, ticketKey string) (*domain.JiraIssue, error) {
			return nil, ErrTokenExpired
		},
	}
	h := NewJiraHandler(mockJira)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jira/issues/PROJ-123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ticket_key")
	c.SetParamValues("PROJ-123")
	c.Set("user_id", "test-user-123")

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
