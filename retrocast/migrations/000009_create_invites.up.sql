CREATE TABLE invites (
    code       VARCHAR(8) PRIMARY KEY,
    guild_id   BIGINT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    channel_id BIGINT REFERENCES channels(id),
    creator_id BIGINT NOT NULL REFERENCES users(id),
    max_uses   INT NOT NULL DEFAULT 0,
    uses       INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
