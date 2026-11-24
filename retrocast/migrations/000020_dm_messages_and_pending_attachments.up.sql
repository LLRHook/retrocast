BEGIN;

ALTER TABLE messages DROP CONSTRAINT messages_channel_id_fkey;

CREATE FUNCTION check_message_channel() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM id FROM channels WHERE id = NEW.channel_id FOR KEY SHARE;
    IF NOT FOUND THEN
        PERFORM id FROM dm_channels WHERE id = NEW.channel_id FOR KEY SHARE;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'message channel % does not exist', NEW.channel_id
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER messages_check_channel
    BEFORE INSERT OR UPDATE OF channel_id ON messages
    FOR EACH ROW EXECUTE FUNCTION check_message_channel();

CREATE FUNCTION delete_channel_messages() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM messages WHERE channel_id = OLD.id;
    RETURN OLD;
END;
$$;
CREATE TRIGGER channels_delete_messages AFTER DELETE ON channels
    FOR EACH ROW EXECUTE FUNCTION delete_channel_messages();
CREATE TRIGGER dm_channels_delete_messages AFTER DELETE ON dm_channels
    FOR EACH ROW EXECUTE FUNCTION delete_channel_messages();

ALTER TABLE attachments ALTER COLUMN message_id DROP NOT NULL;
ALTER TABLE attachments ADD COLUMN uploader_id BIGINT REFERENCES users(id),
    ADD COLUMN channel_id BIGINT REFERENCES channels(id) ON DELETE CASCADE;
UPDATE attachments a SET uploader_id = m.author_id, channel_id = m.channel_id
    FROM messages m WHERE m.id = a.message_id;
ALTER TABLE attachments ALTER COLUMN uploader_id SET NOT NULL,
    ALTER COLUMN channel_id SET NOT NULL;
CREATE INDEX idx_attachments_message ON attachments(message_id);
COMMIT;
