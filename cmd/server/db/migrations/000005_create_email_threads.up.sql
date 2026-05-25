BEGIN;

CREATE TABLE IF NOT EXISTS email_threads (
    id              UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id         TEXT        NOT NULL,
    automation_id   UUID        NOT NULL,
    gmail_thread_id TEXT        NOT NULL,
    ticket_id       TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'open',
    last_synced_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_email_threads_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_email_threads_user ON email_threads (user_id);
CREATE INDEX IF NOT EXISTS idx_email_threads_automation ON email_threads (automation_id);
CREATE INDEX IF NOT EXISTS idx_email_threads_gmail ON email_threads (gmail_thread_id);

COMMIT;
