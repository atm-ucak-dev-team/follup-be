BEGIN;

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id    TEXT        NOT NULL PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL,
    cloud_id   TEXT,
    avatar_url TEXT,
    created_at TIMESTAMP   NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_profiles_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

COMMIT;
