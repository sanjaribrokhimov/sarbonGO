-- Chat: conversations and messages (universal for driver/dispatcher/admin by UUID).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS chat_conversations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_a_id UUID NOT NULL,
  user_b_id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  CONSTRAINT chat_conv_ordered CHECK (user_a_id < user_b_id),
  CONSTRAINT chat_conv_unique UNIQUE (user_a_id, user_b_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_a ON chat_conversations (user_a_id);
CREATE INDEX IF NOT EXISTS idx_chat_conversations_user_b ON chat_conversations (user_b_id);

CREATE TABLE IF NOT EXISTS chat_messages (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
  sender_id UUID NOT NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_created ON chat_messages (conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_sender ON chat_messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_deleted ON chat_messages (deleted_at) WHERE deleted_at IS NULL;
