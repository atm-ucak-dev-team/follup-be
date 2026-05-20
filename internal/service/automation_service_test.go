package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/atm-ucak/follup/internal/domain"
	service2 "github.com/atm-ucak/follup/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockEmailService) SendFollowUpByAutomation(ctx context.Context, automationID string) error {
	args := m.Called(ctx, automationID)
	return args.Error(0)
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

func (m *MockEmailService) PollInbox(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Helper function to create a valid test rule
func createValidTestRule() *domain.AutomationRule {
	return &domain.AutomationRule{
		ID:            "test-id",
		UserID:        "user-123",
		JiraTicketID:  "ticket-456",
		JiraTicketKey: "PROJ-123",
		Recipients:    []string{"test@example.com"},
		CronSchedule:  "0 9 * * 1", // Valid 5-part cron: Every Monday at 9 AM
		Status:        domain.AutomationStatusActive,
	}
}

func TestCreateRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	validRule := createValidTestRule()
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *domain.AutomationRule) bool {
		return r.ID == validRule.ID && r.UserID == validRule.UserID
	})).Return(nil)

	// Execute
	err := automationService.CreateRule(context.Background(), validRule)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCreateRule_InvalidCron(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	invalidRule := createValidTestRule()
	invalidRule.CronSchedule = "invalid-cron"

	// Execute
	err := automationService.CreateRule(context.Background(), invalidRule)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreateRule_InvalidRecipients(t *testing.T) {
	tests := []struct {
		name        string
		recipients  []string
		expectedErr string
	}{
		{
			name:        "No recipients",
			recipients:  []string{},
			expectedErr: "at least one recipient is required",
		},
		{
			name:        "Invalid email format",
			recipients:  []string{"not-an-email"},
			expectedErr: "invalid email address",
		},
		{
			name:        "Empty email",
			recipients:  []string{""},
			expectedErr: "recipient email cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockAutomationRuleRepository)
			mockEmailService := new(MockEmailService)
			automationService := service2.NewAutomationService(mockRepo, mockEmailService)

			invalidRule := createValidTestRule()
			invalidRule.Recipients = tt.recipients

			// Execute
			err := automationService.CreateRule(context.Background(), invalidRule)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
			mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}
}

func TestCreateRule_MissingJiraTicketID(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	invalidRule := createValidTestRule()
	invalidRule.JiraTicketID = ""

	// Execute
	err := automationService.CreateRule(context.Background(), invalidRule)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Jira ticket ID is required")
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreateRule_InvalidStatus(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	invalidRule := createValidTestRule()
	invalidRule.Status = "invalid-status"

	// Execute
	err := automationService.CreateRule(context.Background(), invalidRule)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status must be 'active' or 'paused'")
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreateRule_RepoFails(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	validRule := createValidTestRule()
	mockRepo.On("Create", mock.Anything, validRule).Return(errors.New("database error"))

	// Execute
	err := automationService.CreateRule(context.Background(), validRule)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create automation rule")
	mockRepo.AssertExpectations(t)
}

func TestGetRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	expectedRule := createValidTestRule()
	mockRepo.On("GetByID", mock.Anything, "test-id").Return(expectedRule, nil)

	// Execute
	rule, err := automationService.GetRule(context.Background(), "test-id")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedRule, rule)
	mockRepo.AssertExpectations(t)
}

func TestGetRule_NotFound(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("not found"))

	// Execute
	rule, err := automationService.GetRule(context.Background(), "non-existent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
	mockRepo.AssertExpectations(t)
}

func TestGetRule_EmptyID(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	// Execute
	rule, err := automationService.GetRule(context.Background(), "")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, rule)
	assert.Contains(t, err.Error(), "automation ID cannot be empty")
	mockRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestGetUserRules_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	expectedRules := []*domain.AutomationRule{
		createValidTestRule(),
		{
			ID:            "test-id-2",
			UserID:        "user-123",
			JiraTicketID:  "ticket-789",
			JiraTicketKey: "PROJ-456",
			Recipients:    []string{"another@example.com"},
			CronSchedule:  "0 17 * * 5", // Every Friday at 5 PM
			Status:        domain.AutomationStatusActive,
		},
	}

	mockRepo.On("GetByUserID", mock.Anything, "user-123").Return(expectedRules, nil)

	// Execute
	rules, err := automationService.GetUserRules(context.Background(), "user-123")

	// Assert
	assert.NoError(t, err)
	assert.Len(t, rules, 2)
	assert.Equal(t, expectedRules, rules)
	mockRepo.AssertExpectations(t)
}

func TestUpdateRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()
	updatedRule := createValidTestRule()
	updatedRule.Recipients = []string{"updated@example.com"}

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *domain.AutomationRule) bool {
		return r.Recipients[0] == "updated@example.com"
	})).Return(nil)

	// Execute
	err := automationService.UpdateRule(context.Background(), updatedRule)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateRule_NotOwnedByUser(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()
	existingRule.UserID = "original-user"

	updatedRule := createValidTestRule()
	updatedRule.UserID = "different-user" // Different user trying to update

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)

	// Execute
	err := automationService.UpdateRule(context.Background(), updatedRule)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user does not own this automation rule")
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateRule_InvalidCron(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()
	updatedRule := createValidTestRule()
	updatedRule.CronSchedule = "invalid-cron"

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)

	// Execute
	err := automationService.UpdateRule(context.Background(), updatedRule)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestDeleteRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()
	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)
	mockRepo.On("Delete", mock.Anything, "test-id").Return(nil)

	// Execute
	err := automationService.DeleteRule(context.Background(), "test-id")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteRule_NotFound(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("not found"))

	// Execute
	err := automationService.DeleteRule(context.Background(), "non-existent")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get automation rule")
	mockRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestPauseRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()
	existingRule.Status = domain.AutomationStatusActive

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *domain.AutomationRule) bool {
		return r.Status == domain.AutomationStatusPaused
	})).Return(nil)

	// Execute
	err := automationService.PauseRule(context.Background(), "test-id")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestResumeRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()
	existingRule.Status = domain.AutomationStatusPaused

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(r *domain.AutomationRule) bool {
		return r.Status == domain.AutomationStatusActive
	})).Return(nil)

	// Execute
	err := automationService.ResumeRule(context.Background(), "test-id")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGetActiveRules_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	activeRules := []*domain.AutomationRule{
		createValidTestRule(),
		{
			ID:            "test-id-2",
			UserID:        "user-456",
			JiraTicketID:  "ticket-789",
			JiraTicketKey: "PROJ-456",
			Recipients:    []string{"active@example.com"},
			CronSchedule:  "0 12 * * *",
			Status:        domain.AutomationStatusActive,
		},
	}

	mockRepo.On("GetActiveRules", mock.Anything).Return(activeRules, nil)

	// Execute
	rules, err := automationService.GetActiveRules(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.Len(t, rules, 2)
	mockRepo.AssertExpectations(t)
}

