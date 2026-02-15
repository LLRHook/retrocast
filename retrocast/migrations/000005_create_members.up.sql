CREATE TABLE members (
    guild_id  BIGINT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nickname  VARCHAR(32),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id)
);
