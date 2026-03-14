-- Cargo moderation: freelancer creates -> pending_moderation; admin accepts (searching) or rejects (rejection reason required).
-- Offer reject: dispatcher can set optional rejection_reason.
-- Cargo status: add pending_moderation, rejected, in_progress, completed.
-- Driver recommendations: dispatcher recommends cargo to driver; driver accept/decline.

-- Cargo: moderation and new statuses
ALTER TABLE cargo ADD COLUMN IF NOT EXISTS moderation_rejection_reason TEXT NULL;
-- New statuses used in app: pending_moderation, rejected, in_progress, completed (no DB enum; varchar)

-- Offers: optional rejection reason when dispatcher rejects
ALTER TABLE offers ADD COLUMN IF NOT EXISTS rejection_reason TEXT NULL;

-- Cargo driver recommendations (dispatcher recommends cargo to driver)
CREATE TABLE IF NOT EXISTS cargo_driver_recommendations (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  cargo_id UUID NOT NULL REFERENCES cargo(id) ON DELETE CASCADE,
  driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
  invited_by_dispatcher_id UUID NOT NULL REFERENCES freelance_dispatchers(id) ON DELETE CASCADE,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  UNIQUE(cargo_id, driver_id)
);
CREATE INDEX IF NOT EXISTS idx_cargo_driver_recommendations_cargo ON cargo_driver_recommendations (cargo_id);
CREATE INDEX IF NOT EXISTS idx_cargo_driver_recommendations_driver ON cargo_driver_recommendations (driver_id);
CREATE INDEX IF NOT EXISTS idx_cargo_driver_recommendations_dispatcher ON cargo_driver_recommendations (invited_by_dispatcher_id);