func TestTriggerRule_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)
	mockEmailService.On("SendFollowUpByAutomation", mock.Anything, "test-id").Return(nil)

	// Execute
	err := automationService.TriggerRule(context.Background(), "test-id")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}

func TestTriggerRule_EmailServiceFails(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	existingRule := createValidTestRule()

	mockRepo.On("GetByID", mock.Anything, "test-id").Return(existingRule, nil)
	mockEmailService.On("SendFollowUpByAutomation", mock.Anything, "test-id").Return(errors.New("email service error"))

	// Execute
	err := automationService.TriggerRule(context.Background(), "test-id")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to trigger automation rule")
	mockRepo.AssertExpectations(t)
	mockEmailService.AssertExpectations(t)
}

func TestTriggerRule_NotFound(t *testing.T) {
	// Setup
	mockRepo := new(MockAutomationRuleRepository)
	mockEmailService := new(MockEmailService)
	automationService := service2.NewAutomationService(mockRepo, mockEmailService)

	mockRepo.On("GetByID", mock.Anything, "non-existent").Return(nil, errors.New("not found"))

	// Execute
	err := automationService.TriggerRule(context.Background(), "non-existent")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get automation rule")
	mockRepo.AssertNotCalled(t, "SendFollowUpByAutomation", mock.Anything, mock.Anything)
}
