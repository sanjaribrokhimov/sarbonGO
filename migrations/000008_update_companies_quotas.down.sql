-- Revert companies quotas/limits structure (best-effort).

ALTER TABLE companies
  -- Restore old columns (if you used them previously)
  ADD COLUMN IF NOT EXISTS max_vehicles INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_drivers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_vehicles INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS current_drivers INTEGER NOT NULL DEFAULT 0;

-- Best-effort backfill
UPDATE companies
SET
  max_vehicles = GREATEST(max_vehicles, COALESCE(max_cargo, 0)),
  current_vehicles = GREATEST(current_vehicles, COALESCE(current_cargo, 0));

ALTER TABLE companies
  DROP COLUMN IF EXISTS max_cargo,
  DROP COLUMN IF EXISTS max_dispatchers,
  DROP COLUMN IF EXISTS max_managers,
  DROP COLUMN IF EXISTS max_top_dispatchers,
  DROP COLUMN IF EXISTS max_top_managers,
  DROP COLUMN IF EXISTS current_cargo,
  DROP COLUMN IF EXISTS current_dispatchers,
  DROP COLUMN IF EXISTS current_managers,
  DROP COLUMN IF EXISTS current_top_dispatchers,
  DROP COLUMN IF EXISTS current_top_managers;

