-- Rollback chat attachments and message type/payload.

DROP TABLE IF EXISTS chat_attachments;

ALTER TABLE chat_messages
  DROP COLUMN IF EXISTS payload,
  DROP COLUMN IF EXISTS type;

ALTER TABLE chat_messages
  ALTER COLUMN body SET NOT NULL;

