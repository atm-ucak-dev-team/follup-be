package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		return fmt.Errorf("user ID is required")
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO users (id, jira_account_id) VALUES ($1, $2)
		 ON CONFLICT (id, jira_account_id) DO NOTHING`,
		user.ID, user.JiraAccountID,
	); err != nil {
		return fmt.Errorf("insert users: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO user_profiles (user_id, name, email, cloud_id, avatar_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id) DO UPDATE SET name=$2, email=$3, cloud_id=$4, avatar_url=$5, created_at=$6`,
		user.ID, user.Name, user.Email, user.CloudID, user.AvatarURL, user.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert user_profiles: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.jira_account_id, COALESCE(p.name,''), COALESCE(p.email,''),
		        COALESCE(p.cloud_id,''), COALESCE(p.avatar_url,''), COALESCE(p.created_at, NOW())
		 FROM users u
		 LEFT JOIN user_profiles p ON p.user_id = u.id
		 WHERE u.id = $1`,
		id,
	).Scan(&user.ID, &user.JiraAccountID, &user.Name, &user.Email,
		&user.CloudID, &user.AvatarURL, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.jira_account_id, p.name, p.email,
		        COALESCE(p.cloud_id,''), COALESCE(p.avatar_url,''), p.created_at
		 FROM users u
		 JOIN user_profiles p ON p.user_id = u.id
		 WHERE p.email = $1`,
		email,
	).Scan(&user.ID, &user.JiraAccountID, &user.Name, &user.Email,
		&user.CloudID, &user.AvatarURL, &user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`INSERT INTO users (id, jira_account_id) VALUES ($1, $2)
		 ON CONFLICT (id, jira_account_id) DO UPDATE SET jira_account_id=$2`,
		user.ID, user.JiraAccountID,
	); err != nil {
		return fmt.Errorf("update users: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO user_profiles (user_id, name, email, cloud_id, avatar_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id) DO UPDATE SET name=$2, email=$3, cloud_id=$4, avatar_url=$5`,
		user.ID, user.Name, user.Email, user.CloudID, user.AvatarURL, user.CreatedAt,
	); err != nil {
		return fmt.Errorf("update user_profiles: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
