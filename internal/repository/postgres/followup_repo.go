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

// scanFollowup scans a row into a Followup struct (excludes app-only fields).
func scanFollowup(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.Followup, error) {
	var f domain.Followup
	var cc *string
	err := scanner.Scan(
		&f.ID, &f.JiraTicketID, &f.UserID, &f.To, &cc,
		&f.Subject, &f.EmailBody, &f.StartDateTime, &f.ExpireDateTime,
		&f.Frequency, &f.Repeat, &f.FollowupConfirmation, &f.Status,
	)
	if err != nil {
		return nil, err
	}
	f.Cc = cc
	return &f, nil
}

func (r *FollowupRepository) Create(ctx context.Context, rule *domain.Followup) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Ensure tickets entry exists for FK constraint
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tickets (jira_ticket_id, user_id) VALUES ($1, $2)
		 ON CONFLICT (jira_ticket_id, user_id) DO NOTHING`,
		rule.JiraTicketID, rule.UserID,
	)
	if err != nil {
		return fmt.Errorf("ensure ticket: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO followups
		 (id, jira_ticket_id, user_id, "to", cc, subject, email_body,
		  start_date_time, expire_date_time, frequency, repeat,
		  followup_confirmation, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		rule.ID, rule.JiraTicketID, rule.UserID, rule.To, rule.Cc,
		rule.Subject, rule.EmailBody, rule.StartDateTime, rule.ExpireDateTime,
		rule.Frequency, rule.Repeat, rule.FollowupConfirmation, rule.Status,
	)
	if err != nil {
		return fmt.Errorf("insert followup: %w", err)
	}
	return nil
}

func (r *FollowupRepository) GetByID(ctx context.Context, id string) (*domain.Followup, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, jira_ticket_id, user_id, "to", cc, subject, email_body,
		        start_date_time, expire_date_time, frequency, repeat,
		        followup_confirmation, status
		 FROM followups WHERE id = $1`, id,
	)
	return scanFollowup(row)
}

func (r *FollowupRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Followup, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, jira_ticket_id, user_id, "to", cc, subject, email_body,
		        start_date_time, expire_date_time, frequency, repeat,
		        followup_confirmation, status
		 FROM followups WHERE user_id = $1 ORDER BY id`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query followups: %w", err)
	}
	defer rows.Close()

	var rules []*domain.Followup
	for rows.Next() {
		f, err := scanFollowup(rows)
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
		`SELECT id, jira_ticket_id, user_id, "to", cc, subject, email_body,
		        start_date_time, expire_date_time, frequency, repeat,
		        followup_confirmation, status
		 FROM followups WHERE status = 'ongoing'`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active followups: %w", err)
	}
	defer rows.Close()

	var rules []*domain.Followup
	for rows.Next() {
		f, err := scanFollowup(rows)
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
		 followup_confirmation=$12, status=$13
		 WHERE id=$1`,
		rule.ID, rule.JiraTicketID, rule.UserID, rule.To, rule.Cc,
		rule.Subject, rule.EmailBody, rule.StartDateTime, rule.ExpireDateTime,
		rule.Frequency, rule.Repeat, rule.FollowupConfirmation, rule.Status,
	)
	return err
}

func (r *FollowupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM followups WHERE id = $1`, id)
	return err
}
