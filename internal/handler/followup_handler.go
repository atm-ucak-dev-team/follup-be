package handler

import (
	"net/http"
	"strings"

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
