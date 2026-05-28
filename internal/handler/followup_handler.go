package handler

import (
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/repository"
	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/labstack/echo/v4"
)

var uuidRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

type FollowupHandler struct {
	automationService   service2.AutomationService
	jiraService         service2.JiraService
	emailThreadRepo     repository.EmailThreadRepository
	emailCredentialRepo repository.EmailCredentialRepository
}

func NewFollowupHandler(automationService service2.AutomationService, jiraService service2.JiraService, emailThreadRepo repository.EmailThreadRepository, emailCredentialRepo repository.EmailCredentialRepository) *FollowupHandler {
	return &FollowupHandler{
		automationService:   automationService,
		jiraService:         jiraService,
		emailThreadRepo:     emailThreadRepo,
		emailCredentialRepo: emailCredentialRepo,
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
	if req.JiraTicketTitle == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_TITLE", "jiraTicketTitle is required")
	}
	if req.JiraStakeholder == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_STAKEHOLDER", "jiraStakeholder is required")
	}
	if req.JiraTicketStatus == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_JIRA_TICKET_STATUS", "jiraTicketStatus is required")
	}

	var cc *string
	if req.Cc != "" {
		cc = &req.Cc
	}

	rule := &domain.Followup{
		UserID:               userID,
		JiraTicketID:         req.JiraTicketID,
		JiraTicketTitle:      req.JiraTicketTitle,
		JiraStakeholder:      req.JiraStakeholder,
		JiraTicketStatus:     req.JiraTicketStatus,
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

func (h *FollowupHandler) GetFollowup(c echo.Context) error {
	userID := getUserIDFromContext(c)
	if userID == "" {
		return buildErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
	}

	followupID := c.Param("id")
	if followupID == "" {
		return buildErrorResponse(c, http.StatusBadRequest, "MISSING_FOLLOWUP_ID", "followup ID is required")
	}
	if !uuidRegex.MatchString(followupID) {
		return buildErrorResponse(c, http.StatusBadRequest, "INVALID_FOLLOWUP_ID", "followup ID must be a valid UUID")
	}

	detail, err := h.automationService.GetFollowupDetail(c.Request().Context(), followupID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return buildErrorResponse(c, http.StatusNotFound, "FOLLOWUP_NOT_FOUND", "followup not found")
		}
		return buildErrorResponse(c, http.StatusInternalServerError, "FOLLOWUP_GET_FAILED", "failed to get followup: "+err.Error())
	}

	if detail.Followup.UserID != userID {
		return buildErrorResponse(c, http.StatusForbidden, "FORBIDDEN", "you don't have permission to view this followup")
	}

	// Fetch user's email credential for "from" field
	fromEmail := ""
	emailCred, err := h.emailCredentialRepo.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		log.Printf("Error retrieving email credential for user %s: %v", userID, err)
	} else {
		fromEmail = emailCred.EmailAddress
	}

	var lastFollowUpStr *string
	if detail.Followup.LastRunAt != nil {
		s := detail.Followup.LastRunAt.Format("2006-01-02T15:04:05Z")
		lastFollowUpStr = &s
	}

	var sendEmailEveryStr *string
	if detail.NextFollowUp != nil {
		s := detail.NextFollowUp.Format("2006-01-02T15:04:05Z")
		sendEmailEveryStr = &s
	}

	stakeholderName := "Unassigned"
	if detail.Followup.JiraStakeholder != "" {
		stakeholderName = detail.Followup.JiraStakeholder
	}

	// Query email threads by automation ID
	threads := make([]ThreadItem, 0)
	emailThreads, err := h.emailThreadRepo.GetByAutomationID(c.Request().Context(), followupID)
	if err != nil {
		log.Printf("Error retrieving email threads for followup %s: %v", followupID, err)
	} else {
		log.Printf("Retrieved %d email threads for followup %s", len(emailThreads), followupID)
		for i, thread := range emailThreads {
			log.Printf("Thread %d: ID=%s, GmailID=%s, Status=%s", i+1, thread.ID, thread.GmailThreadID, thread.Status)
		}
	}
	if err == nil && emailThreads != nil {
		for _, et := range emailThreads {
			threads = append(threads, ThreadItem{
				ID:            et.ID,
				GmailThreadID: et.GmailThreadID,
				Status:        et.Status,
				Body:          et.Body,
				LastSyncedAt:  et.LastSyncedAt.Format("2006-01-02T15:04:05Z"),
			})
		}
	}

	// Generate suggestion based on status
	suggestion := ""
	switch detail.EffectiveStatus {
	case "ongoing":
		suggestion = "We're still checking and connecting you to your stakeholder"
	case "replied":
		suggestion = "You're connected and completed this follow up!"
	case "expired":
		suggestion = "Check your email regularly or start new automation"
	}

	resp := FollowupDetailResponse{
		Subject:          detail.Followup.Subject,
		Status:           detail.EffectiveStatus,
		ExpireDateTime:   detail.Followup.ExpireDateTime.Format("2006-01-02T15:04:05Z"),
		LastFollowUp:     lastFollowUpStr,
		StakeholderName:  stakeholderName,
		SendEmailEvery:   sendEmailEveryStr,
		Threads:          threads,
		Suggestion:       suggestion,
		JiraTicketTitle:  detail.Followup.JiraTicketTitle,
		JiraTicketStatus: detail.Followup.JiraTicketStatus,
		From:             fromEmail,
		To:               detail.Followup.To,
		Cc:               detail.Followup.Cc,
	}

	return c.JSON(http.StatusOK, resp)
}

type FollowupItem struct {
	FollowupID      string  `json:"followupId"`
	JiraTicketID    string  `json:"jiraTicketId"`
	Subject         string  `json:"subject"`
	StakeholderName string  `json:"stakeholderName"`
	LastFollowUp    *string `json:"lastFollowUp"`
	NextFollowUp    *string `json:"nextFollowUp"`
	RepliedAt       *string `json:"repliedAt"`
	Status          string  `json:"status"`
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
	JiraTicketTitle      string    `json:"jiraTicketTitle"`
	JiraStakeholder      string    `json:"jiraStakeholder"`
	JiraTicketStatus     string    `json:"jiraTicketStatus"`
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

// ThreadItem represents a single email thread in the response
type ThreadItem struct {
	ID            string `json:"id"`
	GmailThreadID string `json:"gmailThreadId"`
	Status        string `json:"status"`
	Body          string `json:"body"`
	LastSyncedAt  string `json:"lastSyncedAt"`
}

// FollowupDetailResponse represents the response for GET /api/v1/followups/:id
type FollowupDetailResponse struct {
	Subject          string       `json:"subject"`
	Status           string       `json:"status"`
	ExpireDateTime   string       `json:"expireDateTime"`
	LastFollowUp     *string      `json:"lastFollowUp"`
	StakeholderName  string       `json:"stakeholderName"`
	SendEmailEvery   *string      `json:"sendEmailEvery"`
	Threads          []ThreadItem `json:"threads"`
	Suggestion       string       `json:"suggestion"`
	JiraTicketTitle  string       `json:"jiraTicketTitle"`
	JiraTicketStatus string       `json:"jiraTicketStatus"`
	From             string       `json:"from"`
	To               string       `json:"to"`
	Cc               *string      `json:"cc"`
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
			FollowupID:      r.ID,
			JiraTicketID:    ticketID,
			Subject:         r.Subject,
			StakeholderName: r.JiraStakeholder,
			Status:          d.EffectiveStatus,
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
