package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
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

// GetIssues retrieves Jira issues using Atlassian Cloud API JQL search
func (h *JiraHandler) GetIssues(c echo.Context) error {
	// Get cloud ID and access token from headers
	cloudID := getCloudIDFromContext(c)
	if cloudID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "MISSING_CLOUD_ID", "X-User-Cloud-ID header is required")
	}

	accessToken := getAccessTokenFromContext(c)
	if accessToken == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "MISSING_ACCESS_TOKEN", "X-Jira-Access-Token header is required")
	}

	// Parse query parameters
	search := c.QueryParam("search")
	limit := c.QueryParam("limit")

	log.Println(search, limit)

	// Get issues from Jira service
	issues, err := h.jiraService.GetIssues(c.Request().Context(), cloudID, accessToken, search, limit)
	if err != nil {
		// Check if it's a token expired error
		if errors.Is(err, ErrTokenExpired) {
			return buildErrorResponse(c, http.StatusUnauthorized, "JIRA_TOKEN_EXPIRED", "Jira access token has expired")
		}
		// Handle other Jira API errors
		return buildErrorResponse(c, http.StatusBadGateway, "JIRA_API_ERROR", "failed to fetch issues from Jira: "+err.Error())
	}

	// Return array directly instead of wrapped in object
	return c.JSON(http.StatusOK, issues)
}

// GetIssue retrieves a single Jira issue by issue ID using Atlassian Cloud API
func (h *JiraHandler) GetIssue(c echo.Context) error {
	// Get cloud ID and access token from headers
	cloudID := getCloudIDFromContext(c)
	if cloudID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "MISSING_CLOUD_ID", "X-User-Cloud-ID header is required")
	}

	accessToken := getAccessTokenFromContext(c)
	if accessToken == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "MISSING_ACCESS_TOKEN", "X-Jira-Access-Token header is required")
	}

	// Get issue ID from URL parameter
	issueID := c.Param("ticket_key")
	if issueID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_ISSUE_ID", "ticket_key parameter is required")
	}

	// Get issue from Jira service
	issue, err := h.jiraService.GetIssue(c.Request().Context(), cloudID, accessToken, issueID)
	if err != nil {
		// Check if it's a token expired error
		if errors.Is(err, ErrTokenExpired) {
			return buildErrorResponse(c, http.StatusUnauthorized, "JIRA_TOKEN_EXPIRED", "Jira access token has expired")
		}
		// Check if it's a not found error
		if errors.Is(err, ErrIssueNotFound) {
			return buildErrorResponse(c, http.StatusNotFound, "ISSUE_NOT_FOUND", "Jira issue not found")
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

// getCloudIDFromContext extracts the X-User-Cloud-ID header from the request context
func getCloudIDFromContext(c echo.Context) string {
	return c.Request().Header.Get("X-User-Cloud-ID")
}

// getAccessTokenFromContext extracts the X-Jira-Access-Token header from the request context
func getAccessTokenFromContext(c echo.Context) string {
	return c.Request().Header.Get("X-Jira-Access-Token")
}
