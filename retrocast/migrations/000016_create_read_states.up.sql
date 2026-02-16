CREATE TABLE read_states (
    user_id    BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    last_message_id BIGINT NOT NULL DEFAULT 0,
    mention_count   INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, channel_id)
);
CREATE INDEX idx_read_states_user ON read_states(user_id);
