CREATE TABLE channel_overrides (
    channel_id  BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    role_id     BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    allow_perms BIGINT NOT NULL DEFAULT 0,
    deny_perms  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (channel_id, role_id)
);
