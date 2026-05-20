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
	automationKeyPrefix   = "automation:"
	automationIndexPrefix = "automation:index:"
	automationActiveKey   = "automation:active"
)

// AutomationRepository implements AutomationRuleRepository with DragonflyDB
type AutomationRepository struct {
	client *redis.Client
}

// NewAutomationRepository creates a new AutomationRepository instance
func NewAutomationRepository(client *redis.Client) *AutomationRepository {
	return &AutomationRepository{
		client: client,
	}
}

// Create saves a new automation rule to DragonflyDB
func (r *AutomationRepository) Create(ctx context.Context, rule *domain.AutomationRule) error {
	// Generate UUID if ID is empty
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Set creation time
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}

	// Marshal and save the automation rule
	key := automationKeyPrefix + rule.ID
	if err := infra.JSONSet(ctx, r.client, key, rule); err != nil {
		return fmt.Errorf("failed to save automation rule: %w", err)
	}

	// Add to user's index list
	indexKey := automationIndexPrefix + rule.UserID
	if err := r.client.RPush(ctx, indexKey, rule.ID).Err(); err != nil {
		return fmt.Errorf("failed to add to user index: %w", err)
	}

	return nil
}

// GetByID retrieves a single automation rule by ID
func (r *AutomationRepository) GetByID(ctx context.Context, id string) (*domain.AutomationRule, error) {
	key := automationKeyPrefix + id

	var rule domain.AutomationRule
	if err := infra.JSONGet(ctx, r.client, key, &rule); err != nil {
		return nil, fmt.Errorf("failed to get automation rule: %w", err)
	}

	return &rule, nil
}

// GetByUserID retrieves all automation rules for a specific user
func (r *AutomationRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.AutomationRule, error) {
	// Get automation IDs from user's index
	indexKey := automationIndexPrefix + userID
	automationIDs, err := r.client.LRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user index: %w", err)
	}

	if len(automationIDs) == 0 {
		return []*domain.AutomationRule{}, nil
	}

	// Fetch all automation rules
	keys := make([]string, len(automationIDs))
	for i, id := range automationIDs {
		keys[i] = automationKeyPrefix + id
	}

	dataMap, err := infra.JSONGetMultiple(ctx, r.client, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get automation rules: %w", err)
	}

	rules := make([]*domain.AutomationRule, 0, len(dataMap))
	for _, data := range dataMap {
		var rule domain.AutomationRule
		if err := json.Unmarshal(data, &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal automation rule: %w", err)
		}
		rules = append(rules, &rule)
	}

	return rules, nil
}

// GetActiveRules retrieves all active automation rules (for cron scheduler)
func (r *AutomationRepository) GetActiveRules(ctx context.Context) ([]*domain.AutomationRule, error) {
	// Scan all automation keys, excluding index keys
	pattern := automationKeyPrefix + "*"
	var keys []string
	var cursor uint64

	for {
		var batch []string
		var err error

		batch, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan automation keys: %w", err)
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
		return []*domain.AutomationRule{}, nil
	}

	// Fetch all automation rules
	dataMap, err := infra.JSONGetMultiple(ctx, r.client, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to get automation rules: %w", err)
	}

	// Filter for active rules only
	rules := make([]*domain.AutomationRule, 0, len(dataMap))
	for _, data := range dataMap {
		var rule domain.AutomationRule
		if err := json.Unmarshal(data, &rule); err != nil {
			return nil, fmt.Errorf("failed to unmarshal automation rule: %w", err)
		}

		if rule.Status == domain.AutomationStatusActive {
			rules = append(rules, &rule)
		}
	}

	return rules, nil
}

// Update updates an existing automation rule
func (r *AutomationRepository) Update(ctx context.Context, rule *domain.AutomationRule) error {
	// Check if rule exists
	key := automationKeyPrefix + rule.ID
	exists, err := infra.JSONExists(ctx, r.client, key)
	if err != nil {
		return fmt.Errorf("failed to check rule existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("automation rule with ID '%s' not found", rule.ID)
	}

	// If userID changed, update indexes
	var oldRule domain.AutomationRule
	if err := infra.JSONGet(ctx, r.client, key, &oldRule); err == nil {
		if oldRule.UserID != rule.UserID {
			// Remove from old user index
			oldIndexKey := automationIndexPrefix + oldRule.UserID
			r.client.LRem(ctx, oldIndexKey, 0, rule.ID)

			// Add to new user index
			newIndexKey := automationIndexPrefix + rule.UserID
			r.client.RPush(ctx, newIndexKey, rule.ID)
		}
	}

	// Update the rule
	if err := infra.JSONSet(ctx, r.client, key, rule); err != nil {
		return fmt.Errorf("failed to update automation rule: %w", err)
	}

	return nil
}

// Delete removes an automation rule and cleans up indexes
func (r *AutomationRepository) Delete(ctx context.Context, id string) error {
	// First get the rule to find user_id
	key := automationKeyPrefix + id

	var rule domain.AutomationRule
	if err := infra.JSONGet(ctx, r.client, key, &rule); err != nil {
		return fmt.Errorf("failed to get automation rule for deletion: %w", err)
	}

	// Delete the main rule
	if err := infra.JSONDelete(ctx, r.client, key); err != nil {
		return fmt.Errorf("failed to delete automation rule: %w", err)
	}

	// Remove from user's index
	indexKey := automationIndexPrefix + rule.UserID
	if err := r.client.LRem(ctx, indexKey, 0, id).Err(); err != nil {
		return fmt.Errorf("failed to remove from user index: %w", err)
	}

	return nil
}
