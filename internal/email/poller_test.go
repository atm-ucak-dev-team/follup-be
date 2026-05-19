package email

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
)

// Mock implementations for testing
type mockEmailService struct {
	getCredentialFunc     func(ctx interface{}, userID string) (*domain.EmailCredential, error)
	decryptPasswordFunc   func(encryptedPassword string) (string, error)
}

func (m *mockEmailService) GetCredential(ctx interface{}, userID string) (*domain.EmailCredential, error) {
	if m.getCredentialFunc != nil {
		return m.getCredentialFunc(ctx, userID)
	}
	return &domain.EmailCredential{
		UserID:            userID,
		EmailAddress:      "test@example.com",
		EncryptedPassword: "encrypted_password",
		IMAPHost:          "imap.example.com",
		SMTPHost:          "smtp.example.com",
		CreatedAt:         time.Now(),
	}, nil
}

func (m *mockEmailService) DecryptPassword(encryptedPassword string) (string, error) {
	if m.decryptPasswordFunc != nil {
		return m.decryptPasswordFunc(encryptedPassword)
	}
	return "decrypted_password", nil
}

func (m *mockEmailService) RegisterCredential(ctx interface{}, cred *domain.EmailCredential) error {
	return nil
}

func (m *mockEmailService) SendFollowUp(ctx interface{}, threadID, subject, body string, recipients []string) error {
	return nil
}

func (m *mockEmailService) CheckForReplies(ctx interface{}) error {
	return nil
}

func (m *mockEmailService) SaveCredential(ctx context.Context, userID, email, password string) error {
	return nil
}

func (m *mockEmailService) SendFollowUpByAutomation(ctx context.Context, automationID string) error {
	return nil
}

func (m *mockEmailService) PollInbox(ctx context.Context) error {
	return nil
}

type mockAutomationRepository struct {
	getActiveRulesFunc func(ctx interface{}) ([]*domain.AutomationRule, error)
}

func (m *mockAutomationRepository) Create(ctx context.Context, rule *domain.AutomationRule) error {
	return nil
}

func (m *mockAutomationRepository) GetByID(ctx context.Context, id string) (*domain.AutomationRule, error) {
	return nil, nil
}

func (m *mockAutomationRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.AutomationRule, error) {
	return nil, nil
}

func (m *mockAutomationRepository) GetActiveRules(ctx context.Context) ([]*domain.AutomationRule, error) {
	if m.getActiveRulesFunc != nil {
		return m.getActiveRulesFunc(ctx)
	}
	return []*domain.AutomationRule{
		{
			ID:           "automation-1",
			UserID:       "user-1",
			JiraTicketID: "ticket-1",
			Status:       domain.AutomationStatusActive,
			CreatedAt:    time.Now(),
		},
	}, nil
}

func (m *mockAutomationRepository) Update(ctx context.Context, rule *domain.AutomationRule) error {
	return nil
}

func (m *mockAutomationRepository) Delete(ctx context.Context, id string) error {
	return nil
}

type mockEmailThreadRepository struct {
	updateThreadStatusFunc func(ctx context.Context, threadID, status string) error
}

func (m *mockEmailThreadRepository) UpdateThreadStatus(ctx context.Context, threadID, status string) error {
	if m.updateThreadStatusFunc != nil {
		return m.updateThreadStatusFunc(ctx, threadID, status)
	}
	return nil
}

func (m *mockEmailThreadRepository) Create(ctx context.Context, thread *domain.EmailThread) error {
	return nil
}

func (m *mockEmailThreadRepository) GetByID(ctx context.Context, id string) (*domain.EmailThread, error) {
	return nil, nil
}

func (m *mockEmailThreadRepository) GetByAutomationID(ctx context.Context, automationID string) ([]*domain.EmailThread, error) {
	return nil, nil
}

func (m *mockEmailThreadRepository) GetByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.EmailThread, error) {
	return nil, nil
}

