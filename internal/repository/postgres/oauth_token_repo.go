package postgres

import (
	"context"
	"fmt"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthTokenRepository struct {
	pool *pgxpool.Pool
}

func NewOAuthTokenRepository(pool *pgxpool.Pool) *OAuthTokenRepository {
	return &OAuthTokenRepository{pool: pool}
}

func (r *OAuthTokenRepository) Create(ctx context.Context, token *domain.OAuthToken) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO oauth_tokens (user_id, provider, access_token, refresh_token, expires_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		token.UserID, token.Provider, token.AccessToken, token.RefreshToken, token.ExpiresAt,
	)
	return err
}

func (r *OAuthTokenRepository) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*domain.OAuthToken, error) {
	var t domain.OAuthToken
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, provider, access_token, refresh_token, expires_at
		 FROM oauth_tokens WHERE user_id=$1 AND provider=$2`,
		userID, provider,
	).Scan(&t.UserID, &t.Provider, &t.AccessToken, &t.RefreshToken, &t.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("get oauth token: %w", err)
	}
	return &t, nil
}

func (r *OAuthTokenRepository) Update(ctx context.Context, token *domain.OAuthToken) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE oauth_tokens SET access_token=$3, refresh_token=$4, expires_at=$5
		 WHERE user_id=$1 AND provider=$2`,
		token.UserID, token.Provider, token.AccessToken, token.RefreshToken, token.ExpiresAt,
	)
	return err
}

func (r *OAuthTokenRepository) Delete(ctx context.Context, userID, provider string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM oauth_tokens WHERE user_id=$1 AND provider=$2`,
		userID, provider,
	)
	return err
}
