package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSONSerialization(t *testing.T) {
	user := User{
		ID:        "user-123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal User: %v", err)
	}

	var unmarshaled User
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal User: %v", err)
	}

	if unmarshaled.ID != user.ID {
		t.Errorf("Expected ID %s, got %s", user.ID, unmarshaled.ID)
	}
}

func TestOAuthTokenJSONSerialization(t *testing.T) {
	token := OAuthToken{
		UserID:       "user-123",
		Provider:     "jira",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}

	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Failed to marshal OAuthToken: %v", err)
	}

	var unmarshaled OAuthToken
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal OAuthToken: %v", err)
	}

	if unmarshaled.UserID != token.UserID {
		t.Errorf("Expected UserID %s, got %s", token.UserID, unmarshaled.UserID)
	}
}

func TestEmailCredentialJSONSerialization(t *testing.T) {
	cred := EmailCredential{
		UserID:            "user-123",
		EmailAddress:      "user@example.com",
		EncryptedPassword: "encrypted-password",
		IMAPHost:          "imap.example.com",
		SMTPHost:          "smtp.example.com",
		CreatedAt:         time.Now(),
	}

	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("Failed to marshal EmailCredential: %v", err)
	}

	var unmarshaled EmailCredential
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal EmailCredential: %v", err)
	}

	if unmarshaled.UserID != cred.UserID {
		t.Errorf("Expected UserID %s, got %s", cred.UserID, unmarshaled.UserID)
	}
}

func TestFollowupJSONSerialization(t *testing.T) {
	now := time.Now()
	cc := "cc@example.com"
	rule := Followup{
		ID:            "automation-123",
		UserID:        "user-123",
		JiraTicketID:  "ticket-123",
		JiraTicketKey: "PROJ-123",
		To:            "to@example.com",
		Cc:            &cc,
		Subject:       "Follow-up: PROJ-123",
		EmailBody:     "Please review this ticket.",
		Frequency:     "0 9 * * 1",
		Status:        FollowupStatusOngoing,
		LastRunAt:     &now,
		CreatedAt:     time.Now(),
	}

	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("Failed to marshal Followup: %v", err)
	}

	var unmarshaled Followup
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Followup: %v", err)
	}

	if unmarshaled.ID != rule.ID {
		t.Errorf("Expected ID %s, got %s", rule.ID, unmarshaled.ID)
	}

	if unmarshaled.Status != FollowupStatusOngoing {
		t.Errorf("Expected status %s, got %s", FollowupStatusOngoing, unmarshaled.Status)
	}
}

func TestEmailThreadJSONSerialization(t *testing.T) {
	thread := EmailThread{
		ID:            "thread-123",
		UserID:        "user-123",
		AutomationID:  "automation-123",
		GmailThreadID: "gmail-thread-123",
		TicketID:      "ticket-123",
		Status:        EmailThreadStatusOpen,
		LastSyncedAt:  time.Now(),
	}

	data, err := json.Marshal(thread)
	if err != nil {
		t.Fatalf("Failed to marshal EmailThread: %v", err)
	}

	var unmarshaled EmailThread
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal EmailThread: %v", err)
	}

	if unmarshaled.ID != thread.ID {
		t.Errorf("Expected ID %s, got %s", thread.ID, unmarshaled.ID)
	}

	if unmarshaled.Status != EmailThreadStatusOpen {
		t.Errorf("Expected status %s, got %s", EmailThreadStatusOpen, unmarshaled.Status)
	}
}

func TestStorageKeyGenerators(t *testing.T) {
	tests := []struct {
		name     string
		generate func() string
		expected string
	}{
		{"UserKey", func() string { return UserKey("user-123") }, "user:user-123"},
		{"OAuthKey", func() string { return OAuthKey("user-123", "jira") }, "oauth:user-123:jira"},
		{"EmailCredentialKey", func() string { return EmailCredentialKey("user-123") }, "email_credential:user-123"},
		{"AutomationKey", func() string { return AutomationKey("auto-123") }, "automation:auto-123"},
		{"AutomationIndexKey", func() string { return AutomationIndexKey("user-123") }, "automation:index:user-123"},
		{"EmailThreadKey", func() string { return EmailThreadKey("thread-123") }, "email_thread:thread-123"},
		{"EmailThreadIndexKey", func() string { return EmailThreadIndexKey("auto-123") }, "email_thread:index:auto-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.generate()
			if result != tt.expected {
				t.Errorf("Expected key %s, got %s", tt.expected, result)
			}
		})
	}
}
