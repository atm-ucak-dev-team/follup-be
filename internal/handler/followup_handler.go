package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
)

type FollowupHandler struct {
	automationService service2.AutomationService
}

func NewFollowupHandler(automationService service2.AutomationService) *FollowupHandler {
	return &FollowupHandler{
		automationService: automationService,
	}
}

func (h *FollowupHandler) CreateFollowup(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	var req CreateFollowupRequest
	if err := c.Bind(&req); err != nil {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
	}

	if req.JiraTicketID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_ID", "jiraTicketId is required")
	}
	if req.To == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_TO", "to is required")
	}
	if !strings.Contains(req.To, "@") {
		return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_TO", "to must be a valid email address")
	}
	if req.Cc != "" && !strings.Contains(req.Cc, "@") {
		return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_CC", "cc must be a valid email address")
	}
	if len(req.To) > 500 || len(req.Cc) > 500 {
		return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_RECIPIENTS", "to/cc must not exceed 500 characters")
	}
	if req.Subject == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_SUBJECT", "subject is required")
	}
	if len(req.Subject) > 500 {
		return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_SUBJECT", "subject must not exceed 500 characters")
	}
	if req.EmailBody == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_EMAIL_BODY", "emailBody is required")
	}
	if len(req.EmailBody) > 10000 {
		return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_EMAIL_BODY", "emailBody must not exceed 10000 characters")
	}
	if req.Frequency == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FREQUENCY", "frequency is required")
	}
	if req.Repeat < 0 {
		return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_REPEAT", "repeat must not be negative")
	}

	var cc *string
	if req.Cc != "" {
		cc = &req.Cc
	}

	rule := &domain.Followup{
		UserID:               userID,
		JiraTicketID:         req.JiraTicketID,
		To:                   req.To,
		Cc:                   cc,
		Subject:              req.Subject,
		EmailBody:            req.EmailBody,
		StartDateTime:        req.StartDateTime,
		ExpireDateTime:       req.ExpireDateTime,
		Frequency:            req.Frequency,
		Repeat:               req.Repeat,
		FollowupConfirmation: req.FollowupConfirmation,
		ExecutionCount:       0, // Initialize execution count to 0
	}

	if err := h.automationService.CreateRule(c.Request().Context(), rule); err != nil {
		if strings.Contains(err.Error(), "invalid frequency") {
			return buildErrorResponse(c, http.StatusUnprocessableEntity, "INVALID_FREQUENCY", "invalid frequency: "+err.Error())
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_CREATE_FAILED", "failed to create followup: "+err.Error())
	}

	return c.JSON(http.StatusCreated, rule)
}

type FollowupItem struct {
	FollowupID   string  `json:"followupId"`
	JiraTicketID string  `json:"jiraTicketId"`
	Subject      string  `json:"subject"`
	LastFollowUp *string `json:"lastFollowUp"`
	NextFollowUp *string `json:"nextFollowUp"`
	RepliedAt    *string `json:"repliedAt"`
	Status       string  `json:"status"`
}

type FollowupSummaryResponse struct {
	JiraTicketID string `json:"jiraTicketId"`
	JiraTitle    string `json:"jiraTitle"`
	Replied      int    `json:"replied"`
	Ongoing      int    `json:"ongoing"`
	Expired      int    `json:"expired"`
}

// CreateFollowupRequest represents the request body for creating a followup via POST /v1/followups
type CreateFollowupRequest struct {
	JiraTicketID         string    `json:"jiraTicketId"`
	To                   string    `json:"to"`
	Cc                   string    `json:"cc,omitempty"`
	Subject              string    `json:"subject"`
	EmailBody            string    `json:"emailBody"`
	StartDateTime        time.Time `json:"startDateTime"`
	ExpireDateTime       time.Time `json:"expireDateTime"`
	Frequency            string    `json:"frequency"`
	Repeat               int       `json:"repeat"`
	FollowupConfirmation bool      `json:"followupConfirmation"`
}

// StatisticResponse represents the global summary without ticket-specific fields
type StatisticResponse struct {
	Replied int `json:"replied"`
	Ongoing int `json:"ongoing"`
	Expired int `json:"expired"`
}

func formatTime(t *string) *string {
	if t == nil || *t == "" {
		return nil
	}
	return t
}

func (h *FollowupHandler) ListFollowups(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	jiraTicketID := c.QueryParam("jiraTicket")
	if jiraTicketID == "" {
		jiraTicketID = c.Param("jiraTicketID")
	}

	details, err := h.automationService.ListFollowupDetails(c.Request().Context(), userID, jiraTicketID)
	if err != nil {
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_LIST_FAILED", "failed to list followups: "+err.Error())
	}

	items := make([]FollowupItem, 0, len(details))
	for _, d := range details {
		r := d.Followup

		ticketID := r.JiraTicketKey
		if ticketID == "" {
			ticketID = r.JiraTicketID
		}

		item := FollowupItem{
			FollowupID:   r.ID,
			JiraTicketID: ticketID,
			Subject:      r.Subject,
			Status:       d.EffectiveStatus,
		}

		switch d.EffectiveStatus {
		case "ongoing":
			if d.NextFollowUp != nil {
				s := d.NextFollowUp.Format("2006-01-02T15:04:05Z")
				item.NextFollowUp = &s
			}
		case "replied":
			if d.RepliedAt != nil {
				s := d.RepliedAt.Format("2006-01-02T15:04:05Z")
				item.RepliedAt = &s
			}
		case "expired":
			if r.LastRunAt != nil {
				s := r.LastRunAt.Format("2006-01-02T15:04:05Z")
				item.LastFollowUp = &s
			}
		}

		items = append(items, item)
	}

	return c.JSON(http.StatusOK, items)
}

func (h *FollowupHandler) GetFollowupsByTicketID(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	jiraTicketID := c.Param("jiraTicketID")
	if jiraTicketID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_ID", "jiraTicketID is required")
	}

	return h.ListFollowups(c)
}

func (h *FollowupHandler) GetSummary(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	jiraTicketID := c.Param("jiraTicketID")
	if jiraTicketID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_ID", "jiraTicketID is required")
	}

	summary, err := h.automationService.GetSummary(c.Request().Context(), userID, jiraTicketID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return buildErrorResponse(c, http.StatusNotFound, "TICKET_NOT_FOUND", "jira ticket not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "SUMMARY_FAILED", "failed to get summary: "+err.Error())
	}

	resp := FollowupSummaryResponse{
		JiraTicketID: summary.JiraTicketID,
		JiraTitle:    summary.JiraTitle,
		Replied:      summary.Replied,
		Ongoing:      summary.Ongoing,
		Expired:      summary.Expired,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetGlobalSummary returns summary counts across all jira tickets for a user
func (h *FollowupHandler) GetGlobalSummary(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	summary, err := h.automationService.GetGlobalSummary(c.Request().Context(), userID)
	if err != nil {
		return buildErrorResponse(c, http.StatusInternalServerError, "GLOBAL_SUMMARY_FAILED", "failed to get global summary: "+err.Error())
	}

	resp := StatisticResponse{
		Replied: summary.Replied,
		Ongoing: summary.Ongoing,
		Expired: summary.Expired,
	}

	return c.JSON(http.StatusOK, resp)
}
