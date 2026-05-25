package handler

import (
	"net/http"

	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
)

type TicketHandler struct {
	jiraService       service2.JiraService
	automationService service2.AutomationService
}

func NewTicketHandler(jiraService service2.JiraService, automationService service2.AutomationService) *TicketHandler {
	return &TicketHandler{
		jiraService:       jiraService,
		automationService: automationService,
	}
}

type TicketResponse struct {
	JiraTicketId string `json:"jiraTicketId"`
	JiraTitle    string `json:"jiraTitle"`
}

func (h *TicketHandler) GetTickets(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	// Get followups for the user to get distinct ticket IDs
	followups, err := h.automationService.GetUserRules(c.Request().Context(), userID)
	if err != nil {
		return buildErrorResponse(c, http.StatusInternalServerError, "TICKETS_FAILED", "failed to get followups: "+err.Error())
	}

	// Map of ticketID -> ticketKey (to distinct by ticketID)
	ticketMap := make(map[string]string)
	for _, f := range followups {
		ticketMap[f.JiraTicketID] = f.JiraTicketKey
	}

	var results []TicketResponse
	for ticketID, ticketKey := range ticketMap {
		// Construct ticket response from available data
		// jiraTicketId: the database ticket ID (like "10001")
		// jiraTitle: use ticketKey as fallback for title since we don't have real summary
		results = append(results, TicketResponse{
			JiraTicketId: ticketID,
			JiraTitle:    ticketKey,
		})
	}

	return c.JSON(http.StatusOK, results)
}