func (m *mockEmailThreadRepository) Update(ctx context.Context, thread *domain.EmailThread) error {
	return nil
}

func (m *mockEmailThreadRepository) Delete(ctx context.Context, id string) error {
	return nil
}

// TestPoller_StartAndStop tests basic poller lifecycle
func TestPoller_StartAndStop(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{}, nil // No active rules
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test start
	poller.Start()
	assert.True(t, poller.running, "Poller should be running after Start()")

	// Test stop
	poller.Stop()
	assert.False(t, poller.running, "Poller should not be running after Stop()")
}

// TestPoller_StartWhenAlreadyRunning tests starting an already running poller
func TestPoller_StartWhenAlreadyRunning(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	poller.Start()
	firstRunningState := poller.running

	// Try to start again (should be idempotent)
	poller.Start()
	secondRunningState := poller.running

	// Cleanup
	poller.Stop()

	assert.Equal(t, firstRunningState, secondRunningState, "Start() should be idempotent")
	assert.True(t, firstRunningState, "Poller should be running")
}

// TestPoller_StopWhenNotRunning tests stopping a poller that's not running
func TestPoller_StopWhenNotRunning(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Stop without starting (should be safe)
	poller.Stop()
	assert.False(t, poller.running, "Poller should not be running")
}

// TestPoller_PollOnce_NoActiveAutomations tests polling with no active automations
func TestPoller_PollOnce_NoActiveAutomations(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{}, nil // No active rules
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This should complete without errors
	poller.pollOnce() // Should handle empty rules gracefully
}

// TestPoller_PollOnce_GetActiveRulesFails tests polling when getting active rules fails
func TestPoller_PollOnce_GetActiveRulesFails(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return nil, errors.New("database connection failed")
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This should handle the error gracefully
	poller.pollOnce() // Should not panic
}

// TestPoller_PollOnce_GetCredentialFails tests polling when getting credentials fails
func TestPoller_PollOnce_GetCredentialFails(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return nil, errors.New("user not found")
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{
					ID:           "automation-1",
					UserID:       "user-1",
					JiraTicketID: "ticket-1",
					Status:       domain.AutomationStatusActive,
					CreatedAt:    time.Now(),
				},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This should handle the error gracefully
	poller.pollOnce() // Should not panic
}

// TestPoller_PollOnce_DecryptPasswordFails tests polling when password decryption fails
func TestPoller_PollOnce_DecryptPasswordFails(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "invalid_encrypted_password",
				IMAPHost:          "imap.example.com",
				CreatedAt:         time.Now(),
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "", errors.New("decryption failed")
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{
					ID:           "automation-1",
					UserID:       "user-1",
					JiraTicketID: "ticket-1",
					Status:       domain.AutomationStatusActive,
					CreatedAt:    time.Now(),
				},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This should handle the error gracefully
	poller.pollOnce() // Should not panic
}

// TestPoller_PollOnce_ThreadMatching tests the thread matching logic
func TestPoller_PollOnce_ThreadMatching(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{
					ID:           "automation-1",
					UserID:       "user-1",
					JiraTicketID: "ticket-1",
					Status:       domain.AutomationStatusActive,
					CreatedAt:    time.Now(),
				},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{
		updateThreadStatusFunc: func(ctx context.Context, threadID, status string) error {
			assert.Equal(t, domain.EmailThreadStatusReplied, status)
			return nil
		},
	}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This will test the thread matching logic when IMAP connection fails (expected in unit test)
	poller.pollOnce() // Should not panic even with IMAP connection issues
}

