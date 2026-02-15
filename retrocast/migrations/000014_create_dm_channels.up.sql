CREATE TABLE dm_channels (
    id         BIGINT PRIMARY KEY,
    type       INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE dm_recipients (
    channel_id BIGINT NOT NULL REFERENCES dm_channels(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_dm_recipients_user ON dm_recipients(user_id);
