-- Add account_status and photo_path if missing (e.g. table created with older schema).

ALTER TABLE freelance_dispatchers
  ADD COLUMN IF NOT EXISTS account_status VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS photo_path VARCHAR NULL;

ALTER TABLE deleted_freelance_dispatchers
  ADD COLUMN IF NOT EXISTS account_status VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS photo_path VARCHAR NULL;
