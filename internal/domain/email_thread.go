package domain

import "time"

// EmailThread represents an email thread for follow-up tracking.
type EmailThread struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AutomationID  string    `json:"automation_id"`
	GmailThreadID string    `json:"gmail_thread_id"`
	TicketID      string    `json:"ticket_id"`
	Status        string    `json:"status"`
	LastSyncedAt  time.Time `json:"last_synced_at"`
}

const (
	EmailThreadStatusOpen    = "open"
	EmailThreadStatusReplied = "replied"
	EmailThreadStatusClosed  = "closed"
)
