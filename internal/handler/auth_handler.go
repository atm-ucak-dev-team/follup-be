package handler

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
)

//go:embed callback_page.html
var callbackTemplate embed.FS

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService service.AuthService
	config      *domain.Config
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(authService service.AuthService, config *domain.Config) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		config:      config,
	}
}

// ConnectJira returns the Jira OAuth authorization URL as JSON
func (h *AuthHandler) ConnectJira(c echo.Context) error {
	// Generate a state parameter for CSRF protection
	state := generateState()

	// Get the Jira authorization URL from the auth service
	authURL := h.authService.GenerateAuthURL(state)

	// Return JSON response with authorization URL
	response := map[string]interface{}{
		"connectUrl": authURL,
	}

	return c.JSON(http.StatusOK, response)
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

	// Exchange the authorization code for OAuth tokens and user info
	user, tokenInfo, err := h.authService.ExchangeJiraCode(c.Request().Context(), code, state)
	if err != nil {
		return buildErrorResponse(c, http.StatusUnauthorized, "INVALID_CODE", "failed to exchange authorization code: "+err.Error())
	}

	// Build redirect URL with OAuth tokens as query parameters
	redirectURL, err := buildRedirectURL(h.config.FrontendCallbackURL, tokenInfo, user)
	if err != nil {
		log.Printf("Failed to build redirect URL: %v", err)
		// Fallback to JSON response if URL construction fails
		response := map[string]interface{}{
			"access_token":  tokenInfo.AccessToken,
			"refresh_token": tokenInfo.RefreshToken,
			"expires_at":    tokenInfo.ExpiresAt,
			"expires_in":    tokenInfo.ExpiresIn,
			"token_type":    tokenInfo.TokenType,
			"scope":         tokenInfo.Scope,
			"user": map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"email":      user.Email,
				"cloud_id":   user.CloudID,
				"avatar_url": user.AvatarURL,
			},
		}
		return c.JSON(http.StatusOK, response)
	}

	// Load the embedded HTML template
	templateContent, err := callbackTemplate.ReadFile("callback_page.html")
	if err != nil {
		log.Printf("Failed to read embedded template: %v", err)
		// Fallback to JSON response if template loading fails
		response := map[string]interface{}{
			"access_token":  tokenInfo.AccessToken,
			"refresh_token": tokenInfo.RefreshToken,
			"expires_at":    tokenInfo.ExpiresAt,
			"expires_in":    tokenInfo.ExpiresIn,
			"token_type":    tokenInfo.TokenType,
			"scope":         tokenInfo.Scope,
			"user": map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"email":      user.Email,
				"cloud_id":   user.CloudID,
				"avatar_url": user.AvatarURL,
			},
		}
		return c.JSON(http.StatusOK, response)
	}

	// Parse and execute the template
	tmpl, err := template.New("callback").Parse(string(templateContent))
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		// Fallback to JSON response if template parsing fails
		response := map[string]interface{}{
			"access_token":  tokenInfo.AccessToken,
			"refresh_token": tokenInfo.RefreshToken,
			"expires_at":    tokenInfo.ExpiresAt,
			"expires_in":    tokenInfo.ExpiresIn,
			"token_type":    tokenInfo.TokenType,
			"scope":         tokenInfo.Scope,
			"user": map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"email":      user.Email,
				"cloud_id":   user.CloudID,
				"avatar_url": user.AvatarURL,
			},
		}
		return c.JSON(http.StatusOK, response)
	}

	// Prepare template data
	data := struct {
		UserName    string
		RedirectURL string
	}{
		UserName:    user.Name,
		RedirectURL: redirectURL,
	}

	// Execute template and return HTML response
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Response().Writer, data)
}

