ALTER TABLE company_users DROP CONSTRAINT IF EXISTS fk_company_users_company;
DROP INDEX IF EXISTS idx_company_users_company_id;
DROP INDEX IF EXISTS idx_company_users_phone;

ALTER TABLE company_users DROP COLUMN IF EXISTS role;
ALTER TABLE company_users DROP COLUMN IF EXISTS company_id;

ALTER TABLE company_users ADD COLUMN IF NOT EXISTS email VARCHAR(255) UNIQUE;
-- restore data not possible; leave nullable
ALTER TABLE company_users RENAME TO app_users;
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_users_email ON app_users (email) WHERE email IS NOT NULL;
