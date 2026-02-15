CREATE TABLE attachments (
    id           BIGINT PRIMARY KEY,
    message_id   BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename     VARCHAR(256) NOT NULL,
    content_type VARCHAR(128) NOT NULL,
    size         BIGINT NOT NULL,
    storage_key  TEXT NOT NULL
);
