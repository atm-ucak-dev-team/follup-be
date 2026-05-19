package dragonfly

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
	"github.com/bomanarakasura/jira-email-automation/internal/infra"
)

// OAuthTokenRepositoryImpl implements OAuthTokenRepository
type OAuthTokenRepositoryImpl struct {
	client *redis.Client
}

// NewOAuthTokenRepository creates a new OAuthTokenRepository
func NewOAuthTokenRepository(client *redis.Client) *OAuthTokenRepositoryImpl {
	return &OAuthTokenRepositoryImpl{client: client}
}

// Create stores an OAuth token with calculated TTL
func (r *OAuthTokenRepositoryImpl) Create(ctx context.Context, token *domain.OAuthToken) error {
	key := domain.OAuthKey(token.UserID, token.Provider)

	// Calculate TTL: expires_at - now - 5 minutes buffer
	ttl := time.Until(token.ExpiresAt) - 5*time.Minute
	if ttl <= 0 {
		return fmt.Errorf("token expires too soon or has already expired")
	}

	return infra.JSONSetWithTTL(ctx, r.client, key, token, ttl)
}

// GetByUserIDAndProvider retrieves an OAuth token by user ID and provider
func (r *OAuthTokenRepositoryImpl) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*domain.OAuthToken, error) {
	key := domain.OAuthKey(userID, provider)
	var token domain.OAuthToken
	err := infra.JSONGet(ctx, r.client, key, &token)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// Update updates an OAuth token
func (r *OAuthTokenRepositoryImpl) Update(ctx context.Context, token *domain.OAuthToken) error {
	return r.Create(ctx, token)
}

// Delete removes an OAuth token from DragonflyDB
func (r *OAuthTokenRepositoryImpl) Delete(ctx context.Context, userID, provider string) error {
	key := domain.OAuthKey(userID, provider)
	return infra.JSONDelete(ctx, r.client, key)
}