-- Хранение фото диспетчера в БД (загрузка файла вместо только ссылки).

ALTER TABLE freelance_dispatchers
  ADD COLUMN IF NOT EXISTS photo_data BYTEA NULL,
  ADD COLUMN IF NOT EXISTS photo_content_type VARCHAR(50) NULL;

ALTER TABLE deleted_freelance_dispatchers
  ADD COLUMN IF NOT EXISTS photo_data BYTEA NULL,
  ADD COLUMN IF NOT EXISTS photo_content_type VARCHAR(50) NULL;
