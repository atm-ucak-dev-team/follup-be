package service

import (
	"context"

	"github.com/atm-ucak/follup/internal/domain"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// ExchangeJiraCode exchanges the authorization code for access token and returns user with token info
	ExchangeJiraCode(ctx context.Context, code, state string) (*domain.User, *JiraTokenInfo, error)

	// RefreshJiraToken refreshes an expired access token and returns new access token
	RefreshJiraToken(ctx context.Context, refreshToken string) (*domain.TokenResponse, error)

	// GenerateAuthURL creates the Jira OAuth authorization URL
	GenerateAuthURL(state string) string

	// ValidateToken validates if a token is valid and not expired
	ValidateToken(token *domain.OAuthToken) bool
}

// AutomationService defines the interface for automation rule operations
type AutomationService interface {
	// CreateRule creates a new automation rule
	CreateRule(ctx interface{}, rule *domain.AutomationRule) error

	// GetRule retrieves an automation rule by ID
	GetRule(ctx interface{}, id string) (*domain.AutomationRule, error)

	// GetUserRules retrieves all rules for a user
	GetUserRules(ctx interface{}, userID string) ([]*domain.AutomationRule, error)

	// UpdateRule updates an existing automation rule
	UpdateRule(ctx interface{}, rule *domain.AutomationRule) error

	// DeleteRule deletes an automation rule
	DeleteRule(ctx interface{}, id string) error

	// PauseRule pauses an active automation rule
	PauseRule(ctx interface{}, id string) error

	// ResumeRule resumes a paused automation rule
	ResumeRule(ctx interface{}, id string) error

	// GetActiveRules retrieves all active automation rules
	GetActiveRules(ctx interface{}) ([]*domain.AutomationRule, error)

	// TriggerRule manually executes an automation rule
	TriggerRule(ctx interface{}, automationID string) error
}

// EmailService defines the interface for email operations
type EmailService interface {
	// RegisterCredential registers or updates email credentials
	RegisterCredential(ctx interface{}, cred *domain.EmailCredential) error

	// GetCredential retrieves email credentials for a user
	GetCredential(ctx interface{}, userID string) (*domain.EmailCredential, error)

	// SendFollowUp sends a follow-up email
	SendFollowUp(ctx interface{}, threadID, subject, body string, recipients []string) error

	// CheckForReplies checks IMAP for replies to sent emails
	CheckForReplies(ctx interface{}) error

	// DecryptPassword decrypts an encrypted password
	DecryptPassword(encryptedPassword string) (string, error)

	// SaveCredential saves email credentials with encryption
	SaveCredential(ctx context.Context, userID, email, password string) error

	// SendFollowUpByAutomation sends a follow-up email based on automation ID
	SendFollowUpByAutomation(ctx context.Context, automationID string) error

	// PollInbox polls IMAP inbox for replies
	PollInbox(ctx context.Context) error
}

// JiraService defines the interface for Jira operations
type JiraService interface {
	// GetTicket retrieves a Jira ticket by ID
	GetTicket(ctx interface{}, ticketID string) (*domain.JiraTicket, error)

	// UpdateTicketStatus updates the status of a Jira ticket
	UpdateTicketStatus(ctx interface{}, ticketID, status string) error

	// AddComment adds a comment to a Jira ticket
	AddComment(ctx interface{}, ticketID, comment string) error

	// GetAuthenticatedUser retrieves the authenticated Jira user
	GetAuthenticatedUser(ctx interface{}) (*domain.User, error)

	// GetIssues retrieves Jira issues using Atlassian Cloud API JQL search
	GetIssues(ctx context.Context, cloudID, accessToken, search, limit string) ([]domain.JiraIssueResponse, error)

	// GetIssue retrieves a single Jira issue by issue ID using Atlassian Cloud API
	GetIssue(ctx context.Context, cloudID, accessToken, issueID string) (*domain.JiraIssueDetailResponse, error)
}