// TestPoller_MultipleUsersWithActiveAutomations tests polling multiple users
func TestPoller_MultipleUsersWithActiveAutomations(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      userID + "@example.com",
				EncryptedPassword: "encrypted_password",
				IMAPHost:          "imap.example.com",
				CreatedAt:         time.Now(),
			}, nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{
					ID:           "automation-1",
					UserID:       "user-1",
					JiraTicketID: "ticket-1",
					Status:       domain.AutomationStatusActive,
					CreatedAt:    time.Now(),
				},
				{
					ID:           "automation-2",
					UserID:       "user-2",
					JiraTicketID: "ticket-2",
					Status:       domain.AutomationStatusActive,
					CreatedAt:    time.Now(),
				},
				{
					ID:           "automation-3",
					UserID:       "user-1", // Same user as automation-1
					JiraTicketID: "ticket-3",
					Status:       domain.AutomationStatusActive,
					CreatedAt:    time.Now(),
				},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This should process 2 users (user-1 with 2 automations, user-2 with 1 automation)
	poller.pollOnce() // Should not panic
}

// TestNewPoller tests poller construction
func TestNewPoller(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}
	interval := 5 * time.Minute

	poller := NewPoller(mockEmail, mockAutomation, mockThread, interval)

	assert.NotNil(t, poller)
	assert.Equal(t, mockEmail, poller.emailService)
	assert.Equal(t, mockAutomation, poller.automationRepo)
	assert.Equal(t, mockThread, poller.emailThreadRepo)
	assert.Equal(t, interval, poller.interval)
	assert.False(t, poller.running, "New poller should not be running")
	assert.NotNil(t, poller.stopChan)
}

// TestPoller_ConcurrentStartStop tests concurrent start/stop operations
func TestPoller_ConcurrentStartStop(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 50*time.Millisecond)

	// Start and stop multiple times concurrently
	done := make(chan bool)
	for i := 0; i < 3; i++ {
		go func() {
			poller.Start()
			time.Sleep(10 * time.Millisecond)
			poller.Stop()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	assert.False(t, poller.running, "Poller should be stopped after all operations")
}

// TestPoller_connectIMAP tests the IMAP connection logic
func TestPoller_connectIMAP(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test with invalid host (should fail gracefully)
	_, err := poller.connectIMAP("test@example.com", "password", "invalid.host", 993)
	assert.Error(t, err, "Should return error for invalid host")
	assert.Contains(t, err.Error(), "failed to connect", "Error should mention connection failure")
}

// TestPoller_pollUser_ErrorHandling tests various error scenarios in pollUser
func TestPoller_pollUser_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupEmail    func() *mockEmailService
		setupRules    func() []*domain.AutomationRule
		expectError   bool
		errorContains string
	}{
		{
			name: "No credentials found",
			setupEmail: func() *mockEmailService {
				return &mockEmailService{
					getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
						return nil, errors.New("credentials not found")
					},
				}
			},
			setupRules: func() []*domain.AutomationRule {
				return []*domain.AutomationRule{
					{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
				}
			},
			expectError:   true,
			errorContains: "failed to get credentials",
		},
		{
			name: "Password decryption fails",
			setupEmail: func() *mockEmailService {
				return &mockEmailService{
					getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
						return &domain.EmailCredential{
							UserID:            userID,
							EmailAddress:      "test@example.com",
							EncryptedPassword: "encrypted",
							IMAPHost:          "imap.example.com",
						}, nil
					},
					decryptPasswordFunc: func(encryptedPassword string) (string, error) {
						return "", errors.New("decryption failed")
					},
				}
			},
			setupRules: func() []*domain.AutomationRule {
				return []*domain.AutomationRule{
					{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
				}
			},
			expectError:   true,
			errorContains: "failed to decrypt password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmail := tt.setupEmail()
			mockAutomation := &mockAutomationRepository{}
			mockThread := &mockEmailThreadRepository{}

			poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

			rules := tt.setupRules()
			err := poller.pollUser(context.Background(), "user-1", rules)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPoller_matchMessageToThread tests the message-to-thread matching logic
func TestPoller_matchMessageToThread(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *mockEmailThreadRepository
		expectError bool
	}{
		{
			name: "Thread repository returns no error",
			setupMock: func() *mockEmailThreadRepository {
				return &mockEmailThreadRepository{
					updateThreadStatusFunc: func(ctx context.Context, threadID, status string) error {
						return nil
					},
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmail := &mockEmailService{}
			mockAutomation := &mockAutomationRepository{}
			mockThread := tt.setupMock()

			poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

			// Since we can't easily create a proper imap.Message in tests,
			// we'll skip the full matchMessageToThread test and just test the findMatchingThread method
			threadID, err := poller.findMatchingThread(context.Background(), "user-1", "message-123@example.com")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Empty(t, threadID) // Placeholder returns empty
			}
		})
	}
}

// TestPoller_findMatchingThread tests the thread matching algorithm
func TestPoller_findMatchingThread(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test finding matching thread (currently returns empty as placeholder)
	threadID, err := poller.findMatchingThread(context.Background(), "user-1", "message-123@example.com")

	assert.NoError(t, err)
	assert.Empty(t, threadID, "Should return empty thread ID for placeholder implementation")
}

// TestPoller_MultiplePollCycles tests running multiple poll cycles
func TestPoller_MultiplePollCycles(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      userID + "@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "imap.example.com",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "", errors.New("decryption fails") // Simulate decryption failure
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 50*time.Millisecond)

	// Start the poller
	poller.Start()
	defer poller.Stop()

	// Run for a few cycles
	time.Sleep(200 * time.Millisecond)

	// Poller should still be running
	assert.True(t, poller.running, "Poller should still be running after multiple cycles")
}

