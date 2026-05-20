package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/atm-ucak/follup/internal/domain"
	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
)

// AutomationHandler handles automation rule HTTP requests
type AutomationHandler struct {
	automationService service2.AutomationService
}

// NewAutomationHandler creates a new AutomationHandler instance
func NewAutomationHandler(automationService service2.AutomationService) *AutomationHandler {
	return &AutomationHandler{
		automationService: automationService,
	}
}

// Custom errors for automation operations
var (
	ErrAutomationNotFound     = errors.New("automation not found")
	ErrInvalidCronExpression  = errors.New("invalid cron expression")
	ErrInvalidRecipients      = errors.New("invalid recipients")
	ErrUnauthorizedAutomation = errors.New("unauthorized automation access")
)

// CreateAutomationRequest represents the request body for creating an automation rule
type CreateAutomationRequest struct {
	JiraTicketID  string   `json:"jira_ticket_id"`
	JiraTicketKey string   `json:"jira_ticket_key"`
	Recipients    []string `json:"recipients"`
	CronSchedule  string   `json:"cron_schedule"`
	Status        string   `json:"status"` // optional, defaults to "active"
}

// UpdateAutomationRequest represents the request body for updating an automation rule
type UpdateAutomationRequest struct {
	Recipients   []string `json:"recipients,omitempty"`
	CronSchedule string   `json:"cron_schedule,omitempty"`
	Status       string   `json:"status,omitempty"`
}

// CreateAutomation handles POST /automations
func (h *AutomationHandler) CreateAutomation(c echo.Context) error {
	// Get user from JWT context
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Parse request body
	var req CreateAutomationRequest
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
	}

	// Validate required fields
	if req.JiraTicketID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_ID", "jira_ticket_id is required")
	}
	if req.JiraTicketKey == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_KEY", "jira_ticket_key is required")
	}
	if len(req.Recipients) == 0 {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_RECIPIENTS", "at least one recipient is required")
	}
	if req.CronSchedule == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_CRON_SCHEDULE", "cron_schedule is required")
	}

	// Create automation rule
	rule := &domain.AutomationRule{
		UserID:        userID,
		JiraTicketID:  req.JiraTicketID,
		JiraTicketKey: req.JiraTicketKey,
		Recipients:    req.Recipients,
		CronSchedule:  req.CronSchedule,
		Status:        req.Status,
	}

	// Set default status if not provided
	if rule.Status == "" {
		rule.Status = domain.AutomationStatusActive
	}

	// Call service to create rule
	if err := h.automationService.CreateRule(c.Request().Context(), rule); err != nil {
		// Handle validation errors
		if strings.Contains(err.Error(), "invalid cron expression") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_CRON", "invalid cron expression: "+err.Error())
		}
		if strings.Contains(err.Error(), "recipient") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_RECIPIENTS", "invalid recipients: "+err.Error())
		}
		if strings.Contains(err.Error(), "Jira ticket") {
			return buildErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", "validation error: "+err.Error())
		}

		// Handle other errors
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_CREATE_FAILED", "failed to create automation rule: "+err.Error())
	}

	// Return 201 with created rule
	return c.JSON(http.StatusCreated, rule)
}

// ListAutomations handles GET /automations
func (h *AutomationHandler) ListAutomations(c echo.Context) error {
	// Get user from JWT context
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get user's automation rules
	rules, err := h.automationService.GetUserRules(c.Request().Context(), userID)
	if err != nil {
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_LIST_FAILED", "failed to list automation rules: "+err.Error())
	}

	// Return 200 with rules array
	return c.JSON(http.StatusOK, map[string]interface{}{
		"automations": rules,
	})
}

// GetAutomation handles GET /automations/:id
func (h *AutomationHandler) GetAutomation(c echo.Context) error {
	// Get user from JWT context
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get automation ID from URL parameter
	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_AUTOMATION_ID", "automation ID is required")
	}

	// Get automation rule
	rule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get automation rule") {
			return buildErrorResponse(c, http.StatusNotFound, "AUTOMATION_NOT_FOUND", "automation rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_GET_FAILED", "failed to get automation rule: "+err.Error())
	}

	// Verify ownership (user can only view their own automations)
	if rule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to view this automation rule")
	}

	// Return 200 with rule
	return c.JSON(http.StatusOK, rule)
}

