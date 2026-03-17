-- Chat: add message types/payload and attachments.
-- Storage strategy: files are stored on server disk, metadata in DB.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Allow non-text messages: body becomes optional, and message has type/payload.
ALTER TABLE chat_messages
  ALTER COLUMN body DROP NOT NULL;

ALTER TABLE chat_messages
  ADD COLUMN IF NOT EXISTS type VARCHAR(20) NOT NULL DEFAULT 'TEXT',
  ADD COLUMN IF NOT EXISTS payload JSONB NULL;

-- Attachments metadata (file stored on disk).
CREATE TABLE IF NOT EXISTS chat_attachments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  message_id UUID NULL REFERENCES chat_messages(id) ON DELETE SET NULL,
  conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
  uploader_id UUID NOT NULL,
  kind VARCHAR(20) NOT NULL,         -- PHOTO | VOICE | VIDEO | VIDEO_NOTE
  mime VARCHAR(128) NOT NULL,
  size_bytes BIGINT NOT NULL,
  path VARCHAR(1024) NOT NULL,       -- server disk path (relative to storage root)
  thumb_path VARCHAR(1024) NULL,     -- optional thumbnail (relative)
  width INTEGER NULL,
  height INTEGER NULL,
  duration_ms INTEGER NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_attachments_conv_created ON chat_attachments (conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_attachments_message_id ON chat_attachments (message_id);