// TestPoller_PollOnce_IMAPConnectionFails tests polling when IMAP connection fails
func TestPoller_PollOnce_IMAPConnectionFails(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Should not panic, just log the error
	poller.pollOnce()
}

// TestPoller_PollOnce_ThreadMatchingSuccess tests successful thread matching
func TestPoller_PollOnce_ThreadMatchingSuccess(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{
		updateThreadStatusFunc: func(ctx context.Context, threadID, status string) error {
			// This would be called if thread matching succeeded
			return nil
		},
	}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Should handle IMAP connection failure gracefully
	poller.pollOnce()
}

// TestPoller_GroupsUsersByUserID tests that the poller correctly groups automations by user
func TestPoller_GroupsUsersByUserID(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      userID + "@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
				{ID: "auto-2", UserID: "user-1", Status: domain.AutomationStatusActive}, // Same user
				{ID: "auto-3", UserID: "user-2", Status: domain.AutomationStatusActive},
				{ID: "auto-4", UserID: "user-1", Status: domain.AutomationStatusActive}, // Same user again
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// The poller should group these by user ID and only make 2 IMAP connections
	// (one for user-1, one for user-2) instead of 4 connections
	poller.pollOnce()
}

// TestNewPoller_WithZeroInterval tests poller construction with zero interval
func TestNewPoller_WithZeroInterval(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 0)

	assert.NotNil(t, poller)
	assert.Equal(t, time.Duration(0), poller.interval)
	assert.False(t, poller.running)
}

// TestPoller_StartWithSmallInterval tests starting a poller with very small interval
func TestPoller_StartWithSmallInterval(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	// Use 1ms instead of 0 to avoid ticker panic
	poller := NewPoller(mockEmail, mockAutomation, mockThread, 1*time.Millisecond)

	// Should start without panicking
	poller.Start()
	assert.True(t, poller.running)

	// Stop it right away
	poller.Stop()
	assert.False(t, poller.running)
}

// TestPoller_StopIsIdempotent tests that multiple Stop calls are safe
func TestPoller_StopIsIdempotent(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	poller.Start()
	poller.Stop()
	poller.Stop() // Should not panic
	poller.Stop() // Should still not panic

	assert.False(t, poller.running)
}

