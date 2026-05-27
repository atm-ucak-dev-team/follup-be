package dragonfly

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/infra"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	followupKeyPrefix   = "followup:"
	followupIndexPrefix = "followup:index:"
	followupActiveKey   = "followup:active"
)

// FollowupRepository implements FollowupRepository with DragonflyDB/Redis
type FollowupRepository struct {
	client *redis.Client
}

// NewFollowupRepository creates a new FollowupRepository instance
func NewFollowupRepository(client *redis.Client) *FollowupRepository {
	return &FollowupRepository{
		client: client,
	}
}

// Create saves a new automation rule to DragonflyDB
func (r *FollowupRepository) Create(ctx context.Context, rule *domain.Followup) error {
	// Generate UUID if ID is empty
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Set creation time
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}

	// Marshal and save the automation rule
	key := followupKeyPrefix + rule.ID
	if err := infra.JSONSet(ctx, r.client, key, rule); err != nil {
		return fmt.Errorf("failed to save followup: %w", err)
	}

	// Add to user's index list
	indexKey := followupIndexPrefix + rule.UserID
	if err := r.client.RPush(ctx, indexKey, rule.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to user index: %w", err)
	}

	return nil
}

// GetByID retrieves a single followup rule by ID
func (r *FollowupRepository) GetByID(ctx context.Context, id string) (*domain.Followup, error) {
	key := followupKeyPrefix + id

	var rule domain.Followup
	if err := infra.JSONGet(ctx, r.client, key, &rule); err != nil {
		return nil, fmt.Errorf("failed to get followup: %w", err)
	}

	return &rule, nil
}

// GetByUserID retrieves all followups for a specific user
func (r *FollowupRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Followup, error) {
	// Get followup IDs from user's index
	indexKey := followupIndexPrefix + userID
	followupIDs, err := r.client.LRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user index: %w", err)
	}

	if len(followupIDs) == 0 {
		return []*domain.Followup{}, nil
	}

	// Fetch all followup rules
	keys := make([]string, len(followupIDs))
	for i, id := range followupIDs {
		keys[i] = followupKeyPrefix + id
	}

	dataMap, err := infra.JSONGetMultiple(ctx, r.client, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get followups: %w", err)
	}

	rules := make([]*domain.Followup, 0, len(dataMap))
	for _, data := range dataMap {
		var rule domain.Followup
		if err := json.Unmarshal(data, &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal followup: %w", err)
		}
		rules = append(rules, &rule)
	}

	return rules, nil
}

// GetActiveRules retrieves all active (ongoing) followup rules (for cron scheduler)
func (r *FollowupRepository) GetActiveRules(ctx context.Context) ([]*domain.Followup, error) {
	// Scan all followup keys, excluding index keys
	pattern := followupKeyPrefix + "*"
	var keys []string
	var cursor uint64

	for {
		var batch []string
		var err error

		batch, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan followup keys: %w", err)
		}

		// Filter out index keys (they contain ":index:")
		for _, key := range batch {
			if !strings.Contains(key, ":index:") {
				keys = append(keys, key)
			}
		}

		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return []*domain.Followup{}, nil
	}

	// Fetch all followup rules
	dataMap, err := infra.JSONGetMultiple(ctx, r.client, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get followups: %w", err)
	}

	// Filter for active rules only
	rules := make([]*domain.Followup, 0, len(dataMap))
	now := time.Now() // Get current time for startDateTime validation

	for _, data := range dataMap {
		var rule domain.Followup
		if err := json.Unmarshal(data, &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal followup: %w", err)
		}

		// Only include rules that have started (startDateTime <= now)
		if rule.Status == domain.FollowupStatusOngoing && rule.StartDateTime.Before(now) {
			rules = append(rules, &rule)
		}
	}

	return rules, nil
}

// Update updates an existing followup rule
func (r *FollowupRepository) Update(ctx context.Context, rule *domain.Followup) error {
	// Check if rule exists
	key := followupKeyPrefix + rule.ID
	exists, err := infra.JSONExists(ctx, r.client, key)
	if err != nil {
		return fmt.Errorf("failed to check rule existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("followup with ID '%s' not found", rule.ID)
	}

	// If userID changed, update indexes
	var oldRule domain.Followup
	if err := infra.JSONGet(ctx, r.client, key, &oldRule); err == nil {
		if oldRule.UserID != rule.UserID {
			oldIndexKey := followupIndexPrefix + oldRule.UserID
			r.client.LRem(ctx, oldIndexKey, 0, rule.ID)

			newIndexKey := followupIndexPrefix + rule.UserID
			r.client.RPush(ctx, newIndexKey, rule.ID)
		}
	}

	// Update the rule
	if err := infra.JSONSet(ctx, r.client, key, rule); err != nil {
		return fmt.Errorf("failed to update followup: %w", err)
	}

	return nil
}

// Delete removes a followup rule and cleans up indexes
func (r *FollowupRepository) Delete(ctx context.Context, id string) error {
	// First get the rule to find user_id
	key := followupKeyPrefix + id

	var rule domain.Followup
	if err := infra.JSONGet(ctx, r.client, key, &rule); err != nil {
		return fmt.Errorf("failed to get followup for deletion: %w", err)
	}

	// Delete the main rule
	if err := infra.JSONDelete(ctx, r.client, key); err != nil {
		return fmt.Errorf("failed to delete followup: %w", err)
	}

	// Remove from user's index
	indexKey := followupIndexPrefix + rule.UserID
	if err := r.client.LRem(ctx, indexKey, 0, id).Err(); err != nil {
		return fmt.Errorf("failed to remove from user index: %w", err)
	}

	return nil
}
