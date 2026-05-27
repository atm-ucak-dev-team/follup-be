package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EmailThread represents an email thread for follow-up tracking.
type EmailThread struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AutomationID  string    `json:"automation_id"`
	GmailThreadID string    `json:"gmail_thread_id"`
	TicketID      string    `json:"ticket_id"`
	Status        string    `json:"status"`
	Body          string    `json:"body"` // Reply email body content
	LastSyncedAt  time.Time `json:"last_synced_at"`
}

const (
	EmailThreadStatusOpen    = "open"
	EmailThreadStatusReplied = "replied"
	EmailThreadStatusClosed  = "closed"
)

// ValidateAutomationID validates that AutomationID is a valid UUID string
func (t *EmailThread) ValidateAutomationID() error {
	_, err := uuid.Parse(t.AutomationID)
	if err != nil {
		return fmt.Errorf("invalid automation_id format: %w", err)
	}
	return nil
}
