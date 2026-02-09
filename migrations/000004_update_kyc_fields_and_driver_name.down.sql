-- Best-effort rollback: re-add dropped columns (data can't be restored), and rename name -> owner_name.
ALTER TABLE drivers
  ADD COLUMN IF NOT EXISTS power_owner_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_id_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_id_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_id_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_id_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS owner_inn_pinfl VARCHAR NULL;

ALTER TABLE drivers
  DROP COLUMN IF EXISTS power_owner_id,
  DROP COLUMN IF EXISTS power_owner_name,
  DROP COLUMN IF EXISTS trailer_owner_id,
  DROP COLUMN IF EXISTS trailer_owner_name,
  DROP COLUMN IF EXISTS driver_owner;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='drivers' AND column_name='name'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='drivers' AND column_name='owner_name'
  ) THEN
    EXECUTE 'ALTER TABLE drivers RENAME COLUMN name TO owner_name';
  END IF;
END $$;

