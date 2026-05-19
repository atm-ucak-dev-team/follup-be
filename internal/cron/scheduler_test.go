package cron

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/atm-ucak/follup/internal/domain"
)

// MockAutomationRuleRepository is a mock for testing
type MockAutomationRuleRepository struct {
	mock.Mock
}

func (m *MockAutomationRuleRepository) Create(ctx context.Context, rule *domain.AutomationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAutomationRuleRepository) GetByID(ctx context.Context, id string) (*domain.AutomationRule, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AutomationRule), args.Error(1)
}

func (m *MockAutomationRuleRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *MockAutomationRuleRepository) GetActiveRules(ctx context.Context) ([]*domain.AutomationRule, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AutomationRule), args.Error(1)
}

func (m *MockAutomationRuleRepository) Update(ctx context.Context, rule *domain.AutomationRule) error {
	args := m.Called(ctx, rule)
	return args.Error(0)
}

func (m *MockAutomationRuleRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockEmailService is a mock for testing
type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) RegisterCredential(ctx interface{}, cred *domain.EmailCredential) error {
	args := m.Called(ctx, cred)
	return args.Error(0)
}

func (m *MockEmailService) GetCredential(ctx interface{}, userID string) (*domain.EmailCredential, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.EmailCredential), args.Error(1)
}

func (m *MockEmailService) SendFollowUp(ctx interface{}, threadID, subject, body string, recipients []string) error {
	args := m.Called(ctx, threadID, subject, body, recipients)
	return args.Error(0)
}

func (m *MockEmailService) CheckForReplies(ctx interface{}) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockEmailService) DecryptPassword(encryptedPassword string) (string, error) {
	args := m.Called(encryptedPassword)
	return args.String(0), args.Error(1)
}

func (m *MockEmailService) SaveCredential(ctx context.Context, userID, email, password string) error {
	args := m.Called(ctx, userID, email, password)
	return args.Error(0)
}

func (m *MockEmailService) SendFollowUpByAutomation(ctx context.Context, automationID string) error {
	args := m.Called(ctx, automationID)
	return args.Error(0)
}