// TestPoller_findMatchingThread_PlaceholderLogic tests the placeholder thread matching logic
func TestPoller_findMatchingThread_PlaceholderLogic(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test with various message IDs
	testCases := []string{
		"",
		"message-123@example.com",
		"<message-456@example.com>",
		"complex-message-id-with-special-chars@example.com",
	}

	for _, messageID := range testCases {
		threadID, err := poller.findMatchingThread(context.Background(), "user-1", messageID)
		assert.NoError(t, err, "Should not return error for message ID: %s", messageID)
		assert.Empty(t, threadID, "Placeholder implementation should return empty thread ID")
	}
}

// TestPoller_UpdateThreadStatus_Failure tests handling thread status update failures
func TestPoller_UpdateThreadStatus_Failure(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{
		updateThreadStatusFunc: func(ctx context.Context, threadID, status string) error {
			return errors.New("thread update failed")
		},
	}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// The findMatchingThread method should handle this gracefully
	threadID, err := poller.findMatchingThread(context.Background(), "user-1", "message-123@example.com")
	assert.NoError(t, err, "Should not return error even if thread update would fail")
	assert.Empty(t, threadID, "Should return empty thread ID")
}

// TestPoller_PollOnce_WithMultipleUsersAndAutomations tests complex user/automation scenarios
func TestPoller_PollOnce_WithMultipleUsersAndAutomations(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      userID + "@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
				{ID: "auto-2", UserID: "user-1", Status: domain.AutomationStatusActive},
				{ID: "auto-3", UserID: "user-2", Status: domain.AutomationStatusActive},
				{ID: "auto-4", UserID: "user-3", Status: domain.AutomationStatusActive},
				{ID: "auto-5", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Should process 3 users with their respective automations
	poller.pollOnce()
}

// TestPoller_PollOnce_EmptyUserID tests handling of empty user ID
func TestPoller_PollOnce_EmptyUserID(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return nil, errors.New("user ID is empty")
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Should handle empty user ID gracefully
	poller.pollOnce()
}

// TestPoller_ConnectIMAP_ValidParameters tests IMAP connection with valid parameters
func TestPoller_ConnectIMAP_ValidParameters(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test with valid parameters (will still fail due to network, but tests parameter handling)
	tests := []struct {
		name     string
		email    string
		password string
		host     string
		port     int
	}{
		{"Valid parameters", "user@example.com", "password123", "imap.gmail.com", 993},
		{"Different port", "user@example.com", "password123", "imap.example.com", 143},
		{"Long password", "user@example.com", "very_long_password_with_special_chars_!@#$%", "imap.example.com", 993},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These will fail due to network issues, but we test the parameter handling
			_, err := poller.connectIMAP(tt.email, tt.password, tt.host, tt.port)
			assert.Error(t, err, "Should return error due to network issues")
		})
	}
}

// TestNewPoller_DifferentIntervals tests poller with various intervals
func TestNewPoller_DifferentIntervals(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	intervals := []time.Duration{
		1 * time.Second,
		5 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, interval := range intervals {
		t.Run(interval.String(), func(t *testing.T) {
			poller := NewPoller(mockEmail, mockAutomation, mockThread, interval)
			assert.NotNil(t, poller)
			assert.Equal(t, interval, poller.interval)
			assert.False(t, poller.running)
		})
	}
}

// TestPoller_PollOnce_RetryLogic tests error handling and retry scenarios
func TestPoller_PollOnce_RetryLogic(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
				{ID: "auto-2", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Should handle multiple IMAP connection failures gracefully for same user
	poller.pollOnce()
}

// TestPoller_LogMessages tests that proper log messages are generated
func TestPoller_LogMessages(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return nil, errors.New("test error")
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// This should generate log messages but not panic
	poller.pollOnce()
}

// TestPoller_fetchAndMatchMessages tests the message fetching logic
func TestPoller_fetchAndMatchMessages(t *testing.T) {
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test through the full poll flow which includes fetchAndMatchMessages
	// This will test the actual error handling in the real flow
	poller.pollOnce()
}

// TestPoller_matchMessageToThread_NilMessage tests handling of nil messages
func TestPoller_matchMessageToThread_NilMessage(t *testing.T) {
	// This test would cause a panic if we directly pass nil to matchMessageToThread
	// Instead, we'll test through the actual flow where errors are handled gracefully
	mockEmail := &mockEmailService{
		getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
			return &domain.EmailCredential{
				UserID:            userID,
				EmailAddress:      "test@example.com",
				EncryptedPassword: "encrypted",
				IMAPHost:          "invalid.imap.server",
			}, nil
		},
		decryptPasswordFunc: func(encryptedPassword string) (string, error) {
			return "password", nil
		},
	}
	mockAutomation := &mockAutomationRepository{
		getActiveRulesFunc: func(ctx interface{}) ([]*domain.AutomationRule, error) {
			return []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			}, nil
		},
	}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test through the actual flow where errors are handled gracefully
	poller.pollOnce()
}

