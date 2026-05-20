package dragonfly

import (
	"context"
	"fmt"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/infra"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EmailRepository implements basic email-related operations
// Note: For full interface implementations, use the specific repository implementations:
// - NewEmailCredentialRepository for EmailCredentialRepository
// - NewOAuthTokenRepository for OAuthTokenRepository
// - NewEmailThreadRepository for EmailThreadRepository
type EmailRepository struct {
	client *redis.Client
}

// NewEmailRepository creates a new EmailRepository
func NewEmailRepository(client *redis.Client) *EmailRepository {
	return &EmailRepository{
		client: client,
	}
}

// === Email Credential Operations ===

// SaveCredential stores an email credential in DragonflyDB
func (r *EmailRepository) SaveCredential(ctx context.Context, cred domain.EmailCredential) error {
	key := domain.EmailCredentialKey(cred.UserID)
	return infra.JSONSet(ctx, r.client, key, cred)
}

// GetCredential retrieves an email credential by user ID
func (r *EmailRepository) GetCredential(ctx context.Context, userID string) (*domain.EmailCredential, error) {
	key := domain.EmailCredentialKey(userID)
	var cred domain.EmailCredential
	err := infra.JSONGet(ctx, r.client, key, &cred)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// DeleteCredential removes an email credential from DragonflyDB
func (r *EmailRepository) DeleteCredential(ctx context.Context, userID string) error {
	key := domain.EmailCredentialKey(userID)
	return infra.JSONDelete(ctx, r.client, key)
}

// === OAuth Token Operations ===

// SaveOAuthToken stores an OAuth token with calculated TTL
func (r *EmailRepository) SaveOAuthToken(ctx context.Context, token domain.OAuthToken) error {
	key := domain.OAuthKey(token.UserID, token.Provider)

	// Calculate TTL: expires_at - now - 5 minutes buffer
	ttl := time.Until(token.ExpiresAt) - 5*time.Minute
	if ttl <= 0 {
		return fmt.Errorf("token expires too soon or has already expired")
	}

	return infra.JSONSetWithTTL(ctx, r.client, key, token, ttl)
}

// GetOAuthToken retrieves an OAuth token by user ID and provider
func (r *EmailRepository) GetOAuthToken(ctx context.Context, userID, provider string) (*domain.OAuthToken, error) {
	key := domain.OAuthKey(userID, provider)
	var token domain.OAuthToken
	err := infra.JSONGet(ctx, r.client, key, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// DeleteOAuthToken removes an OAuth token from DragonflyDB
func (r *EmailRepository) DeleteOAuthToken(ctx context.Context, userID, provider string) error {
	key := domain.OAuthKey(userID, provider)
	return infra.JSONDelete(ctx, r.client, key)
}

// === Email Thread Operations ===

// SaveThread stores an email thread and updates the automation index
func (r *EmailRepository) SaveThread(ctx context.Context, thread *domain.EmailThread) error {
	// Generate UUID if ID is empty
	if thread.ID == "" {
		thread.ID = uuid.New().String()
	}

	// Store the thread
	threadKey := domain.EmailThreadKey(thread.ID)
	if err := infra.JSONSet(ctx, r.client, threadKey, thread); err != nil {
		return fmt.Errorf("failed to save thread: %w", err)
	}

	// Add thread ID to automation's index
	indexKey := domain.EmailThreadIndexKey(thread.AutomationID)
	if err := r.client.SAdd(ctx, indexKey, thread.ID).Err(); err != nil {
		return fmt.Errorf("failed to update thread index: %w", err)
	}

	return nil
}

// GetThread retrieves an email thread by ID
func (r *EmailRepository) GetThread(ctx context.Context, threadID string) (*domain.EmailThread, error) {
	key := domain.EmailThreadKey(threadID)
	var thread domain.EmailThread
	err := infra.JSONGet(ctx, r.client, key, &thread)
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

// GetThreadsByAutomation retrieves all threads for a specific automation
func (r *EmailRepository) GetThreadsByAutomation(ctx context.Context, automationID string) ([]domain.EmailThread, error) {
	indexKey := domain.EmailThreadIndexKey(automationID)

	// Get all thread IDs from the index
	threadIDs, err := r.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread index: %w", err)
	}

	if len(threadIDs) == 0 {
		return []domain.EmailThread{}, nil
	}

	// Fetch all threads
	threads := make([]domain.EmailThread, 0, len(threadIDs))
	for _, threadID := range threadIDs {
		thread, err := r.GetThread(ctx, threadID)
		if err != nil {
			// Skip threads that don't exist (index inconsistency)
			continue
		}
		threads = append(threads, *thread)
	}

	return threads, nil
}

// UpdateThreadStatus updates the status of an existing thread
func (r *EmailRepository) UpdateThreadStatus(ctx context.Context, threadID, status string) error {
	// Get existing thread
	thread, err := r.GetThread(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", err)
	}

	// Update status
	thread.Status = status

	// Save updated thread
	threadKey := domain.EmailThreadKey(threadID)
	return infra.JSONSet(ctx, r.client, threadKey, thread)
}
