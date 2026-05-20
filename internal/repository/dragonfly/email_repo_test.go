package dragonfly

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/atm-ucak/follup/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// setupTestDB creates a miniredis server for testing
func setupTestDB(t *testing.T) (*miniredis.Miniredis, *EmailRepository) {
	t.Helper()

	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	repo := NewEmailRepository(client)

	return s, repo
}

func TestEmailRepo_SaveAndGetCredential_Success(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	cred := domain.EmailCredential{
		UserID:            "user123",
		EmailAddress:      "test@example.com",
		EncryptedPassword: "encrypted_password_here",
		IMAPHost:          "imap.gmail.com",
		SMTPHost:          "smtp.gmail.com",
		CreatedAt:         time.Now(),
	}

	// Save credential
	err := repo.SaveCredential(ctx, cred)
	if err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}

	// Get credential
	retrieved, err := repo.GetCredential(ctx, "user123")
	if err != nil {
		t.Fatalf("failed to get credential: %v", err)
	}

	// Verify
	if retrieved.UserID != cred.UserID {
		t.Errorf("expected UserID %s, got %s", cred.UserID, retrieved.UserID)
	}
	if retrieved.EmailAddress != cred.EmailAddress {
		t.Errorf("expected EmailAddress %s, got %s", cred.EmailAddress, retrieved.EmailAddress)
	}
	if retrieved.EncryptedPassword != cred.EncryptedPassword {
		t.Errorf("expected EncryptedPassword %s, got %s", cred.EncryptedPassword, retrieved.EncryptedPassword)
	}
}

func TestEmailRepo_SaveOAuthToken_WithTTL(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	expiresAt := time.Now().Add(1 * time.Hour)
	token := domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		ExpiresAt:    expiresAt,
	}

	// Save OAuth token
	err := repo.SaveOAuthToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to save OAuth token: %v", err)
	}

	// Verify token exists
	retrieved, err := repo.GetOAuthToken(ctx, "user123", "jira")
	if err != nil {
		t.Fatalf("failed to get OAuth token: %v", err)
	}

	if retrieved.UserID != token.UserID {
		t.Errorf("expected UserID %s, got %s", token.UserID, retrieved.UserID)
	}
	if retrieved.Provider != token.Provider {
		t.Errorf("expected Provider %s, got %s", token.Provider, retrieved.Provider)
	}
	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("expected AccessToken %s, got %s", token.AccessToken, retrieved.AccessToken)
	}
}

func TestEmailRepo_OAuthTokenExpires(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()

	// Test with token that expires very soon (less than 5 minutes)
	expiresAt := time.Now().Add(2 * time.Minute)
	token := domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		ExpiresAt:    expiresAt,
	}

	// Should fail because TTL would be negative
	err := repo.SaveOAuthToken(ctx, token)
	if err == nil {
		t.Error("expected error when saving token with insufficient TTL, got nil")
	}

	// Test that token actually expires after TTL
	expiresAt = time.Now().Add(1 * time.Hour)
	token.ExpiresAt = expiresAt

	err = repo.SaveOAuthToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to save OAuth token: %v", err)
	}

	// Fast forward time past TTL
	s.FastForward(1 * time.Hour)

	// Token should no longer exist
	_, err = repo.GetOAuthToken(ctx, "user123", "jira")
	if err == nil {
		t.Error("expected error when getting expired token, got nil")
	}
}

func TestEmailRepo_SaveThread_Success(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	thread := domain.EmailThread{
		UserID:        "user123",
		AutomationID:  "automation456",
		GmailThreadID: "gmail_thread_789",
		TicketID:      "ticket-123",
		Status:        domain.EmailThreadStatusOpen,
		LastSyncedAt:  time.Now(),
	}

	// Save thread (ID should be auto-generated)
	err := repo.SaveThread(ctx, &thread)
	if err != nil {
		t.Fatalf("failed to save thread: %v", err)
	}

	// Verify thread was saved
	if thread.ID == "" {
		t.Error("expected thread ID to be generated, got empty string")
	}

	// Get thread
	retrieved, err := repo.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	if retrieved.UserID != thread.UserID {
		t.Errorf("expected UserID %s, got %s", thread.UserID, retrieved.UserID)
	}
	if retrieved.AutomationID != thread.AutomationID {
		t.Errorf("expected AutomationID %s, got %s", thread.AutomationID, retrieved.AutomationID)
	}
	if retrieved.Status != thread.Status {
		t.Errorf("expected Status %s, got %s", thread.Status, retrieved.Status)
	}
}

