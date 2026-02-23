-- Remove current counters and extra columns that were added by mistake.
-- Keep idempotent.

ALTER TABLE companies
  DROP COLUMN IF EXISTS current_cargo,
  DROP COLUMN IF EXISTS current_dispatchers,
  DROP COLUMN IF EXISTS current_managers,
  DROP COLUMN IF EXISTS current_top_dispatchers,
  DROP COLUMN IF EXISTS current_top_managers;

