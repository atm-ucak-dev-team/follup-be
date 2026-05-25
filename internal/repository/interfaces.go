package repository

import (
	"context"
	"github.com/atm-ucak/follup/internal/domain"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}

// OAuthTokenRepository defines the interface for OAuth token operations
type OAuthTokenRepository interface {
	Create(ctx context.Context, token *domain.OAuthToken) error
	GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*domain.OAuthToken, error)
	Update(ctx context.Context, token *domain.OAuthToken) error
	Delete(ctx context.Context, userID, provider string) error
}

// EmailCredentialRepository defines the interface for email credential operations
type EmailCredentialRepository interface {
	Create(ctx context.Context, cred *domain.EmailCredential) error
	GetByUserID(ctx context.Context, userID string) (*domain.EmailCredential, error)
	Update(ctx context.Context, cred *domain.EmailCredential) error
	Delete(ctx context.Context, userID string) error
}

// FollowupRepository defines the interface for follow-up rule operations
type FollowupRepository interface {
	Create(ctx context.Context, rule *domain.Followup) error
	GetByID(ctx context.Context, id string) (*domain.Followup, error)
	GetByUserID(ctx context.Context, userID string) ([]*domain.Followup, error)
	GetActiveRules(ctx context.Context) ([]*domain.Followup, error)
	Update(ctx context.Context, rule *domain.Followup) error
	Delete(ctx context.Context, id string) error
}

// EmailThreadRepository defines the interface for email thread operations
type EmailThreadRepository interface {
	Create(ctx context.Context, thread *domain.EmailThread) error
	GetByID(ctx context.Context, id string) (*domain.EmailThread, error)
	GetByAutomationID(ctx context.Context, automationID string) ([]*domain.EmailThread, error)
	GetByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.EmailThread, error)
	Update(ctx context.Context, thread *domain.EmailThread) error
	UpdateThreadStatus(ctx context.Context, threadID, status string) error
	Delete(ctx context.Context, id string) error
}
