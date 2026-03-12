-- Restore power/trailer columns on drivers (and deleted_drivers) and drop driver_powers, driver_trailers

ALTER TABLE drivers
  ADD COLUMN IF NOT EXISTS power_plate_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_plate_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_tech_series VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_tech_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_scan_status BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS trailer_plate_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_plate_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_tech_series VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_tech_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_scan_status BOOLEAN NULL;

UPDATE drivers d SET
  power_plate_type = p.power_plate_type,
  power_plate_number = p.power_plate_number,
  power_tech_series = p.power_tech_series,
  power_tech_number = p.power_tech_number,
  power_owner_id = p.power_owner_id,
  power_owner_name = p.power_owner_name,
  power_scan_status = p.power_scan_status,
  updated_at = GREATEST(d.updated_at, p.updated_at)
FROM driver_powers p WHERE p.driver_id = d.id;

UPDATE drivers d SET
  trailer_plate_type = t.trailer_plate_type,
  trailer_plate_number = t.trailer_plate_number,
  trailer_tech_series = t.trailer_tech_series,
  trailer_tech_number = t.trailer_tech_number,
  trailer_owner_id = t.trailer_owner_id,
  trailer_owner_name = t.trailer_owner_name,
  trailer_scan_status = t.trailer_scan_status,
  updated_at = GREATEST(d.updated_at, t.updated_at)
FROM driver_trailers t WHERE t.driver_id = d.id;

ALTER TABLE deleted_drivers
  ADD COLUMN IF NOT EXISTS power_plate_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_plate_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_tech_series VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_tech_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS power_scan_status BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS trailer_plate_type VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_plate_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_tech_series VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_tech_number VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_id VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_owner_name VARCHAR NULL,
  ADD COLUMN IF NOT EXISTS trailer_scan_status BOOLEAN NULL;

DROP TABLE IF EXISTS driver_trailers;
DROP TABLE IF EXISTS driver_powers;
