-- Drop title column from cargo (no longer used in API).
ALTER TABLE cargo DROP COLUMN IF EXISTS title;