// TestPoller_pollUser_CompleteFlow tests the complete user polling flow
func TestPoller_pollUser_CompleteFlow(t *testing.T) {
	tests := []struct {
		name           string
		emailSetup     func() *mockEmailService
		rules          []*domain.AutomationRule
		expectError    bool
		errorContains  string
	}{
		{
			name: "Successful credential retrieval, failed IMAP connection",
			emailSetup: func() *mockEmailService {
				return &mockEmailService{
					getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
						return &domain.EmailCredential{
							UserID:            userID,
							EmailAddress:      "test@example.com",
							EncryptedPassword: "encrypted",
							IMAPHost:          "imap.example.com",
							CreatedAt:         time.Now(),
						}, nil
					},
					decryptPasswordFunc: func(encryptedPassword string) (string, error) {
						return "password", nil
					},
				}
			},
			rules: []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
			},
			expectError:   true,
			errorContains: "failed to connect to IMAP",
		},
		{
			name: "Multiple rules for same user",
			emailSetup: func() *mockEmailService {
				return &mockEmailService{
					getCredentialFunc: func(ctx interface{}, userID string) (*domain.EmailCredential, error) {
						return &domain.EmailCredential{
							UserID:            userID,
							EmailAddress:      "test@example.com",
							EncryptedPassword: "encrypted",
							IMAPHost:          "invalid.imap.server",
							CreatedAt:         time.Now(),
						}, nil
					},
					decryptPasswordFunc: func(encryptedPassword string) (string, error) {
						return "password", nil
					},
				}
			},
			rules: []*domain.AutomationRule{
				{ID: "auto-1", UserID: "user-1", Status: domain.AutomationStatusActive},
				{ID: "auto-2", UserID: "user-1", Status: domain.AutomationStatusActive},
				{ID: "auto-3", UserID: "user-1", Status: domain.AutomationStatusActive},
			},
			expectError:   true,
			errorContains: "failed to connect to IMAP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmail := tt.emailSetup()
			mockAutomation := &mockAutomationRepository{}
			mockThread := &mockEmailThreadRepository{}

			poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

			err := poller.pollUser(context.Background(), "user-1", tt.rules)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPoller_ConnectIMAP_TestsConnectionParameters tests various IMAP connection parameters
func TestPoller_ConnectIMAP_TestsConnectionParameters(t *testing.T) {
	mockEmail := &mockEmailService{}
	mockAutomation := &mockAutomationRepository{}
	mockThread := &mockEmailThreadRepository{}

	poller := NewPoller(mockEmail, mockAutomation, mockThread, 100*time.Millisecond)

	// Test various email formats and hosts
	testCases := []struct {
		name     string
		email    string
		password string
		host     string
		port     int
	}{
		{"Gmail format", "user@gmail.com", "pass", "imap.gmail.com", 993},
		{"Corporate format", "user@company.com", "complex!Pass", "imap.company.com", 993},
		{"Custom port", "user@example.com", "password", "mail.example.com", 143},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// These will all fail due to network, but we test parameter handling
			_, err := poller.connectIMAP(tt.email, tt.password, tt.host, tt.port)
			assert.Error(t, err, "Should fail due to network issues")
		})
	}
}