package dragonfly

import (
	"context"
	"fmt"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/infra"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// UserRepository implements repository.UserRepository interface
type UserRepository struct {
	client *redis.Client
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(client *redis.Client) *UserRepository {
	return &UserRepository{
		client: client,
	}
}

// Create stores a new user in DragonflyDB with email indexing
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	// Generate UUID if ID is empty
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	// Set CreatedAt if not set
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	// Store the user
	userKey := domain.UserKey(user.ID)
	if err := infra.JSONSet(ctx, r.client, userKey, user); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	// Create email index
	emailIndexKey := domain.UserEmailIndexKey(user.Email)
	if err := r.client.Set(ctx, emailIndexKey, user.ID, 0).Err(); err != nil {
		return fmt.Errorf("failed to create email index: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	key := domain.UserKey(id)
	var user domain.User
	err := infra.JSONGet(ctx, r.client, key, &user)
	if err != nil {
		return nil, err // Return error as-is (includes "not found" message)
	}
	return &user, nil
}

// GetByEmail retrieves a user by email address using the email index
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Get user ID from email index
	emailIndexKey := domain.UserEmailIndexKey(email)
	userID, err := r.client.Get(ctx, emailIndexKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user with email '%s' not found", email)
		}
		return nil, fmt.Errorf("failed to get email index: %w", err)
	}

	// Get user by ID
	return r.GetByID(ctx, userID)
}

// Update updates an existing user in DragonflyDB
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	// Check if user exists
	existingUser, err := r.GetByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update email index if email changed
	if existingUser.Email != user.Email {
		// Remove old email index
		oldEmailIndexKey := domain.UserEmailIndexKey(existingUser.Email)
		if err := r.client.Del(ctx, oldEmailIndexKey).Err(); err != nil {
			return fmt.Errorf("failed to remove old email index: %w", err)
		}

		// Create new email index
		newEmailIndexKey := domain.UserEmailIndexKey(user.Email)
		if err := r.client.Set(ctx, newEmailIndexKey, user.ID, 0).Err(); err != nil {
			return fmt.Errorf("failed to create new email index: %w", err)
		}
	}

	// Update user record
	userKey := domain.UserKey(user.ID)
	if err := infra.JSONSet(ctx, r.client, userKey, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// Delete removes a user from DragonflyDB including the email index
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	// Get user first to clean up email index
	user, err := r.GetByID(ctx, id)
	if err != nil {
		return err // Return error as-is (includes "not found" message)
	}

	// Remove email index
	emailIndexKey := domain.UserEmailIndexKey(user.Email)
	if err := r.client.Del(ctx, emailIndexKey).Err(); err != nil {
		return fmt.Errorf("failed to remove email index: %w", err)
	}

	// Remove user record
	userKey := domain.UserKey(id)
	if err := infra.JSONDelete(ctx, r.client, userKey); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
