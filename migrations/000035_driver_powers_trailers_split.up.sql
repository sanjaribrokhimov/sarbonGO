-- Variant A: split driver data into 3 tables (drivers, driver_powers, driver_trailers).
-- One row per driver in driver_powers and driver_trailers (driver can change vehicle later).
-- Old API unchanged: repo will read/write from these tables and expose same Driver shape.

CREATE TABLE IF NOT EXISTS driver_powers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  driver_id UUID NOT NULL UNIQUE REFERENCES drivers(id) ON DELETE CASCADE,
  power_plate_type VARCHAR NULL,
  power_plate_number VARCHAR NULL,
  power_tech_series VARCHAR NULL,
  power_tech_number VARCHAR NULL,
  power_owner_id VARCHAR NULL,
  power_owner_name VARCHAR NULL,
  power_scan_status BOOLEAN NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS driver_trailers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  driver_id UUID NOT NULL UNIQUE REFERENCES drivers(id) ON DELETE CASCADE,
  trailer_plate_type VARCHAR NULL,
  trailer_plate_number VARCHAR NULL,
  trailer_tech_series VARCHAR NULL,
  trailer_tech_number VARCHAR NULL,
  trailer_owner_id VARCHAR NULL,
  trailer_owner_name VARCHAR NULL,
  trailer_scan_status BOOLEAN NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_driver_powers_driver_id ON driver_powers (driver_id);
CREATE INDEX IF NOT EXISTS idx_driver_trailers_driver_id ON driver_trailers (driver_id);

-- Migrate existing data from drivers into driver_powers / driver_trailers
INSERT INTO driver_powers (driver_id, power_plate_type, power_plate_number, power_tech_series, power_tech_number, power_owner_id, power_owner_name, power_scan_status, updated_at)
SELECT id, power_plate_type, power_plate_number, power_tech_series, power_tech_number, power_owner_id, power_owner_name, power_scan_status, updated_at
FROM drivers
WHERE power_plate_type IS NOT NULL OR power_plate_number IS NOT NULL OR power_tech_series IS NOT NULL
   OR power_tech_number IS NOT NULL OR power_owner_id IS NOT NULL OR power_owner_name IS NOT NULL OR power_scan_status IS NOT NULL
ON CONFLICT (driver_id) DO NOTHING;

INSERT INTO driver_trailers (driver_id, trailer_plate_type, trailer_plate_number, trailer_tech_series, trailer_tech_number, trailer_owner_id, trailer_owner_name, trailer_scan_status, updated_at)
SELECT id, trailer_plate_type, trailer_plate_number, trailer_tech_series, trailer_tech_number, trailer_owner_id, trailer_owner_name, trailer_scan_status, updated_at
FROM drivers
WHERE trailer_plate_type IS NOT NULL OR trailer_plate_number IS NOT NULL OR trailer_tech_series IS NOT NULL
   OR trailer_tech_number IS NOT NULL OR trailer_owner_id IS NOT NULL OR trailer_owner_name IS NOT NULL OR trailer_scan_status IS NOT NULL
ON CONFLICT (driver_id) DO NOTHING;

-- Drivers with only power_plate_type/trailer_plate_type (from registration step) get a row too
INSERT INTO driver_powers (driver_id, power_plate_type, updated_at)
SELECT id, power_plate_type, updated_at FROM drivers WHERE power_plate_type IS NOT NULL
ON CONFLICT (driver_id) DO UPDATE SET power_plate_type = EXCLUDED.power_plate_type, updated_at = EXCLUDED.updated_at;

INSERT INTO driver_trailers (driver_id, trailer_plate_type, updated_at)
SELECT id, trailer_plate_type, updated_at FROM drivers WHERE trailer_plate_type IS NOT NULL
ON CONFLICT (driver_id) DO UPDATE SET trailer_plate_type = EXCLUDED.trailer_plate_type, updated_at = EXCLUDED.updated_at;

-- Drop power and trailer columns from drivers
ALTER TABLE drivers
  DROP COLUMN IF EXISTS power_plate_type,
  DROP COLUMN IF EXISTS power_plate_number,
  DROP COLUMN IF EXISTS power_tech_series,
  DROP COLUMN IF EXISTS power_tech_number,
  DROP COLUMN IF EXISTS power_owner_id,
  DROP COLUMN IF EXISTS power_owner_name,
  DROP COLUMN IF EXISTS power_scan_status,
  DROP COLUMN IF EXISTS trailer_plate_type,
  DROP COLUMN IF EXISTS trailer_plate_number,
  DROP COLUMN IF EXISTS trailer_tech_series,
  DROP COLUMN IF EXISTS trailer_tech_number,
  DROP COLUMN IF EXISTS trailer_owner_id,
  DROP COLUMN IF EXISTS trailer_owner_name,
  DROP COLUMN IF EXISTS trailer_scan_status;

-- Keep deleted_drivers in sync (same columns as drivers for INSERT ... SELECT *)
ALTER TABLE deleted_drivers
  DROP COLUMN IF EXISTS power_plate_type,
  DROP COLUMN IF EXISTS power_plate_number,
  DROP COLUMN IF EXISTS power_tech_series,
  DROP COLUMN IF EXISTS power_tech_number,
  DROP COLUMN IF EXISTS power_owner_id,
  DROP COLUMN IF EXISTS power_owner_name,
  DROP COLUMN IF EXISTS power_scan_status,
  DROP COLUMN IF EXISTS trailer_plate_type,
  DROP COLUMN IF EXISTS trailer_plate_number,
  DROP COLUMN IF EXISTS trailer_tech_series,
  DROP COLUMN IF EXISTS trailer_tech_number,
  DROP COLUMN IF EXISTS trailer_owner_id,
  DROP COLUMN IF EXISTS trailer_owner_name,
  DROP COLUMN IF EXISTS trailer_scan_status;
