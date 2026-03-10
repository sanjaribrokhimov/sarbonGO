-- Cargo, RoutePoint, Payment, Offer tables (run with migrations; also ensured on startup via infra.EnsureCargoTables).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS cargo (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  weight DOUBLE PRECISION NOT NULL CHECK (weight > 0),
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
  status VARCHAR NOT NULL DEFAULT 'created' CHECK (status IN ('created','searching','assigned','in_transit','delivered','cancelled')),
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_cargo_status ON cargo (status);
CREATE INDEX IF NOT EXISTS idx_cargo_created_at ON cargo (created_at);
CREATE INDEX IF NOT EXISTS idx_cargo_weight ON cargo (weight);
CREATE INDEX IF NOT EXISTS idx_cargo_truck_type ON cargo (truck_type);
CREATE INDEX IF NOT EXISTS idx_cargo_deleted_at ON cargo (deleted_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS route_points (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL REFERENCES cargo(id) ON DELETE CASCADE,
  type VARCHAR NOT NULL CHECK (type IN ('load','unload','customs','transit')),
  address VARCHAR NOT NULL,
  lat DOUBLE PRECISION NOT NULL,
  lng DOUBLE PRECISION NOT NULL,
  comment VARCHAR NULL,
  point_order INTEGER NOT NULL,
  is_main_load BOOLEAN NOT NULL DEFAULT false,
  is_main_unload BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_route_points_cargo_id ON route_points (cargo_id);

CREATE TABLE IF NOT EXISTS payments (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL UNIQUE REFERENCES cargo(id) ON DELETE CASCADE,
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_cargo_id ON payments (cargo_id);

CREATE TABLE IF NOT EXISTS offers (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL REFERENCES cargo(id) ON DELETE CASCADE,
  carrier_id UUID NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  currency VARCHAR NOT NULL,
  comment VARCHAR NULL,
  status VARCHAR NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected')),
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_offers_cargo_id ON offers (cargo_id);
CREATE INDEX IF NOT EXISTS idx_offers_status ON offers (status);
