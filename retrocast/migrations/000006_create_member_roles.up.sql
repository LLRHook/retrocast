CREATE TABLE member_roles (
    guild_id BIGINT NOT NULL,
    user_id  BIGINT NOT NULL,
    role_id  BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (guild_id, user_id, role_id),
    FOREIGN KEY (guild_id, user_id) REFERENCES members(guild_id, user_id) ON DELETE CASCADE
);
