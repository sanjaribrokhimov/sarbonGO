ALTER TABLE cargo
  DROP CONSTRAINT IF EXISTS fk_cargo_cargo_type,
  DROP COLUMN IF EXISTS cargo_type_id,
  DROP COLUMN IF EXISTS name;

DROP TABLE IF EXISTS cargo_types;

