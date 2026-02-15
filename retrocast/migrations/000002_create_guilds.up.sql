CREATE TABLE guilds (
    id         BIGINT PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    icon_hash  VARCHAR(64),
    owner_id   BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
