package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureDriversTable makes backend self-bootstrapping:
// - if table `drivers` doesn't exist -> creates it
// - if table exists but misses some columns -> adds them (non-destructive)
// - ensures `deleted_drivers` exists for hard-delete archiving
func EnsureDriversTable(ctx context.Context, pg *pgxpool.Pool) error {
	// extension for uuid default
	if _, err := pg.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`); err != nil {
		return err
	}

	// Create table with the latest known schema (idempotent).
	// Note: ALTER TABLE below is still needed for older existing DBs.
	_, err := pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS drivers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  phone VARCHAR NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  last_online_at TIMESTAMP NULL,
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  push_token VARCHAR NULL,
  registration_step VARCHAR NULL,
  registration_status VARCHAR NULL,
  name VARCHAR NULL,
  driver_type VARCHAR NULL,
  rating DOUBLE PRECISION NULL,
  work_status VARCHAR NULL,
  freelancer_id UUID NULL,
  company_id UUID NULL,
  account_status VARCHAR NULL,

  driver_passport_series VARCHAR NULL,
  driver_passport_number VARCHAR NULL,
  driver_pinfl VARCHAR NULL,
  driver_scan_status BOOLEAN NULL,

  power_plate_type VARCHAR NULL,
  power_plate_number VARCHAR NULL,
  power_tech_series VARCHAR NULL,
  power_tech_number VARCHAR NULL,
  power_owner_id VARCHAR NULL,
  power_owner_name VARCHAR NULL,
  power_scan_status BOOLEAN NULL,

  trailer_plate_type VARCHAR NULL,
  trailer_plate_number VARCHAR NULL,
  trailer_tech_series VARCHAR NULL,
  trailer_tech_number VARCHAR NULL,
  trailer_owner_id VARCHAR NULL,
  trailer_owner_name VARCHAR NULL,
  trailer_scan_status BOOLEAN NULL,

  driver_owner BOOLEAN NULL,
  kyc_status VARCHAR NULL
);
`)
	if err != nil {
		return err
	}

	// Non-destructive upgrades for older schemas.
	// (No DROP COLUMN here on purpose.)
	_, err = pg.Exec(ctx, `
ALTER TABLE drivers
  ADD COLUMN IF NOT EXISTS push_token VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_plate_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_plate_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS driver_owner BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS driver_scan_status BOOLEAN NULL;
`)
	if err != nil {
		return err
	}

	// Index (idempotent)
	if _, err = pg.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_drivers_phone ON drivers (phone);`); err != nil {
		return err
	}

	// Archive table for hard deletes (clone structure without constraints).
	// We intentionally do NOT include constraints/indexes to avoid UNIQUE conflicts on phone.
	_, err = pg.Exec(ctx, `CREATE TABLE IF NOT EXISTS deleted_drivers (LIKE drivers INCLUDING DEFAULTS);`)
	if err != nil {
		return err
	}
	_, err = pg.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_deleted_drivers_phone ON deleted_drivers (phone);`)
	return err
}

