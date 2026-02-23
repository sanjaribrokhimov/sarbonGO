-- Make companies.created_by reference admins(id).
-- Also drop old users FK if it was ever created.

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_created_by_users') THEN
    ALTER TABLE companies DROP CONSTRAINT fk_companies_created_by_users;
  END IF;

  IF to_regclass('public.admins') IS NOT NULL THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_companies_created_by_admins') THEN
      ALTER TABLE companies
        ADD CONSTRAINT fk_companies_created_by_admins
        FOREIGN KEY (created_by) REFERENCES admins(id);
    END IF;
  END IF;
END$$;

