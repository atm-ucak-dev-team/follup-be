BEGIN;

CREATE TABLE IF NOT EXISTS oauth_tokens (
    user_id       TEXT        NOT NULL,
    provider      TEXT        NOT NULL,
    access_token  TEXT        NOT NULL,
    refresh_token TEXT        NOT NULL,
    expires_at    TIMESTAMP   NOT NULL,
    PRIMARY KEY (user_id, provider),
    CONSTRAINT fk_oauth_tokens_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

COMMIT;
