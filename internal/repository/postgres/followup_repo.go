package postgres

import (
	"context"
	"fmt"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FollowupRepository struct {
	pool *pgxpool.Pool
}

func NewFollowupRepository(pool *pgxpool.Pool) *FollowupRepository {
	return &FollowupRepository{pool: pool}
}

// scanFollowup scans a row into a Followup struct.
func scanFollowup(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.Followup, error) {
	var f domain.Followup
	var cc *string
	var executionCount *int // Use pointer for NULL handling
	err := scanner.Scan(
		&f.ID, &f.JiraTicketID, &f.UserID, &f.To, &cc,
		&f.Subject, &f.EmailBody, &f.StartDateTime, &f.ExpireDateTime,
		&f.Frequency, &f.Repeat, &f.FollowupConfirmation, &f.Status,
		&executionCount, &f.JiraTicketKey, &f.LastRunAt, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.Cc = cc
	// Code-level default: NULL execution_count means 0
	if executionCount != nil {
		f.ExecutionCount = *executionCount
	} else {
		f.ExecutionCount = 0
	}
	return &f, nil
}

// scanFollowupWithTicket scans a row into a Followup struct including ticket metadata.
func scanFollowupWithTicket(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.Followup, error) {
	var f domain.Followup
	var cc *string
	var executionCount *int
	var title, stakeholder, status *string // NULLable from LEFT JOIN

	err := scanner.Scan(
		&f.ID, &f.JiraTicketID, &f.UserID, &f.To, &cc,
		&f.Subject, &f.EmailBody, &f.StartDateTime, &f.ExpireDateTime,
		&f.Frequency, &f.Repeat, &f.FollowupConfirmation, &f.Status,
		&executionCount, &f.JiraTicketKey, &f.LastRunAt, &f.CreatedAt,
		&title, &stakeholder, &status, // NEW: ticket metadata
	)
	if err != nil {
		return nil, err
	}
	f.Cc = cc
	if executionCount != nil {
		f.ExecutionCount = *executionCount
	} else {
		f.ExecutionCount = 0
	}
	// Populate ticket metadata if available
	if title != nil {
		f.JiraTicketTitle = *title
	}
	if stakeholder != nil {
		f.JiraStakeholder = *stakeholder
	}
	if status != nil {
		f.JiraTicketStatus = *status
	}
	return &f, nil
}

func (r *FollowupRepository) Create(ctx context.Context, rule *domain.Followup) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Ensure tickets entry exists for FK constraint
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tickets (jira_ticket_id, user_id, title, stakeholder, status)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (jira_ticket_id, user_id)
		 DO UPDATE SET title = EXCLUDED.title,
		              stakeholder = EXCLUDED.stakeholder,
		              status = EXCLUDED.status`,
		rule.JiraTicketID, rule.UserID,
		rule.JiraTicketTitle, rule.JiraStakeholder, rule.JiraTicketStatus,
	)
	if err != nil {
		return fmt.Errorf("ensure ticket: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO followups
		 (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
		  start_date_time, expire_date_time, frequency, repeat,
		  followup_confirmation, status, execution_count, jira_ticket_key, last_run_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		rule.ID, rule.JiraTicketID, rule.UserID, rule.To, rule.Cc,
		rule.Subject, rule.EmailBody, rule.StartDateTime, rule.ExpireDateTime,
		rule.Frequency, rule.Repeat, rule.FollowupConfirmation, rule.Status,
		rule.ExecutionCount, rule.JiraTicketKey, rule.LastRunAt, rule.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert followup: %w", err)
	}
	return nil
}

func (r *FollowupRepository) GetByID(ctx context.Context, id string) (*domain.Followup, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT f.id, f.jira_ticket_id, f.user_id, f."to", f.cc, f.subject, f.email_body,
		        f.start_date_time, f.expire_date_time, f.frequency, f.repeat,
		        f.followup_confirmation, f.status, f.execution_count,
		        f.jira_ticket_key, f.last_run_at, f.created_at,
		        t.title, t.stakeholder, t.status
		 FROM followups f
		 LEFT JOIN tickets t ON f.jira_ticket_id = t.jira_ticket_id AND f.user_id = t.user_id
		 WHERE f.id = $1`, id,
	)
	return scanFollowupWithTicket(row)
}

func (r *FollowupRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Followup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.jira_ticket_id, f.user_id, f."to", f.cc, f.subject, f.email_body,
		        f.start_date_time, f.expire_date_time, f.frequency, f.repeat,
		        f.followup_confirmation, f.status, f.execution_count,
		        f.jira_ticket_key, f.last_run_at, f.created_at,
		        t.title, t.stakeholder, t.status
		 FROM followups f
		 LEFT JOIN tickets t ON f.jira_ticket_id = t.jira_ticket_id AND f.user_id = t.user_id
		 WHERE f.user_id = $1 ORDER BY f.id`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query followups: %w", err)
	}
	defer rows.Close()

	var rules []*domain.Followup
	for rows.Next() {
		f, err := scanFollowupWithTicket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan followup: %w", err)
		}
		rules = append(rules, f)
	}
	if rules == nil {
		rules = []*domain.Followup{}
	}
	return rules, nil
}

func (r *FollowupRepository) GetActiveRules(ctx context.Context) ([]*domain.Followup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT f.id, f.jira_ticket_id, f.user_id, f."to", f.cc, f.subject, f.email_body,
		        f.start_date_time, f.expire_date_time, f.frequency, f.repeat,
		        f.followup_confirmation, f.status, f.execution_count,
		        f.jira_ticket_key, f.last_run_at, f.created_at,
		        t.title, t.stakeholder, t.status
		 FROM followups f
		 LEFT JOIN tickets t ON f.jira_ticket_id = t.jira_ticket_id AND f.user_id = t.user_id
		 WHERE f.status = 'ongoing'
		 AND f.start_date_time <= NOW()`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active followups: %w", err)
	}
	defer rows.Close()

	var rules []*domain.Followup
	for rows.Next() {
		f, err := scanFollowupWithTicket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan followup: %w", err)
		}
		rules = append(rules, f)
	}
	if rules == nil {
		rules = []*domain.Followup{}
	}
	return rules, nil
}

func (r *FollowupRepository) Update(ctx context.Context, rule *domain.Followup) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE followups SET
		 jira_ticket_id=$2, user_id=$3, "to"=$4, cc=$5, subject=$6, email_body=$7,
		 start_date_time=$8, expire_date_time=$9, frequency=$10, repeat=$11,
		 followup_confirmation=$12, status=$13, execution_count=$14, jira_ticket_key=$15, last_run_at=$16
		 WHERE id=$1`,
		rule.ID, rule.JiraTicketID, rule.UserID, rule.To, rule.Cc,
		rule.Subject, rule.EmailBody, rule.StartDateTime, rule.ExpireDateTime,
		rule.Frequency, rule.Repeat, rule.FollowupConfirmation, rule.Status,
		rule.ExecutionCount, rule.JiraTicketKey, rule.LastRunAt,
	)
	return err
}

func (r *FollowupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM followups WHERE id = $1`, id)
	return err
}
