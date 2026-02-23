-- Update companies quotas/limits structure.
-- Keep idempotent to be safe across environments.

ALTER TABLE companies
  -- New quotas
  ADD COLUMN IF NOT EXISTS max_cargo INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_dispatchers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_managers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_top_dispatchers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_top_managers INTEGER NOT NULL DEFAULT 0,

  -- New current counters
  ADD COLUMN IF NOT EXISTS current_cargo INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_dispatchers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_managers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_top_dispatchers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_top_managers INTEGER NOT NULL DEFAULT 0;

-- Backfill from old columns if they exist (best-effort).
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'companies' AND column_name = 'max_vehicles'
  ) THEN
    EXECUTE 'UPDATE companies SET max_cargo = GREATEST(max_cargo, COALESCE(max_vehicles, 0))';
  END IF;
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'companies' AND column_name = 'current_vehicles'
  ) THEN
    EXECUTE 'UPDATE companies SET current_cargo = GREATEST(current_cargo, COALESCE(current_vehicles, 0))';
  END IF;
END$$;

-- Remove obsolete columns if present.
ALTER TABLE companies
  DROP COLUMN IF EXISTS current_vehicles,
  DROP COLUMN IF EXISTS current_drivers,
  DROP COLUMN IF EXISTS max_vehicles,
  DROP COLUMN IF EXISTS max_drivers;

