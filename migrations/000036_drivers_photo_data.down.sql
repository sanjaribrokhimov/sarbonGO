ALTER TABLE drivers DROP COLUMN IF EXISTS photo_data, DROP COLUMN IF EXISTS photo_content_type;
ALTER TABLE deleted_drivers DROP COLUMN IF EXISTS photo_data, DROP COLUMN IF EXISTS photo_content_type;
