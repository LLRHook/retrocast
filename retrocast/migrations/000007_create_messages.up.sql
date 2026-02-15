CREATE TABLE messages (
    id         BIGINT PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id  BIGINT NOT NULL REFERENCES users(id),
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at  TIMESTAMPTZ
);

CREATE INDEX idx_messages_channel_id ON messages(channel_id, id DESC);
