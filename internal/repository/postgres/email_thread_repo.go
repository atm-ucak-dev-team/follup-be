package postgres

import (
	"context"
	"fmt"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailThreadRepository struct {
	pool *pgxpool.Pool
}

func NewEmailThreadRepository(pool *pgxpool.Pool) *EmailThreadRepository {
	return &EmailThreadRepository{pool: pool}
}

func (r *EmailThreadRepository) Create(ctx context.Context, thread *domain.EmailThread) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_threads
		 (id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		thread.ID, thread.UserID, thread.AutomationID, thread.GmailThreadID,
		thread.TicketID, thread.Status, thread.LastSyncedAt,
	)
	return err
}

func (r *EmailThreadRepository) GetByID(ctx context.Context, id string) (*domain.EmailThread, error) {
	var t domain.EmailThread
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at
		 FROM email_threads WHERE id=$1`, id,
	).Scan(&t.ID, &t.UserID, &t.AutomationID, &t.GmailThreadID,
		&t.TicketID, &t.Status, &t.LastSyncedAt)
	if err != nil {
		return nil, fmt.Errorf("get email thread: %w", err)
	}
	return &t, nil
}

func (r *EmailThreadRepository) GetByAutomationID(ctx context.Context, automationID string) ([]*domain.EmailThread, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at
		 FROM email_threads WHERE automation_id=$1 ORDER BY last_synced_at DESC`,
		automationID,
	)
	if err != nil {
		return nil, fmt.Errorf("query threads: %w", err)
	}
	defer rows.Close()

	var threads []*domain.EmailThread
	for rows.Next() {
		var t domain.EmailThread
		if err := rows.Scan(&t.ID, &t.UserID, &t.AutomationID, &t.GmailThreadID,
			&t.TicketID, &t.Status, &t.LastSyncedAt); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		threads = append(threads, &t)
	}
	if threads == nil {
		threads = []*domain.EmailThread{}
	}
	return threads, nil
}

func (r *EmailThreadRepository) GetByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.EmailThread, error) {
	var t domain.EmailThread
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, automation_id, gmail_thread_id, ticket_id, status, last_synced_at
		 FROM email_threads WHERE gmail_thread_id=$1`, gmailThreadID,
	).Scan(&t.ID, &t.UserID, &t.AutomationID, &t.GmailThreadID,
		&t.TicketID, &t.Status, &t.LastSyncedAt)
	if err != nil {
		return nil, fmt.Errorf("get thread by gmail id: %w", err)
	}
	return &t, nil
}

func (r *EmailThreadRepository) Update(ctx context.Context, thread *domain.EmailThread) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_threads SET user_id=$2, automation_id=$3, gmail_thread_id=$4,
		 ticket_id=$5, status=$6, last_synced_at=$7 WHERE id=$1`,
		thread.ID, thread.UserID, thread.AutomationID, thread.GmailThreadID,
		thread.TicketID, thread.Status, thread.LastSyncedAt,
	)
	return err
}

func (r *EmailThreadRepository) UpdateThreadStatus(ctx context.Context, threadID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_threads SET status=$2, last_synced_at=NOW() WHERE id=$1`,
		threadID, status,
	)
	return err
}

func (r *EmailThreadRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_threads WHERE id=$1`, id)
	return err
}
