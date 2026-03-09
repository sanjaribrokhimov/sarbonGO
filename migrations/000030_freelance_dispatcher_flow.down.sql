DROP TABLE IF EXISTS trips;
DROP TABLE IF EXISTS driver_invitations;
DROP TABLE IF EXISTS dispatcher_invitations;
DROP TABLE IF EXISTS dispatcher_company_roles;
ALTER TABLE companies DROP COLUMN IF EXISTS owner_dispatcher_id;
