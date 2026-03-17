-- Chat media dedup: unique media_files by content_hash.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS media_files (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  content_hash VARCHAR(64) NOT NULL UNIQUE, -- sha256 hex
  kind VARCHAR(20) NOT NULL,               -- PHOTO | VOICE | VIDEO | VIDEO_NOTE | THUMB
  mime VARCHAR(128) NOT NULL,
  size_bytes BIGINT NOT NULL,
  path VARCHAR(1024) NOT NULL,             -- relative to storage root (or absolute; but must match nginx internal alias)
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_media_files_kind_created ON media_files (kind, created_at DESC);

-- Attachments now reference media_files for main and thumb.
ALTER TABLE chat_attachments
  ADD COLUMN IF NOT EXISTS media_file_id UUID NULL REFERENCES media_files(id) ON DELETE RESTRICT,
  ADD COLUMN IF NOT EXISTS thumb_media_file_id UUID NULL REFERENCES media_files(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_chat_attachments_media_file ON chat_attachments (media_file_id);

-- Backfill: keep old columns (path/thumb_path) as legacy; new uploads should fill media_file_id.

