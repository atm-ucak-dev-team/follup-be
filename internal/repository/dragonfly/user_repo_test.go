package dragonfly

import (
	"context"
	"testing"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/infra"
	"github.com/redis/go-redis/v9"
)

const (
	testUserRepoDragonflyAddr = "localhost:6379"
	testUserRepoDragonflyDB   = 15 // Use separate DB for testing
)

// setupTestClient creates a test DragonflyDB client
func setupTestClient(t *testing.T) *redis.Client {
	cfg := &domain.Config{
		DragonflyAddr:     testUserRepoDragonflyAddr,
		DragonflyPassword: "",
		DragonflyDB:       testUserRepoDragonflyDB,
	}

	client := infra.NewDragonflyClient(cfg)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := infra.Ping(ctx, client); err != nil {
		t.Logf("DragonflyDB not available for testing: %v", err)
		t.Skip("Skipping test - DragonflyDB not available")
	}

	return client
}

// cleanupTestUser cleans up a test user from the database
func cleanupTestUser(t *testing.T, ctx context.Context, repo *UserRepository, user *domain.User) {
	if user != nil && user.ID != "" {
		// Try to delete the user (ignore errors during cleanup)
		_ = repo.Delete(ctx, user.ID)
	}
}

func TestUserRepo_CreateAndGet_Success(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test user
	testUser := &domain.User{
		Name:  "Test User",
		Email: "test@example.com",
	}

	// Clean up on test completion
	defer cleanupTestUser(t, ctx, repo, testUser)

	// Create user
	if err := repo.Create(ctx, testUser); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify ID was generated
	if testUser.ID == "" {
		t.Error("Expected ID to be generated")
	}

	// Verify CreatedAt was set
	if testUser.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	// Get user by ID
	retrieved, err := repo.GetByID(ctx, testUser.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	// Verify user data
	if retrieved.Name != testUser.Name {
		t.Errorf("Expected Name %s, got %s", testUser.Name, retrieved.Name)
	}
	if retrieved.Email != testUser.Email {
		t.Errorf("Expected Email %s, got %s", testUser.Email, retrieved.Email)
	}
	if retrieved.ID != testUser.ID {
		t.Errorf("Expected ID %s, got %s", testUser.ID, retrieved.ID)
	}
}

func TestUserRepo_GetByEmail_Success(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test user
	testUser := &domain.User{
		Name:  "Email Test User",
		Email: "emailtest@example.com",
	}

	// Clean up on test completion
	defer cleanupTestUser(t, ctx, repo, testUser)

	// Create user
	if err := repo.Create(ctx, testUser); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get user by email
	retrieved, err := repo.GetByEmail(ctx, testUser.Email)
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	// Verify user data
	if retrieved.Name != testUser.Name {
		t.Errorf("Expected Name %s, got %s", testUser.Name, retrieved.Name)
	}
	if retrieved.Email != testUser.Email {
		t.Errorf("Expected Email %s, got %s", testUser.Email, retrieved.Email)
	}
	if retrieved.ID != testUser.ID {
		t.Errorf("Expected ID %s, got %s", testUser.ID, retrieved.ID)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get non-existent user
	_, err := repo.GetByID(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error when getting non-existent user, got nil")
	}
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get user with non-existent email
	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	if err == nil {
		t.Error("Expected error when getting user with non-existent email, got nil")
	}
}

func TestUserRepo_Update_Success(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test user
	testUser := &domain.User{
		Name:  "Original Name",
		Email: "original@example.com",
	}

	// Clean up on test completion
	defer cleanupTestUser(t, ctx, repo, testUser)

	// Create user
	if err := repo.Create(ctx, testUser); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update user
	testUser.Name = "Updated Name"
	if err := repo.Update(ctx, testUser); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetByID(ctx, testUser.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Expected Name 'Updated Name', got %s", retrieved.Name)
	}
}

func TestUserRepo_Update_EmailChange(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test user
	originalEmail := "original@example.com"
	testUser := &domain.User{
		Name:  "Email Change User",
		Email: originalEmail,
	}

	// Clean up on test completion
	defer cleanupTestUser(t, ctx, repo, testUser)

	// Create user
	if err := repo.Create(ctx, testUser); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update email
	newEmail := "newemail@example.com"
	testUser.Email = newEmail
	if err := repo.Update(ctx, testUser); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify old email doesn't work
	_, err := repo.GetByEmail(ctx, originalEmail)
	if err == nil {
		t.Error("Expected error when getting user by old email, got nil")
	}

	// Verify new email works
	retrieved, err := repo.GetByEmail(ctx, newEmail)
	if err != nil {
		t.Fatalf("GetByEmail with new email failed: %v", err)
	}

	if retrieved.Email != newEmail {
		t.Errorf("Expected Email %s, got %s", newEmail, retrieved.Email)
	}
}

func TestUserRepo_Update_NotFound(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to update non-existent user
	nonExistentUser := &domain.User{
		ID:    "nonexistent-id",
		Name:  "Nonexistent User",
		Email: "nonexistent@example.com",
	}

	err := repo.Update(ctx, nonExistentUser)
	if err == nil {
		t.Error("Expected error when updating non-existent user, got nil")
	}
}

func TestUserRepo_Delete_Success(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create test user
	testUser := &domain.User{
		Name:  "Delete Test User",
		Email: "deletetest@example.com",
	}

	// Create user
	if err := repo.Create(ctx, testUser); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete user
	if err := repo.Delete(ctx, testUser.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify user is gone
	_, err := repo.GetByID(ctx, testUser.ID)
	if err == nil {
		t.Error("Expected error when getting deleted user, got nil")
	}

	// Verify email index is gone
	_, err = repo.GetByEmail(ctx, testUser.Email)
	if err == nil {
		t.Error("Expected error when getting deleted user by email, got nil")
	}
}

func TestUserRepo_Delete_NotFound(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to delete non-existent user
	err := repo.Delete(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error when deleting non-existent user, got nil")
	}
}

func TestUserRepo_Create_IDPreservation(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create user with predefined ID
	predefinedID := "test-predefined-id"
	testUser := &domain.User{
		ID:    predefinedID,
		Name:  "ID Preservation Test",
		Email: "idpreservation@example.com",
	}

	// Clean up on test completion
	defer cleanupTestUser(t, ctx, repo, testUser)

	// Create user
	if err := repo.Create(ctx, testUser); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify ID was preserved
	if testUser.ID != predefinedID {
		t.Errorf("Expected ID to be preserved as %s, got %s", predefinedID, testUser.ID)
	}

	// Verify user can be retrieved with that ID
	retrieved, err := repo.GetByID(ctx, predefinedID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved.ID != predefinedID {
		t.Errorf("Expected retrieved ID %s, got %s", predefinedID, retrieved.ID)
	}
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	client := setupTestClient(t)
	defer infra.Close(client)

	repo := NewUserRepository(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create first user
	firstUser := &domain.User{
		Name:  "First User",
		Email: "duplicate@example.com",
	}

	// Create second user with same email
	secondUser := &domain.User{
		Name:  "Second User",
		Email: "duplicate@example.com", // Same email
	}

	// Clean up both users
	defer func() {
		cleanupTestUser(t, ctx, repo, firstUser)
		cleanupTestUser(t, ctx, repo, secondUser)
	}()

	// Create first user
	if err := repo.Create(ctx, firstUser); err != nil {
		t.Fatalf("Create first user failed: %v", err)
	}

	// Create second user (should overwrite email index)
	if err := repo.Create(ctx, secondUser); err != nil {
		t.Fatalf("Create second user failed: %v", err)
	}

	// Verify email lookup returns second user
	retrieved, err := repo.GetByEmail(ctx, "duplicate@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}

	if retrieved.ID != secondUser.ID {
		t.Errorf("Expected email lookup to return second user ID %s, got %s", secondUser.ID, retrieved.ID)
	}
}