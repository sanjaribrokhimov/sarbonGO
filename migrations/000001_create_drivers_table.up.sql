-- Single-table design: all user data in one table only
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS drivers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  phone VARCHAR NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  last_online_at TIMESTAMP NULL,
  latitude DOUBLE PRECISION NULL,
  longitude DOUBLE PRECISION NULL,
  push_token VARCHAR NULL,
  registration_step VARCHAR NULL,     -- name-oferta | geo-push | transport-type
  registration_status VARCHAR NULL,   -- start | basic | full
  name VARCHAR NULL,
  driver_type VARCHAR NULL,           -- company | freelancer | driver
  rating DOUBLE PRECISION NULL,       -- max 5
  work_status VARCHAR NULL,           -- svoboden | zanyat
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
  power_owner_id VARCHAR NULL,        -- INN or PINFL (stored as string)
  power_owner_name VARCHAR NULL,      -- name or organization
  power_scan_status BOOLEAN NULL,

  trailer_plate_type VARCHAR NULL,
  trailer_plate_number VARCHAR NULL,
  trailer_tech_series VARCHAR NULL,
  trailer_tech_number VARCHAR NULL,
  trailer_owner_id VARCHAR NULL,      -- INN or PINFL (stored as string)
  trailer_owner_name VARCHAR NULL,    -- name or organization
  trailer_scan_status BOOLEAN NULL,

  driver_owner BOOLEAN NULL,
  kyc_status VARCHAR NULL
);

CREATE INDEX IF NOT EXISTS idx_drivers_phone ON drivers (phone);

