-- Create freelance_dispatchers and deleted_freelance_dispatchers tables (if not exists).
-- uuid-ossp extension is created in 000001, but keep table creation idempotent.

CREATE TABLE IF NOT EXISTS freelance_dispatchers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR NOT NULL,
  phone VARCHAR NOT NULL UNIQUE,
  password VARCHAR NOT NULL,
  passport_series VARCHAR NULL,
  passport_number VARCHAR NULL,
  pinfl VARCHAR NULL,
  cargo_id UUID NULL,
  driver_id UUID NULL,
  rating DOUBLE PRECISION NULL,
  work_status VARCHAR NULL,
  account_status VARCHAR NULL,
  photo_path VARCHAR NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_freelance_dispatchers_phone ON freelance_dispatchers (phone);
CREATE INDEX IF NOT EXISTS idx_freelance_dispatchers_pinfl ON freelance_dispatchers (pinfl);
CREATE INDEX IF NOT EXISTS idx_freelance_dispatchers_cargo_id ON freelance_dispatchers (cargo_id);
CREATE INDEX IF NOT EXISTS idx_freelance_dispatchers_driver_id ON freelance_dispatchers (driver_id);

CREATE TABLE IF NOT EXISTS deleted_freelance_dispatchers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name VARCHAR NULL,
  phone VARCHAR NULL,
  password VARCHAR NULL,
  passport_series VARCHAR NULL,
  passport_number VARCHAR NULL,
  pinfl VARCHAR NULL,
  cargo_id UUID NULL,
  driver_id UUID NULL,
  rating DOUBLE PRECISION NULL,
  work_status VARCHAR NULL,
  account_status VARCHAR NULL,
  photo_path VARCHAR NULL,
  created_at TIMESTAMP NULL,
  updated_at TIMESTAMP NULL,
  deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_deleted_freelance_dispatchers_phone ON deleted_freelance_dispatchers (phone);
CREATE INDEX IF NOT EXISTS idx_deleted_freelance_dispatchers_pinfl ON deleted_freelance_dispatchers (pinfl);
CREATE INDEX IF NOT EXISTS idx_deleted_freelance_dispatchers_cargo_id ON deleted_freelance_dispatchers (cargo_id);
CREATE INDEX IF NOT EXISTS idx_deleted_freelance_dispatchers_driver_id ON deleted_freelance_dispatchers (driver_id);

