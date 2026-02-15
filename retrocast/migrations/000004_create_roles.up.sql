CREATE TABLE roles (
    id          BIGINT PRIMARY KEY,
    guild_id    BIGINT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    color       INT NOT NULL DEFAULT 0,
    permissions BIGINT NOT NULL DEFAULT 0,
    position    INT NOT NULL DEFAULT 0,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE
);
