-- Align KYC fields to simplified contract and keep driver's name separately.
DO $$
BEGIN
  -- rename owner_name -> name (driver's name)
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='drivers' AND column_name='owner_name'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name='drivers' AND column_name='name'
  ) THEN
    EXECUTE 'ALTER TABLE drivers RENAME COLUMN owner_name TO name';
  END IF;
END $$;

ALTER TABLE drivers
  ADD COLUMN IF NOT EXISTS power_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS driver_owner BOOLEAN NULL;

-- Drop deprecated columns (were replaced by *_owner_id/name)
ALTER TABLE drivers
  DROP COLUMN IF EXISTS power_owner_type,
  DROP COLUMN IF EXISTS power_owner_id_type,
  DROP COLUMN IF EXISTS power_owner_id_number,
  DROP COLUMN IF EXISTS trailer_owner_type,
  DROP COLUMN IF EXISTS trailer_owner_id_type,
  DROP COLUMN IF EXISTS trailer_owner_id_number,
  DROP COLUMN IF EXISTS owner_inn_pinfl;

