-- 000001_create_tables.up.sql
-- Creates: users, tickets, followups

BEGIN;

-- ============================================
-- USERS
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id              TEXT NOT NULL,
    jira_account_id TEXT NOT NULL,

    PRIMARY KEY (id, jira_account_id)
);

-- users.id must be unique so tickets can FK reference it
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_id ON users (id);

-- ============================================
-- TICKETS
-- ============================================
CREATE TABLE IF NOT EXISTS tickets (
    jira_ticket_id TEXT NOT NULL,
    user_id        TEXT NOT NULL,

    PRIMARY KEY (jira_ticket_id, user_id),

    CONSTRAINT fk_tickets_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

-- ============================================
-- FOLLOWUPS
-- ============================================
CREATE TABLE IF NOT EXISTS followups (
    id                     UUID        NOT NULL DEFAULT gen_random_uuid(),
    jira_ticket_id         TEXT        NOT NULL,
    user_id                TEXT        NOT NULL,
    "to"                   TEXT        NOT NULL,
    cc                     TEXT,
    subject                TEXT        NOT NULL,
    email_body             TEXT        NOT NULL,
    start_date_time        TIMESTAMP   NOT NULL,
    expire_date_time       TIMESTAMP   NOT NULL,
    frequency              TEXT        NOT NULL,
    repeat                 INTEGER     NOT NULL DEFAULT 0,
    followup_confirmation  BOOLEAN     NOT NULL DEFAULT FALSE,
    status                 TEXT        NOT NULL DEFAULT 'ongoing',

    PRIMARY KEY (id),

    CONSTRAINT fk_followups_ticket
        FOREIGN KEY (jira_ticket_id, user_id)
        REFERENCES tickets (jira_ticket_id, user_id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_followups_ticket ON followups (jira_ticket_id, user_id);
CREATE INDEX IF NOT EXISTS idx_followups_status ON followups (status);

COMMIT;