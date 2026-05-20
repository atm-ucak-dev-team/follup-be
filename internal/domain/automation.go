package domain

import "time"

// AutomationRule represents an automation rule for follow-up emails
type AutomationRule struct {
	ID            string     `json:"id"`
	UserID        string     `json:"user_id"`
	JiraTicketID  string     `json:"jira_ticket_id"`
	JiraTicketKey string     `json:"jira_ticket_key"` // e.g., "PROJ-123"
	Recipients    []string   `json:"recipients"`
	CronSchedule  string     `json:"cron_schedule"` // cron expression, e.g., "0 9 * * 1"
	Status        string     `json:"status"`        // "active" | "paused"
	LastRunAt     *time.Time `json:"last_run_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// AutomationRuleStatus constants
const (
	AutomationStatusActive = "active"
	AutomationStatusPaused = "paused"
)

// EmailThread represents an email thread for follow-up tracking
type EmailThread struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AutomationID  string    `json:"automation_id"`
	GmailThreadID string    `json:"gmail_thread_id"` // IMAP thread/message ID
	TicketID      string    `json:"ticket_id"`       // bound jira ticket
	Status        string    `json:"status"`          // "open" | "replied" | "closed"
	LastSyncedAt  time.Time `json:"last_synced_at"`
}

// EmailThreadStatus constants
const (
	EmailThreadStatusOpen    = "open"
	EmailThreadStatusReplied = "replied"
	EmailThreadStatusClosed  = "closed"
)
