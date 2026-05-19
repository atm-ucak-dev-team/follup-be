package dragonfly

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestAutomationRepository creates a test repository with miniredis
func setupTestAutomationRepository(t *testing.T) (*AutomationRepository, *miniredis.Miniredis) {
	t.Helper()

	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	repo := NewAutomationRepository(client)

	return repo, s
}

func TestAutomationRepo_CreateAndGet_Success(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	rule := &domain.AutomationRule{
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com", "admin@example.com"},
		CronSchedule:  "0 9 * * 1",
		Status:        domain.AutomationStatusActive,
		CreatedAt:     time.Now(),
	}

	// Create the rule
	err := repo.Create(ctx, rule)
	require.NoError(t, err)
	assert.NotEmpty(t, rule.ID, "ID should be generated")

	// Get the rule by ID
	retrieved, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)

	assert.Equal(t, rule.ID, retrieved.ID)
	assert.Equal(t, rule.UserID, retrieved.UserID)
	assert.Equal(t, rule.JiraTicketID, retrieved.JiraTicketID)
	assert.Equal(t, rule.JiraTicketKey, retrieved.JiraTicketKey)
	assert.Equal(t, rule.Recipients, retrieved.Recipients)
	assert.Equal(t, rule.CronSchedule, retrieved.CronSchedule)
	assert.Equal(t, rule.Status, retrieved.Status)
	assert.WithinDuration(t, rule.CreatedAt, retrieved.CreatedAt, time.Second)
}

func TestAutomationRepo_GetByUser_Success(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()
	userID := "user123"

	// Create multiple rules for the same user
	rule1 := &domain.AutomationRule{
		UserID:        userID,
		JiraTicketID:  "ticket1",
		JiraTicketKey: "PROJ-1",
		Recipients:    []string{"test1@example.com"},
		CronSchedule:  "0 9 * * 1",
		Status:        domain.AutomationStatusActive,
	}

	rule2 := &domain.AutomationRule{
		UserID:        userID,
		JiraTicketID:  "ticket2",
		JiraTicketKey: "PROJ-2",
		Recipients:    []string{"test2@example.com"},
		CronSchedule:  "0 10 * * 2",
		Status:        domain.AutomationStatusPaused,
	}

	err := repo.Create(ctx, rule1)
	require.NoError(t, err)

	err = repo.Create(ctx, rule2)
	require.NoError(t, err)

	// Get rules by user ID
	rules, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)

	assert.Len(t, rules, 2, "Should return both rules for the user")

	// Verify we got both rules
	ruleIDs := make(map[string]bool)
	for _, rule := range rules {
		ruleIDs[rule.ID] = true
		assert.Equal(t, userID, rule.UserID)
	}

	assert.Contains(t, ruleIDs, rule1.ID)
	assert.Contains(t, ruleIDs, rule2.ID)
}

func TestAutomationRepo_GetActive_OnlyActiveRules(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Create active rules
	activeRule1 := &domain.AutomationRule{
		UserID:        "user1",
		JiraTicketID:  "ticket1",
		JiraTicketKey: "PROJ-1",
		Recipients:    []string{"active1@example.com"},
		CronSchedule:  "0 9 * * 1",
		Status:        domain.AutomationStatusActive,
	}

	activeRule2 := &domain.AutomationRule{
		UserID:        "user2",
		JiraTicketID:  "ticket2",
		JiraTicketKey: "PROJ-2",
		Recipients:    []string{"active2@example.com"},
		CronSchedule:  "0 10 * * 2",
		Status:        domain.AutomationStatusActive,
	}

	// Create paused rules
	pausedRule := &domain.AutomationRule{
		UserID:        "user3",
		JiraTicketID:  "ticket3",
		JiraTicketKey: "PROJ-3",
		Recipients:    []string{"paused@example.com"},
		CronSchedule:  "0 11 * * 3",
		Status:        domain.AutomationStatusPaused,
	}

	err := repo.Create(ctx, activeRule1)
	require.NoError(t, err)

	err = repo.Create(ctx, pausedRule)
	require.NoError(t, err)

	err = repo.Create(ctx, activeRule2)
	require.NoError(t, err)

	// Get active rules only
	activeRules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)

	assert.Len(t, activeRules, 2, "Should only return active rules")

	// Verify all returned rules are active
	for _, rule := range activeRules {
		assert.Equal(t, domain.AutomationStatusActive, rule.Status, "All rules should be active")
	}
}

