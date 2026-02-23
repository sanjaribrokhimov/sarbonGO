-- Create companies table (if not exists).
-- Keep it idempotent to support repeated `migrate up` in dev/stage.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS companies (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  name VARCHAR NOT NULL,
  inn VARCHAR NULL,
  address VARCHAR NULL,
  phone VARCHAR NULL,
  email VARCHAR NULL,
  website VARCHAR NULL,
  license_number VARCHAR NULL,

  status VARCHAR NOT NULL DEFAULT 'pending',

  -- Quotas and limits
  max_vehicles INTEGER NOT NULL DEFAULT 0,
  max_drivers INTEGER NOT NULL DEFAULT 0,
  current_vehicles INTEGER NOT NULL DEFAULT 0,
  current_drivers INTEGER NOT NULL DEFAULT 0,

  rating DOUBLE PRECISION NULL,

  completed_orders INTEGER NOT NULL DEFAULT 0,
  cancelled_orders INTEGER NOT NULL DEFAULT 0,
  total_revenue NUMERIC(18,2) NOT NULL DEFAULT 0,

  created_by UUID NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP NULL,

  CONSTRAINT companies_status_check CHECK (status IN ('active', 'inactive', 'blocked', 'pending'))
);

CREATE INDEX IF NOT EXISTS idx_companies_name ON companies (name);
CREATE INDEX IF NOT EXISTS idx_companies_status ON companies (status);
CREATE INDEX IF NOT EXISTS idx_companies_created_by ON companies (created_by);

-- Uniqueness (optional fields) — partial unique indexes.
CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_inn_unique ON companies (inn)
  WHERE inn IS NOT NULL AND inn <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_license_number_unique ON companies (license_number)
  WHERE license_number IS NOT NULL AND license_number <> '';

-- Optional FK to users(id) if/when users table exists.
DO $$
BEGIN
  IF to_regclass('public.users') IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1
      FROM pg_constraint
      WHERE conname = 'fk_companies_created_by_users'
    ) THEN
      ALTER TABLE companies
        ADD CONSTRAINT fk_companies_created_by_users
        FOREIGN KEY (created_by) REFERENCES users(id);
    END IF;
  END IF;
END$$;

