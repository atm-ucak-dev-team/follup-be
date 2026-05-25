package postgres

import (
	"context"
	"fmt"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailCredentialRepository struct {
	pool *pgxpool.Pool
}

func NewEmailCredentialRepository(pool *pgxpool.Pool) *EmailCredentialRepository {
	return &EmailCredentialRepository{pool: pool}
}

func (r *EmailCredentialRepository) Create(ctx context.Context, cred *domain.EmailCredential) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_credentials (user_id, email_address, encrypted_password, imap_host, smtp_host, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		cred.UserID, cred.EmailAddress, cred.EncryptedPassword,
		cred.IMAPHost, cred.SMTPHost, cred.CreatedAt,
	)
	return err
}

func (r *EmailCredentialRepository) GetByUserID(ctx context.Context, userID string) (*domain.EmailCredential, error) {
	var c domain.EmailCredential
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, email_address, encrypted_password, imap_host, smtp_host, created_at
		 FROM email_credentials WHERE user_id=$1`,
		userID,
	).Scan(&c.UserID, &c.EmailAddress, &c.EncryptedPassword,
		&c.IMAPHost, &c.SMTPHost, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get email credential: %w", err)
	}
	return &c, nil
}

func (r *EmailCredentialRepository) Update(ctx context.Context, cred *domain.EmailCredential) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_credentials SET email_address=$2, encrypted_password=$3,
		 imap_host=$4, smtp_host=$5, created_at=$6
		 WHERE user_id=$1`,
		cred.UserID, cred.EmailAddress, cred.EncryptedPassword,
		cred.IMAPHost, cred.SMTPHost, cred.CreatedAt,
	)
	return err
}

func (r *EmailCredentialRepository) Delete(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_credentials WHERE user_id=$1`, userID)
	return err
}
