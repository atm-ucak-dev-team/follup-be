package handler

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/atm-ucak/follup/internal/service"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// ConnectJira redirects the user to Jira OAuth consent page
func (h *AuthHandler) ConnectJira(c echo.Context) error {
	// Generate a state parameter for CSRF protection
	state := generateState()

	// Get the Jira authorization URL from the auth service
	authURL := h.authService.GenerateAuthURL(state)

	// Redirect to Jira OAuth consent page
	return c.Redirect(http.StatusFound, authURL)
}

// JiraCallback handles the OAuth callback from Jira
func (h *AuthHandler) JiraCallback(c echo.Context) error {
	// Parse query parameters
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	// Validate required parameters
	if code == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_CODE", "authorization code is required")
	}
	if state == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_STATE", "state parameter is required")
	}

	// Exchange the authorization code for JWT and user info
	user, accessToken, err := h.authService.ExchangeJiraCode(c.Request().Context(), code, state)
	if err != nil {
		return buildErrorResponse(c, http.StatusUnauthorized, "INVALID_CODE", "failed to exchange authorization code: "+err.Error())
	}

	// Build success response
	response := map[string]interface{}{
		"access_token": accessToken,
		"user": map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	}

	return c.JSON(http.StatusOK, response)
}

// RefreshToken refreshes the Jira OAuth access token
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	// Get user from JWT context (set by middleware)
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Refresh the token
	newAccessToken, err := h.authService.RefreshJiraToken(c.Request().Context(), userID)
	if err != nil {
		return buildErrorResponse(c, http.StatusUnauthorized, "TOKEN_REFRESH_FAILED", "failed to refresh token: "+err.Error())
	}

	// Build success response
	response := map[string]interface{}{
		"access_token": newAccessToken,
	}

	return c.JSON(http.StatusOK, response)
}

// ErrorResponse represents the error response format
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// buildErrorResponse builds a standardized error response
func buildErrorResponse(c echo.Context, status int, code, message string) error {
	errResp := ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	}
	return c.JSON(status, errResp)
}

// getUserIDFromContext extracts user ID from JWT context
func getUserIDFromContext(c echo.Context) string {
	// This will be set by JWT middleware
	// For now, return empty string - middleware implementation will be needed
	if userID, ok := c.Get("user_id").(string); ok {
		return userID
	}
	return ""
}

// generateState generates a random state parameter for CSRF protection
func generateState() string {
	// Simple implementation - in production, use crypto/rand
	return time.Now().Format("20060102150405")
}