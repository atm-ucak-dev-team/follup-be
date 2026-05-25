BEGIN;

CREATE TABLE IF NOT EXISTS email_credentials (
    user_id            TEXT        NOT NULL PRIMARY KEY,
    email_address      TEXT        NOT NULL,
    encrypted_password TEXT        NOT NULL,
    imap_host          TEXT        NOT NULL,
    smtp_host          TEXT        NOT NULL,
    created_at         TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_email_credentials_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

COMMIT;
