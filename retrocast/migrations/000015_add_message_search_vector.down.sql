DROP TRIGGER IF EXISTS trg_messages_search_vector ON messages;
DROP FUNCTION IF EXISTS messages_search_vector_update();
DROP INDEX IF EXISTS idx_messages_search_vector;
ALTER TABLE messages DROP COLUMN IF EXISTS search_vector;
