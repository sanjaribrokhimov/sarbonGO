package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureCompaniesTable makes backend self-bootstrapping for `companies`.
// It is intentionally idempotent and avoids destructive changes except dropping
// legacy quota columns that were explicitly removed from the spec.
func EnsureCompaniesTable(ctx context.Context, pg *pgxpool.Pool) error {
	// extension for uuid default
	if _, err := pg.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`); err != nil {
		return err
	}

	// Create table with the latest known schema (idempotent).
	_, err := pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS companies (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

  name VARCHAR NOT NULL,
  inn VARCHAR NULL,
  address VARCHAR NULL,
  phone VARCHAR NULL,
  email VARCHAR NULL,
  website VARCHAR NULL,
  license_number VARCHAR NULL,

  status VARCHAR NOT NULL DEFAULT 'active',

  -- Quotas and limits
  max_vehicles INTEGER NOT NULL DEFAULT 0,
  max_drivers INTEGER NOT NULL DEFAULT 0,
  max_cargo INTEGER NOT NULL DEFAULT 0,
  max_dispatchers INTEGER NOT NULL DEFAULT 0,
  max_managers INTEGER NOT NULL DEFAULT 0,
  max_top_dispatchers INTEGER NOT NULL DEFAULT 0,
  max_top_managers INTEGER NOT NULL DEFAULT 0,

  rating DOUBLE PRECISION NULL,

  completed_orders INTEGER NOT NULL DEFAULT 0,
  cancelled_orders INTEGER NOT NULL DEFAULT 0,
  total_revenue NUMERIC(18,2) NOT NULL DEFAULT 0,

  created_by UUID NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP NULL
);
`)
	if err != nil {
		return err
	}

	// Non-destructive upgrades for older schemas.
	_, err = pg.Exec(ctx, `
ALTER TABLE companies
  ADD COLUMN IF NOT EXISTS inn VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS address VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS phone VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS email VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS website VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS license_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS status VARCHAR NOT NULL DEFAULT 'active',

  ADD COLUMN IF NOT EXISTS max_vehicles INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_drivers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_cargo INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_dispatchers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_managers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_top_dispatchers INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS max_top_managers INTEGER NOT NULL DEFAULT 0,

  ADD COLUMN IF NOT EXISTS rating DOUBLE PRECISION NULL,
  ADD COLUMN IF NOT EXISTS completed_orders INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS cancelled_orders INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS total_revenue NUMERIC(18,2) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS created_by UUID NULL,
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;
`)
	if err != nil {
		return err
	}

	// Explicitly remove legacy quota columns from an earlier draft (safe and idempotent).
	if _, err := pg.Exec(ctx, `
ALTER TABLE companies
  DROP COLUMN IF EXISTS current_vehicles,
  DROP COLUMN IF EXISTS current_drivers,
  DROP COLUMN IF EXISTS current_cargo,
  DROP COLUMN IF EXISTS current_dispatchers,
  DROP COLUMN IF EXISTS current_managers,
  DROP COLUMN IF EXISTS current_top_dispatchers,
  DROP COLUMN IF EXISTS current_top_managers;
`); err != nil {
		return err
	}

	// Constraint + FK in a safe, idempotent way.
	_, err = pg.Exec(ctx, `
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'companies_status_check'
  ) THEN
    ALTER TABLE companies
      ADD CONSTRAINT companies_status_check
      CHECK (status IN ('active', 'inactive', 'blocked', 'pending'));
  END IF;

  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_created_by_users') THEN
    ALTER TABLE companies DROP CONSTRAINT fk_companies_created_by_users;
  END IF;

  IF to_regclass('public.admins') IS NOT NULL THEN
    IF NOT EXISTS (
      SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_created_by_admins'
    ) THEN
      ALTER TABLE companies
        ADD CONSTRAINT fk_companies_created_by_admins
        FOREIGN KEY (created_by) REFERENCES admins(id);
    END IF;
  END IF;
END$$;
`)
	if err != nil {
		return err
	}

	// Indexes (idempotent).
	if _, err := pg.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_companies_name ON companies (name);`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_companies_status ON companies (status);`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_companies_created_by ON companies (created_by);`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_inn_unique ON companies (inn) WHERE inn IS NOT NULL AND inn <> '';`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_license_number_unique ON companies (license_number) WHERE license_number IS NOT NULL AND license_number <> '';`); err != nil {
		return err
	}

	return nil
}

