package domain

import "time"

// Followup represents a follow-up automation rule.
// Matches the followups table in PostgreSQL migration.
// Extra fields (JiraTicketKey, LastRunAt, CreatedAt) are app-only, stored in Redis.
type Followup struct {
	ID                   string     `json:"id"`
	JiraTicketID         string     `json:"jira_ticket_id"`
	JiraTicketKey        string     `json:"jira_ticket_key"`
	JiraTicketTitle      string     `json:"jiraTicketTitle"`
	JiraStakeholder      string     `json:"jiraStakeholder"`
	JiraTicketStatus     string     `json:"jiraTicketStatus"`
	UserID               string     `json:"user_id"`
	To                   string     `json:"to"`
	Cc                   *string    `json:"cc,omitempty"`
	Subject              string     `json:"subject"`
	EmailBody            string     `json:"email_body"`
	StartDateTime        time.Time  `json:"start_date_time"`
	ExpireDateTime       time.Time  `json:"expire_date_time"`
	Frequency            string     `json:"frequency"`
	Repeat               int        `json:"repeat"`
	FollowupConfirmation bool       `json:"followup_confirmation"`
	Status               string     `json:"status"`
	ExecutionCount       int        `json:"execution_count"` // Number of successful executions
	LastRunAt            *time.Time `json:"last_run_at"`
	CreatedAt            time.Time  `json:"created_at"`
}

const (
	FollowupStatusOngoing   = "ongoing"
	FollowupStatusCompleted = "completed"
	FollowupStatusStopped   = "stopped"
	FollowupStatusExpired   = "expired"
	FollowupStatusReplied   = "replied"
)
