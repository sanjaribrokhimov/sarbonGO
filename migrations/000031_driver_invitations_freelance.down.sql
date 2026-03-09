ALTER TABLE driver_invitations DROP COLUMN IF EXISTS invited_by_dispatcher_id;
ALTER TABLE driver_invitations ALTER COLUMN company_id SET NOT NULL;
