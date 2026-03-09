-- Freelance dispatcher flow: dispatcher-owned companies, dispatcher_company_roles, invitations, trips, driver_invitations.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1) companies: optional owner_dispatcher_id (freelance_dispatchers.id)
ALTER TABLE companies ADD COLUMN IF NOT EXISTS owner_dispatcher_id UUID NULL;
DO $$
BEGIN
  IF to_regclass('public.freelance_dispatchers') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_owner_dispatcher') THEN
      ALTER TABLE companies ADD CONSTRAINT fk_companies_owner_dispatcher
        FOREIGN KEY (owner_dispatcher_id) REFERENCES freelance_dispatchers(id);
    END IF;
  END IF;
END$$;
CREATE INDEX IF NOT EXISTS idx_companies_owner_dispatcher_id ON companies (owner_dispatcher_id) WHERE owner_dispatcher_id IS NOT NULL;

-- 2) dispatcher_company_roles: dispatcher linked to company (owner or invited as dispatcher/top_dispatcher)
CREATE TABLE IF NOT EXISTS dispatcher_company_roles (
  dispatcher_id UUID NOT NULL,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  role VARCHAR(50) NOT NULL,
  invited_by UUID NULL,
  accepted_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  PRIMARY KEY (dispatcher_id, company_id)
);
CREATE INDEX IF NOT EXISTS idx_dispatcher_company_roles_dispatcher ON dispatcher_company_roles (dispatcher_id);
CREATE INDEX IF NOT EXISTS idx_dispatcher_company_roles_company ON dispatcher_company_roles (company_id);
DO $$
BEGIN
  IF to_regclass('public.freelance_dispatchers') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_dcr_dispatcher') THEN
      ALTER TABLE dispatcher_company_roles ADD CONSTRAINT fk_dcr_dispatcher
        FOREIGN KEY (dispatcher_id) REFERENCES freelance_dispatchers(id) ON DELETE CASCADE;
    END IF;
  END IF;
END$$;

-- 3) dispatcher_invitations: company invites dispatcher by phone
CREATE TABLE IF NOT EXISTS dispatcher_invitations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  token VARCHAR(64) UNIQUE NOT NULL,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  role VARCHAR(50) NOT NULL,
  phone VARCHAR(20) NOT NULL,
  invited_by UUID NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_dispatcher_invitations_token ON dispatcher_invitations (token);
CREATE INDEX IF NOT EXISTS idx_dispatcher_invitations_phone ON dispatcher_invitations (phone);
CREATE INDEX IF NOT EXISTS idx_dispatcher_invitations_company ON dispatcher_invitations (company_id);

-- 4) driver_invitations: company (dispatcher) invites driver by phone
CREATE TABLE IF NOT EXISTS driver_invitations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  token VARCHAR(64) UNIQUE NOT NULL,
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  phone VARCHAR(20) NOT NULL,
  invited_by UUID NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_driver_invitations_token ON driver_invitations (token);
CREATE INDEX IF NOT EXISTS idx_driver_invitations_phone ON driver_invitations (phone);
CREATE INDEX IF NOT EXISTS idx_driver_invitations_company ON driver_invitations (company_id);

-- 5) trips: рейс после принятия ставки; статусы pending_driver -> assigned -> loading -> en_route -> unloading -> completed
CREATE TABLE IF NOT EXISTS trips (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL REFERENCES cargo(id) ON DELETE CASCADE,
  offer_id UUID NOT NULL,
  driver_id UUID NULL,
  status VARCHAR(50) NOT NULL DEFAULT 'pending_driver',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_trips_cargo_id ON trips (cargo_id);
CREATE INDEX IF NOT EXISTS idx_trips_offer_id ON trips (offer_id);
CREATE INDEX IF NOT EXISTS idx_trips_driver_id ON trips (driver_id) WHERE driver_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips (status);
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_trips_offer') THEN
    ALTER TABLE trips ADD CONSTRAINT fk_trips_offer FOREIGN KEY (offer_id) REFERENCES offers(id);
  END IF;
  IF to_regclass('public.drivers') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_trips_driver') THEN
      ALTER TABLE trips ADD CONSTRAINT fk_trips_driver FOREIGN KEY (driver_id) REFERENCES drivers(id);
    END IF;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'trips_status_check') THEN
    ALTER TABLE trips ADD CONSTRAINT trips_status_check CHECK (status IN (
      'pending_driver', 'assigned', 'loading', 'en_route', 'unloading', 'completed', 'cancelled'
    ));
  END IF;
END$$;
