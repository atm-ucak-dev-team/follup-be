package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/bomanarakasura/jira-email-automation/internal/service"
)

// EmailHandler handles email credential-related HTTP requests
type EmailHandler struct {
	emailService service.EmailService
}

// NewEmailHandler creates a new EmailHandler instance
func NewEmailHandler(emailService service.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

// SaveCredentialsRequest represents the request body for saving email credentials
type SaveCredentialsRequest struct {
	EmailAddress string `json:"email_address" validate:"required,email"`
	Password     string `json:"password" validate:"required"`
}

// CredentialsResponse represents the response for getting credentials (excluding password)
type CredentialsResponse struct {
	EmailAddress string `json:"email_address"`
	CreatedAt    string `json:"created_at"`
}

// SaveCredentials registers or updates user's IMAP/SMTP credentials
func (h *EmailHandler) SaveCredentials(c echo.Context) error {
	// Get user from JWT context (set by middleware)
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Parse request body
	var req SaveCredentialsRequest
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
	}

	// Validate required fields
	if req.EmailAddress == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_EMAIL_ADDRESS", "email_address is required")
	}
	if req.Password == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_PASSWORD", "password is required")
	}

	// Call email service to save credentials
	err := h.emailService.SaveCredential(c.Request().Context(), userID, req.EmailAddress, req.Password)
	if err != nil {
		// Log security event for credential operations
		logSecurityEvent("credential_save_failed", userID, req.EmailAddress, err.Error())
		return buildErrorResponse(c, http.StatusInternalServerError, "CREDENTIAL_SAVE_FAILED", "failed to save email credentials: "+err.Error())
	}

	// Log security event for successful credential save
	logSecurityEvent("credential_saved", userID, req.EmailAddress, "")

	// Build success response - never include password in response
	response := map[string]interface{}{
		"message":      "email credential saved",
		"email_address": req.EmailAddress,
	}

	return c.JSON(http.StatusCreated, response)
}

// GetCredentials retrieves user's email credentials (without password)
func (h *EmailHandler) GetCredentials(c echo.Context) error {
	// Get user from JWT context (set by middleware)
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get credential from email service
	cred, err := h.emailService.GetCredential(c.Request().Context(), userID)
	if err != nil {
		// Check if credential not found
		if contains(err.Error(), "not found") || contains(err.Error(), "no credential") {
			return buildErrorResponse(c, http.StatusNotFound, "EMAIL_CREDENTIAL_NOT_FOUND", "email credential not found")
		}
		// Log security event for credential retrieval failure
		logSecurityEvent("credential_get_failed", userID, "", err.Error())
		return buildErrorResponse(c, http.StatusInternalServerError, "CREDENTIAL_GET_FAILED", "failed to retrieve email credentials: "+err.Error())
	}

	// Log security event for successful credential retrieval
	logSecurityEvent("credential_retrieved", userID, cred.EmailAddress, "")

	// Build response - never include password in response
	response := CredentialsResponse{
		EmailAddress: cred.EmailAddress,
		CreatedAt:    cred.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return c.JSON(http.StatusOK, response)
}

// logSecurityEvent logs security-related events for credential operations
func logSecurityEvent(event, userID, email, details string) {
	// In production, this would log to a secure audit system
	// For now, we'll use a simple implementation
	// Format: [EVENT] user_id: email_address: details
	message := formatSecurityMessage(event, userID, email, details)
	// In a real system, this would go to a secure logging service
	// For development, we can just note it
	if details != "" {
		// Log with details for errors
		message += " | " + details
	}
	// This would typically be sent to a structured logging system
	_ = message
}

// formatSecurityMessage formats a security event message
func formatSecurityMessage(event, userID, email, details string) string {
	if email != "" {
		return "[" + event + "] user: " + userID + " | email: " + email
	}
	return "[" + event + "] user: " + userID
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}