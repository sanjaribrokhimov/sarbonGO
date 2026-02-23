-- Create admins table (minimal).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS admins (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  login VARCHAR NOT NULL UNIQUE,
  password VARCHAR NOT NULL,
  name VARCHAR NOT NULL,
  status VARCHAR NOT NULL DEFAULT 'active',
  type VARCHAR NOT NULL DEFAULT 'creator',
  CONSTRAINT admins_status_check CHECK (status IN ('active', 'inactive', 'blocked'))
);

CREATE INDEX IF NOT EXISTS idx_admins_login ON admins (login);

