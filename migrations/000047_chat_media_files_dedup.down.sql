ALTER TABLE chat_attachments
  DROP COLUMN IF EXISTS thumb_media_file_id,
  DROP COLUMN IF EXISTS media_file_id;

DROP TABLE IF EXISTS media_files;