// UpdateAutomation handles PATCH /automations/:id
func (h *AutomationHandler) UpdateAutomation(c echo.Context) error {
	// Get user from JWT context
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get automation ID from URL parameter
	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_AUTOMATION_ID", "automation ID is required")
	}

	// Parse request body
	var req UpdateAutomationRequest
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
	}

	// Get existing automation rule first
	existingRule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get automation rule") {
			return buildErrorResponse(c, http.StatusNotFound, "AUTOMATION_NOT_FOUND", "automation rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_GET_FAILED", "failed to get automation rule: "+err.Error())
	}

	// Verify ownership before update
	if existingRule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to update this automation rule")
	}

	// Apply updates to existing rule
	if req.Recipients != nil {
		existingRule.Recipients = req.Recipients
	}
	if req.CronSchedule != "" {
		existingRule.CronSchedule = req.CronSchedule
	}
	if req.Status != "" {
		existingRule.Status = req.Status
	}

	// Call service to update rule
	if err := h.automationService.UpdateRule(c.Request().Context(), existingRule); err != nil {
		// Handle validation errors
		if strings.Contains(err.Error(), "invalid cron expression") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_CRON", "invalid cron expression: "+err.Error())
		}
		if strings.Contains(err.Error(), "recipient") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_RECIPIENTS", "invalid recipients: "+err.Error())
		}
		if strings.Contains(err.Error(), "does not own") {
			return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to update this automation rule")
		}

		// Handle other errors
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_UPDATE_FAILED", "failed to update automation rule: "+err.Error())
	}

	// Return 200 with updated rule
	return c.JSON(http.StatusOK, existingRule)
}

// DeleteAutomation handles DELETE /automations/:id
func (h *AutomationHandler) DeleteAutomation(c echo.Context) error {
	// Get user from JWT context
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get automation ID from URL parameter
	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_AUTOMATION_ID", "automation ID is required")
	}

	// Get existing automation rule first to verify ownership
	existingRule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get automation rule") {
			return buildErrorResponse(c, http.StatusNotFound, "AUTOMATION_NOT_FOUND", "automation rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_GET_FAILED", "failed to get automation rule: "+err.Error())
	}

	// Verify ownership before delete
	if existingRule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to delete this automation rule")
	}

	// Call service to delete rule
	if err := h.automationService.DeleteRule(c.Request().Context(), automationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return buildErrorResponse(c, http.StatusNotFound, "AUTOMATION_NOT_FOUND", "automation rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_DELETE_FAILED", "failed to delete automation rule: "+err.Error())
	}

	// Return 204 No Content
	return c.NoContent(http.StatusNoContent)
}

// TriggerAutomation handles POST /automations/:id/trigger
func (h *AutomationHandler) TriggerAutomation(c echo.Context) error {
	// Get user from JWT context
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get automation ID from URL parameter
	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_AUTOMATION_ID", "automation ID is required")
	}

	// Get existing automation rule first to verify ownership
	existingRule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get automation rule") {
			return buildErrorResponse(c, http.StatusNotFound, "AUTOMATION_NOT_FOUND", "automation rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_GET_FAILED", "failed to get automation rule: "+err.Error())
	}

	// Verify ownership before trigger
	if existingRule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to trigger this automation rule")
	}

	// Call service to trigger rule
	if err := h.automationService.TriggerRule(c.Request().Context(), automationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return buildErrorResponse(c, http.StatusNotFound, "AUTOMATION_NOT_FOUND", "automation rule not found")
		}
		// Handle email service errors (credential issues, connection problems, etc.)
		if strings.Contains(err.Error(), "credential") || strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "token") {
			return buildErrorResponse(c, http.StatusBadGateway, "AUTOMATION_TRIGGER_FAILED", "failed to send email: "+err.Error())
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "AUTOMATION_TRIGGER_FAILED", "failed to trigger automation rule: "+err.Error())
	}

	// Return 202 Accepted
	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":       "automation rule triggered successfully",
		"automation_id": automationID,
	})
}
