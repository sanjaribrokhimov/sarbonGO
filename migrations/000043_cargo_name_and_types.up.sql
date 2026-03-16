CREATE TABLE IF NOT EXISTS cargo_types (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code VARCHAR(128) NOT NULL UNIQUE,
  name_ru VARCHAR(255) NOT NULL,
  name_uz VARCHAR(255) NOT NULL,
  name_en VARCHAR(255) NOT NULL,
  name_tr VARCHAR(255) NOT NULL,
  name_zh VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

ALTER TABLE cargo
  ADD COLUMN IF NOT EXISTS name VARCHAR(255),
  ADD COLUMN IF NOT EXISTS cargo_type_id UUID;

ALTER TABLE cargo
  ADD CONSTRAINT fk_cargo_cargo_type
  FOREIGN KEY (cargo_type_id) REFERENCES cargo_types(id);

