BEGIN;
LOCK TABLE channels, dm_channels, messages, attachments IN ACCESS EXCLUSIVE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM messages m WHERE NOT EXISTS
        (SELECT 1 FROM channels c WHERE c.id = m.channel_id)) THEN
        RAISE EXCEPTION 'cannot downgrade: remove DM messages before migration 000020 down';
    END IF;
    IF EXISTS (SELECT 1 FROM attachments WHERE message_id IS NULL) THEN
        RAISE EXCEPTION 'cannot downgrade: link or remove pending attachments before migration 000020 down';
    END IF;
END;
$$;
DROP TRIGGER messages_check_channel ON messages;
DROP FUNCTION check_message_channel();
DROP TRIGGER channels_delete_messages ON channels;
DROP TRIGGER dm_channels_delete_messages ON dm_channels;
DROP FUNCTION delete_channel_messages();
ALTER TABLE messages ADD CONSTRAINT messages_channel_id_fkey
    FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE;
DROP INDEX idx_attachments_message;
ALTER TABLE attachments DROP COLUMN uploader_id, DROP COLUMN channel_id,
    ALTER COLUMN message_id SET NOT NULL;
COMMIT;
