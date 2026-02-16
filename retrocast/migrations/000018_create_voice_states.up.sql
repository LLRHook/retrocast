CREATE TABLE voice_states (
    guild_id   BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    session_id TEXT NOT NULL,
    self_mute  BOOLEAN NOT NULL DEFAULT false,
    self_deaf  BOOLEAN NOT NULL DEFAULT false,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id)
);

CREATE INDEX idx_voice_states_channel ON voice_states(channel_id);
