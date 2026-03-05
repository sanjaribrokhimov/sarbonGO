-- Переименовать app_users в company_users, убрать email, добавить company_id и role.

ALTER TABLE app_users RENAME TO company_users;

DROP INDEX IF EXISTS idx_app_users_email;
ALTER TABLE company_users DROP COLUMN IF EXISTS email;

ALTER TABLE company_users ADD COLUMN IF NOT EXISTS company_id UUID NULL;
ALTER TABLE company_users ADD COLUMN IF NOT EXISTS role VARCHAR(50) NULL;

CREATE INDEX IF NOT EXISTS idx_company_users_phone ON company_users (phone) WHERE phone IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_company_users_company_id ON company_users (company_id) WHERE company_id IS NOT NULL;

-- FK company_id -> companies (добавить после того как таблица companies есть)
DO $$
BEGIN
  IF to_regclass('public.companies') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_company_users_company') THEN
      ALTER TABLE company_users
        ADD CONSTRAINT fk_company_users_company
        FOREIGN KEY (company_id) REFERENCES companies(id);
    END IF;
  END IF;
END$$;

-- GoAdmin menu: обновить пункт App Users на Company Users
UPDATE goadmin_menu SET uri = '/info/company_users', title = 'Company Users' WHERE uri = '/info/app_users';
