package dragonfly

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/atm-ucak/follup/internal/domain"
	"github.com/atm-ucak/follup/internal/infra"
)

// EmailCredentialRepositoryImpl implements EmailCredentialRepository
type EmailCredentialRepositoryImpl struct {
	client *redis.Client
}

// NewEmailCredentialRepository creates a new EmailCredentialRepository
func NewEmailCredentialRepository(client *redis.Client) *EmailCredentialRepositoryImpl {
	return &EmailCredentialRepositoryImpl{client: client}
}

// Create stores an email credential in DragonflyDB
func (r *EmailCredentialRepositoryImpl) Create(ctx context.Context, cred *domain.EmailCredential) error {
	key := domain.EmailCredentialKey(cred.UserID)
	return infra.JSONSet(ctx, r.client, key, cred)
}

// GetByUserID retrieves an email credential by user ID
func (r *EmailCredentialRepositoryImpl) GetByUserID(ctx context.Context, userID string) (*domain.EmailCredential, error) {
	key := domain.EmailCredentialKey(userID)
	var cred domain.EmailCredential
	err := infra.JSONGet(ctx, r.client, key, &cred)
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

// Update updates an email credential
func (r *EmailCredentialRepositoryImpl) Update(ctx context.Context, cred *domain.EmailCredential) error {
	key := domain.EmailCredentialKey(cred.UserID)
	return infra.JSONSet(ctx, r.client, key, cred)
}

// Delete removes an email credential from DragonflyDB
func (r *EmailCredentialRepositoryImpl) Delete(ctx context.Context, userID string) error {
	key := domain.EmailCredentialKey(userID)
	return infra.JSONDelete(ctx, r.client, key)
}