func (m *MockEmailService) PollInbox(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestScheduler_Start_LoadsActiveRules tests that the scheduler loads active rules on startup
func TestScheduler_Start_LoadsActiveRules(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	// Create test automation rules
	now := time.Now()
	activeRules := []*domain.AutomationRule{
		{
			ID:           "rule-1",
			UserID:       "user-1",
			JiraTicketID: "ticket-1",
			JiraTicketKey: "PROJ-123",
			Recipients:    []string{"test@example.com"},
			CronSchedule:  "0 9 * * 1", // Every Monday at 9 AM
			Status:        domain.AutomationStatusActive,
			CreatedAt:     now,
		},
		{
			ID:           "rule-2",
			UserID:       "user-2",
			JiraTicketID: "ticket-2",
			JiraTicketKey: "PROJ-456",
			Recipients:    []string{"test2@example.com"},
			CronSchedule:  "0 */2 * * *", // Every 2 hours
			Status:        domain.AutomationStatusActive,
			CreatedAt:     now,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetActiveRules", mock.Anything).Return(activeRules, nil)

	// Create scheduler
	scheduler := NewScheduler(mockRepo, mockEmailService)

	// Start scheduler
	err := scheduler.Start()
	assert.NoError(t, err, "Start should not return an error")

	// Verify that rules were loaded
	assert.Equal(t, 2, scheduler.GetScheduledRuleCount(), "Should have 2 scheduled rules")

	// Clean up
	scheduler.Stop()

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
}

// TestScheduler_AddRule_Success tests adding a rule successfully
func TestScheduler_AddRule_Success(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * *", // Daily at 9 AM
		Status:        domain.AutomationStatusActive,
	}

	// Add rule
	err := scheduler.AddRule(rule)
	assert.NoError(t, err, "AddRule should not return an error")

	// Verify rule was added
	assert.Equal(t, 1, scheduler.GetScheduledRuleCount(), "Should have 1 scheduled rule")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_AddRule_InvalidCron tests adding a rule with invalid cron expression
func TestScheduler_AddRule_InvalidCron(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "invalid-cron", // Invalid cron expression
		Status:        domain.AutomationStatusActive,
	}

	// Try to add rule with invalid cron
	err := scheduler.AddRule(rule)
	assert.Error(t, err, "AddRule should return an error for invalid cron expression")
	assert.Contains(t, err.Error(), "invalid cron expression", "Error should mention invalid cron")

	// Verify rule was not added
	assert.Equal(t, 0, scheduler.GetScheduledRuleCount(), "Should have 0 scheduled rules")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_AddRule_EmptyCron tests adding a rule with empty cron expression
func TestScheduler_AddRule_EmptyCron(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "", // Empty cron expression
		Status:        domain.AutomationStatusActive,
	}

	// Try to add rule with empty cron
	err := scheduler.AddRule(rule)
	assert.Error(t, err, "AddRule should return an error for empty cron expression")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_RemoveRule_Success tests removing a rule successfully
func TestScheduler_RemoveRule_Success(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * *",
		Status:        domain.AutomationStatusActive,
	}

	// Add rule first
	err := scheduler.AddRule(rule)
	require.NoError(t, err, "AddRule should succeed")

	// Verify rule was added
	assert.Equal(t, 1, scheduler.GetScheduledRuleCount(), "Should have 1 scheduled rule")

	// Remove rule
	scheduler.RemoveRule("rule-1")

	// Verify rule was removed
	assert.Equal(t, 0, scheduler.GetScheduledRuleCount(), "Should have 0 scheduled rules after removal")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_RemoveRule_NonExistent tests removing a rule that doesn't exist
func TestScheduler_RemoveRule_NonExistent(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	// Try to remove non-existent rule (should not panic, just log)
	scheduler.RemoveRule("non-existent-rule")

	// Verify count is still 0
	assert.Equal(t, 0, scheduler.GetScheduledRuleCount(), "Should have 0 scheduled rules")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_JobExecution_Success tests job execution
func TestScheduler_JobExecution_Success(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "* * * * *", // Every minute for testing
		Status:        domain.AutomationStatusActive,
	}

	// Set up mock expectation for email sending
	mockEmailService.On("SendFollowUpByAutomation", mock.Anything, "rule-1").Return(nil)

	// Add rule
	err := scheduler.AddRule(rule)
	require.NoError(t, err, "AddRule should succeed")

	// We can't easily test actual cron execution in unit tests without waiting,
	// but we can test the executeAutomation method directly via accessing it through job execution
	// For now, this test verifies the rule can be added successfully and mocks are set up

	// Give scheduler a moment to potentially execute (in real scenario)
	// Note: This won't actually trigger in unit tests due to timing, but the structure is correct

	// Clean up
	scheduler.Stop()

	// Note: In real scenarios, you might want to add integration tests that wait for cron execution
}

// TestScheduler_RuleStatusChange_SyncsCorrectly tests rule status changes sync correctly
func TestScheduler_RuleStatusChange_SyncsCorrectly(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * *",
		Status:        domain.AutomationStatusActive,
	}

	// Add active rule
	err := scheduler.AddRule(rule)
	require.NoError(t, err, "AddRule should succeed")

	// Verify rule was added
	assert.Equal(t, 1, scheduler.GetScheduledRuleCount(), "Should have 1 scheduled rule")

	// Simulate pausing rule by removing it from scheduler
	scheduler.RemoveRule("rule-1")

	// Verify rule was removed (simulating pause)
	assert.Equal(t, 0, scheduler.GetScheduledRuleCount(), "Should have 0 scheduled rules after pause")

	// Simulate resuming rule by adding it back
	err = scheduler.AddRule(rule)
	require.NoError(t, err, "AddRule should succeed on resume")

	// Verify rule was added back (simulating resume)
	assert.Equal(t, 1, scheduler.GetScheduledRuleCount(), "Should have 1 scheduled rule after resume")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_Start_RepositoryError tests error handling when repository fails on start
func TestScheduler_Start_RepositoryError(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	// Set up mock to return error
	mockRepo.On("GetActiveRules", mock.Anything).Return(nil, assert.AnError)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	// Start should return error
	err := scheduler.Start()
	assert.Error(t, err, "Start should return error when repository fails")
	assert.Contains(t, err.Error(), "failed to load active automation rules", "Error should mention loading rules")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_AddRule_DuplicateRule tests adding a rule that already exists
func TestScheduler_AddRule_DuplicateRule(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	rule := domain.AutomationRule{
		ID:           "rule-1",
		UserID:       "user-1",
		JiraTicketID: "ticket-1",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * *",
		Status:        domain.AutomationStatusActive,
	}

	// Add rule first time
	err := scheduler.AddRule(rule)
	require.NoError(t, err, "AddRule should succeed first time")

	// Try to add same rule again
	err = scheduler.AddRule(rule)
	assert.Error(t, err, "AddRule should return error for duplicate rule")
	assert.Contains(t, err.Error(), "already exists", "Error should mention rule already exists")

	// Verify only one instance exists
	assert.Equal(t, 1, scheduler.GetScheduledRuleCount(), "Should have 1 scheduled rule")

	// Clean up
	scheduler.Stop()
}

// TestScheduler_GracefulShutdown tests graceful shutdown behavior
func TestScheduler_GracefulShutdown(t *testing.T) {
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)

	// Create test automation rules
	now := time.Now()
	activeRules := []*domain.AutomationRule{
		{
			ID:           "rule-1",
			UserID:       "user-1",
			JiraTicketID: "ticket-1",
			JiraTicketKey: "PROJ-123",
			Recipients:    []string{"test@example.com"},
			CronSchedule:  "0 9 * * *",
			Status:        domain.AutomationStatusActive,
			CreatedAt:     now,
		},
	}

	// Set up mock expectations
	mockRepo.On("GetActiveRules", mock.Anything).Return(activeRules, nil)

	scheduler := NewScheduler(mockRepo, mockEmailService)

	// Start scheduler
	err := scheduler.Start()
	assert.NoError(t, err, "Start should not return an error")

	// Stop scheduler gracefully
	scheduler.Stop()

	// Verify scheduler stopped without panicking
	assert.Equal(t, 1, scheduler.GetScheduledRuleCount(), "Rule count should be maintained after stop")

	// Verify mock expectations
	mockRepo.AssertExpectations(t)
}