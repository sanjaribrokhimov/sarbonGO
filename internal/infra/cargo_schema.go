package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureCargoTables creates cargo, route_points, payments, offers if not exist
// and adds any missing columns (idempotent). Call on app startup so tables
// are created/updated automatically without running migrate manually.
func EnsureCargoTables(ctx context.Context, pg *pgxpool.Pool) error {
	if _, err := pg.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`); err != nil {
		return err
	}

	_, err := pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS cargo (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  weight DOUBLE PRECISION NOT NULL,
  volume DOUBLE PRECISION NOT NULL,
  ready_enabled BOOLEAN NOT NULL DEFAULT false,
  ready_at TIMESTAMP NULL,
  load_comment VARCHAR NULL,
  truck_type VARCHAR NOT NULL,
  temp_min DOUBLE PRECISION NULL,
  temp_max DOUBLE PRECISION NULL,
  adr_enabled BOOLEAN NOT NULL DEFAULT false,
  adr_class VARCHAR NULL,
  loading_types TEXT[] NULL,
  requirements TEXT[] NULL,
  shipment_type VARCHAR NULL,
  belts_count INTEGER NULL,
  documents JSONB NULL,
  contact_name VARCHAR NULL,
  contact_phone VARCHAR NULL,
  status VARCHAR NOT NULL DEFAULT 'created',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP NULL
);
`)
	if err != nil {
		return err
	}

	_, err = pg.Exec(ctx, `
ALTER TABLE cargo
  ADD COLUMN IF NOT EXISTS weight DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS volume DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS ready_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS ready_at TIMESTAMP NULL,
  ADD COLUMN IF NOT EXISTS load_comment VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS truck_type VARCHAR,
  ADD COLUMN IF NOT EXISTS temp_min DOUBLE PRECISION NULL,
  ADD COLUMN IF NOT EXISTS temp_max DOUBLE PRECISION NULL,
  ADD COLUMN IF NOT EXISTS adr_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS adr_class VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS loading_types TEXT[] NULL,
  ADD COLUMN IF NOT EXISTS requirements TEXT[] NULL,
  ADD COLUMN IF NOT EXISTS shipment_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS belts_count INTEGER NULL,
  ADD COLUMN IF NOT EXISTS documents JSONB NULL,
  ADD COLUMN IF NOT EXISTS contact_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS contact_phone VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS status VARCHAR NOT NULL DEFAULT 'created',
  ADD COLUMN IF NOT EXISTS created_at TIMESTAMP NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL,
  ADD COLUMN IF NOT EXISTS created_by_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS created_by_id UUID NULL,
  ADD COLUMN IF NOT EXISTS company_id UUID NULL;
`)
	if err != nil {
		return err
	}

	_, err = pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS route_points (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL,
  type VARCHAR NOT NULL,
  address VARCHAR NOT NULL,
  lat DOUBLE PRECISION NOT NULL,
  lng DOUBLE PRECISION NOT NULL,
  comment VARCHAR NULL,
  point_order INTEGER NOT NULL,
  is_main_load BOOLEAN NOT NULL DEFAULT false,
  is_main_unload BOOLEAN NOT NULL DEFAULT false
);
`)
	if err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS cargo_id UUID;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS type VARCHAR;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS address VARCHAR;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS lat DOUBLE PRECISION;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS lng DOUBLE PRECISION;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS comment VARCHAR NULL;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS point_order INTEGER;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS is_main_load BOOLEAN NOT NULL DEFAULT false;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS is_main_unload BOOLEAN NOT NULL DEFAULT false;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS city_code VARCHAR(20) NULL;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS region_code VARCHAR(20) NULL;`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `ALTER TABLE route_points ADD COLUMN IF NOT EXISTS orientir VARCHAR(500) NULL;`); err != nil {
		return err
	}
	_, err = pg.Exec(ctx, `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'route_points_cargo_id_fkey') THEN
    ALTER TABLE route_points ADD CONSTRAINT route_points_cargo_id_fkey FOREIGN KEY (cargo_id) REFERENCES cargo(id) ON DELETE CASCADE;
  END IF;
END$$;
`)
	if err != nil {
		return err
	}

	_, err = pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL,
  is_negotiable BOOLEAN NOT NULL DEFAULT false,
  price_request BOOLEAN NOT NULL DEFAULT false,
  total_amount DOUBLE PRECISION NULL,
  total_currency VARCHAR NULL,
  with_prepayment BOOLEAN NOT NULL DEFAULT false,
  without_prepayment BOOLEAN NOT NULL DEFAULT true,
  prepayment_amount DOUBLE PRECISION NULL,
  prepayment_currency VARCHAR NULL,
  prepayment_type VARCHAR NULL,
  remaining_amount DOUBLE PRECISION NULL,
  remaining_currency VARCHAR NULL,
  remaining_type VARCHAR NULL
);
`)
	if err != nil {
		return err
	}
	_, err = pg.Exec(ctx, `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payments_cargo_id_fkey') THEN
    ALTER TABLE payments ADD CONSTRAINT payments_cargo_id_fkey FOREIGN KEY (cargo_id) REFERENCES cargo(id) ON DELETE CASCADE;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payments_cargo_id_key') THEN
    ALTER TABLE payments ADD CONSTRAINT payments_cargo_id_key UNIQUE (cargo_id);
  END IF;
