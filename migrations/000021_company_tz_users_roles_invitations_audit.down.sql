DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS user_company_roles;
ALTER TABLE companies DROP COLUMN IF EXISTS owner_id;
ALTER TABLE companies DROP COLUMN IF EXISTS company_type;
ALTER TABLE companies DROP COLUMN IF EXISTS auto_approve_limit;
DROP TABLE IF EXISTS app_roles;
DROP TABLE IF EXISTS app_users;
