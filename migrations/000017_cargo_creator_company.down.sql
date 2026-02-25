ALTER TABLE cargo DROP CONSTRAINT IF EXISTS fk_cargo_company_id;
DROP INDEX IF EXISTS idx_cargo_company_id;
DROP INDEX IF EXISTS idx_cargo_created_by_id;
ALTER TABLE cargo
  DROP COLUMN IF EXISTS created_by_type,
  DROP COLUMN IF EXISTS created_by_id,
  DROP COLUMN IF EXISTS company_id;
