CREATE TABLE channels (
    id        BIGINT PRIMARY KEY,
    guild_id  BIGINT NOT NULL REFERENCES guilds(id) ON DELETE CASCADE,
    name      VARCHAR(100) NOT NULL,
    type      SMALLINT NOT NULL DEFAULT 0,
    position  INT NOT NULL DEFAULT 0,
    topic     VARCHAR(1024),
    parent_id BIGINT REFERENCES channels(id),
    UNIQUE(guild_id, name)
);
