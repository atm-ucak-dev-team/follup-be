package dragonfly

import (
	"context"
	"fmt"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/infra"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// EmailThreadRepositoryImpl implements EmailThreadRepository
type EmailThreadRepositoryImpl struct {
	client *redis.Client
}

// NewEmailThreadRepository creates a new EmailThreadRepository
func NewEmailThreadRepository(client *redis.Client) *EmailThreadRepositoryImpl {
	return &EmailThreadRepositoryImpl{client: client}
}

// Create stores an email thread and updates the automation index
func (r *EmailThreadRepositoryImpl) Create(ctx context.Context, thread *domain.EmailThread) error {
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

// GetByID retrieves an email thread by ID
func (r *EmailThreadRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.EmailThread, error) {
	key := domain.EmailThreadKey(id)
	var thread domain.EmailThread
	err := infra.JSONGet(ctx, r.client, key, &thread)
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

// GetByAutomationID retrieves all threads for a specific automation
func (r *EmailThreadRepositoryImpl) GetByAutomationID(ctx context.Context, automationID string) ([]*domain.EmailThread, error) {
	indexKey := domain.EmailThreadIndexKey(automationID)

	// Get all thread IDs from the index
	threadIDs, err := r.client.SMembers(ctx, indexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get thread index: %w", err)
	}

	if len(threadIDs) == 0 {
		return []*domain.EmailThread{}, nil
	}

	// Fetch all threads
	threads := make([]*domain.EmailThread, 0, len(threadIDs))
	for _, threadID := range threadIDs {
		thread, err := r.GetByID(ctx, threadID)
		if err != nil {
			// Skip threads that don't exist (index inconsistency)
			continue
		}
		threads = append(threads, thread)
	}

	return threads, nil
}

// GetByGmailThreadID implements EmailThreadRepository.GetByGmailThreadID
func (r *EmailThreadRepositoryImpl) GetByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.EmailThread, error) {
	// This would require a reverse lookup by GmailThreadID
	// For now, return not found since we don't have this indexed
	return nil, fmt.Errorf("thread not found")
}

// Update updates an existing thread
func (r *EmailThreadRepositoryImpl) Update(ctx context.Context, thread *domain.EmailThread) error {
	threadKey := domain.EmailThreadKey(thread.ID)
	return infra.JSONSet(ctx, r.client, threadKey, thread)
}

// UpdateThreadStatus updates the status of an existing thread
func (r *EmailThreadRepositoryImpl) UpdateThreadStatus(ctx context.Context, threadID, status string) error {
	// Get existing thread
	thread, err := r.GetByID(ctx, threadID)
	if err != nil {
		return fmt.Errorf("thread not found: %w", err)
	}

	// Update status
	thread.Status = status

	// Save updated thread
	threadKey := domain.EmailThreadKey(threadID)
	return infra.JSONSet(ctx, r.client, threadKey, thread)
}

// Delete removes a thread
func (r *EmailThreadRepositoryImpl) Delete(ctx context.Context, id string) error {
	threadKey := domain.EmailThreadKey(id)
	return infra.JSONDelete(ctx, r.client, threadKey)
}
