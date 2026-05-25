BEGIN;

ALTER TABLE followups
    DROP COLUMN IF EXISTS jira_ticket_key,
    DROP COLUMN IF EXISTS last_run_at,
    DROP COLUMN IF EXISTS created_at;

COMMIT;