func TestEmailRepo_GetThreadsByAutomation_Success(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	automationID := "automation456"

	// Create multiple threads for the same automation
	threads := []domain.EmailThread{
		{
			UserID:        "user123",
			AutomationID:  automationID,
			GmailThreadID: "gmail_thread_1",
			TicketID:      "ticket-1",
			Status:        domain.EmailThreadStatusOpen,
			LastSyncedAt:  time.Now(),
		},
		{
			UserID:        "user123",
			AutomationID:  automationID,
			GmailThreadID: "gmail_thread_2",
			TicketID:      "ticket-2",
			Status:        domain.EmailThreadStatusOpen,
			LastSyncedAt:  time.Now(),
		},
	}

	// Save threads
	for i := range threads {
		err := repo.SaveThread(ctx, &threads[i])
		if err != nil {
			t.Fatalf("failed to save thread %d: %v", i, err)
		}
	}

	// Get threads by automation
	retrieved, err := repo.GetThreadsByAutomation(ctx, automationID)
	if err != nil {
		t.Fatalf("failed to get threads by automation: %v", err)
	}

	if len(retrieved) != len(threads) {
		t.Errorf("expected %d threads, got %d", len(threads), len(retrieved))
	}

	// Verify all threads are retrieved
	for i := range retrieved {
		if retrieved[i].AutomationID != automationID {
			t.Errorf("expected AutomationID %s, got %s", automationID, retrieved[i].AutomationID)
		}
	}
}

func TestEmailRepo_UpdateThreadStatus_Success(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	thread := domain.EmailThread{
		UserID:        "user123",
		AutomationID:  "automation456",
		GmailThreadID: "gmail_thread_789",
		TicketID:      "ticket-123",
		Status:        domain.EmailThreadStatusOpen,
		LastSyncedAt:  time.Now(),
	}

	// Save thread
	err := repo.SaveThread(ctx, &thread)
	if err != nil {
		t.Fatalf("failed to save thread: %v", err)
	}

	// Update thread status
	err = repo.UpdateThreadStatus(ctx, thread.ID, domain.EmailThreadStatusReplied)
	if err != nil {
		t.Fatalf("failed to update thread status: %v", err)
	}

	// Verify status was updated
	retrieved, err := repo.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	if retrieved.Status != domain.EmailThreadStatusReplied {
		t.Errorf("expected Status %s, got %s", domain.EmailThreadStatusReplied, retrieved.Status)
	}

	// Test updating to closed status
	err = repo.UpdateThreadStatus(ctx, thread.ID, domain.EmailThreadStatusClosed)
	if err != nil {
		t.Fatalf("failed to update thread status to closed: %v", err)
	}

	retrieved, err = repo.GetThread(ctx, thread.ID)
	if err != nil {
		t.Fatalf("failed to get thread after second update: %v", err)
	}

	if retrieved.Status != domain.EmailThreadStatusClosed {
		t.Errorf("expected Status %s, got %s", domain.EmailThreadStatusClosed, retrieved.Status)
	}
}

func TestEmailRepo_GetThreadsByAutomation_EmptySet(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()

	// Get threads for automation that doesn't exist
	threads, err := repo.GetThreadsByAutomation(ctx, "nonexistent_automation")
	if err != nil {
		t.Fatalf("failed to get threads by automation: %v", err)
	}

	if len(threads) != 0 {
		t.Errorf("expected 0 threads, got %d", len(threads))
	}
}

func TestEmailRepo_GetThread_NotFound(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()

	// Try to get non-existent thread
	_, err := repo.GetThread(ctx, "nonexistent_thread_id")
	if err == nil {
		t.Error("expected error when getting non-existent thread, got nil")
	}
}

func TestEmailRepo_GetCredential_NotFound(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()

	// Try to get non-existent credential
	_, err := repo.GetCredential(ctx, "nonexistent_user_id")
	if err == nil {
		t.Error("expected error when getting non-existent credential, got nil")
	}
}

func TestEmailRepo_GetOAuthToken_NotFound(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()

	// Try to get non-existent OAuth token
	_, err := repo.GetOAuthToken(ctx, "nonexistent_user_id", "jira")
	if err == nil {
		t.Error("expected error when getting non-existent OAuth token, got nil")
	}
}

