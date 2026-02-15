CREATE TABLE users (
    id            BIGINT PRIMARY KEY,
    username      VARCHAR(32) NOT NULL UNIQUE,
    display_name  VARCHAR(32) NOT NULL,
    avatar_hash   VARCHAR(64),
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);
