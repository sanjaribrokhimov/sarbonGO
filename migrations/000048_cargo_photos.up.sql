-- Cargo photos: store photo files on disk, metadata in DB.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS cargo_photos (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL REFERENCES cargo(id) ON DELETE CASCADE,
  uploader_id UUID NULL,
  mime VARCHAR(128) NOT NULL,
  size_bytes BIGINT NOT NULL,
  path VARCHAR(1024) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cargo_photos_cargo_created ON cargo_photos (cargo_id, created_at DESC);

