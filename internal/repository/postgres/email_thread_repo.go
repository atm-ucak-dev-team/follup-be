package postgres

import (
	"context"
	"fmt"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailThreadRepository struct {
	pool *pgxpool.Pool
}

func NewEmailThreadRepository(pool *pgxpool.Pool) *EmailThreadRepository {
	return &EmailThreadRepository{pool: pool}
}

func (r *EmailThreadRepository) Create(ctx context.Context, thread *domain.EmailThread) error {
	// Parse automation ID as UUID for PostgreSQL
	automationUUID, err := uuid.Parse(thread.AutomationID)
	if err != nil {
		return fmt.Errorf("invalid automation UUID: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO email_threads
		 (id, user_id, automation_id, gmail_thread_id, ticket_id, status, body, last_synced_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		thread.ID, thread.UserID, automationUUID, thread.GmailThreadID,
		thread.TicketID, thread.Status, thread.Body, thread.LastSyncedAt,
	)
	return err
}

func (r *EmailThreadRepository) GetByID(ctx context.Context, id string) (*domain.EmailThread, error) {
	var t domain.EmailThread
	var automationUUID uuid.UUID

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, automation_id, gmail_thread_id, ticket_id, status, body, last_synced_at
		 FROM email_threads WHERE id=$1`, id,
	).Scan(&t.ID, &t.UserID, &automationUUID, &t.GmailThreadID,
		&t.TicketID, &t.Status, &t.Body, &t.LastSyncedAt)
	if err != nil {
		return nil, fmt.Errorf("get email thread: %w", err)
	}

	// Convert UUID back to string for domain model
	t.AutomationID = automationUUID.String()
	return &t, nil
}

func (r *EmailThreadRepository) GetByAutomationID(ctx context.Context, automationID string) ([]*domain.EmailThread, error) {
	// Parse automation ID as UUID for PostgreSQL query
	automationUUID, err := uuid.Parse(automationID)
	if err != nil {
		return nil, fmt.Errorf("invalid automation UUID: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, automation_id, gmail_thread_id, ticket_id, status, body, last_synced_at
		 FROM email_threads WHERE automation_id=$1 ORDER BY last_synced_at DESC`,
		automationUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("query threads: %w", err)
	}
	defer rows.Close()

	var threads []*domain.EmailThread
	for rows.Next() {
		var t domain.EmailThread
		var automationUUID uuid.UUID
		if err := rows.Scan(&t.ID, &t.UserID, &automationUUID, &t.GmailThreadID,
			&t.TicketID, &t.Status, &t.Body, &t.LastSyncedAt); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		// Convert UUID back to string for domain model
		t.AutomationID = automationUUID.String()
		threads = append(threads, &t)
	}
	if threads == nil {
		threads = []*domain.EmailThread{}
	}
	return threads, nil
}

func (r *EmailThreadRepository) GetByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.EmailThread, error) {
	var t domain.EmailThread
	var automationUUID uuid.UUID

	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, automation_id, gmail_thread_id, ticket_id, status, body, last_synced_at
		 FROM email_threads WHERE gmail_thread_id=$1`, gmailThreadID,
	).Scan(&t.ID, &t.UserID, &automationUUID, &t.GmailThreadID,
		&t.TicketID, &t.Status, &t.Body, &t.LastSyncedAt)
	if err != nil {
		return nil, fmt.Errorf("get thread by gmail id: %w", err)
	}

	// Convert UUID back to string for domain model
	t.AutomationID = automationUUID.String()
	return &t, nil
}

func (r *EmailThreadRepository) Update(ctx context.Context, thread *domain.EmailThread) error {
	// Parse automation ID as UUID for PostgreSQL
	automationUUID, err := uuid.Parse(thread.AutomationID)
	if err != nil {
		return fmt.Errorf("invalid automation UUID: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE email_threads SET user_id=$2, automation_id=$3, gmail_thread_id=$4,
		 ticket_id=$5, status=$6, body=$7, last_synced_at=$8 WHERE id=$1`,
		thread.ID, thread.UserID, automationUUID, thread.GmailThreadID,
		thread.TicketID, thread.Status, thread.Body, thread.LastSyncedAt,
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
