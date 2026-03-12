-- Фото водителя в БД (необязательно при регистрации; можно добавить/обновить/удалить когда угодно).

ALTER TABLE drivers
  ADD COLUMN IF NOT EXISTS photo_data BYTEA NULL,
  ADD COLUMN IF NOT EXISTS photo_content_type VARCHAR(50) NULL;

ALTER TABLE deleted_drivers
  ADD COLUMN IF NOT EXISTS photo_data BYTEA NULL,
  ADD COLUMN IF NOT EXISTS photo_content_type VARCHAR(50) NULL;
