package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/atm-ucak/follup/internal/domain"
	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
)

// AutomationHandler handles followup rule HTTP requests
type AutomationHandler struct {
	automationService service2.AutomationService
}

// NewAutomationHandler creates a new AutomationHandler instance
func NewAutomationHandler(automationService service2.AutomationService) *AutomationHandler {
	return &AutomationHandler{
		automationService: automationService,
	}
}

var (
	ErrAutomationNotFound     = errors.New("followup not found")
	ErrInvalidFrequency       = errors.New("invalid frequency")
	ErrInvalidRecipients      = errors.New("invalid recipients")
	ErrUnauthorizedAutomation = errors.New("unauthorized followup access")
)

// CreateAutomationRequest represents the request body for creating a followup rule
type CreateAutomationRequest struct {
	JiraTicketID  string `json:"jira_ticket_id"`
	JiraTicketKey string `json:"jira_ticket_key"`
	To            string `json:"to"`
	Cc            string `json:"cc,omitempty"`
	Subject       string `json:"subject"`
	EmailBody     string `json:"email_body"`
	Frequency     string `json:"frequency"`
	Status        string `json:"status"`
}

// UpdateAutomationRequest represents the request body for updating a followup rule
type UpdateAutomationRequest struct {
	To       string `json:"to,omitempty"`
	Cc       string `json:"cc,omitempty"`
	Subject  string `json:"subject,omitempty"`
	EmailBody string `json:"email_body,omitempty"`
	Frequency string `json:"frequency,omitempty"`
	Status   string `json:"status,omitempty"`
}

// CreateAutomation handles POST /automations
func (h *AutomationHandler) CreateAutomation(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	var req CreateAutomationRequest
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
	}

	if req.JiraTicketID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_ID", "jira_ticket_id is required")
	}
	if req.To == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_TO", "to is required")
	}
	if req.Subject == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_SUBJECT", "subject is required")
	}
	if req.EmailBody == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_EMAIL_BODY", "email_body is required")
	}
	if req.Frequency == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FREQUENCY", "frequency is required")
	}

	var cc *string
	if req.Cc != "" {
		cc = &req.Cc
	}

	rule := &domain.Followup{
		UserID:        userID,
		JiraTicketID:  req.JiraTicketID,
		JiraTicketKey: req.JiraTicketKey,
		To:            req.To,
		Cc:            cc,
		Subject:       req.Subject,
		EmailBody:     req.EmailBody,
		Frequency:     req.Frequency,
		Status:        req.Status,
	}

	if rule.Status == "" {
		rule.Status = domain.FollowupStatusOngoing
	}

	if err := h.automationService.CreateRule(c.Request().Context(), rule); err != nil {
		if strings.Contains(err.Error(), "invalid frequency") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_FREQUENCY", "invalid frequency: "+err.Error())
		}
		if strings.Contains(err.Error(), "recipient") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_RECIPIENTS", "invalid recipients: "+err.Error())
		}
		if strings.Contains(err.Error(), "Jira ticket") {
			return buildErrorResponse(c, http.StatusBadRequest, "VALIDATION_ERROR", "validation error: "+err.Error())
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_CREATE_FAILED", "failed to create followup rule: "+err.Error())
	}

	return c.JSON(http.StatusCreated, rule)
}

// ListAutomations handles GET /automations
func (h *AutomationHandler) ListAutomations(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	rules, err := h.automationService.GetUserRules(c.Request().Context(), userID)
	if err != nil {
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_LIST_FAILED", "failed to list followup rules: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"automations": rules,
	})
}

// GetAutomation handles GET /automations/:id
func (h *AutomationHandler) GetAutomation(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FOLLOWUP_ID", "followup ID is required")
	}

	rule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get followup") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_GET_FAILED", "failed to get followup rule: "+err.Error())
	}

	if rule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to view this followup rule")
	}

	return c.JSON(http.StatusOK, rule)
}

// UpdateAutomation handles PATCH /automations/:id
func (h *AutomationHandler) UpdateAutomation(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FOLLOWUP_ID", "followup ID is required")
	}

	var req UpdateAutomationRequest
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
	}

	existingRule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get followup") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_GET_FAILED", "failed to get followup rule: "+err.Error())
	}

	if existingRule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to update this followup rule")
	}

	if req.To != "" {
		existingRule.To = req.To
	}
	if req.Cc != "" {
		existingRule.Cc = &req.Cc
	}
	if req.Subject != "" {
		existingRule.Subject = req.Subject
	}
	if req.EmailBody != "" {
		existingRule.EmailBody = req.EmailBody
	}
	if req.Frequency != "" {
		existingRule.Frequency = req.Frequency
	}
	if req.Status != "" {
		existingRule.Status = req.Status
	}

	if err := h.automationService.UpdateRule(c.Request().Context(), existingRule); err != nil {
		if strings.Contains(err.Error(), "invalid frequency") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_FREQUENCY", "invalid frequency: "+err.Error())
		}
		if strings.Contains(err.Error(), "recipient") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_RECIPIENTS", "invalid recipients: "+err.Error())
		}
		if strings.Contains(err.Error(), "does not own") {
			return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to update this followup rule")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_UPDATE_FAILED", "failed to update followup rule: "+err.Error())
	}

	return c.JSON(http.StatusOK, existingRule)
}

// DeleteAutomation handles DELETE /automations/:id
func (h *AutomationHandler) DeleteAutomation(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FOLLOWUP_ID", "followup ID is required")
	}

	existingRule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get followup") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_GET_FAILED", "failed to get followup rule: "+err.Error())
	}

	if existingRule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to delete this followup rule")
	}

	if err := h.automationService.DeleteRule(c.Request().Context(), automationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_DELETE_FAILED", "failed to delete followup rule: "+err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// TriggerAutomation handles POST /automations/:id/trigger
func (h *AutomationHandler) TriggerAutomation(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	automationID := c.Param("id")
	if automationID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FOLLOWUP_ID", "followup ID is required")
	}

	existingRule, err := h.automationService.GetRule(c.Request().Context(), automationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "failed to get followup") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup rule not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_GET_FAILED", "failed to get followup rule: "+err.Error())
	}

	if existingRule.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to trigger this followup rule")
	}

	if err := h.automationService.TriggerRule(c.Request().Context(), automationID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup rule not found")
		}
		if strings.Contains(err.Error(), "credential") || strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "token") {
			return buildErrorResponse(c, http.StatusBadGateway, "FOLLOWUP_TRIGGER_FAILED", "failed to send email: "+err.Error())
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_TRIGGER_FAILED", "failed to trigger followup rule: "+err.Error())
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"message":       "followup rule triggered successfully",
		"automation_id": automationID,
	})
}
