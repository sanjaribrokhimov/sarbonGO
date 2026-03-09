-- Freelance dispatcher can invite drivers without company (driver works with dispatcher via freelancer_id).

ALTER TABLE driver_invitations ALTER COLUMN company_id DROP NOT NULL;
ALTER TABLE driver_invitations ADD COLUMN IF NOT EXISTS invited_by_dispatcher_id UUID NULL;

DO $$
BEGIN
  IF to_regclass('public.freelance_dispatchers') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_driver_invitations_dispatcher') THEN
      ALTER TABLE driver_invitations
        ADD CONSTRAINT fk_driver_invitations_dispatcher
        FOREIGN KEY (invited_by_dispatcher_id) REFERENCES freelance_dispatchers(id);
    END IF;
  END IF;
END$$;

CREATE INDEX IF NOT EXISTS idx_driver_invitations_dispatcher ON driver_invitations (invited_by_dispatcher_id) WHERE invited_by_dispatcher_id IS NOT NULL;

-- Ensure exactly one of company_id or invited_by_dispatcher_id is set (for new rows)
-- We don't add CHECK for backward compat with existing company-only rows; application enforces.