END$$;
`)
	if err != nil {
		return err
	}

	_, err = pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS cities (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code VARCHAR(20) NOT NULL UNIQUE,
  name_ru VARCHAR(255) NOT NULL,
  name_en VARCHAR(255) NULL,
  country_code VARCHAR(3) NOT NULL,
  lat DOUBLE PRECISION NULL,
  lng DOUBLE PRECISION NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
`)
	if err != nil {
		return err
	}
	_, err = pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS regions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  code VARCHAR(20) NOT NULL,
  name_ru VARCHAR(255) NOT NULL,
  name_en VARCHAR(255) NULL,
  country_code VARCHAR(3) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(country_code, code)
);
`)
	if err != nil {
		return err
	}
	for _, q := range []string{
		`CREATE INDEX IF NOT EXISTS idx_cities_country ON cities (country_code)`,
		`CREATE INDEX IF NOT EXISTS idx_regions_country ON regions (country_code)`,
	} {
		if _, err := pg.Exec(ctx, q); err != nil {
			return err
		}
	}

	_, err = pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS offers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL,
  carrier_id UUID NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  currency VARCHAR NOT NULL,
  comment VARCHAR NULL,
  status VARCHAR NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
`)
	if err != nil {
		return err
	}
	_, err = pg.Exec(ctx, `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'offers_cargo_id_fkey') THEN
    ALTER TABLE offers ADD CONSTRAINT offers_cargo_id_fkey FOREIGN KEY (cargo_id) REFERENCES cargo(id) ON DELETE CASCADE;
  END IF;
END$$;
`)
	if err != nil {
		return err
	}

	_, err = pg.Exec(ctx, `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cargo_company_id') AND to_regclass('public.companies') IS NOT NULL THEN
    ALTER TABLE cargo ADD CONSTRAINT fk_cargo_company_id FOREIGN KEY (company_id) REFERENCES companies(id);
  END IF;
END$$;
`)
	if err != nil {
		return err
	}

	for _, q := range []string{
		`CREATE INDEX IF NOT EXISTS idx_cargo_status ON cargo (status)`,
		`CREATE INDEX IF NOT EXISTS idx_cargo_created_at ON cargo (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_cargo_weight ON cargo (weight)`,
		`CREATE INDEX IF NOT EXISTS idx_cargo_truck_type ON cargo (truck_type)`,
		`CREATE INDEX IF NOT EXISTS idx_cargo_deleted_at ON cargo (deleted_at) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cargo_created_by_id ON cargo (created_by_id) WHERE created_by_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_cargo_company_id ON cargo (company_id) WHERE company_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_route_points_cargo_id ON route_points (cargo_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_cargo_id ON payments (cargo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_offers_cargo_id ON offers (cargo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_offers_status ON offers (status)`,
	} {
		if _, err := pg.Exec(ctx, q); err != nil {
			return err
		}
	}

	return nil
}
