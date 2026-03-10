-- Replace capacity (tons) with volume (m3) as required field.
-- 1) Ensure volume exists, backfill NULLs, make NOT NULL.
ALTER TABLE cargo ADD COLUMN IF NOT EXISTS volume DOUBLE PRECISION;
UPDATE cargo SET volume = 0 WHERE volume IS NULL;
ALTER TABLE cargo ALTER COLUMN volume SET NOT NULL;

-- 2) Drop capacity column (no longer used).
ALTER TABLE cargo DROP COLUMN IF EXISTS capacity;

