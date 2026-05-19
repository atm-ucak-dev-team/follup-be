package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/bomanarakasura/jira-email-automation/internal/service"
)

// JiraHandler handles Jira-related HTTP requests
type JiraHandler struct {
	jiraService service.JiraService
}

// NewJiraHandler creates a new JiraHandler instance
func NewJiraHandler(jiraService service.JiraService) *JiraHandler {
	return &JiraHandler{
		jiraService: jiraService,
	}
}

// GetIssues retrieves Jira issues for the authenticated user with optional filters
func (h *JiraHandler) GetIssues(c echo.Context) error {
	// Get user from JWT context (set by middleware)
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Parse query parameters
	project := c.QueryParam("project")
	status := c.QueryParam("status")

	// Get issues from Jira service
	issues, err := h.jiraService.GetIssues(c.Request().Context(), userID, project, status)
	if err != nil {
		// Check if it's a token expiration error
		if errors.Is(err, ErrTokenExpired) {
			return buildErrorResponse(c, http.StatusUnauthorized, "JIRA_TOKEN_EXPIRED", "Jira access token has expired")
		}
		// Handle other Jira API errors
		return buildErrorResponse(c, http.StatusBadGateway, "JIRA_API_ERROR", "failed to fetch issues from Jira: "+err.Error())
	}

	// Build success response
	response := map[string]interface{}{
		"issues": issues,
	}

	return c.JSON(http.StatusOK, response)
}

// GetIssue retrieves a single Jira issue by ticket key
func (h *JiraHandler) GetIssue(c echo.Context) error {
	// Get user from JWT context (set by middleware)
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get ticket key from URL parameter
	ticketKey := c.Param("ticket_key")
	if ticketKey == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_TICKET_KEY", "ticket_key parameter is required")
	}

	// Get issue from Jira service
	issue, err := h.jiraService.GetIssue(c.Request().Context(), userID, ticketKey)
	if err != nil {
		// Check if it's a not found error
		if errors.Is(err, ErrIssueNotFound) {
			return buildErrorResponse(c, http.StatusNotFound, "ISSUE_NOT_FOUND", "Jira issue not found")
		}
		// Check if it's a token expiration error
		if errors.Is(err, ErrTokenExpired) {
			return buildErrorResponse(c, http.StatusUnauthorized, "JIRA_TOKEN_EXPIRED", "Jira access token has expired")
		}
		// Handle other Jira API errors
		return buildErrorResponse(c, http.StatusBadGateway, "JIRA_API_ERROR", "failed to fetch issue from Jira: "+err.Error())
	}

	// Build success response
	return c.JSON(http.StatusOK, issue)
}

// Custom errors for Jira operations
var (
	ErrTokenExpired  = errors.New("token expired")
	ErrIssueNotFound = errors.New("issue not found")
)