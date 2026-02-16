CREATE TABLE reactions (
    message_id BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    emoji      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message ON reactions(message_id);