func TestEmailRepo_GetThreadsByAutomation_InconsistentIndex(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	automationID := "automation456"

	// Create a thread
	thread := domain.EmailThread{
		UserID:        "user123",
		AutomationID:  automationID,
		GmailThreadID: "gmail_thread_1",
		TicketID:      "ticket-1",
		Status:        domain.EmailThreadStatusOpen,
		LastSyncedAt:  time.Now(),
	}

	err := repo.SaveThread(ctx, &thread)
	if err != nil {
		t.Fatalf("failed to save thread: %v", err)
	}

	// Manually corrupt the index by adding a non-existent thread ID
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	indexKey := domain.EmailThreadIndexKey(automationID)
	err = client.SAdd(ctx, indexKey, "nonexistent_thread_id").Err()
	if err != nil {
		t.Fatalf("failed to corrupt index: %v", err)
	}

	// Get threads by automation should skip the non-existent thread
	threads, err := repo.GetThreadsByAutomation(ctx, automationID)
	if err != nil {
		t.Fatalf("failed to get threads by automation: %v", err)
	}

	// Should return 1 thread (the valid one) despite corrupted index
	if len(threads) != 1 {
		t.Errorf("expected 1 thread, got %d", len(threads))
	}
}

func TestEmailRepo_SaveThread_WithExistingID(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	threadID := uuid.New().String()
	thread := domain.EmailThread{
		ID:            threadID, // Pre-existing ID
		UserID:        "user123",
		AutomationID:  "automation456",
		GmailThreadID: "gmail_thread_789",
		TicketID:      "ticket-123",
		Status:        domain.EmailThreadStatusOpen,
		LastSyncedAt:  time.Now(),
	}

	// Save thread with pre-existing ID
	err := repo.SaveThread(ctx, &thread)
	if err != nil {
		t.Fatalf("failed to save thread with existing ID: %v", err)
	}

	// Verify the ID wasn't changed
	if thread.ID != threadID {
		t.Errorf("expected ID to remain %s, got %s", threadID, thread.ID)
	}

	// Verify thread can be retrieved
	retrieved, err := repo.GetThread(ctx, threadID)
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	if retrieved.ID != threadID {
		t.Errorf("expected retrieved thread ID to be %s, got %s", threadID, retrieved.ID)
	}
}

func TestEmailRepo_UpdateThreadStatus_ThreadNotFound(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()

	// Try to update non-existent thread
	err := repo.UpdateThreadStatus(ctx, "nonexistent_thread_id", domain.EmailThreadStatusReplied)
	if err == nil {
		t.Error("expected error when updating non-existent thread, got nil")
	}
}

func TestEmailRepo_DeleteCredential_Success(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	cred := domain.EmailCredential{
		UserID:            "user123",
		EmailAddress:      "test@example.com",
		EncryptedPassword: "encrypted_password_here",
		IMAPHost:          "imap.gmail.com",
		SMTPHost:          "smtp.gmail.com",
		CreatedAt:         time.Now(),
	}

	// Save credential
	err := repo.SaveCredential(ctx, cred)
	if err != nil {
		t.Fatalf("failed to save credential: %v", err)
	}

	// Delete credential
	err = repo.DeleteCredential(ctx, "user123")
	if err != nil {
		t.Fatalf("failed to delete credential: %v", err)
	}

	// Verify credential is deleted
	_, err = repo.GetCredential(ctx, "user123")
	if err == nil {
		t.Error("expected error when getting deleted credential, got nil")
	}
}

func TestEmailRepo_DeleteOAuthToken_Success(t *testing.T) {
	s, repo := setupTestDB(t)
	defer s.Close()

	ctx := context.Background()
	expiresAt := time.Now().Add(1 * time.Hour)
	token := domain.OAuthToken{
		UserID:       "user123",
		Provider:     "jira",
		AccessToken:  "access_token",
		RefreshToken: "refresh_token",
		ExpiresAt:    expiresAt,
	}

	// Save token
	err := repo.SaveOAuthToken(ctx, token)
	if err != nil {
		t.Fatalf("failed to save OAuth token: %v", err)
	}

	// Delete token
	err = repo.DeleteOAuthToken(ctx, "user123", "jira")
	if err != nil {
		t.Fatalf("failed to delete OAuth token: %v", err)
	}

	// Verify token is deleted
	_, err = repo.GetOAuthToken(ctx, "user123", "jira")
	if err == nil {
		t.Error("expected error when getting deleted token, got nil")
	}
}