func TestAutomationRepo_Delete_RemovesFromIndex(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	rule := &domain.AutomationRule{
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * 1",
		Status:        domain.AutomationStatusActive,
	}

	// Create the rule
	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	// Verify it exists
	_, err = repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)

	// Verify it's in user index
	rules, err := repo.GetByUserID(ctx, rule.UserID)
	require.NoError(t, err)
	assert.Len(t, rules, 1)

	// Delete the rule
	err = repo.Delete(ctx, rule.ID)
	require.NoError(t, err)

	// Verify it's removed from main storage
	_, err = repo.GetByID(ctx, rule.ID)
	assert.Error(t, err, "Should not find deleted rule")

	// Verify it's removed from user index
	rules, err = repo.GetByUserID(ctx, rule.UserID)
	require.NoError(t, err)
	assert.Len(t, rules, 0, "User index should be empty after deletion")
}

func TestAutomationRepo_UpdateExistingRule(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Create initial rule
	rule := &domain.AutomationRule{
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * 1",
		Status:        domain.AutomationStatusActive,
	}

	err := repo.Create(ctx, rule)
	require.NoError(t, err)

	// Update the rule
	updatedRule := *rule // Copy the rule
	updatedRule.Recipients = []string{"updated@example.com", "new@example.com"}
	updatedRule.Status = domain.AutomationStatusPaused
	updatedRule.CronSchedule = "0 10 * * 2"

	err = repo.Update(ctx, &updatedRule)
	require.NoError(t, err)

	// Verify the update
	retrieved, err := repo.GetByID(ctx, rule.ID)
	require.NoError(t, err)

	assert.Equal(t, []string{"updated@example.com", "new@example.com"}, retrieved.Recipients)
	assert.Equal(t, domain.AutomationStatusPaused, retrieved.Status)
	assert.Equal(t, "0 10 * * 2", retrieved.CronSchedule)
	assert.Equal(t, rule.ID, retrieved.ID, "ID should not change")
	assert.Equal(t, rule.UserID, retrieved.UserID, "UserID should not change")
}

func TestAutomationRepo_UpdateNonExistentRule(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Try to update non-existent rule
	rule := &domain.AutomationRule{
		ID:            "nonexistent-id",
		UserID:        "user123",
		JiraTicketID:  "ticket456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * 1",
		Status:        domain.AutomationStatusActive,
	}

	err := repo.Update(ctx, rule)
	assert.Error(t, err, "Should return error for non-existent rule")
	assert.Contains(t, err.Error(), "not found")
}

func TestAutomationRepo_GetByUser_EmptyResult(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Try to get rules for user with no rules
	rules, err := repo.GetByUserID(ctx, "nonexistent-user")
	require.NoError(t, err)
	assert.Len(t, rules, 0, "Should return empty slice for user with no rules")
}

func TestAutomationRepo_GetActive_EmptyResult(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Get active rules when none exist
	rules, err := repo.GetActiveRules(ctx)
	require.NoError(t, err)
	assert.Len(t, rules, 0, "Should return empty slice when no active rules exist")
}

func TestAutomationRepo_GetByID_NotFound(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Try to get non-existent rule
	_, err := repo.GetByID(ctx, "nonexistent-id")
	assert.Error(t, err, "Should return error for non-existent rule")
	assert.Contains(t, err.Error(), "not found")
}

func TestAutomationRepo_Delete_NotFound(t *testing.T) {
	repo, s := setupTestAutomationRepository(t)
	defer s.Close()

	ctx := context.Background()

	// Try to delete non-existent rule
	err := repo.Delete(ctx, "nonexistent-id")
	assert.Error(t, err, "Should return error for non-existent rule")
}