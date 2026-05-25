BEGIN;

ALTER TABLE followups
    ADD COLUMN IF NOT EXISTS jira_ticket_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_run_at     TIMESTAMP,
    ADD COLUMN IF NOT EXISTS created_at      TIMESTAMP NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_followups_jira_key ON followups (jira_ticket_key);

COMMIT;