// DummyJiraCallback returns a dummy OAuth callback response for frontend testing
// This endpoint is only available in development environment
func (h *AuthHandler) DummyJiraCallback(c echo.Context) error {
	// Security check: only allow in development environment
	if h.config.Env != "development" {
		return buildErrorResponse(c, http.StatusForbidden, "NOT_AVAILABLE", "dummy endpoint only available in development")
	}

	// Create hardcoded dummy OAuth token info
	expiresAt := time.Now().Add(3600 * time.Second)
	tokenInfo := &service.JiraTokenInfo{
		AccessToken:  "dummy_access_token_for_testing",
		RefreshToken: "dummy_refresh_token_for_testing",
		ExpiresAt:    expiresAt,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
		Scope:        "read:jira-user read:jira-work",
	}

	// Create hardcoded dummy user data
	user := &domain.User{
		ID:        "dummy-user-123",
		Name:      "Test User",
		Email:     "test@example.com",
		CloudID:   "dummy-cloud-456",
		AvatarURL: "https://example.com/avatar.png",
		CreatedAt: time.Now(),
	}

	// Build redirect URL with dummy data
	redirectURL, err := buildRedirectURL(h.config.FrontendCallbackURL, tokenInfo, user)
	if err != nil {
		log.Printf("Failed to build redirect URL: %v", err)
		// Fallback to JSON response if URL construction fails
		response := map[string]interface{}{
			"access_token":  tokenInfo.AccessToken,
			"refresh_token": tokenInfo.RefreshToken,
			"expires_at":    tokenInfo.ExpiresAt.Format(time.RFC3339),
			"expires_in":    tokenInfo.ExpiresIn,
			"token_type":    tokenInfo.TokenType,
			"scope":         tokenInfo.Scope,
			"user": map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"email":      user.Email,
				"cloud_id":   user.CloudID,
				"avatar_url": user.AvatarURL,
			},
		}
		return c.JSON(http.StatusOK, response)
	}

	// Load the embedded HTML template
	templateContent, err := callbackTemplate.ReadFile("callback_page.html")
	if err != nil {
		log.Printf("Failed to read embedded template: %v", err)
		// Fallback to JSON response if template loading fails
		response := map[string]interface{}{
			"access_token":  tokenInfo.AccessToken,
			"refresh_token": tokenInfo.RefreshToken,
			"expires_at":    tokenInfo.ExpiresAt.Format(time.RFC3339),
			"expires_in":    tokenInfo.ExpiresIn,
			"token_type":    tokenInfo.TokenType,
			"scope":         tokenInfo.Scope,
			"user": map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"email":      user.Email,
				"cloud_id":   user.CloudID,
				"avatar_url": user.AvatarURL,
			},
		}
		return c.JSON(http.StatusOK, response)
	}

	// Parse and execute the template
	tmpl, err := template.New("callback").Parse(string(templateContent))
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		// Fallback to JSON response if template parsing fails
		response := map[string]interface{}{
			"access_token":  tokenInfo.AccessToken,
			"refresh_token": tokenInfo.RefreshToken,
			"expires_at":    tokenInfo.ExpiresAt.Format(time.RFC3339),
			"expires_in":    tokenInfo.ExpiresIn,
			"token_type":    tokenInfo.TokenType,
			"scope":         tokenInfo.Scope,
			"user": map[string]interface{}{
				"id":         user.ID,
				"name":       user.Name,
				"email":      user.Email,
				"cloud_id":   user.CloudID,
				"avatar_url": user.AvatarURL,
			},
		}
		return c.JSON(http.StatusOK, response)
	}

	// Prepare template data
	data := struct {
		UserName    string
		RedirectURL string
	}{
		UserName:    user.Name,
		RedirectURL: redirectURL,
	}

	// Execute template and return HTML response
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Response().Writer, data)
}

// RefreshToken refreshes the Jira OAuth access token
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	// Parse request body
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
	}

	// Validate refresh token provided
	if req.RefreshToken == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_REFRESH_TOKEN", "refreshToken is required")
	}

	// Refresh the token using provided refresh token
	tokenResp, err := h.authService.RefreshJiraToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return buildErrorResponse(c, http.StatusUnauthorized, "TOKEN_REFRESH_FAILED", "failed to refresh token: "+err.Error())
	}

	return c.JSON(http.StatusOK, tokenResp)
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

// buildRedirectURL constructs the frontend callback URL with OAuth tokens as query parameters
func buildRedirectURL(baseURL string, tokenInfo *service.JiraTokenInfo, user *domain.User) (string, error) {
	// Parse the base URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Build query parameters with all OAuth data
	params := url.Values{}
	params.Add("access_token", tokenInfo.AccessToken)
	params.Add("refresh_token", tokenInfo.RefreshToken)
	params.Add("expires_at", tokenInfo.ExpiresAt.Format(time.RFC3339))
	params.Add("expires_in", fmt.Sprintf("%d", tokenInfo.ExpiresIn))
	params.Add("token_type", tokenInfo.TokenType)
	params.Add("scope", tokenInfo.Scope)
	params.Add("user_id", user.ID)
	params.Add("user_name", user.Name)
	params.Add("user_email", user.Email)
	params.Add("cloud_id", user.CloudID)
	params.Add("avatar_url", user.AvatarURL)

	// Set query parameters
	parsedURL.RawQuery = params.Encode()

	return parsedURL.String(), nil
}
