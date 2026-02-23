-- Add max limits that were requested later.
-- Keep idempotent.

ALTER TABLE companies
  ADD COLUMN IF NOT EXISTS max_vehicles INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_drivers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_top_dispatchers INTEGER NOT NULL DEFAULT 0;

